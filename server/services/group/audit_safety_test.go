package group

import (
	"github.com/Jinnrry/pmail/db"
	"github.com/Jinnrry/pmail/models"
	"github.com/Jinnrry/pmail/utils/context"
	"path/filepath"
	"testing"
	"xorm.io/xorm"
)

func newGroupTestDB(t *testing.T) *xorm.Engine {
	t.Helper()
	engine, err := xorm.NewEngine("sqlite", filepath.Join(t.TempDir(), "group.db"))
	if err != nil {
		t.Fatal(err)
	}
	if err = engine.Sync2(&models.Group{}, &models.UserEmail{}); err != nil {
		t.Fatal(err)
	}
	old := db.Instance
	db.Instance = engine
	t.Cleanup(func() { db.Instance = old; engine.Close() })
	return engine
}

func TestAuditGroupOwnershipAndRollback(t *testing.T) {
	engine := newGroupTestDB(t)
	alice, bob := &context.Context{UserID: 1}, &context.Context{UserID: 2}
	owned, err := CreateGroup(bob, "Private", 0)
	if err != nil {
		t.Fatal(err)
	}
	ue := models.UserEmail{UserID: 2, EmailID: 1, GroupId: owned.ID}
	if _, err = engine.Insert(&ue); err != nil {
		t.Fatal(err)
	}
	if _, err = CreateGroup(alice, "Child", owned.ID); err == nil {
		t.Fatal("foreign parent accepted")
	}
	if MoveMailToGroup(alice, []int{1}, owned.ID) {
		t.Fatal("foreign destination accepted")
	}
	if _, err = DelGroup(alice, owned.ID); err == nil {
		t.Fatal("foreign deletion accepted")
	}
	var saved models.UserEmail
	engine.ID(ue.ID).Get(&saved)
	if saved.GroupId != owned.ID {
		t.Fatal("foreign mail was moved")
	}
	if _, err = engine.Exec("CREATE TRIGGER reject_group_delete BEFORE DELETE ON `group` BEGIN SELECT RAISE(ABORT, 'test rollback'); END"); err != nil {
		t.Fatal(err)
	}
	if _, err = DelGroup(bob, owned.ID); err == nil {
		t.Fatal("expected deletion failure")
	}
	saved = models.UserEmail{}
	engine.ID(ue.ID).Get(&saved)
	if saved.GroupId != owned.ID {
		t.Fatal("partial move was not rolled back")
	}
	if _, err = engine.Exec("DROP TRIGGER reject_group_delete"); err != nil {
		t.Fatal(err)
	}
	if ok, err := DelGroup(bob, owned.ID); err != nil || !ok {
		t.Fatalf("own delete: %v %v", ok, err)
	}
	saved = models.UserEmail{}
	engine.ID(ue.ID).Get(&saved)
	if saved.GroupId != 0 {
		t.Fatal("own mail not returned to inbox")
	}
}
