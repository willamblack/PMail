package rule

import (
	"github.com/Jinnrry/pmail/db"
	"github.com/Jinnrry/pmail/dto"
	"github.com/Jinnrry/pmail/dto/parsemail"
	"github.com/Jinnrry/pmail/models"
	"github.com/Jinnrry/pmail/utils/context"
	"path/filepath"
	"strconv"
	"testing"
	"xorm.io/xorm"
)

func TestAuditRuleMovePreservesSharedEmail(t *testing.T) {
	engine, err := xorm.NewEngine("sqlite", filepath.Join(t.TempDir(), "rule-move.db"))
	if err != nil {
		t.Fatal(err)
	}
	old := db.Instance
	db.Instance = engine
	t.Cleanup(func() { db.Instance = old; engine.Close() })
	if err = engine.Sync2(&models.Group{}, &models.UserEmail{}, &models.Email{}); err != nil {
		t.Fatal(err)
	}
	email := models.Email{Type: 0, Subject: "shared"}
	engine.Insert(&email)
	first := models.UserEmail{UserID: 1, EmailID: email.Id}
	second := models.UserEmail{UserID: 2, EmailID: email.Id}
	engine.Insert(&first)
	engine.Insert(&second)
	rule := &dto.Rule{UserId: 1, Params: strconv.Itoa(models.Sent)}
	doMove(&context.Context{UserID: 99}, rule, &parsemail.Email{MessageId: int64(email.Id)}, &models.User{ID: 1})
	var got models.Email
	engine.ID(email.Id).Get(&got)
	if got.Type != 0 {
		t.Fatal("shared mail was reclassified")
	}
	var alice, bob models.UserEmail
	engine.ID(first.ID).Get(&alice)
	engine.ID(second.ID).Get(&bob)
	if alice.Status != 1 || bob.Status != 0 {
		t.Fatalf("wrong mailbox changed: alice=%d bob=%d", alice.Status, bob.Status)
	}
	target := models.Group{UserId: 2, Name: "foreign", FullPath: "foreign"}
	engine.Insert(&target)
	rule.Params = strconv.Itoa(target.ID)
	doMove(&context.Context{}, rule, &parsemail.Email{MessageId: int64(email.Id)}, &models.User{ID: 1})
	alice = models.UserEmail{}
	engine.ID(first.ID).Get(&alice)
	if alice.GroupId != 0 {
		t.Fatal("foreign rule destination accepted")
	}
}
