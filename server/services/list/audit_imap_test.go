package list

import (
	"github.com/Jinnrry/pmail/db"
	"github.com/Jinnrry/pmail/models"
	"github.com/Jinnrry/pmail/utils/context"
	"github.com/emersion/go-imap/v2"
	"path/filepath"
	"testing"
	"xorm.io/xorm"
)

func TestAuditIMAPSearchAndMailboxRanges(t *testing.T) {
	engine, err := xorm.NewEngine("sqlite", filepath.Join(t.TempDir(), "imap-list.db"))
	if err != nil {
		t.Fatal(err)
	}
	old := db.Instance
	db.Instance = engine
	t.Cleanup(func() { db.Instance = old; engine.Close() })
	if err = engine.Sync2(&models.Group{}, &models.UserEmail{}, &models.Email{}); err != nil {
		t.Fatal(err)
	}
	ctx := &context.Context{UserID: 42}
	mailbox := models.Group{UserId: 42, Name: "Child", FullPath: "Parent/Child"}
	engine.Insert(&mailbox)
	var ids []int
	for _, subject := range []string{"red round", "red square", "blue round"} {
		email := models.Email{Subject: subject}
		engine.Insert(&email)
		ue := models.UserEmail{UserID: 42, EmailID: email.Id, GroupId: mailbox.ID}
		engine.Insert(&ue)
		ids = append(ids, ue.ID)
	}
	rows := GetEmailListByGroup(ctx, "Parent/Child", ImapListReq{Star: 2, End: 3}, false)
	if len(rows) != 2 || rows[0].SerialNumber != 2 {
		t.Fatalf("custom mailbox range: %+v", rows)
	}
	rows = GetEmailListByGroup(ctx, "Parent/Child", ImapListReq{Star: ids[2], End: ids[2]}, true)
	if len(rows) != 1 || rows[0].SerialNumber != 3 {
		t.Fatal("UID selection renumbered")
	}
	criteria := &imap.SearchCriteria{Or: [][2]imap.SearchCriteria{
		{{Text: []string{"red"}}, {Text: []string{"orange"}}},
		{{Text: []string{"round"}}, {Text: []string{"triangle"}}},
	}}
	matched, err := SearchEmails(ctx, "Parent/Child", criteria)
	if err != nil || len(matched) != 1 || matched[0].ID != ids[0] {
		t.Fatalf("OR pairs not ANDed: %+v %v", matched, err)
	}
	matched, err = SearchEmails(ctx, "Parent/Child", &imap.SearchCriteria{SeqNum: []imap.SeqSet{{{Start: 0, Stop: 0}}}})
	if err != nil || len(matched) != 1 || matched[0].ID != ids[2] {
		t.Fatalf("star search: %+v %v", matched, err)
	}
	matched, err = SearchEmails(ctx, "Parent/Child", &imap.SearchCriteria{UID: []imap.UIDSet{imap.UIDSetNum(imap.UID(ids[0])), imap.UIDSetNum(imap.UID(ids[2]))}})
	if err != nil || len(matched) != 0 {
		t.Fatal("adjacent UID criteria must intersect")
	}
	if _, err = engine.Exec("DROP TABLE email"); err != nil {
		t.Fatal(err)
	}
	if _, err = SearchEmails(ctx, "Parent/Child", criteria); err == nil {
		t.Fatal("query failure silently returned search matches")
	}
}
