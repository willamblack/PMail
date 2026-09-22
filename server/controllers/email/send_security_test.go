package email

import (
	"testing"

	"github.com/Jinnrry/pmail/config"
	"github.com/Jinnrry/pmail/utils/context"
)

func TestValidateSendRequest(t *testing.T) {
	old := config.Instance
	config.Instance = &config.Config{Domains: []string{"example.com"}, AcceptSubdomains: true}
	t.Cleanup(func() { config.Instance = old })
	for _, tc := range []struct {
		name, from, sender, subject string
		admin, valid                bool
	}{
		{"own identity", "alice@example.com", "", "hello", false, true},
		{"admin dynamic", "order@a.b.example.com", "bounce@example.com", "中文", true, true},
		{"foreign envelope sender", "alice@example.com", "victim@evil.org", "hello", true, false},
		{"other user sender", "alice@example.com", "admin@example.com", "hello", false, false},
		{"suffix confusion", "a@evilexample.com", "", "hello", true, false},
		{"subject injection", "alice@example.com", "", "hello\r\nBcc: victim@evil.org", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := validateSendRequest(&context.Context{UserAccount: "alice", IsAdmin: tc.admin}, &sendRequest{
				From: user{Email: tc.from}, Sender: user{Email: tc.sender}, Subject: tc.subject,
				To: []user{{Email: "receiver@outside.net"}},
			})
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v err=%v", tc.valid, err)
			}
		})
	}
}

func TestDecodeAttachment(t *testing.T) {
	for _, input := range []string{"", "hello", "data:text/plain", "data:text/plain;base64,%%%", "data:text/plain\r\nX-Test:yes;base64,QQ=="} {
		if _, _, err := decodeAttachment(input); err == nil {
			t.Fatalf("accepted %q", input)
		}
	}
	typ, body, err := decodeAttachment("data:text/plain;base64,SGVsbG8=")
	if err != nil || typ != "text/plain" || string(body) != "Hello" {
		t.Fatalf("%q %q %v", typ, body, err)
	}
}

func TestValidateSendRequestCanonicalMailbox(t *testing.T) {
	old := config.Instance
	config.Instance = &config.Config{Domains: []string{"example.com"}, AcceptSubdomains: true}
	t.Cleanup(func() { config.Instance = old })
	for _, tc := range []struct{ input, want string }{
		{"<alice@EXAMPLE.COM>", "alice@example.com"},
		{`"quoted@local"@EXAMPLE.COM`, `"quoted@local"@example.com`},
		{"user@例子.中国", "user@xn--fsqu00a.xn--fiqs8s"},
	} {
		r := &sendRequest{From: user{Email: "admin@example.com"}, To: []user{{Email: tc.input}}}
		if err := validateSendRequest(&context.Context{IsAdmin: true}, r); err != nil {
			t.Fatalf("%q: %v", tc.input, err)
		}
		if r.To[0].Email != tc.want {
			t.Fatalf("canonical %q = %q, want %q", tc.input, r.To[0].Email, tc.want)
		}
	}
	for _, input := range []string{"a b@example.com", "a@example.com, b@example.com", "Display <a@example.com>", "a@example.com\r\nBcc: b@example.com"} {
		r := &sendRequest{From: user{Email: "admin@example.com"}, To: []user{{Email: input}}}
		if err := validateSendRequest(&context.Context{IsAdmin: true}, r); err == nil {
			t.Fatalf("invalid mailbox accepted: %q", input)
		}
	}
}
