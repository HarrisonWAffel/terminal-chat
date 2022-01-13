package client

import (
	"bytes"
	"fmt"
	"github.com/HarrisonWAffel/terminal-chat/internal/pion"
	"github.com/HarrisonWAffel/terminal-chat/internal/server"
	"github.com/pion/webrtc/v3"
	"github.com/pkg/errors"
	"io/ioutil"
	"net/http"
	"time"
)

type HTTPHost struct {
	*Client
}

type HTTPReceiver struct {
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

func NewHTTPReceiverClient(appCtx *server.AppCtx) ReceivingClient {
	return &HTTPReceiver{Client: newReceiverClient(appCtx)}
}

func NewHTTPOfferClient(appCtx *server.AppCtx) HostClient {
	return &HTTPHost{Client: newOfferClient(appCtx)}
}

type ConnectionConfig struct {
	CustomToken string
}

func (c *HTTPReceiver) ConnectToConversationToken(appCtx *server.AppCtx, conversationToken string) {
	// get the remote host connection info for the given token
	req, _ := http.NewRequest(http.MethodGet, appCtx.ServerURL+"/get", nil)
	req.Header.Set("conn-token", conversationToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}

	if resp.StatusCode != http.StatusOK {
		panic("bad status received from server, " + resp.Status + ", check conversation ID")
	}

	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	c.ConnectToOffer(b)

	fmt.Println("Gathering ICE Candidates")
	<-webrtc.GatheringCompletePromise(c.PeerConnection)
	fmt.Println("Done gathering ICE candidates")

	connInfo, err := pion.Encode(c.LocalDescription())
	if err != nil {
		panic("could not encode local description: " + err.Error())
	}

	fmt.Println("Attempting to join ", conversationToken)
	req, _ = http.NewRequest(http.MethodPost, appCtx.ServerURL+"/join", bytes.NewReader([]byte(connInfo)))
	req.Header.Set("conn-token", conversationToken)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		panic(errors.Wrap(err, "could not join conversation"))
	}

	select {}
}

func (c *HTTPHost) HostNewConversation(appCtx *server.AppCtx, connConfig ...ConnectionConfig) {
	connectionName := ReadCustomConnectionName(connConfig...)

	desc := c.GatherICECandidate()
	encoded, err := pion.Encode(desc)
	if err != nil {
		panic("could not encode local description: " + err.Error())
	}

	req, err := http.NewRequest(http.MethodPost, appCtx.ServerURL+"/host", bytes.NewReader([]byte(encoded)))
	if err != nil {
		panic(err)
	}
	if connectionName != "" {
		req.Header.Set("req-conn-id", connectionName)
	}

	client := http.Client{
		Timeout: 11 * time.Minute,
	}

	info := ""
	resp, err := client.Do(req)
	if err != nil {
		panic(err.Error() + ": is this a gRPC server?")
	}

	if resp.StatusCode != http.StatusOK {
		panic("status " + resp.Status + " != 200 OK")
	}

	fmt.Println("connection name: '" + connectionName + "' was accepted by the server, it will be valid for 10 minutes, waiting for connection from peer.\n\n")

	for {
		b, err := ioutil.ReadAll(resp.Body)
		if err == nil {
			if len(b) > 0 {
				info = string(b)
				break
			}
		}
		time.Sleep(1 * time.Second)
	}

	c.DecodeAndSetDescription(info)

	select {}
}
