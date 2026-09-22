package send

import (
	stdcontext "context"
	"errors"
	"fmt"
	"github.com/Jinnrry/pmail/dto/parsemail"
	"github.com/Jinnrry/pmail/utils/context"
	"net"
	"net/textproto"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
)

func TestTestDomainUsesOrdinaryDNSRouting(t *testing.T) {
	oldResolver := net.DefaultResolver
	t.Cleanup(func() { net.DefaultResolver = oldResolver })
	var lookups atomic.Int32
	net.DefaultResolver = &net.Resolver{PreferGo: true, Dial: func(stdcontext.Context, string, string) (net.Conn, error) {
		lookups.Add(1)
		return nil, errors.New("network disabled for DNS routing test")
	}}
	err, failures := doSend(&context.Context{}, "example.com", []byte("test body"), []*parsemail.User{{EmailAddress: "recipient@test.domain"}}, "sender@example.com")
	if err == nil {
		t.Fatal("blocked DNS unexpectedly delivered message")
	}
	if lookups.Load() == 0 {
		t.Fatal("test.domain bypassed ordinary DNS routing")
	}
	if _, ok := failures["test.domain"]; !ok {
		t.Fatalf("delivery reported a substituted domain: %v", failures)
	}
	if _, ok := failures["localhost"]; ok {
		t.Fatalf("test domain was routed to localhost: %v", failures)
	}
}

func TestDoSendRejectsEmptyOrInvalidRecipients(t *testing.T) {
	for _, recipients := range [][]*parsemail.User{nil, {nil}, {{EmailAddress: "not-an-address"}}} {
		if err, _ := doSend(&context.Context{}, "example.com", nil, recipients, "sender@example.com"); err == nil {
			t.Fatalf("invalid recipients returned success: %+v", recipients)
		}
	}
}

func TestSMTPDeliveryUsesOnlyEnvelopeRecipients(t *testing.T) {
	email := &parsemail.Email{
		To:         []*parsemail.User{{EmailAddress: "header-only@example.net"}},
		Cc:         []*parsemail.User{{EmailAddress: "header-cc@example.net"}},
		Bcc:        []*parsemail.User{{EmailAddress: "bcc@example.net"}},
		EnvelopeTo: []string{"envelope-only@example.net"},
	}
	got := deliveryRecipients(email)
	if len(got) != 1 || got[0].EmailAddress != "envelope-only@example.net" {
		t.Fatalf("SMTP recipients must not be expanded from headers: %+v", got)
	}
	email.EnvelopeTo = nil
	if got = deliveryRecipients(email); len(got) != 3 {
		t.Fatalf("web/API recipient behavior changed: %+v", got)
	}
	email.EnvelopeTo = []string{}
	if got = deliveryRecipients(email); len(got) != 0 {
		t.Fatalf("empty SMTP envelope expanded headers: %+v", got)
	}
}

