package main

import (
	"fmt"
	"github.com/HarrisonWAffel/terminal-chat/cmd"
	"github.com/HarrisonWAffel/terminal-chat/internal/client"
)

func main() {
	ctx, create, connectionId := cmd.RegisterFlags()
	if create {
		client.NewOfferClient(ctx).HostNewConversation(ctx)
	} else {
		fmt.Println("Attempting to  connect to " + connectionId)
		client.NewReceiverClient(ctx).ConnectToConversationId(ctx, connectionId)
	}
}
