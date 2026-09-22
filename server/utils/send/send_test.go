package send

import (
	stdcontext "context"
	"errors"
	"fmt"
	"github.com/Jinnrry/pmail/dto/parsemail"
	"github.com/Jinnrry/pmail/utils/context"
	"net"
	"net/textproto"
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
