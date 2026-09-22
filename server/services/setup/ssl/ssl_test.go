package ssl

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Jinnrry/pmail/config"
	"github.com/go-acme/lego/v4/certificate"
)

func TestSaveCertificateUsesConfiguredPaths(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	keyBytes, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{SerialNumber: big.NewInt(1), DNSNames: []string{"mail.example.com"}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour)}
	certBytes, err := x509.CreateCertificate(rand.Reader, template, template, key.Public(), key)
	if err != nil {
		t.Fatal(err)
	}
	resource := &certificate.Resource{PrivateKey: pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes}), Certificate: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes})}
	cfg := &config.Config{SSLPrivateKeyPath: filepath.Join(t.TempDir(), "keys/server.key"), SSLPublicKeyPath: filepath.Join(t.TempDir(), "certs/fullchain.crt")}
	if err = saveCertificate(cfg, resource); err != nil {
		t.Fatal(err)
	}
	if _, err = tls.LoadX509KeyPair(cfg.SSLPublicKeyPath, cfg.SSLPrivateKeyPath); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(cfg.SSLPrivateKeyPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("TLS private key permissions = %o", info.Mode().Perm())
	}
	resource.PrivateKey = []byte("invalid")
	if err = saveCertificate(cfg, resource); err == nil {
		t.Fatal("invalid certificate pair was saved")
	}
	if _, err = tls.LoadX509KeyPair(cfg.SSLPublicKeyPath, cfg.SSLPrivateKeyPath); err != nil {
		t.Fatalf("invalid renewal overwrote existing keypair: %v", err)
	}
}

func TestCheckSSLCrtInfo(t *testing.T) {
	config.Init()

	got, got1, match, err := CheckSSLCrtInfo()

	fmt.Println(got, got1, match, err)
}

func TestCertificateDomainsUsesExplicitTLSNames(t *testing.T) {
	cfg := &config.Config{
		WebDomain: "web.example.com",
		Domains:   []string{"one.example", "two.example"},
		TLSNames:  []string{" Mail.Example.COM. ", "mail.example.com", "web.example.com"},
	}
	want := []string{"mail.example.com", "web.example.com"}
	got := certificateDomains(cfg)
	if len(got) != len(want) {
		t.Fatalf("certificateDomains() = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("certificateDomains() = %#v, want %#v", got, want)
		}
	}
}