func TestIsPermanentSMTPResponse(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil"},
		{name: "network error", err: errors.New("connection refused")},
		{name: "temporary SMTP response", err: &textproto.Error{Code: 451, Msg: "try later"}},
		{name: "wrapped temporary SMTP response", err: fmt.Errorf("delivery: %w", &textproto.Error{Code: 421, Msg: "service unavailable"})},
		{name: "permanent SMTP response", err: &textproto.Error{Code: 550, Msg: "mailbox unavailable"}, want: true},
		{name: "wrapped permanent SMTP response", err: fmt.Errorf("delivery: %w", &textproto.Error{Code: 554, Msg: "transaction failed"}), want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isPermanentSMTPResponse(tt.err); got != tt.want {
				t.Fatalf("isPermanentSMTPResponse() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDeliveryFailureCausePreservesTemporaryMXError(t *testing.T) {
	tests := []struct {
		name      string
		lookupErr *net.DNSError
	}{
		{
			name:      "SERVFAIL before fallback NXDOMAIN",
			lookupErr: &net.DNSError{Err: "server misbehaving", Name: "example.com", IsTemporary: true},
		},
		{
			name:      "timeout before fallback NXDOMAIN",
			lookupErr: &net.DNSError{Err: "i/o timeout", Name: "example.com", IsTimeout: true},
		},
	}

	fallbackErr := &net.OpError{
		Op:  "dial",
		Net: "tcp",
		Err: &net.DNSError{Err: "no such host", Name: "smtp.example.com", IsNotFound: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := deliveryFailureCause(tt.lookupErr, fallbackErr)

			var dnsErr *net.DNSError
			if !errors.As(err, &dnsErr) {
				t.Fatalf("deliveryFailureCause() = %T, want wrapped DNS error", err)
			}
			if dnsErr.IsNotFound {
				t.Fatalf("deliveryFailureCause() selected fallback NXDOMAIN: %v", dnsErr)
			}
			if !dnsErr.IsTemporary && !dnsErr.IsTimeout {
				t.Fatalf("deliveryFailureCause() selected non-temporary DNS error: %v", dnsErr)
			}
		})
	}
}

func TestDeliveryFailureCauseKeepsExplicitSMTPRejection(t *testing.T) {
	lookupErr := &net.DNSError{Err: "server misbehaving", Name: "example.com", IsTemporary: true}
	rejection := &textproto.Error{Code: 550, Msg: "mailbox unavailable"}

	if got := deliveryFailureCause(lookupErr, rejection); got != rejection {
		t.Fatalf("deliveryFailureCause() = %v, want explicit SMTP rejection %v", got, rejection)
	}
}

func TestDoSendGroupsRecipientsBeforeRotatingMXLookup(t *testing.T) {
	var lookups atomic.Int32
	lookup := func(domain string) ([]*net.MX, error) {
		if domain != "example.net" {
			t.Errorf("lookup domain = %q, want normalized example.net", domain)
		}
		if lookups.Add(1)%2 == 1 {
			return []*net.MX{{Host: "mx1.example.net.", Pref: 10}, {Host: "mx2.example.net.", Pref: 10}}, nil
		}
		return []*net.MX{{Host: "mx2.example.net.", Pref: 10}, {Host: "mx1.example.net.", Pref: 10}}, nil
	}
	var mu sync.Mutex
	var groups [][]string
	deliver := func(_ *context.Context, _, _, _, _ string, recipients []string, _ []byte) error {
		mu.Lock()
		groups = append(groups, append([]string(nil), recipients...))
		mu.Unlock()
		return nil
	}
	recipients := []*parsemail.User{{EmailAddress: "first@EXAMPLE.NET"}, {EmailAddress: "second@example.net."}}
	err, failures := doSendWithRouting(&context.Context{}, "example.com", nil, recipients, "sender@example.com", lookup, deliver)
	if err != nil || len(failures) != 0 {
		t.Fatalf("delivery failed: %v, %v", err, failures)
	}
	if lookups.Load() != 1 || len(groups) != 1 || !reflect.DeepEqual(groups[0], buildAddress(recipients)) {
		t.Fatalf("same domain split by rotating MX: lookups=%d groups=%v", lookups.Load(), groups)
	}
}

func TestDoSendKeepsTemporaryFailureForWholeDomain(t *testing.T) {
	var lookups atomic.Int32
	lookup := func(string) ([]*net.MX, error) {
		first, second := "mx1.example.net.", "mx2.example.net."
		if lookups.Add(1)%2 == 0 {
			first, second = second, first
		}
		return []*net.MX{{Host: first, Pref: 10}, {Host: second, Pref: 10}}, nil
	}
	var deliveries atomic.Int32
	temporary := &textproto.Error{Code: 451, Msg: "try later"}
	deliver := func(_ *context.Context, _, _, _, _ string, recipients []string, _ []byte) error {
		deliveries.Add(1)
		if len(recipients) != 2 {
			// Separate tasks could overwrite the domain's temporary failure with
			// a later permanent rejection for another recipient.
			return &textproto.Error{Code: 550, Msg: "split recipient rejected"}
		}
		return temporary
	}
	err, failures := doSendWithRouting(&context.Context{}, "example.com", nil,
		[]*parsemail.User{{EmailAddress: "first@example.net"}, {EmailAddress: "second@example.net"}},
		"sender@example.com", lookup, deliver)
	if err == nil || len(failures) != 1 || failures["example.net"] != temporary {
		t.Fatalf("temporary domain failure lost: error=%v failures=%v", err, failures)
	}
	if lookups.Load() != 1 || deliveries.Load() != 2 {
		t.Fatalf("want one lookup and both MX attempts, got lookups=%d deliveries=%d", lookups.Load(), deliveries.Load())
	}
}

func TestDoSendKeepsDifferentDomainsAndFailuresIndependent(t *testing.T) {
	lookups := map[string]int{}
	lookup := func(domain string) ([]*net.MX, error) {
		lookups[domain]++
		return []*net.MX{{Host: "mx1." + domain, Pref: 10}, {Host: "mx2." + domain, Pref: 20}}, nil
	}
	var mu sync.Mutex
	attempts := map[string]int{}
	permanent := &textproto.Error{Code: 550, Msg: "mailbox unavailable"}
	temporary := &textproto.Error{Code: 451, Msg: "try later"}
	deliver := func(_ *context.Context, _, domain, _, _ string, _ []string, _ []byte) error {
		mu.Lock()
		attempts[domain]++
		mu.Unlock()
		if domain == "example.net" {
			return permanent
		}
		return temporary
	}
	err, failures := doSendWithRouting(&context.Context{}, "example.com", nil,
		[]*parsemail.User{{EmailAddress: "first@example.net"}, {EmailAddress: "second@example.org"}},
		"sender@example.com", lookup, deliver)
	if err == nil || len(failures) != 2 || failures["example.net"] != permanent || failures["example.org"] != temporary {
		t.Fatalf("independent failures lost: error=%v failures=%v", err, failures)
	}
	if lookups["example.net"] != 1 || lookups["example.org"] != 1 || attempts["example.net"] != 1 || attempts["example.org"] != 2 {
		t.Fatalf("MX rejection/failover policy changed: lookups=%v attempts=%v", lookups, attempts)
	}
}

func TestDoSendValidatesAllRecipientsBeforeRouting(t *testing.T) {
	for _, invalid := range []*parsemail.User{nil, {EmailAddress: "invalid"}} {
		var lookups, deliveries atomic.Int32
		lookup := func(string) ([]*net.MX, error) {
			lookups.Add(1)
			return nil, nil
		}
		deliver := func(_ *context.Context, _, _, _, _ string, _ []string, _ []byte) error {
			deliveries.Add(1)
			return nil
		}
		err, failures := doSendWithRouting(&context.Context{}, "example.com", nil,
			[]*parsemail.User{{EmailAddress: "valid@example.net"}, invalid},
			"sender@example.com", lookup, deliver)
		if err == nil || len(failures) != 0 || lookups.Load() != 0 || deliveries.Load() != 0 {
			t.Fatalf("invalid list reached DNS/delivery: err=%v failures=%v lookups=%d deliveries=%d", err, failures, lookups.Load(), deliveries.Load())
		}
	}
}
