package main

import (
	"fmt"
	"github.com/HarrisonWAffel/terminal-chat/internal"
	"github.com/HarrisonWAffel/terminal-chat/internal/client"
	"github.com/HarrisonWAffel/terminal-chat/internal/server"
	"github.com/pion/webrtc/v3"
	"log"
	"os"
	"testing"
	"time"
)

var ctx = &internal.AppCtx{
	Log:       *log.New(os.Stdout, "", 0),
	ServerURL: "http://localhost:9999",
	IsTest:    true,
	ServerCtx: &internal.ServerCtx{
		Port: ":9999",
	},
}

func Test(t *testing.T) {
	go server.StartServer(ctx)
	h, recv := NewTestOfferClient(ctx)
	r := NewTestReceiverClient(ctx)

	go h.HostNewConversation(ctx, client.ConnectionConfig{CustomToken: "testing"})
	time.Sleep(10 * time.Second) // just needs to be longer than the offer clients ICE candidate gathering
	go r.ConnectToConversationId(ctx, "testing")

L:
	for {
		select {
		case m := <-recv:
			t.Log(m)
			break L
		case <-time.After(time.Minute * 1):
			t.FailNow()
		}
	}
}

func NewTestReceiverClient(appCtx *internal.AppCtx) client.ReceivingClient {
	pc, err := webrtc.NewPeerConnection(client.Config)
	if err != nil {
		panic(err)
	}
	c := &client.Client{
		AppCtx:           appCtx,
		PeerConnection:   pc,
		ICECandidateChan: make(chan webrtc.SessionDescription, 1),
	}

	c.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
		fmt.Printf("Peer Connection State has changed: %s\n", s.String())
		if s == webrtc.PeerConnectionStateFailed {
			fmt.Println("Peer Connection has gone to failed, exiting")
			os.Exit(0)
		}
	})

	c.OnDataChannel(func(d *webrtc.DataChannel) {
		fmt.Printf("DataChannel %s-%d open\n\n", d.Label(), d.ID())
		d.OnOpen(func() {
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
		})
		d.OnMessage(func(msg webrtc.DataChannelMessage) {
			fmt.Println(string(msg.Data))
		})
	})

	return &client.Receiver{Client: c}
}

func NewTestOfferClient(appCtx *internal.AppCtx) (client.HostClient, chan string) {
	pc, err := webrtc.NewPeerConnection(client.Config)
	if err != nil {
		panic(err)
	}
	c := &client.Client{
		AppCtx:           appCtx,
		PeerConnection:   pc,
		ICECandidateChan: make(chan webrtc.SessionDescription, 1),
	}

	d, err := c.CreateDataChannel("conversation", nil)
	if err != nil {
		panic(err)
	}

	d.OnClose(func() {
		fmt.Println("data channel has closed")
	})

	c.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
		fmt.Printf("Peer Connection State has changed: %s\n", s.String())
		if s == webrtc.PeerConnectionStateFailed {
			fmt.Println("Peer Connection has gone to failed, exiting")
			os.Exit(0)
		}
	})

	d.OnOpen(func() {})
	recv := make(chan string)
	d.OnMessage(func(msg webrtc.DataChannelMessage) {
		fmt.Println(string(msg.Data))
		recv <- string(msg.Data)
	})

	return &client.Host{Client: c}, recv
}
