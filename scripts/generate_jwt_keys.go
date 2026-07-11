package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <private-key-file>\n", os.Args[0])
		os.Exit(2)
	}

	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		fail("generate Ed25519 key", err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		fail("encode Ed25519 key as PKCS#8", err)
	}

	path := os.Args[1]
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		fail("create private key file", err)
	}
	if err := pem.Encode(file, &pem.Block{Type: "PRIVATE KEY", Bytes: der}); err != nil {
		file.Close()
		os.Remove(path)
		fail("write private key file", err)
	}
	if err := file.Close(); err != nil {
		os.Remove(path)
		fail("close private key file", err)
	}

	fmt.Printf("Created Ed25519 private key at %s\n", path)
	fmt.Printf("Set LICENSE_JWT_PRIVATE_KEY_FILE=%s\n", path)
}

func fail(action string, err error) {
	fmt.Fprintf(os.Stderr, "%s: %v\n", action, err)
	os.Exit(1)
}
