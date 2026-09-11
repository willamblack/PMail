package setup

import (
	"fmt"
	"strings"

	"github.com/Jinnrry/pmail/config"
	"github.com/Jinnrry/pmail/i18n"
	"github.com/Jinnrry/pmail/services/auth"
	"github.com/Jinnrry/pmail/utils/context"
	"github.com/Jinnrry/pmail/utils/errors"
	"github.com/Jinnrry/pmail/utils/ip"
	"github.com/Jinnrry/pmail/utils/maildomain"
)

type DNSItem struct {
	Type     string `json:"type"`
	Host     string `json:"host"`
	Value    string `json:"value"`
	Priority int    `json:"priority,omitempty"`
	TTL      int    `json:"ttl"`
	Tips     string `json:"tips"`
}

func GetDNSSettings(ctx *context.Context) (map[string][]*DNSItem, error) {
	configData, err := config.ReadConfig()
	if err != nil {
		return nil, errors.Wrap(err)
	}

	serviceHostname, err := canonicalServiceHostname(configData)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	lang := ""
	if ctx != nil {
		lang = ctx.Lang
	}

	return buildDNSSettings(configData, serviceHostname, ip.GetIp(), auth.DkimGen(), lang), nil
}

func canonicalServiceHostname(configData *config.Config) (string, error) {
	for _, candidate := range []string{
		configData.OutboundHostname,
		configData.WebDomain,
		fmt.Sprintf("smtp.%s", configData.Domain),
	} {
		if strings.TrimSpace(candidate) == "" || candidate == "smtp." {
			continue
		}
		return maildomain.Normalize(candidate)
	}
	return "", errors.New("mail service hostname is required")
}

func buildDNSSettings(configData *config.Config, serviceHostname, serverIP, dkimPublicKey, lang string) map[string][]*DNSItem {
	roots := configData.Domains
	if len(roots) == 0 && configData.Domain != "" {
		roots = []string{configData.Domain}
	}

	ret := make(map[string][]*DNSItem, len(roots)+1)
	serviceRoot, serviceIsInConfiguredRoot := maildomain.MatchRoot(serviceHostname, roots, true)
	if !serviceIsInConfiguredRoot {
		ret[serviceHostname] = []*DNSItem{
			{
				Type:  "A",
				Host:  serviceHostname,
				Value: serverIP,
				TTL:   3600,
				Tips:  i18n.GetText(lang, "service_ip_tips"),
			},
		}
	}

	for _, rawRoot := range roots {
		root, err := maildomain.Normalize(rawRoot)
		if err != nil {
			continue
		}

		items := make([]*DNSItem, 0, 7)
		if serviceIsInConfiguredRoot && root == serviceRoot {
			items = append(items, &DNSItem{
				Type:  "A",
				Host:  relativeDNSHost(serviceHostname, root),
				Value: serverIP,
				TTL:   3600,
				Tips:  i18n.GetText(lang, "service_ip_tips"),
			})
		}

		items = append(items,
			&DNSItem{Type: "MX", Host: "@", Value: serviceHostname, Priority: 10, TTL: 3600, Tips: i18n.GetText(lang, "root_mx_tips")},
		)
		if configData.AcceptSubdomains {
			items = append(items,
				&DNSItem{Type: "MX", Host: "*", Value: serviceHostname, Priority: 10, TTL: 3600, Tips: i18n.GetText(lang, "wildcard_mx_tips")},
			)
		}

		items = append(items,
			&DNSItem{Type: "TXT", Host: "@", Value: "v=spf1 mx ~all", TTL: 3600, Tips: i18n.GetText(lang, "root_spf_tips")},
		)
		if configData.AcceptSubdomains {
			items = append(items,
				&DNSItem{Type: "TXT", Host: "*", Value: "v=spf1 mx ~all", TTL: 3600, Tips: i18n.GetText(lang, "wildcard_spf_tips")},
			)
		}

		items = append(items,
			&DNSItem{Type: "TXT", Host: "default._domainkey", Value: dkimPublicKey, TTL: 3600, Tips: i18n.GetText(lang, "dkim_tips")},
			&DNSItem{Type: "TXT", Host: "_dmarc", Value: "v=DMARC1; p=none; sp=none; adkim=r; aspf=r", TTL: 3600, Tips: i18n.GetText(lang, "dmarc_tips")},
		)

		ret[root] = items
	}

	return ret
}

func relativeDNSHost(host, root string) string {
	if host == root {
		return "@"
	}
	return strings.TrimSuffix(host, "."+root)
}
