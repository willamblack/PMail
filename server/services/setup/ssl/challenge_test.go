package ssl

import (
	"sync"
	"testing"
)

func TestChallengeConcurrentAccess(t *testing.T) {
	httpChallenge := &HttpChallenge{}
	dnsChallenge := &DNSChallenge{}
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				httpChallenge.Present("example.com", "token", "value")
				httpChallenge.Lookup("token")
				httpChallenge.CleanUp("example.com", "token", "value")
				dnsChallenge.Present("example.com", "token", "value")
				dnsChallenge.GetDNSSettings(nil)
				dnsChallenge.CleanUp("example.com", "token", "value")
			}
		}()
	}
	wg.Wait()
	if _, ok := httpChallenge.Lookup("token"); ok {
		t.Fatal("HTTP token remained after cleanup")
	}
	if len(dnsChallenge.GetDNSSettings(nil)) != 0 {
		t.Fatal("DNS token remained after cleanup")
	}
}
