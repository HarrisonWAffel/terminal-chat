package client

import (
	"bufio"
	"fmt"
	"github.com/pion/webrtc/v3"
	"os"
	"strings"
)

func RenderOutput(msg webrtc.DataChannelMessage, name string) {
	fmt.Printf("\033[2K\r%s\n", string(msg.Data))
}

func HandleInput(name string, d *webrtc.DataChannel) {
	fmt.Println("Connected to Peer!")
	fmt.Println("Ready to send messages")
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print(name + ": ")
		l, err := reader.ReadString('\n')
		if err != nil {
			panic(err)
		}
		fmt.Printf("\033[2K\r")
		l = strings.ReplaceAll(l, "\n", "")

		err = d.SendText(name + ": " + l)
		if err != nil {
			panic(err)
		}
	}
}
