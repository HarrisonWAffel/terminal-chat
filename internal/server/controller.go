package server

import (
	"encoding/json"
	"fmt"
	"google.golang.org/grpc"
	"log"
	"net"
	"net/http"
)

func StartServer(ctx *AppCtx) {
	if ctx.GRPCEnabled {
		StartGRPCServer(ctx)
	} else {
		StartHTTPServer(ctx)
	}
}

func StartHTTPServer(ctx *AppCtx) {
	m := http.DefaultServeMux
	ctx.Log.Println("HTTP Server listening on " + ctx.ServerCtx.Port)
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

	panic(http.ListenAndServe(ctx.ServerCtx.Port, m))
}

func StartGRPCServer(ctx *AppCtx) {
	// start grpc server
	lis, err := net.Listen("tcp", fmt.Sprintf("%s", ctx.ServerCtx.Port))
	if err != nil {
		log.Fatalf("grpc server failed to listen: %v", err)
	}
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	RegisterDiscoveryServer(grpcServer, &DiscoveryServerImpl{})
	ctx.Log.Println("GRPC Server listening on ", ctx.ServerCtx.Port)
	panic(grpcServer.Serve(lis))
}
