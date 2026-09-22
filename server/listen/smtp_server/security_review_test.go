package smtp_server

import (
	"net"
	"net/textproto"
	"strings"
	"testing"
	"time"

	"github.com/Jinnrry/pmail/utils/context"
)

func TestSubmissionRejectsPlaintextAuthentication(t *testing.T) {
	server := newSMTPServer(":587")
	if server.AllowInsecureAuth {
		t.Fatal("submission server permits plaintext AUTH")
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	go server.Serve(listener)
	defer server.Close()
	conn, err := net.DialTimeout("tcp", listener.Addr().String(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(2 * time.Second))
	client := textproto.NewConn(conn)
	if _, _, err = client.ReadResponse(220); err != nil {
		t.Fatal(err)
	}
	if err = client.PrintfLine("EHLO client.example"); err != nil {
		t.Fatal(err)
	}
	_, capabilities, err := client.ReadResponse(250)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(capabilities, "AUTH") {
		t.Fatalf("plaintext EHLO advertises AUTH: %s", capabilities)
	}
	if err = client.PrintfLine("AUTH PLAIN AGFkbWluAHBhc3M="); err != nil {
		t.Fatal(err)
	}
	code, _, _ := client.ReadResponse(0)
	if code != 523 {
		t.Fatalf("plaintext AUTH status = %d, want go-smtp TLS-required response 523", code)
	}
}

func TestMalformedDataRejectedWithoutPanic(t *testing.T) {
	session := &Session{Ctx: &context.Context{}}
	assertSMTPCode(t, session.Data(strings.NewReader("not a header\r\n\r\nbody")), 554)
}

func TestSPFIdentityUsesEnvelopeAndNullSenderHelo(t *testing.T) {
	for _, test := range []struct {
		from, helo, domain, sender string
		ok                         bool
	}{
		{"bounce@EXAMPLE.COM", "mx.unrelated.test", "example.com", "bounce@EXAMPLE.COM", true},
		{"", "MX.EXAMPLE.COM.", "mx.example.com", "postmaster@mx.example.com", true},
		{"", "", "", "", false},
		{"bad-address", "mx.example.com", "", "bad-address", false},
	} {
		domain, sender, ok := spfIdentity(test.from, test.helo)
		if domain != test.domain || sender != test.sender || ok != test.ok {
			t.Fatalf("spfIdentity(%q, %q) = %q, %q, %v", test.from, test.helo, domain, sender, ok)
		}
	}
	if spfCheck("invalid remote", "bounce@example.com", "mx.example.com") {
		t.Fatal("invalid remote passed SPF")
	}
}
