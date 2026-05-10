package main

import (
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

func main() {
	priv, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
	fmt.Println("X25519_PRIVATE_KEY=" + base64.RawURLEncoding.EncodeToString(priv.Bytes()))
	fmt.Println("X25519_PUBLIC_KEY=" + base64.RawURLEncoding.EncodeToString(priv.PublicKey().Bytes()))
}
