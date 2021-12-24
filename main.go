package main

import (
	"fmt"
	"github.com/HarrisonWAffel/terminal-chat/cmd"
	"github.com/HarrisonWAffel/terminal-chat/internal/client"
)

func main() {
	ctx, create, connectionId := cmd.RegisterFlags()
	if create {
		if ctx.IsTest {
			client.HostNewConversation(ctx, client.ConnectionConfig{CustomToken: "testing"})
		} else {
			client.HostNewConversation(ctx)
		}
	} else {
		fmt.Println("Attempting to  connect to " + connectionId)
		client.ConnectToConversationId(ctx, connectionId)
	}
}
