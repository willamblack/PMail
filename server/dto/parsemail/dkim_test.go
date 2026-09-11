package parsemail

import (
	"strings"
	"testing"

	"github.com/Jinnrry/pmail/config"
)

func TestCheck(t *testing.T) {

	res := Check(nil, strings.NewReader(`Received: from jdl.ac.cn ([159.226.42.8])

xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
`))
	if res != false {
		t.Errorf("DKIM Error")
	}

}

func TestSigningDomainForAddressUsesMatchedRoot(t *testing.T) {
	oldConfig := config.Instance
	t.Cleanup(func() { config.Instance = oldConfig })
	config.Instance = &config.Config{
		Domain:           "primary.example",
		Domains:          []string{"primary.example", "117799.xyz"},
		AcceptSubdomains: true,
	}

	tests := map[string]string{
		"admin@117799.xyz":        "117799.xyz",
		"order@a.b.117799.xyz":    "117799.xyz",
		"admin@primary.example":   "primary.example",
		"admin@unconfigured.test": "primary.example",
	}
	for address, want := range tests {
		if got := signingDomainForAddress(address); got != want {
			t.Errorf("signingDomainForAddress(%q) = %q, want %q", address, got, want)
		}
	}
}
