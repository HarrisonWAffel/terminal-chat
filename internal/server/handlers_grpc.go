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
	err := pion.Decode(connInfo.ConnInfoBase64, &sd)
	if err != nil {
		return err
	}
	connToken := createNewConnectionToken(connInfo.GetToken())

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
			j, err := pion.Encode(msg)
			if err != nil {
				return err
			}
			err = ds.Send(&ConnectionInfo{ConnInfoBase64: j, Token: connToken})
			if err != nil {
				return err
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
		return &ConnectionInfo{ConnInfoBase64: string(j), Token: cToken.GetToken()}, nil
	}

	return nil, errors.New(fmt.Sprintf("%d", http.StatusNotFound))
}

func (d *DiscoveryServerImpl) JoinConversation(ctx context.Context, cToken *ConnectionInfo) (*Empty, error) {
	token := webrtc.SessionDescription{}
	err := pion.Decode(cToken.GetConnInfoBase64(), &token)
	if err != nil {
		return nil, err
	}

	connectionMap.Lock()
	defer connectionMap.Unlock()

	c, ok := connectionMap.m[cToken.GetToken()]
	if ok {
		c.snd <- token
	} else {
		return &Empty{}, errors.New("conversation id not found, token = " + cToken.GetToken())
	}

	return &Empty{}, nil
}

func (d *DiscoveryServerImpl) mustEmbedUnimplementedDiscoveryServer() {}
