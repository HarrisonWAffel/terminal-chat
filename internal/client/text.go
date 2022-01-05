package client

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/pion/webrtc/v3"
	"io"
	"strconv"
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
