package ssl

import (
	"fmt"
	"testing"

	"github.com/Jinnrry/pmail/config"
)

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
