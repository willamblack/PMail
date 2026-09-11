package setup

import (
	"github.com/Jinnrry/pmail/config"
	"github.com/Jinnrry/pmail/utils/array"
	"github.com/Jinnrry/pmail/utils/errors"
	"github.com/Jinnrry/pmail/utils/maildomain"
	"strings"
)

func GetDomainSettings() (string, string, []string, error) {
	configData, err := config.ReadConfig()
	if err != nil {
		return "", "", []string{}, errors.Wrap(err)
	}

	return configData.Domain, configData.WebDomain, array.Difference(configData.Domains, []string{configData.Domain}), nil
}

func SetDomainSettings(smtpDomain, webDomain, multiDomains string) error {
	configData, err := config.ReadConfig()
	if err != nil {
		return errors.Wrap(err)
	}

	normalizedSMTPDomain, err := maildomain.Normalize(smtpDomain)
	if err != nil {
		return errors.New("invalid smtp domain")
	}

	normalizedWebDomain, err := maildomain.Normalize(webDomain)
	if err != nil {
		return errors.New("invalid web domain")
	}

	configData.Domains = []string{}

	if multiDomains != "" {
		domains := strings.Split(multiDomains, ",")
		for _, rawDomain := range domains {
			domain, normalizeErr := maildomain.Normalize(rawDomain)
			if normalizeErr != nil {
				return errors.New("invalid additional domain")
			}
			if !array.InArray(domain, configData.Domains) {
				configData.Domains = append(configData.Domains, domain)
			}
		}
	}

	if !array.InArray(normalizedSMTPDomain, configData.Domains) {
		configData.Domains = append(configData.Domains, normalizedSMTPDomain)
	}

	configData.Domain = normalizedSMTPDomain
	configData.WebDomain = normalizedWebDomain
	if configData.CatchAllAccount == "" {
		configData.CatchAllAccount = "admin"
	}
	configData.AcceptSubdomains = true
	if configData.OutboundHostname == "" {
		configData.OutboundHostname = normalizedWebDomain
	}
	if len(configData.TLSNames) == 0 {
		configData.TLSNames = []string{normalizedWebDomain}
	}

	// 检查域名是否指向本机 todo

	err = config.WriteConfig(configData)
	if err != nil {
		return errors.Wrap(err)
	}
	return nil
}
