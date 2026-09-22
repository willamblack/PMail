package ssl

import (
	"github.com/Jinnrry/pmail/utils/context"
	"github.com/go-acme/lego/v4/challenge/dns01"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

type authInfo struct {
	Domain  string
	Token   string
	KeyAuth string
}

type HttpChallenge struct {
	mu       sync.RWMutex
	authInfo map[string]*authInfo
}

var instance = &HttpChallenge{authInfo: map[string]*authInfo{}}

func (h *HttpChallenge) Present(domain, token, keyAuth string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.authInfo == nil {
		h.authInfo = map[string]*authInfo{}
	}
	h.authInfo[token] = &authInfo{
		Domain:  domain,
		Token:   token,
		KeyAuth: keyAuth,
	}

	return nil
}

func (h *HttpChallenge) CleanUp(domain, token, keyAuth string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.authInfo, token)
	return nil
}

func GetHttpChallengeInstance() *HttpChallenge {
	return instance
}

func (h *HttpChallenge) Lookup(token string) (string, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	info, ok := h.authInfo[token]
	if !ok {
		return "", false
	}
	return info.KeyAuth, true
}

type DNSChallenge struct {
	mu       sync.RWMutex
	authInfo map[string]*authInfo
}

var dnsInstance = &DNSChallenge{authInfo: map[string]*authInfo{}}

func GetDnsChallengeInstance() *DNSChallenge {
	return dnsInstance
}

func (h *DNSChallenge) Present(domain, token, keyAuth string) error {
	info := dns01.GetChallengeInfo(domain, keyAuth)
	log.Infof("Presenting challenge Info : %+v", info)
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.authInfo == nil {
		h.authInfo = map[string]*authInfo{}
	}
	h.authInfo[token] = &authInfo{
		Domain:  info.FQDN,
		Token:   token,
		KeyAuth: info.Value,
	}
	log.Infof("SSL Log:%s %s %s", domain, token, keyAuth)
	return nil
}

func (h *DNSChallenge) CleanUp(domain, token, keyAuth string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.authInfo, token)
	return nil
}

func (h *DNSChallenge) Timeout() (timeout, interval time.Duration) {
	return 60 * time.Minute, 5 * time.Second
}

type DNSItem struct {
	Type  string `json:"type"`
	Host  string `json:"host"`
	Value string `json:"value"`
	TTL   int    `json:"ttl"`
	Tips  string `json:"tips"`
}

func (h *DNSChallenge) GetDNSSettings(ctx *context.Context) []*DNSItem {
	ret := []*DNSItem{}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, info := range h.authInfo {
		ret = append(ret, &DNSItem{
			Type:  "TXT",
			Host:  info.Domain,
			Value: info.KeyAuth,
			TTL:   3600,
		})
	}

	return ret
}
