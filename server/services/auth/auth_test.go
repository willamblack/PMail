package auth

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
)

func TestDKIMGenerationPreservesPrivateKeyWhenPublicFileMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "custom", "signing.key")
	public, err := DkimGen(path)
	if err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	block, _ := pem.Decode(original)
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil || key.(*rsa.PrivateKey).N.BitLen() != 2048 {
		t.Fatal("new DKIM key must be RSA-2048")
	}
	if err = os.Remove(filepath.Join(filepath.Dir(path), "dkim.public")); err != nil {
		t.Fatal(err)
	}
	again, err := DkimGen(path)
	if err != nil || again != public {
		t.Fatalf("public changed: %v", err)
	}
	current, _ := os.ReadFile(path)
	if !bytes.Equal(original, current) {
		t.Fatal("private key rotated unexpectedly")
	}
}

func TestDKIMInvalidPrivateKeyIsNotOverwritten(t *testing.T) {
	path := filepath.Join(t.TempDir(), "invalid.key")
	original := []byte("not a private key")
	if err := os.WriteFile(path, original, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := DkimGen(path); err == nil {
		t.Fatal("invalid key accepted")
	}
	current, _ := os.ReadFile(path)
	if !bytes.Equal(original, current) {
		t.Fatal("invalid key was overwritten")
	}
}

func TestDKIMRejectsPrivatePublicPathCollision(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dkim.public")
	if _, err := DkimGen(path); err == nil {
		t.Fatal("accepted colliding private/public paths")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("created a private key at public output path")
	}
}
