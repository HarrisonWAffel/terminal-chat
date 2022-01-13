package client

import (
	"context"
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

func (c *GRPCReceiver) ConnectToConversationToken(appCtx *server.AppCtx, ConversationToken string) {
	// get the remote host connection info for the given token
	resp, err := appCtx.DiscoveryClient.GetConnectionInfoForToken(context.Background(), &server.ConnectionToken{Token: ConversationToken})
	if err != nil {
		panic(errors.Wrap(err, "error parsing response from server, fatal"))
	}

	c.ConnectToOffer([]byte(resp.GetConnInfoBase64()))

	<-webrtc.GatheringCompletePromise(c.PeerConnection)

	connInfo, err := pion.Encode(c.LocalDescription())
	if err != nil {
		panic("could not encode connection information: " + err.Error())
	}

	x := server.ConnectionInfo{ConnInfoBase64: connInfo, Token: ConversationToken}
	_, err = appCtx.DiscoveryClient.JoinConversation(context.Background(), &x)
	if err != nil {
		panic(err)
	}

	select {}
}

func (c *GRPCHost) HostNewConversation(appCtx *server.AppCtx, connConfig ...ConnectionConfig) {
	connectionName := ReadCustomConnectionName(connConfig...)

	desc := c.GatherICECandidate()
	encode, err := pion.Encode(desc)
	if err != nil {
		panic("could not encode connection information: " + err.Error())
	}

	recv, err := appCtx.DiscoveryClient.PostConnectionInfo(context.Background(), &server.ConnectionInfo{
		ConnInfoBase64: encode,
		Token:          connectionName,
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
		}
	}

	c.DecodeAndSetDescription(info)

	select {}
}
