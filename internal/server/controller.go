package server

import (
	"encoding/json"
	"github.com/HarrisonWAffel/terminal-chat/internal"
	"net/http"
)

func StartServer(ctx *internal.AppCtx) {
	m := http.DefaultServeMux
	ctx.Log.Println("Server listening on " + ctx.ServerCtx.Port)
	m.HandleFunc("/host", CreateConnectionToken)
	m.HandleFunc("/get", GetInfoForToken)
	m.HandleFunc("/join", ConnectWithToken)
	m.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		j, err := json.Marshal(struct {
			TotalTokensCreated          int64 `json:"total_tokens_created"`
			TotalConversationsConnected int64 `json:"total_conversations_connected"`
			TotalConversationsExpired   int64 `json:"total_conversations_expired"`
		}{
			TotalTokensCreated:          connectionMap.TotalTokensCreated,
			TotalConversationsConnected: connectionMap.TotalTokensCompleted,
			TotalConversationsExpired:   connectionMap.TotalTokensExpired,
		})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Write(j)
	})
	CreateAndMonitorConnectionMap()

	panic(http.ListenAndServe(ctx.ServerCtx.Port, m))
}
