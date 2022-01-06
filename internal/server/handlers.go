package server

import (
	"encoding/json"
	"fmt"
	"github.com/HarrisonWAffel/terminal-chat/internal/pion"
	"github.com/google/uuid"
	"github.com/pion/webrtc/v3"
	"io"
	"net/http"
	"sync"
	"time"
)

type val struct {
	timeAdded      time.Time
	connectionInfo webrtc.SessionDescription
	snd            chan webrtc.SessionDescription
}

type ConnectionMap struct {
	sync.Mutex
	m                    map[string]val
	duration             time.Duration
	TotalTokensCreated   int64
	TotalTokensCompleted int64
	TotalTokensExpired   int64
}

var connectionMap ConnectionMap

func init() {
	CreateAndMonitorConnectionMap()
}

func CreateAndMonitorConnectionMap() {
	connectionMap.m = make(map[string]val)
	connectionMap.duration = 1 * time.Minute
	go func() {
		for {
			select {
			case <-time.After(time.Minute * 10):
				connectionMap.Lock()
				var outDatedKeys []string
				for k, v := range connectionMap.m {
					if time.Since(v.timeAdded) >= connectionMap.duration {
						outDatedKeys = append(outDatedKeys, k)
					}
				}
				for _, key := range outDatedKeys {
					connectionMap.m[key].snd <- webrtc.SessionDescription{SDP: "close"}
					close(connectionMap.m[key].snd)
					delete(connectionMap.m, key)
					connectionMap.TotalTokensExpired++
				}
				connectionMap.Unlock()
			}
		}
	}()
}

func createNewConnectionToken(token string) string {
	if token == "" {
		token = uuid.New().String()
	}
	for {
		connectionMap.Lock()
		_, ok := connectionMap.m[token]
		if !ok {
			connectionMap.Unlock()
			return token
		}
		token = uuid.New().String()
		connectionMap.Unlock()
	}
}

func CreateConnectionToken(w http.ResponseWriter, r *http.Request) {
	b, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	sd := webrtc.SessionDescription{}
	pion.Decode(string(b), &sd)

	connToken := createNewConnectionToken(r.Header.Get("req-conn-id"))
	connectionMap.Lock()
	connectionMap.m[connToken] = val{
		timeAdded:      time.Now(),
		connectionInfo: sd,
		snd:            make(chan webrtc.SessionDescription),
	}
	connectionMap.TotalTokensCreated++
	s := connectionMap.m[connToken].snd
	connectionMap.Unlock()

	fmt.Println("connection token: '" + r.Header.Get("req-conn-id") + "' has been created. Waiting for incoming connection...")
	w.WriteHeader(http.StatusOK)
	w.(http.Flusher).Flush()
	for {
		select {
		case msg := <-s:
			if msg.SDP == "close" {
				w.Write([]byte("token timed out"))
				return
			}
			j := pion.Encode(msg)
			w.Write([]byte(j))
			fmt.Println("Connection information has been shared between parties")
			connectionMap.Lock()
			connectionMap.TotalTokensCompleted++
			delete(connectionMap.m, connToken)
			connectionMap.Unlock()
			return
		}
	}

}

func GetInfoForToken(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("conn-token")
	if token == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	connectionMap.Lock()
	defer connectionMap.Unlock()
	_, ok := connectionMap.m[token]
	if ok {
		t := connectionMap.m[token].connectionInfo
		j, err := json.Marshal(t)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Write(j)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

func ConnectWithToken(w http.ResponseWriter, r *http.Request) {
	connectionToken := r.Header.Get("conn-token")
	if connectionToken == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// 1. p2 reqs conn info for p1
	// 2. p2 sends conn info
	// 3. p1 conn info is forwarded to p2
	// 4. p1 and p2 connect to each other
	b, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	token := webrtc.SessionDescription{}
	pion.Decode(string(b), &token)

	connectionMap.Lock()
	c, ok := connectionMap.m[connectionToken]
	if ok {
		c.snd <- token
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
	connectionMap.Unlock()
}
