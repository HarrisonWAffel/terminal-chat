package client

import (
	"encoding/json"
	"fmt"
	"github.com/HarrisonWAffel/terminal-chat/internal/pion"
	"github.com/HarrisonWAffel/terminal-chat/internal/server"
	"github.com/pion/webrtc/v3"
	"github.com/pkg/errors"
	"os"
)

type Client struct {
	AppCtx *server.AppCtx
	Key    string
	*webrtc.PeerConnection
	ICECandidateChan chan webrtc.SessionDescription
	GuiInfo          *GUI
}

func (c *Client) ConnectToOffer(offerBytes []byte) {
	offer := webrtc.SessionDescription{}
	err := json.Unmarshal(offerBytes, &offer)
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
}

func (c *Client) GatherICECandidate() webrtc.SessionDescription {
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
	return desc
}

func (c *Client) DecodeAndSetDescription(info string) {
	if info == "token timed out" {
		fmt.Println("connection failed: token timed out before peer connected")
		os.Exit(1)
	}
	descr := webrtc.SessionDescription{}
	pion.Decode(info, &descr)
	err := c.SetRemoteDescription(descr)
	if err != nil {
		fmt.Println("error setting remote description")
		panic(err)
	}
}

type HostClient interface {
	HostNewConversation(appCtx *server.AppCtx, connConfig ...ConnectionConfig)
}

type ReceivingClient interface {
	ConnectToConversationToken(appCtx *server.AppCtx, conversationToken string)
}

func NewOfferClient(ctx *server.AppCtx) HostClient {
	if ctx.GRPCEnabled {
		return NewGRPCOfferClient(ctx)
	}
	return NewHTTPOfferClient(ctx)
}

func NewReceiverClient(ctx *server.AppCtx) ReceivingClient {
	if ctx.GRPCEnabled {
		return NewGRPCReceiverClient(ctx)
	}
	return NewHTTPReceiverClient(ctx)
}

func newOfferClient(ctx *server.AppCtx) *Client {
	pc, err := webrtc.NewPeerConnection(Config)
	if err != nil {
		panic(err)
	}
	c := &Client{
		AppCtx:           ctx,
		PeerConnection:   pc,
		ICECandidateChan: make(chan webrtc.SessionDescription, 1),
		GuiInfo: &GUI{
			InputChan:   make(chan string),
			OutputChan:  make(chan string),
			NetworkChan: make(chan string),
			Username:    ctx.ScreenName,
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
			c.GuiInfo.OutputChan <- "\n\n***************\nPeer has disconnected. Either wait for possible reconnection or exit application\n***************\n"
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
	return c
}

func newReceiverClient(ctx *server.AppCtx) *Client {
	pc, err := webrtc.NewPeerConnection(Config)
	if err != nil {
		panic(err)
	}
	c := &Client{
		AppCtx:           ctx,
		PeerConnection:   pc,
		ICECandidateChan: make(chan webrtc.SessionDescription, 1),
		GuiInfo: &GUI{
			InputChan:   make(chan string),
			OutputChan:  make(chan string),
			NetworkChan: make(chan string),
			Username:    ctx.ScreenName,
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

	return c
}
