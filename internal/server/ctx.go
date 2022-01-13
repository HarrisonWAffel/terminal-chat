package server

import (
	"log"
)

type AppCtx struct {
	DiscoveryClient DiscoveryClient
	ScreenName      string
	ServerURL       string
	GRPCEnabled     bool
	ServerCtx       *ServerCtx
	Log             log.Logger
}

type ServerCtx struct {
	Port string
}
