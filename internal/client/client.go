package client

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/HarrisonWAffel/terminal-chat/internal"
	"github.com/HarrisonWAffel/terminal-chat/internal/pion"
	"github.com/google/uuid"
	"github.com/pion/webrtc/v3"
	"github.com/pkg/errors"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

type Client struct {
	AppCtx *internal.AppCtx
	Key    string
	*webrtc.PeerConnection
	ICECandidateChan chan webrtc.SessionDescription
	GuiInfo          *GUI
}

type HostClient interface {
	HostNewConversation(appCtx *internal.AppCtx, connConfig ...ConnectionConfig)
}

type ReceivingClient interface {
	ConnectToConversationId(appCtx *internal.AppCtx, conversationId string)
}

type Host struct {
	*Client
}

type Receiver struct {
	*Client
}

var urls = []string{
	"stun:stun.l.google.com:19302",
	"stun:stun.l.google.com:19302",
	"stun:stun1.l.google.com:19302",
	"stun:stun2.l.google.com:19302",
	"stun:stun3.l.google.com:19302",
	"stun:stun4.l.google.com:19302",
	"stun:stun.ekiga.net",
	"stun:stun.ideasip.com",
	"stun:stun.rixtelecom.se",
	"stun:stun.schlund.de",
	"stun:stun.stunprotocol.org:3478",
	"stun:stun.voiparound.com",
	"stun:stun.voipbuster.com",
	"stun:stun.voipstunt.com",
	"stun:stun.voxgratia.org",
}

var Config = webrtc.Configuration{
	ICEServers: []webrtc.ICEServer{
		{
			URLs: urls,
		},
	},
}

func NewReceiverClient(appCtx *internal.AppCtx) ReceivingClient {
	pc, err := webrtc.NewPeerConnection(Config)
	if err != nil {
		panic(err)
	}
	c := &Client{
		AppCtx:           appCtx,
		PeerConnection:   pc,
		ICECandidateChan: make(chan webrtc.SessionDescription, 1),
		GuiInfo: &GUI{
			InputChan:   make(chan string),
			OutputChan:  make(chan string),
			NetworkChan: make(chan string),
			Username:    appCtx.ScreenName,
			PeerConn:    pc,
		},
	}

	// Set the handler for Peer connection state
	// This will notify you when the peer has connected/disconnected
	c.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
		if s == webrtc.PeerConnectionStateFailed {
			c.GuiInfo.OutputChan <- "\n\n***************\nPeer has disconnected. Please exit the application\n***************\n"
		}
		if s == webrtc.PeerConnectionStateDisconnected {
			c.GuiInfo.OutputChan <- "\n\n***************\nPeer has disconnected. Either wait for reconnection or exit application\n***************\n"
		}
	})

	// Register data channel creation handling
	c.OnDataChannel(func(d *webrtc.DataChannel) {
		fmt.Printf("DataChannel %s-%d open\n\n", d.Label(), d.ID())
		if c.Key == "" {
			setKey(c, d)
		}

		// Register channel opening handling
		d.OnOpen(func() {
			go func() {
				for {
					select {
					case msg := <-c.GuiInfo.NetworkChan:
						d.SendText(encryptText(msg, c.Key))
					}
				}
			}()
			err := c.GuiInfo.StartGUI()
			if err != nil {
				os.Exit(1)
			}
		})

		// Register text message handling
		d.OnMessage(func(msg webrtc.DataChannelMessage) {
			c.GuiInfo.OutputChan <- decryptText(string(msg.Data), c.Key)
		})
	})

	return &Receiver{Client: c}
}

func NewOfferClient(appCtx *internal.AppCtx) HostClient {
	pc, err := webrtc.NewPeerConnection(Config)
	if err != nil {
		panic(err)
	}
	c := &Client{
		AppCtx:           appCtx,
		PeerConnection:   pc,
		ICECandidateChan: make(chan webrtc.SessionDescription, 1),
		GuiInfo: &GUI{
			InputChan:   make(chan string),
			OutputChan:  make(chan string),
			NetworkChan: make(chan string),
			Username:    appCtx.ScreenName,
			PeerConn:    pc,
		},
	}

	d, err := c.CreateDataChannel("conversation", nil)
	if err != nil {
		panic(err)
	}

	d.OnClose(func() {
		fmt.Println("data channel has closed")
	})

	c.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
		if s == webrtc.PeerConnectionStateFailed {
			c.GuiInfo.OutputChan <- "\n\n***************\nPeer has disconnected. Please exit the application\n***************\n"
		}
		if s == webrtc.PeerConnectionStateDisconnected {
			c.GuiInfo.OutputChan <- "\n\n***************\nPeer has disconnected. Either wait for reconnection or exit application\n***************\n"
		}
	})

	d.OnOpen(func() {
		if c.Key == "" {
			setKey(c, d)
		}
		go func() {
			for {
				select {
				case msg := <-c.GuiInfo.NetworkChan:
					d.SendText(encryptText(msg, c.Key))
				}
			}
		}()
		err := c.GuiInfo.StartGUI()
		if err != nil {
			os.Exit(1)
		}
	})

	d.OnMessage(func(msg webrtc.DataChannelMessage) {
		c.GuiInfo.OutputChan <- decryptText(string(msg.Data), c.Key)
	})

	return &Host{Client: c}
}

