package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

func main() {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}

	fmt.Println("PUBLIC_KEY_BASE64 =", base64.StdEncoding.EncodeToString(pub))
	fmt.Println("PRIVATE_KEY_BASE64 =", base64.StdEncoding.EncodeToString(priv))
}
