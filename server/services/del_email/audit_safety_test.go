package del_email

import (
	"github.com/Jinnrry/pmail/db"
	"github.com/Jinnrry/pmail/models"
	"github.com/Jinnrry/pmail/utils/context"
	"path/filepath"
	"testing"
	"xorm.io/xorm"
)

func TestAuditUIDDeletionIsAtomicAndOwnerScoped(t *testing.T) {
	engine, err := xorm.NewEngine("sqlite", filepath.Join(t.TempDir(), "delete.db"))
	if err != nil {
		t.Fatal(err)
	}
	old := db.Instance
	db.Instance = engine
	t.Cleanup(func() { db.Instance = old; engine.Close() })
	if err = engine.Sync2(&models.Email{}, &models.UserEmail{}); err != nil {
		t.Fatal(err)
	}
	email := models.Email{Subject: "keep"}
	engine.Insert(&email)
	first := models.UserEmail{UserID: 1, EmailID: email.Id}
	second := models.UserEmail{UserID: 2, EmailID: email.Id}
	engine.Insert(&first)
	engine.Insert(&second)
	ctx := &context.Context{UserID: 1}
	if err = DelByUID(ctx, []int{9999, second.ID, first.ID}); err != nil {
		t.Fatal(err)
	}
	if has, _ := engine.ID(first.ID).Exist(&models.UserEmail{}); has {
		t.Fatal("missing ID aborted later deletes")
	}
	if has, _ := engine.ID(second.ID).Exist(&models.UserEmail{}); !has {
		t.Fatal("foreign UID deleted")
	}
	if has, _ := engine.ID(email.Id).Exist(&models.Email{}); !has {
		t.Fatal("shared email removed")
	}
	if _, err = engine.Exec("CREATE TRIGGER reject_email_delete BEFORE DELETE ON email BEGIN SELECT RAISE(ABORT, 'rollback'); END"); err != nil {
		t.Fatal(err)
	}
	if err = DelByUID(&context.Context{UserID: 2}, []int{second.ID}); err == nil {
		t.Fatal("expected failure")
	}
	if has, _ := engine.ID(second.ID).Exist(&models.UserEmail{}); !has {
		t.Fatal("relationship deletion was not rolled back")
	}
}
