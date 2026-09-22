package group

import (
	"fmt"
	"github.com/Jinnrry/pmail/db"
	"github.com/Jinnrry/pmail/utils/context"
	"testing"
)

func TestGetGroupStatus(t *testing.T) {
	newGroupTestDB(t)
	db.Instance.ShowSQL(true)
	ctx := &context.Context{
		UserID:      1,
		UserName:    "admin",
		UserAccount: "admin",
	}
	ret, _ := GetGroupStatus(ctx, "INBOX", []string{"MESSAGES", "UIDNEXT", "UIDVALIDITY", "UNSEEN"})
	fmt.Println(ret)
}
