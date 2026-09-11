package setup

import (
	"testing"

	"github.com/Jinnrry/pmail/config"
)

func TestBuildDNSSettingsUsesCanonicalHostAndWildcardRecords(t *testing.T) {
	cfg := &config.Config{
		Domain:           "117799.xyz",
		Domains:          []string{"117799.xyz", "Example.NET."},
		AcceptSubdomains: true,
	}

	records := buildDNSSettings(cfg, "mail.117799.xyz", "192.0.2.10", "v=DKIM1; k=rsa; p=PUBLIC", "en")
	primary := records["117799.xyz"]
	secondary := records["example.net"]

	assertDNSRecord(t, primary, "A", "mail", "192.0.2.10")
	assertDNSRecord(t, primary, "MX", "@", "mail.117799.xyz")
	assertDNSRecord(t, primary, "MX", "*", "mail.117799.xyz")
	assertDNSPriority(t, primary, "@", 10)
	assertDNSPriority(t, primary, "*", 10)
	assertDNSRecord(t, primary, "TXT", "@", "v=spf1 mx ~all")
	assertDNSRecord(t, primary, "TXT", "*", "v=spf1 mx ~all")
	assertDNSRecord(t, primary, "TXT", "default._domainkey", "v=DKIM1; k=rsa; p=PUBLIC")
	assertDNSRecord(t, primary, "TXT", "_dmarc", "v=DMARC1; p=none; sp=none; adkim=r; aspf=r")

	assertDNSRecord(t, secondary, "MX", "@", "mail.117799.xyz")
	assertDNSRecord(t, secondary, "MX", "*", "mail.117799.xyz")
	assertNoDNSRecord(t, secondary, "A", "mail")
}

func TestBuildDNSSettingsOmitsWildcardRecordsWhenSubdomainsDisabled(t *testing.T) {
	cfg := &config.Config{
		Domain:           "example.com",
		Domains:          []string{"example.com"},
		AcceptSubdomains: false,
	}

	records := buildDNSSettings(cfg, "mail.example.com", "192.0.2.10", "dkim", "en")
	items := records["example.com"]

	assertDNSRecord(t, items, "MX", "@", "mail.example.com")
	assertNoDNSRecord(t, items, "MX", "*")
	assertNoDNSRecord(t, items, "TXT", "*")
}

func TestBuildDNSSettingsShowsExternalServiceHostnameOnce(t *testing.T) {
	cfg := &config.Config{
		Domain:           "example.com",
		Domains:          []string{"example.com", "example.net"},
		AcceptSubdomains: true,
	}

	records := buildDNSSettings(cfg, "mail.service.test", "192.0.2.10", "dkim", "en")
	serviceRecords := records["mail.service.test"]
	assertDNSRecord(t, serviceRecords, "A", "mail.service.test", "192.0.2.10")
	assertNoDNSRecord(t, records["example.com"], "A", "mail.service.test")
	assertNoDNSRecord(t, records["example.net"], "A", "mail.service.test")
}

func TestCanonicalServiceHostnameFallbacks(t *testing.T) {
	tests := []struct {
		name string
		cfg  *config.Config
		want string
	}{
		{name: "explicit outbound hostname", cfg: &config.Config{OutboundHostname: "MAIL.SERVICE.TEST."}, want: "mail.service.test"},
		{name: "web domain", cfg: &config.Config{WebDomain: "Web.Example.COM."}, want: "web.example.com"},
		{name: "legacy smtp hostname", cfg: &config.Config{Domain: "Example.NET."}, want: "smtp.example.net"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := canonicalServiceHostname(tt.cfg)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("canonicalServiceHostname() = %q, want %q", got, tt.want)
			}
		})
	}
}

func assertDNSRecord(t *testing.T, items []*DNSItem, recordType, host, value string) {
	t.Helper()
	for _, item := range items {
		if item.Type == recordType && item.Host == host && item.Value == value {
			return
		}
	}
	t.Fatalf("record %s %s %q not found in %#v", recordType, host, value, items)
}

func assertNoDNSRecord(t *testing.T, items []*DNSItem, recordType, host string) {
	t.Helper()
	for _, item := range items {
		if item.Type == recordType && item.Host == host {
			t.Fatalf("unexpected record %s %s found: %#v", recordType, host, item)
		}
	}
}

func assertDNSPriority(t *testing.T, items []*DNSItem, host string, priority int) {
	t.Helper()
	for _, item := range items {
		if item.Type == "MX" && item.Host == host {
			if item.Priority != priority {
				t.Fatalf("MX %s priority = %d, want %d", host, item.Priority, priority)
			}
			return
		}
	}
	t.Fatalf("MX %s not found", host)
}
