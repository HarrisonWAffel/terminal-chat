package server

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/HarrisonWAffel/terminal-chat/internal/pion"
	"github.com/pion/webrtc/v3"
	"github.com/pkg/errors"
	"net/http"
	"time"
)

type DiscoveryServerImpl struct {
	UnimplementedDiscoveryServer
}

func (d *DiscoveryServerImpl) PostConnectionInfo(connInfo *ConnectionInfo, ds Discovery_PostConnectionInfoServer) error {
	sd := webrtc.SessionDescription{}
	pion.Decode(connInfo.ConnInfoBase64, &sd)
	connToken := createNewConnectionToken(connInfo.Token.GetToken())

	connectionMap.Lock()
	connectionMap.m[connToken] = val{
		timeAdded:      time.Now(),
		connectionInfo: sd,
		snd:            make(chan webrtc.SessionDescription),
	}
	connectionMap.TotalTokensCreated++
	s := connectionMap.m[connToken].snd
	connectionMap.Unlock()

	for {
		select {
		case msg := <-s:
			if msg.SDP == "close" {
				connInfo.ConnInfoBase64 = "token timed out"
				return nil
			}
			j := pion.Encode(msg)
			err := ds.Send(&ConnectionInfo{ConnInfoBase64: j, Token: &ConnectionToken{Token: connToken}})
			if err != nil {
				fmt.Println("err: ", err.Error())
			} else {
				fmt.Println("Connection information has been shared between parties")
			}
			connectionMap.Lock()
			connectionMap.TotalTokensCompleted++
			delete(connectionMap.m, connToken)
			connectionMap.Unlock()
			return err
		}
	}
}

func (d *DiscoveryServerImpl) GetConnectionInfoForToken(ctx context.Context, cToken *ConnectionToken) (*ConnectionInfo, error) {
	connectionMap.Lock()
	defer connectionMap.Unlock()

	_, ok := connectionMap.m[cToken.GetToken()]
	if ok {
		t := connectionMap.m[cToken.GetToken()].connectionInfo
		j, err := json.Marshal(t)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("%d", http.StatusInternalServerError))
		}
		return &ConnectionInfo{ConnInfoBase64: string(j), Token: cToken}, nil
	}

	return nil, errors.New(fmt.Sprintf("%d", http.StatusNotFound))
}

func (d *DiscoveryServerImpl) JoinConversation(ctx context.Context, cToken *ConnectionInfo) (*Empty, error) {
	token := webrtc.SessionDescription{}
	pion.Decode(cToken.GetConnInfoBase64(), &token)

	connectionMap.Lock()
	defer connectionMap.Unlock()

	c, ok := connectionMap.m[cToken.GetToken().GetToken()]
	if ok {
		c.snd <- token
	} else {
		return &Empty{}, errors.New("conversation id not found, token = " + cToken.GetToken().GetToken())
	}

	return &Empty{}, nil
}

func (d *DiscoveryServerImpl) mustEmbedUnimplementedDiscoveryServer() {}
