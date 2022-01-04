package client

import (
	"bufio"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/pion/webrtc/v3"
	"io"
	"os"
	"strconv"
	"strings"
)

func padKey(b []byte) []byte {
	for i := len(b); i < 32; i++ {
		a := [1]byte{byte(i)}
		b = append(b, a[0])
	}
	return b
}

func setKey(c *Client, d *webrtc.DataChannel) {
	b := []byte(d.Label() + strconv.Itoa(int(*d.ID())))
	b = padKey(b)
	c.Key = hex.EncodeToString(b)
}

func encryptText(s, keyString string) string {
	key, _ := hex.DecodeString(keyString)
	plaintext := []byte(s)
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err.Error())
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		panic(err.Error())
	}
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		panic(err.Error())
	}
	ciphertext := aesGCM.Seal(nonce, nonce, plaintext, nil)
	return fmt.Sprintf("%x", ciphertext)
	return ""
}

func decryptText(s, keyString string) string {
	key, _ := hex.DecodeString(keyString)
	enc, _ := hex.DecodeString(s)
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err.Error())
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		panic(err.Error())
	}
	nonceSize := aesGCM.NonceSize()
	nonce, ciphertext := enc[:nonceSize], enc[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		panic(err.Error())
	}
	return fmt.Sprintf("%s", plaintext)
}

func (c *Client) RenderOutput(msg webrtc.DataChannelMessage) {
	fmt.Printf("\033[2K\r%s\n", decryptText(string(msg.Data), c.Key))
}

func (c *Client) HandleInput(d *webrtc.DataChannel) {
	fmt.Println("Connected to Peer!")
	fmt.Println("Ready to send messages\n\n")
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print(c.AppCtx.ScreenName + ": ")
		l, err := reader.ReadString('\n')
		if err != nil {
			panic(err)
		}

		l = strings.ReplaceAll(l, "\n", "")

		err = d.SendText(encryptText(c.AppCtx.ScreenName+": "+l, c.Key))
		if err != nil {
			panic(err)
		}
	}
}
