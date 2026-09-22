package hooks

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Jinnrry/pmail/dto/parsemail"
	"github.com/Jinnrry/pmail/utils/context"
)

type auditResponseBody struct {
	io.Reader
	closed bool
}

func (b *auditResponseBody) Close() error { b.closed = true; return nil }

type auditRoundTripper func(*http.Request) (*http.Response, error)

func (f auditRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestAuditHookHTTPBodiesClosed(t *testing.T) {
	for _, name := range []string{"ReceiveSaveAfter", "SendBefore", "SendAfter", "ReceiveParseBefore", "ReceiveParseAfter", "GetName", "SettingsHtml"} {
		t.Run(name, func(t *testing.T) {
			body := &auditResponseBody{Reader: strings.NewReader("{}")}
			h := &HookSender{httpc: http.Client{Transport: auditRoundTripper(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: body}, nil
			})}}
			ctx, email := &context.Context{}, &parsemail.Email{}
			switch name {
			case "ReceiveSaveAfter":
				h.ReceiveSaveAfter(ctx, email, nil)
			case "SendBefore":
				h.SendBefore(ctx, email)
			case "SendAfter":
				h.SendAfter(ctx, email, nil)
			case "ReceiveParseBefore":
				raw := []byte("test")
				h.ReceiveParseBefore(ctx, &raw)
			case "ReceiveParseAfter":
				h.ReceiveParseAfter(ctx, email)
			case "GetName":
				h.GetName(ctx)
			case "SettingsHtml":
				h.SettingsHtml(ctx, "index.html", "")
			}
			if !body.closed {
				t.Fatal("HTTP response body was not closed")
			}
		})
	}
}
