package main

import (
	"fmt"
	"github.com/HarrisonWAffel/terminal-chat/cmd"
	"github.com/HarrisonWAffel/terminal-chat/internal/client"
)

func main() {
	ctx, create, connectionId, roomName := cmd.RegisterFlags()
	if create {
		if roomName == "" {
			client.NewOfferClient(ctx).HostNewConversation(ctx)
		} else {
			client.NewOfferClient(ctx).HostNewConversation(ctx, client.ConnectionConfig{CustomToken: roomName})
		}
	} else {
		fmt.Println("Attempting to  connect to " + connectionId)
		client.NewReceiverClient(ctx).ConnectToConversationId(ctx, connectionId)
	}
}
