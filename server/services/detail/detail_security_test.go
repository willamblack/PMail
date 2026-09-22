package detail

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/Jinnrry/pmail/db"
	"github.com/Jinnrry/pmail/models"
	"github.com/Jinnrry/pmail/utils/context"
	_ "modernc.org/sqlite"
	"xorm.io/xorm"
)

func TestDisplaySanitizesExistingOutgoingMailAndDoesNotMarkForeignMail(t *testing.T) {
	old := db.Instance
	engine, err := xorm.NewEngine("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	engine.SetMaxOpenConns(1)
	db.Instance = engine
	t.Cleanup(func() { _ = engine.Close(); db.Instance = old })
	if err = engine.Sync2(&models.Email{}, &models.UserEmail{}); err != nil {
		t.Fatal(err)
	}
	email := models.Email{Html: sql.NullString{String: `<p>Hello</p><script>alert(1)</script><img src="x" onerror="alert(2)">`, Valid: true}}
	if _, err = engine.Insert(&email); err != nil {
		t.Fatal(err)
	}
	link := models.UserEmail{EmailID: email.Id, UserID: 2}
	if _, err = engine.Insert(&link); err != nil {
		t.Fatal(err)
	}
	if _, err = GetEmailDetail(&context.Context{UserID: 1}, email.Id, true); err == nil {
		t.Fatal("foreign mail access allowed")
	}
	var unchanged models.UserEmail
	if _, err = engine.ID(link.ID).Get(&unchanged); err != nil || unchanged.IsRead != 0 {
		t.Fatal("unauthorized read changed state")
	}
	res, err := GetEmailDetail(&context.Context{UserID: 2}, email.Id, true)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(res.Html.String, "<script") || strings.Contains(res.Html.String, "onerror") || !strings.Contains(res.Html.String, "Hello") {
		t.Fatalf("bad display HTML %q", res.Html.String)
	}
	if _, err = GetEmailDetail(&context.Context{UserID: 1, IsAdmin: true}, 999, true); err == nil {
		t.Fatal("nonexistent email accepted")
	}
}
