package maildomain

import (
	"errors"
	"strings"

	"golang.org/x/net/idna"
)

var lookupProfile = idna.New(
	idna.MapForLookup(),
	idna.StrictDomainName(true),
	idna.ValidateLabels(true),
	idna.VerifyDNSLength(true),
)

// Normalize converts a mail domain to a lower-case DNS A-label without a
// trailing root dot. Wildcards are deliberately rejected: configured values
// are roots, not DNS wildcard owner names.
func Normalize(raw string) (string, error) {
	domain := strings.TrimSpace(raw)
	domain = strings.TrimSuffix(domain, ".")
	if domain == "" || strings.Contains(domain, "*") {
		return "", errors.New("invalid mail domain")
	}

	ascii, err := lookupProfile.ToASCII(domain)
	if err != nil {
		return "", err
	}
	return strings.ToLower(ascii), nil
}

// SplitAddress separates an SMTP address at its final @. The SMTP server
// library has already performed envelope syntax parsing before Session hooks;
// using the final @ also keeps quoted local-parts containing @ intact.
func SplitAddress(raw string) (localPart, domain string, err error) {
	address := strings.TrimSpace(raw)
	at := strings.LastIndexByte(address, '@')
	if at <= 0 || at == len(address)-1 {
		return "", "", errors.New("invalid mail address")
	}

	domain, err = Normalize(address[at+1:])
	if err != nil {
		return "", "", err
	}
	return address[:at], domain, nil
}

func EqualAddress(left, right string) bool {
	leftLocal, leftDomain, leftErr := SplitAddress(left)
	rightLocal, rightDomain, rightErr := SplitAddress(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	return leftLocal == rightLocal && leftDomain == rightDomain
}

// MatchRoot returns the longest configured root matching rawDomain. A
// subdomain match always requires a dot label boundary, so evil-example.com
// never matches example.com.
func MatchRoot(rawDomain string, configuredRoots []string, acceptSubdomains bool) (string, bool) {
	domain, err := Normalize(rawDomain)
	if err != nil {
		return "", false
	}

	best := ""
	for _, rawRoot := range configuredRoots {
		root, normalizeErr := Normalize(rawRoot)
		if normalizeErr != nil {
			continue
		}
		if domain == root || (acceptSubdomains && strings.HasSuffix(domain, "."+root)) {
			if len(root) > len(best) {
				best = root
			}
		}
	}
	return best, best != ""
}
