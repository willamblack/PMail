package setup

import (
	"path/filepath"
	"testing"

	"github.com/Jinnrry/pmail/config"
)

func TestSetDomainSettingsEnablesSingleAdminCatchAllDefaults(t *testing.T) {
	oldRoot := config.ROOT_PATH
	config.ROOT_PATH = filepath.ToSlash(t.TempDir()) + "/"
	t.Cleanup(func() { config.ROOT_PATH = oldRoot })

	if err := SetDomainSettings(
		"117799.XYZ.",
		"MAIL.EXAMPLE-SERVER.COM.",
		"Example.NET, example.net ,ABC.ORG.",
	); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.ReadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Domain != "117799.xyz" || cfg.WebDomain != "mail.example-server.com" {
		t.Fatalf("normalized domains = primary:%q web:%q", cfg.Domain, cfg.WebDomain)
	}
	if len(cfg.Domains) != 3 {
		t.Fatalf("Domains = %#v, want three unique roots", cfg.Domains)
	}
	if !cfg.AcceptSubdomains || cfg.CatchAllAccount != "admin" {
		t.Fatalf("catch-all defaults = accept:%t account:%q", cfg.AcceptSubdomains, cfg.CatchAllAccount)
	}
	if cfg.OutboundHostname != cfg.WebDomain {
		t.Fatalf("OutboundHostname = %q, want %q", cfg.OutboundHostname, cfg.WebDomain)
	}
	if len(cfg.TLSNames) != 1 || cfg.TLSNames[0] != cfg.WebDomain {
		t.Fatalf("TLSNames = %#v, want [%q]", cfg.TLSNames, cfg.WebDomain)
	}
}
