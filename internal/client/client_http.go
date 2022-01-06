package client

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/HarrisonWAffel/terminal-chat/internal/pion"
	"github.com/HarrisonWAffel/terminal-chat/internal/server"
	"github.com/google/uuid"
	"github.com/pion/webrtc/v3"
	"github.com/pkg/errors"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
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

	req, err := http.NewRequest(http.MethodPost, appCtx.ServerURL+"/host", bytes.NewReader([]byte(pion.Encode(desc))))
	if err != nil {
		panic(err)
	}
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
