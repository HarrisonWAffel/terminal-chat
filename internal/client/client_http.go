package client

import (
	"bytes"
	"encoding/json"
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
	fmt.Println(appCtx.ServerURL + "/get")
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
	fmt.Println("Gathering ICE Candidates")
	<-webrtc.GatheringCompletePromise(c.PeerConnection)
	fmt.Println("Done gathering ICE candidates")
	connInfo := pion.Encode(c.LocalDescription())
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

	req, err := http.NewRequest(http.MethodPost, appCtx.ServerURL+"/host", bytes.NewReader([]byte(pion.Encode(desc))))
	if err != nil {
		panic(err)
	}
	if connectionName != "" {
		req.Header.Set("req-conn-id", connectionName)
	}

	client := http.Client{
		CheckRedirect: nil,
		Jar:           nil,
		Timeout:       0,
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
