package main

import (
	"fmt"
	"github.com/HarrisonWAffel/terminal-chat/cmd"
	"github.com/HarrisonWAffel/terminal-chat/internal/client"
)

func main() {
	ctx, create, roomName := cmd.RegisterFlags()
	if create {
		c := client.NewOfferClient(ctx)
		if roomName != "" {
			config := &client.ConnectionConfig{CustomToken: roomName}
			c.HostNewConversation(ctx, *config)
		}
		c.HostNewConversation(ctx)
	}

	fmt.Println("Attempting to  connect to " + roomName)

	client.NewReceiverClient(ctx).ConnectToConversationToken(ctx, roomName)
}
