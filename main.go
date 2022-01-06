package main

import (
	"fmt"
	"github.com/HarrisonWAffel/terminal-chat/cmd"
	"github.com/HarrisonWAffel/terminal-chat/internal/client"
)

func main() {
	ctx, create, connectionId, roomName := cmd.RegisterFlags()
	if create {
		var config *client.ConnectionConfig
		if roomName != "" {
			config = &client.ConnectionConfig{CustomToken: roomName}
		}
		c := client.NewOfferClient(ctx)
		c.HostNewConversation(ctx, *config)
	}

	fmt.Println("Attempting to  connect to " + connectionId)

	client.NewReceiverClient(ctx).ConnectToConversationToken(ctx, connectionId)
}
