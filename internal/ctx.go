package internal

import "log"

type AppCtx struct {
	ScreenName string
	ServerURL  string
	IsTest     bool
	ServerCtx  *ServerCtx
	Log        log.Logger
}

type ServerCtx struct {
	Port string
}
