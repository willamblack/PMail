package pop3_server

import (
	"bytes"
	"github.com/Jinnrry/gopop"
	"github.com/Jinnrry/pmail/db"
	"github.com/Jinnrry/pmail/models"
	"github.com/Jinnrry/pmail/utils/context"
	"github.com/Jinnrry/pmail/utils/password"
	log "github.com/sirupsen/logrus"
	"path/filepath"
	"strings"
	"testing"
	"xorm.io/xorm"
)

func newPOP3TestDB(t *testing.T) {
	t.Helper()
	engine, err := xorm.NewEngine("sqlite", filepath.Join(t.TempDir(), "pop3.db"))
	if err != nil {
		t.Fatal(err)
	}
	if err = engine.Sync2(&models.User{}, &models.Email{}, &models.UserEmail{}); err != nil {
		t.Fatal(err)
	}
	old := db.Instance
	db.Instance = engine
	t.Cleanup(func() { db.Instance = old; engine.Close() })
}

func TestAuditPOP3TLSAndPasswordLogging(t *testing.T) {
	newPOP3TestDB(t)
	const secret = "audit-secret-not-for-logs"
	user := models.User{Account: "admin", Password: password.Encode(secret)}
	if _, err := db.Instance.Insert(&user); err != nil {
		t.Fatal(err)
	}
	a := action{}
	s := &gopop.Session{User: "admin", Ctx: &context.Context{}}
	var output bytes.Buffer
	logger := log.StandardLogger()
	oldOutput, oldLevel := logger.Out, logger.GetLevel()
	logger.SetOutput(&output)
	logger.SetLevel(log.DebugLevel)
	t.Cleanup(func() { logger.SetOutput(oldOutput); logger.SetLevel(oldLevel) })
	if err := a.Pass(s, secret); err == nil {
		t.Fatal("cleartext authentication accepted")
	}
	// gopop sets InTls for its implicit-TLS listener (port 995).
	s.InTls = true
	if err := a.Pass(s, secret); err != nil {
		t.Fatal(err)
	}
	if s.Status != gopop.TRANSACTION || s.Ctx.(*context.Context).UserID != user.ID {
		t.Fatal("TLS authentication failed")
	}
	if err := a.User(s, "other"); err == nil {
		t.Fatal("authenticated identity can be changed")
	}
	if err := a.Apop(s, "admin", password.Md5Encode(user.Password)); err == nil {
		t.Fatal("static APOP digest accepted")
	}
	if strings.Contains(output.String(), secret) {
		t.Fatal("password appeared in logs")
	}
	caps, err := a.Capa(s)
	if err != nil {
		t.Fatal(err)
	}
	for _, cap := range caps {
		if cap == "APOP" || cap == "STLS" {
			t.Fatalf("unsupported capability %s", cap)
		}
	}
}

func TestAuditPOP3TOPRejectsNegativeLines(t *testing.T) {
	if _, err := (action{}).Top(&gopop.Session{}, 1, -1000); err == nil {
		t.Fatal("negative line count accepted")
	}
}
