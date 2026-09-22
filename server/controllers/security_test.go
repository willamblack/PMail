package controllers

import (
	"mime"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Jinnrry/pmail/utils/context"
)

func TestAttachmentHeadersNeverExecuteActiveContent(t *testing.T) {
	for _, typ := range []string{"text/html", "image/svg+xml", "application/xml", "text/html\r\nX-Test:yes"} {
		w := httptest.NewRecorder()
		setAttachmentHeaders(w, typ, "发票\r\nX-Test:yes.html", true)
		if w.Header().Get("Content-Type") != "application/octet-stream" ||
			w.Header().Get("Content-Security-Policy") != "sandbox; default-src 'none'" ||
			w.Header().Get("X-Content-Type-Options") != "nosniff" {
			t.Fatalf("unsafe headers: %v", w.Header())
		}
		disposition, params, err := mime.ParseMediaType(w.Header().Get("Content-Disposition"))
		if err != nil || disposition != "attachment" || strings.ContainsAny(params["filename"], "\r\n") {
			t.Fatalf("invalid download filename: %v %v", w.Header(), err)
		}
	}
	w := httptest.NewRecorder()
	setAttachmentHeaders(w, "image/png", "image.png", true)
	if w.Header().Get("Content-Type") != "image/png" || w.Header().Get("Content-Disposition") != "" {
		t.Fatal("inline image broken")
	}
}

func TestNegativeAttachmentIndexRejectedBeforeDatabase(t *testing.T) {
	w := httptest.NewRecorder()
	Download(&context.Context{}, w, httptest.NewRequest("GET", "/attachments/download/1/-1", nil))
	if w.Code != 404 {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestMalformedMutationsDoNotPanicOrReachDatabase(t *testing.T) {
	for _, handler := range []HandlerFunc{AddGroup, DelGroup, UpsertRule, DelRule, CreateUser, EditUser, ModifyPassword, Setup, Login} {
		for _, body := range []string{"null", "{", "[]"} {
			w := httptest.NewRecorder()
			handler(&context.Context{IsAdmin: true}, w, httptest.NewRequest("POST", "/api/test", strings.NewReader(body)))
			if !strings.Contains(w.Body.String(), `"errorNo":100`) {
				t.Fatalf("body=%q response=%s", body, w.Body)
			}
		}
	}
}
