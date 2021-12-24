package main

import (
	"fmt"
	"github.com/HarrisonWAffel/terminal-chat/internal"
	"github.com/HarrisonWAffel/terminal-chat/internal/client"
	"github.com/HarrisonWAffel/terminal-chat/internal/server"
	"log"
	"os"
	"testing"
	"time"
)

func Test(t *testing.T) {
	ctx := &internal.AppCtx{
		Log:       *log.New(os.Stdout, "", 0),
		ServerURL: "http://localhost:9999",
		IsTest:    true,
		ServerCtx: &internal.ServerCtx{
			Port: ":9999",
		},
	}
	go server.StartServer(ctx)

	go client.HostNewConversation(ctx, client.ConnectionConfig{CustomToken: "testing"})
	time.Sleep(2 * time.Second)
	fmt.Println()
	fmt.Println()
	fmt.Println()
	//go client.ConnectToConversationId(ctx, "testing")

	time.Sleep(510 * time.Minute)
}
