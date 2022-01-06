package client

import (
	"fmt"
	"github.com/HarrisonWAffel/terminal-chat/internal/server"
	"github.com/pion/webrtc/v3"
	"net/http"
	"os"
	"strings"
)

type Client struct {
	AppCtx *server.AppCtx
	Key    string
	*webrtc.PeerConnection
	ICECandidateChan chan webrtc.SessionDescription
	GuiInfo          *GUI
}

type HostClient interface {
	HostNewConversation(appCtx *server.AppCtx, connConfig ...ConnectionConfig)
}

type ReceivingClient interface {
	ConnectToConversationToken(appCtx *server.AppCtx, conversationToken string)
}

func NewOfferClient(ctx *server.AppCtx) HostClient {
	EnsureServerMode(ctx)
	if ctx.GRPCEnabled {
		return NewGRPCOfferClient(ctx)
	}
	return NewHTTPOfferClient(ctx)
}

func NewReceiverClient(ctx *server.AppCtx) ReceivingClient {
	EnsureServerMode(ctx)
	if ctx.GRPCEnabled {
		return NewGRPCReceiverClient(ctx)
	}
	return NewHTTPReceiverClient(ctx)
}

func EnsureServerMode(ctx *server.AppCtx) {
	serverMode := GetServerMode(ctx)
	if ctx.GRPCEnabled {
		if serverMode != "gRPC" {
			fmt.Println("\n\nremote server is not running in gRPC mode, omit '-grpc' flag")
			os.Exit(1)
		}
	} else {
		if serverMode != "HTTP" {
			fmt.Println("\n\nremote server is not running in HTTP mode, rerun command with the '-grpc' flag")
			os.Exit(1)
		}
	}
}

func GetServerMode(ctx *server.AppCtx) string {
	if !strings.Contains(ctx.ServerURL, "http://") {
		ctx.ServerURL = "https://" + ctx.ServerURL
	}

	resp, err := http.Get(ctx.ServerURL + "/get")
	if err != nil {
		return "gRPC"
	}
	_ = resp
	return "HTTP"
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

	return c
}
