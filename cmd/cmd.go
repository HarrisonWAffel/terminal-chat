package cmd

import (
	"flag"
	"fmt"
	"github.com/HarrisonWAffel/terminal-chat/internal"
	"github.com/HarrisonWAffel/terminal-chat/internal/server"
	"log"
	"os"
)

func RegisterFlags() (*internal.AppCtx, bool, string) {
	description := `run the application in server mode. Clients connect to the server to
change pion connection information between hosts before establishing the p2p connection`
	runAsServer := flag.Bool("server", false, description)
	serverPort := flag.String("server-port", ":8080", "The server port that should be used, should begin with : (e.g. :8080) ")
	serverURL := flag.String("server-url", "", "The URL of the server you want to connect to communicate p2p connection info with another client")

	screenName := flag.String("screen-name", "", "The name you wish to use in the conversation")
	create := flag.Bool("create", false, "set to TRUE if you wish to begin a conversation")
	connect := flag.String("connect", "", "the connection ID of the other person")

	flag.Parse()

	if *runAsServer {
		server.StartServer(&internal.AppCtx{
			Log: *log.New(os.Stdout, "", 0),
			ServerCtx: &internal.ServerCtx{
				Port: *serverPort,
			},
		})
	}

	if *serverURL == "" {
		*serverURL = os.Getenv("TERMINAL_CHAT_URL")
		if *serverURL == "" {
			fmt.Println("must supply a TERMINAL_CHAT_URL environment variable or pass the --server-url flag with a valid server URL")
			os.Exit(1)
		}
	}

	if *screenName == "" {
		fmt.Println("must supply screen name using the --screen-name flag")
		os.Exit(1)
	}

	return &internal.AppCtx{
		ServerURL:  *serverURL,
		ScreenName: *screenName,
	}, *create, *connect
}
