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
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"
)

type Client struct {
	*webrtc.PeerConnection
	c chan webrtc.SessionDescription
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

func NewReceiverClient(name string, isTest bool) *Client {
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: urls,
			},
		},
	}

	pc, err := webrtc.NewPeerConnection(config)
	if err != nil {
		panic(err)
	}
	c := &Client{
		PeerConnection: pc,
		c:              make(chan webrtc.SessionDescription, 1),
	}

	// Set the handler for Peer connection state
	// This will notify you when the peer has connected/disconnected
	c.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
		fmt.Printf("Peer Connection State has changed: %s\n", s.String())

		if s == webrtc.PeerConnectionStateFailed {
			// Wait until PeerConnection has had no network activity for 30 seconds or another failure. It may be reconnected using an ICE Restart.
			// Use webrtc.PeerConnectionStateDisconnected if you are interested in detecting faster timeout.
			// Note that the PeerConnection may come back from PeerConnectionStateDisconnected.
			fmt.Println("Peer Connection has gone to failed, exiting")
			os.Exit(0)
		}
	})

	// Register data channel creation handling
	c.OnDataChannel(func(d *webrtc.DataChannel) {
		fmt.Printf("New DataChannel %s %d\n", d.Label(), d.ID())

		// Register channel opening handling
		d.OnOpen(func() {
			if isTest {
				fmt.Printf("Data channel '%s'-'%d' open. Random messages will now be sent to any connected DataChannels every 5 seconds\n", d.Label(), d.ID())
				for range time.NewTicker(5 * time.Second).C {
					message := "hello world"
					fmt.Printf("Sending '%s'\n", message)

					// Send the message as text
					sendErr := d.SendText(message)
					if sendErr != nil {
						panic(sendErr)
					}
				}
			} else {
				HandleInput(name, d)
			}
		})

		// Register text message handling
		d.OnMessage(func(msg webrtc.DataChannelMessage) {
			RenderOutput(msg, name)
		})
	})

	return c
}

func NewOfferClient(name string, isTest bool) *Client {
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: urls,
			},
		},
	}

	pc, err := webrtc.NewPeerConnection(config)
	if err != nil {
		panic(err)
	}
	c := &Client{
		PeerConnection: pc,
		c:              make(chan webrtc.SessionDescription, 1),
	}

	sendChannel, err := c.CreateDataChannel("conversation", nil)
	if err != nil {
		panic(err)
	}

	sendChannel.OnClose(func() {
		fmt.Println("conversation data channel has closed")
	})

	sendChannel.OnOpen(func() {
		fmt.Println("conversation data channel opened")
		if !isTest {
			HandleInput(name, sendChannel)
		}
	})

	sendChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
		RenderOutput(msg, name)
	})

	return c
}

type ConnectionConfig struct {
	CustomToken string
}

func ConnectToConversationId(appCtx *internal.AppCtx, conversationId string) {
	c := NewReceiverClient(appCtx.ScreenName, appCtx.IsTest)

	// get the remote host connection info for the given token
	req, _ := http.NewRequest(http.MethodGet, appCtx.ServerURL+"/get", nil)
	req.Header.Set("conn-token", conversationId)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	offer := webrtc.SessionDescription{}
	err = json.Unmarshal(b, &offer)
	if err != nil {
		fmt.Println("json error")
		fmt.Println(string(b))
		panic(err)
	}

	err = c.SetRemoteDescription(offer)
	if err != nil {
		fmt.Println("set remote offer error")
		panic(err)
	}

	answer, err := c.CreateAnswer(nil)
	if err != nil {
		fmt.Println("create answer error")
		panic(err)
	}

	err = c.SetLocalDescription(answer)
	if err != nil {
		fmt.Println("cannot set local description")
		panic(err)
	}

	<-webrtc.GatheringCompletePromise(c.PeerConnection)

	connInfo := pion.Encode(c.LocalDescription())
	req, _ = http.NewRequest(http.MethodPost, appCtx.ServerURL+"/join", bytes.NewReader([]byte(connInfo)))
	req.Header.Set("conn-token", conversationId)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}

	select {}
}

func HostNewConversation(appCtx *internal.AppCtx, connConfig ...ConnectionConfig) {
	connectionName := ""
	if len(connConfig) == 0 {
		fmt.Println("Would you like to use a custom connection name? (y/n)")
		reader := bufio.NewReader(os.Stdin)
		text := ""
		for {
			t, _ := reader.ReadString('\n')
			text = strings.ReplaceAll(strings.ToLower(t), "\n", "")
			if text == "y" || text == "yes" || text == "n" || text == "no" {
				break
			}
			fmt.Println("Please enter yes or no (y/n)")
		}
		switch text {
		case "y", "yes":
			fmt.Println("Please enter the custom connection name now")
			connectionName, _ = reader.ReadString('\n')
			connectionName = strings.ReplaceAll(connectionName, "\n", "")
		default:
			connectionName = uuid.New().String()
			break
		}
	} else {
		connectionName = connConfig[0].CustomToken
	}

	c := NewOfferClient(appCtx.ScreenName, appCtx.IsTest)
	// Create offer
	offer, err := c.CreateOffer(nil)
	if err != nil {
		panic(err)
	}

	if err := c.SetLocalDescription(offer); err != nil {
		panic(err)
	}

	// Add handlers for setting up the connection.
	c.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		fmt.Println("New ICE state: ", state)
	})

	c.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate != nil {
			go func() { c.c <- *c.LocalDescription() }()
		}
	})

	var desc webrtc.SessionDescription
	time.Sleep(85 * time.Millisecond)
	select {
	case m := <-c.c:
		desc = m
	}
	j := pion.Encode(desc)
	url := appCtx.ServerURL + "/host"
	req, _ := http.NewRequest(http.MethodPost, url, bytes.NewReader([]byte(j)))
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

	fmt.Println("connection name: '" + connectionName + "' was accepted by the server, it will be valid for 10 minutes, waiting for connection from peer.")

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
