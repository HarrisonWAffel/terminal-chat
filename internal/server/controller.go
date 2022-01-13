package server

import (
	"encoding/json"
	"fmt"
	"google.golang.org/grpc"
	"log"
	"net"
	"net/http"
	"os"
	"time"
)

func StartServer(ctx *AppCtx) {
	if ctx.GRPCEnabled {
		StartGRPCServer(ctx)
	} else {
		StartHTTPServer(ctx)
	}
}

type HTTPHandler struct {
	F   func(ctx *AppCtx, w http.ResponseWriter, r *http.Request)
	Ctx *AppCtx
}

func (H *HTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	H.F(&AppCtx{
		DiscoveryClient: H.Ctx.DiscoveryClient,
		ScreenName:      H.Ctx.ScreenName,
		ServerURL:       H.Ctx.ServerURL,
		GRPCEnabled:     H.Ctx.GRPCEnabled,
		ServerCtx:       H.Ctx.ServerCtx,
		Log:             *log.New(os.Stdout, time.Now().Format("Monday Jan _2")+" | "+fmt.Sprintf("%s %s", r.Method, r.URL)+" | ", 2),
	}, w, r)
}

func StartHTTPServer(ctx *AppCtx) {
	m := http.DefaultServeMux
	m.Handle("/host", &HTTPHandler{F: CreateConnectionToken, Ctx: ctx})
	m.Handle("/get", &HTTPHandler{F: GetInfoForToken, Ctx: ctx})
	m.Handle("/join", &HTTPHandler{F: ConnectWithToken, Ctx: ctx})
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

	ctx.Log.Println("HTTP Server listening on " + ctx.ServerCtx.Port)
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
