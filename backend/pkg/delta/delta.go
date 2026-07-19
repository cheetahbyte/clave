package delta

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/kr/binarydist"
)

const (
	Schema    = "clave.delta/v1"
	Algorithm = "bsdiff"
)

type Engine interface {
	Algorithm() string
	Create(old, new io.Reader, patch io.Writer) error
	Apply(old io.Reader, patch io.Reader, output io.Writer) error
}

type BSDiffEngine struct{}

func (BSDiffEngine) Algorithm() string { return Algorithm }
func (BSDiffEngine) Create(old, new io.Reader, patch io.Writer) error {
	return binarydist.Diff(old, new, patch)
}
func (BSDiffEngine) Apply(old io.Reader, patch io.Reader, output io.Writer) error {
	return binarydist.Patch(old, output, patch)
}

type Contract struct {
	Schema       string `json:"schema"`
	Algorithm    string `json:"algorithm"`
	FromVersion  string `json:"fromVersion"`
	ToVersion    string `json:"toVersion"`
	BaseSHA256   string `json:"baseSha256"`
	TargetSHA256 string `json:"targetSha256"`
	PatchSHA256  string `json:"patchSha256"`
	TargetSize   int64  `json:"targetSize"`
}

func (c Contract) Validate() error {
	if c.Schema != Schema {
		return fmt.Errorf("unsupported delta schema %q", c.Schema)
	}
	if c.Algorithm != Algorithm {
		return fmt.Errorf("unsupported delta algorithm %q", c.Algorithm)
	}
	if c.FromVersion == "" || c.ToVersion == "" {
		return errors.New("delta versions are required")
	}
	if c.TargetSize <= 0 {
		return errors.New("target size must be positive")
	}
	for name, value := range map[string]string{"baseSha256": c.BaseSHA256, "targetSha256": c.TargetSHA256, "patchSha256": c.PatchSHA256} {
		raw, err := hex.DecodeString(value)
		if err != nil || len(raw) != 32 {
			return fmt.Errorf("%s must be a SHA-256 hex digest", name)
		}
	}
	return nil
}

func CanonicalContract(contract Contract) ([]byte, error) {
	if err := contract.Validate(); err != nil {
		return nil, err
	}
	canonical := struct {
		Algorithm    string `json:"algorithm"`
		BaseSHA256   string `json:"baseSha256"`
		FromVersion  string `json:"fromVersion"`
		PatchSHA256  string `json:"patchSha256"`
		Schema       string `json:"schema"`
		TargetSHA256 string `json:"targetSha256"`
		TargetSize   int64  `json:"targetSize"`
		ToVersion    string `json:"toVersion"`
	}{
		Algorithm: contract.Algorithm, BaseSHA256: contract.BaseSHA256,
		FromVersion: contract.FromVersion, PatchSHA256: contract.PatchSHA256,
		Schema: contract.Schema, TargetSHA256: contract.TargetSHA256,
		TargetSize: contract.TargetSize, ToVersion: contract.ToVersion,
	}
	return json.Marshal(canonical)
}

func signatureMessage(contract Contract) ([]byte, error) {
	raw, err := CanonicalContract(contract)
	if err != nil {
		return nil, err
	}
	message := make([]byte, 0, len(Schema)+1+len(raw))
	message = append(message, Schema...)
	message = append(message, 0)
	message = append(message, raw...)
	return message, nil
}

func SignContract(key ed25519.PrivateKey, contract Contract) (string, error) {
	if len(key) != ed25519.PrivateKeySize {
		return "", errors.New("invalid Ed25519 private key")
	}
	message, err := signatureMessage(contract)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(ed25519.Sign(key, message)), nil
}

func VerifyContract(key ed25519.PublicKey, contract Contract, encodedSignature string) error {
	if len(key) != ed25519.PublicKeySize {
		return errors.New("invalid Ed25519 public key")
	}
	signature, err := base64.StdEncoding.DecodeString(encodedSignature)
	if err != nil || len(signature) != ed25519.SignatureSize {
		return errors.New("invalid delta signature encoding")
	}
	message, err := signatureMessage(contract)
	if err != nil {
		return err
	}
	if !ed25519.Verify(key, message, signature) {
		return errors.New("invalid delta signature")
	}
	return nil
}
