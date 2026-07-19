package config

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadEd25519PrivateKeyFile(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	path := writePKCS8Key(t, privateKey)

	loaded, err := loadEd25519PrivateKeyFile(path)
	if err != nil {
		t.Fatalf("loadEd25519PrivateKeyFile() error = %v", err)
	}
	derived := loaded.Public().(ed25519.PublicKey)
	if !derived.Equal(publicKey) {
		t.Fatal("derived public key does not match generated public key")
	}
}

func TestGetEnvInt(t *testing.T) {
	t.Setenv("CLAVE_TEST_INT", "")
	if got, err := getEnvInt("CLAVE_TEST_INT", 20); err != nil || got != 20 {
		t.Fatalf("default = %d, %v", got, err)
	}
	t.Setenv("CLAVE_TEST_INT", "42")
	if got, err := getEnvInt("CLAVE_TEST_INT", 20); err != nil || got != 42 {
		t.Fatalf("parsed = %d, %v", got, err)
	}
	for _, value := range []string{"-1", "invalid"} {
		t.Setenv("CLAVE_TEST_INT", value)
		if _, err := getEnvInt("CLAVE_TEST_INT", 20); err == nil {
			t.Fatalf("expected %q to fail", value)
		}
	}
}

func TestLoadEd25519PrivateKeyFileErrors(t *testing.T) {
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		path    func(*testing.T) string
		wantErr string
	}{
		{name: "missing path", path: func(*testing.T) string { return "" }, wantErr: "is required"},
		{name: "unreadable file", path: func(t *testing.T) string { return filepath.Join(t.TempDir(), "missing.pem") }, wantErr: "read LICENSE_JWT_PRIVATE_KEY_FILE"},
		{name: "malformed PEM", path: func(t *testing.T) string { return writeFile(t, []byte("not pem")) }, wantErr: "PEM-encoded PKCS#8"},
		{name: "wrong PEM type", path: func(t *testing.T) string { return writePEM(t, "ENCRYPTED PRIVATE KEY", []byte("data")) }, wantErr: "unsupported PEM block"},
		{name: "invalid PKCS8", path: func(t *testing.T) string { return writePEM(t, "PRIVATE KEY", []byte("data")) }, wantErr: "parse LICENSE_JWT_PRIVATE_KEY_FILE"},
		{name: "wrong key algorithm", path: func(t *testing.T) string { return writePKCS8Key(t, rsaKey) }, wantErr: "must contain an Ed25519 private key"},
		{name: "multiple PEM blocks", path: func(t *testing.T) string {
			path := writePKCS8Key(t, rsaKey)
			file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
			if err != nil {
				t.Fatal(err)
			}
			if err := pem.Encode(file, &pem.Block{Type: "PRIVATE KEY", Bytes: []byte("other")}); err != nil {
				t.Fatal(err)
			}
			if err := file.Close(); err != nil {
				t.Fatal(err)
			}
			return path
		}, wantErr: "exactly one PEM block"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := loadEd25519PrivateKeyFile(tt.path(t))
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want error containing %q", err, tt.wantErr)
			}
		})
	}
}

func writePKCS8Key(t *testing.T, key any) string {
	t.Helper()
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	return writePEM(t, "PRIVATE KEY", der)
}

func writePEM(t *testing.T, blockType string, der []byte) string {
	t.Helper()
	return writeFile(t, pem.EncodeToMemory(&pem.Block{Type: blockType, Bytes: der}))
}

func writeFile(t *testing.T, contents []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "key.pem")
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
