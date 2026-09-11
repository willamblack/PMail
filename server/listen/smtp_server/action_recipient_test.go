package smtp_server

import (
	"errors"
	"testing"

	"github.com/Jinnrry/pmail/config"
	pmailcontext "github.com/Jinnrry/pmail/utils/context"
	smtp "github.com/emersion/go-smtp"
)

func TestRcptAcceptsConfiguredRootAndSubdomains(t *testing.T) {
	restoreRecipientTestConfig(t)
	session := &Session{Ctx: &pmailcontext.Context{}}

	for _, recipient := range []string{
		"user@117799.xyz",
		"random@117799.xyz",
		"abc@a.117799.xyz",
		"abc@a.b.117799.xyz",
		"abc@a.b.c.117799.xyz",
		"user@A.117799.XYZ.",
	} {
		if err := session.Rcpt(recipient, nil); err != nil {
			t.Errorf("Rcpt(%q) error = %v", recipient, err)
		}
	}
}

func TestRcptRejectsUnconfiguredDomains(t *testing.T) {
	restoreRecipientTestConfig(t)
	session := &Session{Ctx: &pmailcontext.Context{}}

	for _, recipient := range []string{
		"user@gmail.com",
		"user@evil.com",
		"user@evil117799.xyz",
		"user@117799.xyz.evil.com",
	} {
		err := session.Rcpt(recipient, nil)
		var smtpErr *smtp.SMTPError
		if !errors.As(err, &smtpErr) || smtpErr.Code != 550 {
			t.Errorf("Rcpt(%q) error = %v, want SMTP 550", recipient, err)
		}
	}
}

func TestAuthenticatedRcptMayTargetExternalDomain(t *testing.T) {
	restoreRecipientTestConfig(t)
	session := &Session{Ctx: &pmailcontext.Context{UserID: 1, UserAccount: "admin", IsAdmin: true}}
	if err := session.Rcpt("recipient@example.net", nil); err != nil {
		t.Fatalf("authenticated Rcpt() error = %v", err)
	}
}

func TestAdminSenderIsLimitedToConfiguredRoots(t *testing.T) {
	restoreRecipientTestConfig(t)
	session := &Session{Ctx: &pmailcontext.Context{UserID: 1, UserAccount: "admin", IsAdmin: true}}

	for _, sender := range []string{"admin@117799.xyz", "order-1@shop.117799.xyz"} {
		if err := session.Mail(sender, nil); err != nil {
			t.Errorf("Mail(%q) error = %v", sender, err)
		}
	}

	err := session.Mail("admin@evil.com", nil)
	var smtpErr *smtp.SMTPError
	if !errors.As(err, &smtpErr) || smtpErr.Code != 553 {
		t.Fatalf("external Mail() error = %v, want SMTP 553", err)
	}
}

func TestResetClearsEnvelopeStateButKeepsAuthentication(t *testing.T) {
	restoreRecipientTestConfig(t)
	session := &Session{
		Ctx:  &pmailcontext.Context{UserID: 1, UserAccount: "admin", IsAdmin: true},
		From: "admin@117799.xyz",
		To:   []string{"one@117799.xyz"},
	}
	session.Reset()
	if session.From != "" || len(session.To) != 0 {
		t.Fatalf("Reset() left envelope state: From=%q To=%v", session.From, session.To)
	}
	if session.Ctx.UserID != 1 {
		t.Fatal("Reset() cleared authentication state")
	}
}

func restoreRecipientTestConfig(t *testing.T) {
	t.Helper()
	oldConfig := config.Instance
	config.Instance = &config.Config{
		Domain:           "117799.xyz",
		Domains:          []string{"117799.xyz"},
		AcceptSubdomains: true,
		CatchAllAccount:  "admin",
	}
	t.Cleanup(func() { config.Instance = oldConfig })
}
