package parsemail

import (
	"fmt"
	stdmail "net/mail"
	"strings"
	"testing"
	"time"
)

// v2.9.1 fixed explicitly labelled attachments. Named inline parts and the
// legacy Content-Type name parameter must not replace the actual body either.
func TestNamedTextPartsDoNotReplaceBody(t *testing.T) {
	for _, mediaType := range []string{"text/plain", "text/html"} {
		for _, tc := range []struct {
			name        string
			contentType string
			disposition string
			filename    string
		}{
			{"attachment", mediaType, `attachment; filename="notes.txt"`, "notes.txt"},
			{"inline filename", mediaType, `inline; filename="notes.txt"`, "notes.txt"},
			{"legacy name", mediaType + `; name="notes.txt"`, "", "notes.txt"},
			{"encoded filename", mediaType, `inline; filename*=utf-8''%E6%96%87%E4%BB%B6.txt`, "文件.txt"},
			{"disposition filename wins", mediaType + `; name="old.txt"`, `attachment; filename="notes.txt"`, "notes.txt"},
		} {
			t.Run(mediaType+"/"+tc.name, func(t *testing.T) {
				raw := "From: sender@example.com\r\nTo: admin@example.net\r\n" +
					"MIME-Version: 1.0\r\nContent-Type: multipart/mixed; boundary=test-boundary\r\n\r\n" +
					"--test-boundary\r\nContent-Type: " + mediaType + "\r\n\r\noriginal body\r\n" +
					"--test-boundary\r\nContent-Type: " + tc.contentType + "\r\n"
				if tc.disposition != "" {
					raw += "Content-Disposition: " + tc.disposition + "\r\n"
				}
				raw += "\r\nattachment content\r\n--test-boundary--\r\n"
				email := NewEmailFromReader(nil, strings.NewReader(raw), len(raw))
				if email == nil {
					t.Fatal("message could not be parsed")
				}
				body := email.Text
				if mediaType == "text/html" {
					body = email.HTML
				}
				if string(body) != "original body" {
					t.Errorf("body overwritten: %q", body)
				}
				if len(email.Attachments) != 1 {
					t.Fatalf("got %d attachments, want 1", len(email.Attachments))
				}
				if got := email.Attachments[0]; got.Filename != tc.filename || string(got.Content) != "attachment content" {
					t.Errorf("attachment lost or renamed: %+v", got)
				}
			})
		}
	}
}

func TestInlineBodyIsNotAttachmentBySubstring(t *testing.T) {
	raw := "From: sender@example.com\r\nContent-Type: text/plain\r\n" +
		"Content-Disposition: inline; description=attachment\r\n\r\noriginal body"
	email := NewEmailFromReader(nil, strings.NewReader(raw), len(raw))
	if email == nil || string(email.Text) != "original body" || len(email.Attachments) != 0 {
		t.Fatalf("inline body incorrectly classified as attachment: %+v", email)
	}
}

func TestNamedInlineBodyRemainsVisible(t *testing.T) {
	for _, mediaType := range []string{"text/plain", "text/html"} {
		for _, multipart := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/multipart=%v", mediaType, multipart), func(t *testing.T) {
				part := "Content-Type: " + mediaType + "; name=body.txt\r\n" +
					"Content-Disposition: inline; filename=body.txt\r\n\r\noriginal body"
				raw := "From: sender@example.com\r\n"
				if multipart {
					raw += "Content-Type: multipart/alternative; boundary=alt\r\n\r\n--alt\r\n" + part + "\r\n--alt--\r\n"
				} else {
					raw += part
				}
				email := NewEmailFromReader(nil, strings.NewReader(raw), len(raw))
				if email == nil {
					t.Fatal("message could not be parsed")
				}
				body := email.Text
				if mediaType == "text/html" {
					body = email.HTML
				}
				if string(body) != "original body" || len(email.Attachments) != 0 {
					t.Fatalf("named inline body became an attachment: %+v", email)
				}
			})
		}
	}
}

func TestNamedHTMLFileDoesNotHidePlainBody(t *testing.T) {
	raw := "From: sender@example.com\r\nContent-Type: multipart/mixed; boundary=mixed\r\n\r\n" +
		"--mixed\r\nContent-Type: text/plain\r\n\r\noriginal body\r\n" +
		"--mixed\r\nContent-Type: text/html; name=notes.html\r\n\r\n<p>file content</p>\r\n--mixed--\r\n"
	email := NewEmailFromReader(nil, strings.NewReader(raw), len(raw))
	if email == nil || string(email.Text) != "original body" || len(email.HTML) != 0 || len(email.Attachments) != 1 {
		t.Fatalf("HTML file replaced the plain body: %+v", email)
	}
}

func TestMailDateSyntaxPreservesInstant(t *testing.T) {
	for _, date := range []string{
		"Sun, 9 Aug 2026 11:51:08 +0800",
		"9 Aug 2026 11:51:08 +0800",
		"Sun, 09 Aug 2026 11:51:08 GMT",
		"Sun, 09 Aug 2026 11:51:08 +0800 (CST)",
	} {
		t.Run(date, func(t *testing.T) {
			want, err := stdmail.ParseDate(date)
			if err != nil {
				t.Fatal(err)
			}
			raw := fmt.Sprintf("From: sender@example.com\r\nTo: admin@example.net\r\nDate: %s\r\nContent-Type: text/plain\r\n\r\nbody", date)
			email := NewEmailFromReader(nil, strings.NewReader(raw), len(raw))
			if email == nil {
				t.Fatal("message could not be parsed")
			}
			got, err := time.Parse(time.RFC3339, email.Date)
			if err != nil || !got.Equal(want) {
				t.Fatalf("Date replaced: got %q, want instant %s; error=%v", email.Date, want, err)
			}
			if rebuilt := builtMessageDate(t, email.BuildBytes(nil, false)); !rebuilt.Equal(want) {
				t.Fatalf("rebuilt Date changed: got %s, want %s", rebuilt, want)
			}
		})
	}
}
