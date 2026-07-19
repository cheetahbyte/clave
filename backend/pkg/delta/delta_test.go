package delta

import (
	"archive/zip"
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"testing"
	"time"
)

func TestBSDiffRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		old  []byte
		new  []byte
	}{
		{name: "binary", old: bytes.Repeat([]byte{0, 1, 2, 3}, 1024), new: append(bytes.Repeat([]byte{0, 1, 2, 3}, 900), []byte("changed-tail")...)},
		{name: "zip", old: deterministicZIP(t, "version=1\n"), new: deterministicZIP(t, "version=2\n")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var patch bytes.Buffer
			engine := BSDiffEngine{}
			if err := engine.Create(bytes.NewReader(tt.old), bytes.NewReader(tt.new), &patch); err != nil {
				t.Fatal(err)
			}
			var got bytes.Buffer
			if err := engine.Apply(bytes.NewReader(tt.old), bytes.NewReader(patch.Bytes()), &got); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got.Bytes(), tt.new) {
				t.Fatal("patched output differs from target")
			}
		})
	}
}

func TestBSDiffRejectsCorruptPatch(t *testing.T) {
	var got bytes.Buffer
	if err := (BSDiffEngine{}).Apply(bytes.NewReader([]byte("old")), bytes.NewReader([]byte("not a patch")), &got); err == nil {
		t.Fatal("expected corrupt patch error")
	}
}

func TestContractSignature(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	contract := testContract()
	signature, err := SignContract(privateKey, contract)
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyContract(publicKey, contract, signature); err != nil {
		t.Fatal(err)
	}
	contract.TargetSize++
	if err := VerifyContract(publicKey, contract, signature); err == nil {
		t.Fatal("expected tampered contract to fail verification")
	}
}

func TestCanonicalContractUsesDocumentedKeyOrder(t *testing.T) {
	raw, err := CanonicalContract(testContract())
	if err != nil {
		t.Fatal(err)
	}
	wantPrefix := `{"algorithm":"bsdiff","baseSha256":`
	if !bytes.HasPrefix(raw, []byte(wantPrefix)) {
		t.Fatalf("canonical contract = %s", raw)
	}
}

func testContract() Contract {
	hash := sha256.Sum256([]byte("content"))
	digest := fmt.Sprintf("%x", hash)
	return Contract{Schema: Schema, Algorithm: Algorithm, FromVersion: "1.0.0", ToVersion: "1.1.0", BaseSHA256: digest, TargetSHA256: digest, PatchSHA256: digest, TargetSize: 7}
}

func deterministicZIP(t *testing.T, contents string) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	header := &zip.FileHeader{Name: "app/version.txt", Method: zip.Deflate}
	header.SetModTime(time.Unix(0, 0).UTC())
	header.SetMode(0o644)
	f, err := w.CreateHeader(header)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write([]byte(contents)); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
