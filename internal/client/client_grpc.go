package client

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/HarrisonWAffel/terminal-chat/internal/pion"
	"github.com/HarrisonWAffel/terminal-chat/internal/server"

	"github.com/pion/webrtc/v3"
	"github.com/pkg/errors"
)

type GRPCHost struct {
	*Client
}

type GRPCReceiver struct {
	*Client
}

func NewGRPCReceiverClient(appCtx *server.AppCtx) ReceivingClient {
	return &GRPCReceiver{Client: newReceiverClient(appCtx)}
}

func NewGRPCOfferClient(appCtx *server.AppCtx) HostClient {
	return &GRPCHost{Client: newOfferClient(appCtx)}
}

func (c *GRPCReceiver) ConnectToConversationId(appCtx *server.AppCtx, conversationId string) {
	// get the remote host connection info for the given token

	resp, err := appCtx.DiscoveryClient.GetConnectionInfoForToken(context.Background(), &server.ConnectionToken{Token: conversationId})
	if err != nil {
		panic(errors.Wrap(err, "error parsing response from server, fatal"))
	}

	offer := webrtc.SessionDescription{}
	err = json.Unmarshal([]byte(resp.GetConnInfoBase64()), &offer)
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
	x := server.ConnectionInfo{ConnInfoBase64: connInfo, Token: &server.ConnectionToken{Token: conversationId}}
	_, err = appCtx.DiscoveryClient.JoinConversation(context.Background(), &x)
	if err != nil {
		panic(err)
	}

	select {}
}

func (c *GRPCHost) HostNewConversation(appCtx *server.AppCtx, connConfig ...ConnectionConfig) {
	connectionName := ""
	if len(connConfig) == 0 {
		connectionName = ReadCustomConnectionName()
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

	recv, err := appCtx.DiscoveryClient.PostConnectionInfo(context.Background(), &server.ConnectionInfo{
		ConnInfoBase64: pion.Encode(desc),
		Token:          &server.ConnectionToken{Token: connectionName},
	})
	if err != nil {
		panic(err)
	}

	fmt.Println("connection name: '" + connectionName + "' was accepted by the server, it will be valid for 10 minutes, waiting for connection from peer.\n\n")

	info := ""
	for {
		b, err := recv.Recv()
		if err == nil {
			info = b.ConnInfoBase64
			break
		} else {
			panic(err)
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
