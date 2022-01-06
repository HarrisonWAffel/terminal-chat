package cmd

import (
	"flag"
	"fmt"
	"github.com/HarrisonWAffel/terminal-chat/internal/server"
	"google.golang.org/grpc"
	"log"
	"os"
)

func RegisterFlags() (*server.AppCtx, bool, string, string) {
	description := `run the application in server mode. Clients connect to the server to
exchange pion connection information between hosts before establishing the p2p connection`
	runAsServer := flag.Bool("server", false, description)
	serverPort := flag.String("server-port", ":8080", "The server port that should be used, should begin with : (e.g. :8080) ")
	serverURL := flag.String("server-url", "", "The URL of the server you want to connect to communicate p2p connection info with another client")
	enableGrpc := flag.Bool("grpc", false, "pass this flag to start or connect to a gRPC server")

	screenName := flag.String("screen-name", "", "The name you wish to use in the conversation")
	create := flag.Bool("create", false, "pass this flag if you wish to begin a conversation and wait for a peer to connect")
	connect := flag.String("connect", "", "the room ID created by the other peer you wish to connect to")
	roomName := flag.String("room-name", "", "immediately supply the room name to be created")
	flag.Parse()

	if *runAsServer {
		server.StartServer(&server.AppCtx{
			Log:         *log.New(os.Stdout, "", 0),
			GRPCEnabled: *enableGrpc,
			ServerCtx: &server.ServerCtx{
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

	if *enableGrpc {
		conn, err := grpc.Dial(*serverURL, grpc.WithInsecure())
		if err != nil {
			panic(err)
		}
		return &server.AppCtx{
			DiscoveryClient: server.NewDiscoveryClient(conn),
			GRPCEnabled:     *enableGrpc,
			ServerURL:       *serverURL,
			ScreenName:      *screenName,
		}, *create, *connect, *roomName
	}

	return &server.AppCtx{
		ServerURL:  *serverURL,
		ScreenName: *screenName,
	}, *create, *connect, *roomName
}
