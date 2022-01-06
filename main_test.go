package main

import (
	"fmt"
	"github.com/HarrisonWAffel/terminal-chat/internal/client"
	"github.com/HarrisonWAffel/terminal-chat/internal/server"
	"github.com/pion/webrtc/v3"
	"google.golang.org/grpc"
	"log"
	"os"
	"sync"
	"testing"
	"time"
)

var httpCtx = &server.AppCtx{
	Log:       *log.New(os.Stdout, "", 0),
	ServerURL: "http://localhost:9999",
	ServerCtx: &server.ServerCtx{
		Port: ":9999",
	},
}

var GrpcCtx = &server.AppCtx{
	GRPCEnabled: true,
	Log:         *log.New(os.Stdout, "", 0),
	ServerURL:   "http://localhost:9998",
	ServerCtx: &server.ServerCtx{
		Port: ":9998",
	},
}

func init() {
	go server.StartHTTPServer(httpCtx)
	go server.StartGRPCServer(GrpcCtx)
	conn, err := grpc.Dial(GrpcCtx.ServerCtx.Port, grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	GrpcCtx.DiscoveryClient = server.NewDiscoveryClient(conn)
}

func TestHTTP(t *testing.T) {
	h, recv := NewHTTPTestHostClient(httpCtx)
	r := NewHTTPTestReceiverClient(httpCtx)

	go h.HostNewConversation(httpCtx, client.ConnectionConfig{CustomToken: "testing"})
	time.Sleep(10 * time.Second) // just needs to be longer than the offer clients ICE candidate gathering
	go r.ConnectToConversationId(httpCtx, "testing")

	ReceiveMessageOrFailTest(t, recv)
}

func TestGRPC(t *testing.T) {
	h, recv := NewGRPCTestHostClient(GrpcCtx)
	r := NewGRPCTestReceiverClient(GrpcCtx)

	go h.HostNewConversation(GrpcCtx, client.ConnectionConfig{CustomToken: "testing"})
	time.Sleep(10 * time.Second) // just needs to be longer than the offer clients ICE candidate gathering
	go r.ConnectToConversationId(GrpcCtx, "testing")

	ReceiveMessageOrFailTest(t, recv)
}

func TestMultipleHTTP(t *testing.T) {
	wg := &sync.WaitGroup{}
	for i := 0; i < 25; i++ {
		go func(i int, wg *sync.WaitGroup) {
			wg.Add(1)
			h1, recv1 := NewHTTPTestHostClient(httpCtx)
			r := NewHTTPTestReceiverClient(httpCtx)
			go h1.HostNewConversation(httpCtx, client.ConnectionConfig{CustomToken: fmt.Sprintf("test%d", i)})
			time.Sleep(10 * time.Second)
			go r.ConnectToConversationId(httpCtx, fmt.Sprintf("test%d", i))
			ReceiveMessageOrFailTest(t, recv1)
			wg.Done()
		}(i, wg)
	}
	wg.Wait()
}

func TestMutlipleGRPC(t *testing.T) {
	h1, recv1 := NewGRPCTestHostClient(GrpcCtx)
	r := NewGRPCTestReceiverClient(GrpcCtx)

	h2, recv2 := NewGRPCTestHostClient(GrpcCtx)
	r2 := NewGRPCTestReceiverClient(GrpcCtx)

	go h1.HostNewConversation(GrpcCtx, client.ConnectionConfig{CustomToken: "test1"})
	go h2.HostNewConversation(GrpcCtx, client.ConnectionConfig{CustomToken: "test2"})

	time.Sleep(10 * time.Second)

	go r.ConnectToConversationId(GrpcCtx, "test1")
	go r2.ConnectToConversationId(GrpcCtx, "test2")

	ReceiveMessageOrFailTest(t, recv1)
	ReceiveMessageOrFailTest(t, recv2)
}

func ReceiveMessageOrFailTest(t *testing.T, channel chan string) {
L:
	for {
		select {
		case <-channel:
			break L
		case <-time.After(time.Second * 25):
			t.Log("failed to receive message after 25 seconds")
			t.FailNow()
		}
	}
}

func NewHTTPTestReceiverClient(appCtx *server.AppCtx) client.ReceivingClient {
	return &client.HTTPReceiver{Client: NewTestReceiverClient(appCtx)}
}

func NewHTTPTestHostClient(appCtx *server.AppCtx) (client.HostClient, chan string) {
	c, recv := NewTestOfferClient(appCtx)
	return &client.HTTPHost{Client: c}, recv
}

func NewGRPCTestReceiverClient(appCtx *server.AppCtx) client.ReceivingClient {
	return &client.GRPCReceiver{Client: NewTestReceiverClient(appCtx)}
}

func NewGRPCTestHostClient(appCtx *server.AppCtx) (client.HostClient, chan string) {
	c, recv := NewTestOfferClient(appCtx)
	return &client.GRPCHost{Client: c}, recv
}

func NewTestReceiverClient(appCtx *server.AppCtx) *client.Client {
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

	return c
}

func NewTestOfferClient(appCtx *server.AppCtx) (*client.Client, chan string) {
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

	return c, recv
}
