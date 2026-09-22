package parsemail

import (
	"strings"
	"testing"
)

func TestDKIMTestDomainDoesNotBypassVerification(t *testing.T) {
	// This signature is structurally invalid (From is not signed), so the
	// verifier rejects it without any external DNS lookup.
	message := "DKIM-Signature: v=1; a=rsa-sha256; d=test.domain; s=default; h=subject; bh=AAAA; b=AAAA\r\n" +
		"From: attacker@example.net\r\nSubject: forged\r\n\r\nbody"
	if Check(nil, strings.NewReader(message)) {
		t.Fatal("invalid signature from test.domain bypassed DKIM verification")
	}
}

func TestReaderMalformedHeadersReturnsNil(t *testing.T) {
	if got := NewEmailFromReader(nil, strings.NewReader("not a header\r\n\r\nbody"), 0); got != nil {
		t.Fatalf("malformed header returned email: %+v", got)
	}
}

func TestRecipientListKeepsQuotedNamesAndLocalParts(t *testing.T) {
	got := buildUsers([]string{`"Last, First" <first@example.com>, "a,b"@example.com`})
	if len(got) != 2 || got[0].EmailAddress != "first@example.com" || got[0].Name != "Last, First" || got[1].EmailAddress != "a,b@example.com" {
		t.Fatalf("quoted address list parsed incorrectly: %+v", got)
	}
}

func TestReaderMissingSenderFallsBackToFrom(t *testing.T) {
	got := NewEmailFromReader(nil, strings.NewReader("From: from@example.com\r\n\r\nbody"), 0)
	if got == nil || got.Sender == nil || got.Sender.EmailAddress != "from@example.com" {
		t.Fatalf("missing Sender did not fall back to From: %+v", got)
	}
}

func TestRelatedMIMEPreservesCIDOnlyForInlineImages(t *testing.T) {
	raw := "From: from@example.com\r\nTo: to@example.com\r\nContent-Type: multipart/related; boundary=related\r\n\r\n" +
		"--related\r\nContent-Type: text/html\r\n\r\n" +
		`<img src="cid:part1"><a href="cid:part1">link</a><img src="/api/del_email"><img src="javascript:alert(1)">` + "\r\n" +
		"--related\r\nContent-Type: image/png\r\nContent-ID: <part1>\r\nContent-Transfer-Encoding: base64\r\n\r\naGVsbG8=\r\n--related--\r\n"
	email := NewEmailFromReader([]string{"to@example.com"}, strings.NewReader(raw), len(raw))
	if email == nil || !strings.Contains(string(email.HTML), `src="cid:part1"`) {
		t.Fatalf("inline image stripped: %+v", email)
	}
	if len(email.Attachments) != 1 || email.Attachments[0].ContentID != "part1" {
		t.Fatalf("duplicated/missing related attachment: %+v", email.Attachments)
	}
	if strings.Contains(string(email.HTML), `href="cid:`) || strings.Contains(string(email.HTML), "/api/del_email") || strings.Contains(string(email.HTML), "javascript:") {
		t.Fatalf("unsafe URL passed sanitizer: %s", email.HTML)
	}
	display := SanitizeHTMLForDisplay(strings.ReplaceAll(string(email.HTML), "cid:part1", "/attachments/42/part1"))
	if !strings.Contains(display, `src="/attachments/42/part1"`) {
		t.Fatalf("display sanitizer removed generated attachment URL: %s", display)
	}
}
