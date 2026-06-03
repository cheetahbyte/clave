package services

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/hkdf"
	"crypto/sha256"
)

type EncryptionService struct {
	privateKey *ecdh.PrivateKey
}

func NewEncryptionService(privKeyBytes []byte) (*EncryptionService, error) {
	curve := ecdh.X25519()
	priv, err := curve.NewPrivateKey(privKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("invalid X25519 private key: %w", err)
	}
	return &EncryptionService{privateKey: priv}, nil
}

func (s *EncryptionService) PublicKeyBase64() string {
	return base64.RawURLEncoding.EncodeToString(s.privateKey.PublicKey().Bytes())
}

func (s *EncryptionService) SharedSecret(clientPubKeyBytes []byte) ([]byte, error) {
	curve := ecdh.X25519()
	clientPub, err := curve.NewPublicKey(clientPubKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("invalid client public key: %w", err)
	}
	rawSecret, err := s.privateKey.ECDH(clientPub)
	if err != nil {
		return nil, fmt.Errorf("ECDH failed: %w", err)
	}

	aesKey := make([]byte, 32)
	kdf := hkdf.New(sha256.New, rawSecret, nil, []byte("clave-v1"))
	if _, err := io.ReadFull(kdf, aesKey); err != nil {
		return nil, fmt.Errorf("HKDF failed: %w", err)
	}
	return aesKey, nil
}

func (s *EncryptionService) Encrypt(plaintext, aesKey []byte) ([]byte, error) {
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

func (s *EncryptionService) Decrypt(data, aesKey []byte) ([]byte, error) {
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(data) < gcm.NonceSize() {
		return nil, errors.New("ciphertext too short")
	}
	nonce, ciphertext := data[:gcm.NonceSize()], data[gcm.NonceSize():]
	return gcm.Open(nil, nonce, ciphertext, nil)
}