type ConnectionConfig struct {
	CustomToken string
}

func (c *Receiver) ConnectToConversationId(appCtx *internal.AppCtx, conversationId string) {
	// get the remote host connection info for the given token
	req, _ := http.NewRequest(http.MethodGet, appCtx.ServerURL+"/get", nil)
	req.Header.Set("conn-token", conversationId)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	if resp.StatusCode != http.StatusOK {
		panic("bad status received from server, check conversation ID")
	}

	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	offer := webrtc.SessionDescription{}
	err = json.Unmarshal(b, &offer)
	if err != nil {
		panic(errors.Wrap(err, "error parsing response from server, fatal"))
	}

	err = c.SetRemoteDescription(offer)
	if err != nil {
		panic(errors.Wrap(err, "set remote offer error, fatal"))
	}

	answer, err := c.CreateAnswer(nil)
	if err != nil {
		panic(errors.Wrap(err, "create answer error"))
	}

	err = c.SetLocalDescription(answer)
	if err != nil {
		panic(errors.Wrap(err, "cannot set local description"))
	}

	<-webrtc.GatheringCompletePromise(c.PeerConnection)

	connInfo := pion.Encode(c.LocalDescription())
	req, _ = http.NewRequest(http.MethodPost, appCtx.ServerURL+"/join", bytes.NewReader([]byte(connInfo)))
	req.Header.Set("conn-token", conversationId)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		panic(errors.Wrap(err, "could not join conversation"))
	}

	select {}
}

func (c *Host) HostNewConversation(appCtx *internal.AppCtx, connConfig ...ConnectionConfig) {
	connectionName := ""
	if len(connConfig) == 0 {
		fmt.Print("Would you like to use a custom connection name? (y/n): ")
		reader := bufio.NewReader(os.Stdin)
		text := ""
		for {
			t, _ := reader.ReadString('\n')
			text = strings.ReplaceAll(strings.ToLower(t), "\n", "")
			if text == "y" || text == "yes" || text == "n" || text == "no" {
				break
			}
			fmt.Print("Please enter yes or no (y/n): ")
		}
		switch text {
		case "y", "yes":
			fmt.Print("Please enter the custom connection name now: ")
			connectionName, _ = reader.ReadString('\n')
			connectionName = strings.ReplaceAll(connectionName, "\n", "")
		default:
			connectionName = uuid.New().String()
			break
		}
	} else {
		connectionName = connConfig[0].CustomToken
	}

	offer, err := c.CreateOffer(nil)
	if err != nil {
		panic(err)
	}

	if err := c.SetLocalDescription(offer); err != nil {
		panic(err)
	}
	fmt.Println("Gathering ICE Candidates before continuing...")
	c.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			go func() { c.ICECandidateChan <- *c.LocalDescription() }()
		}
	})

	// Client will be available when the first valid ICE
	// candidate is found and added to the LocalDescription
	var desc webrtc.SessionDescription
	select {
	case m := <-c.ICECandidateChan:
		desc = m
	}
	fmt.Println("Done gathering!")

	req, _ := http.NewRequest(http.MethodPost, appCtx.ServerURL+"/host", bytes.NewReader([]byte(pion.Encode(desc))))
	if connectionName != "" {
		req.Header.Set("req-conn-id", connectionName)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		panic("status " + resp.Status + " != 200 OK")
	}

	fmt.Println("connection name: '" + connectionName + "' was accepted by the server, it will be valid for 10 minutes, waiting for connection from peer.\n\n")

	info := ""
	for {
		b, err := ioutil.ReadAll(resp.Body)
		if err == nil {
			if len(b) > 0 {
				info = string(b)
				break
			}
		}
	}

	if info == "token timed out" {
		fmt.Println("connection failed: token timed out before peer connected")
		return
	}

	descr := webrtc.SessionDescription{}
	pion.Decode(info, &descr)
	err = c.SetRemoteDescription(descr)
	if err != nil {
		fmt.Println("error setting remote description")
		panic(err)
	}

	select {}
}
