package server

import (
	"github.com/HarrisonWAffel/terminal-chat/internal"
	"net/http"
)

func StartServer(ctx *internal.AppCtx) {
	m := http.DefaultServeMux
	ctx.Log.Println("Server listening on " + ctx.ServerCtx.Port)
	m.HandleFunc("/host", CreateConnectionToken)
	m.HandleFunc("/get", GetInfoForToken)
	m.HandleFunc("/join", ConnectWithToken)
	CreateAndMonitorConnectionMap()

	panic(http.ListenAndServe(ctx.ServerCtx.Port, m))
}
