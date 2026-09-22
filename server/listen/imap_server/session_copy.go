package imap_server

import (
	"github.com/Jinnrry/pmail/db"
	"github.com/Jinnrry/pmail/dto/response"
	"github.com/Jinnrry/pmail/models"
	"github.com/Jinnrry/pmail/services/group"
	"github.com/Jinnrry/pmail/utils/context"
	"github.com/emersion/go-imap/v2"
	"xorm.io/xorm"
)

func (s *serverSession) Copy(numSet imap.NumSet, dest string) (*imap.CopyData, error) {
	return transferMessages(s.ctx, s.messages(numSet), dest, false)
}

// Copy and move commit all selected UID relationships together. Moving by
// email_id would incorrectly move other copies of the same message as well.
func transferMessages(ctx *context.Context, mails []*response.EmailResponseData, dest string, move bool) (*imap.CopyData, error) {
	groupID, validity := 0, models.GroupNameToCode[dest]
	status, defaultBox := map[string]int8{"INBOX": 0, "Sent Messages": 1, "Deleted Messages": 3, "Drafts": 4, "Junk": 5}[dest]
	if !defaultBox {
		info, err := group.GetGroupByFullPath(ctx, dest)
		if err != nil {
			return nil, err
		}
		if info == nil || info.ID == 0 {
			return nil, &imap.Error{Type: imap.StatusResponseTypeNo, Text: "destination mailbox not found"}
		}
		groupID, validity = info.ID, info.ID
	}
	if len(mails) == 0 {
		return nil, nil
	}
	data := &imap.CopyData{UIDValidity: uint32(validity)}
	err := db.Transaction(db.Instance, func(tx *xorm.Session) error {
		data.SourceUIDs = nil
		data.DestUIDs = nil
		for _, email := range mails {
			var source models.UserEmail
			found, err := tx.Where("id=? and user_id=?", email.UeId, ctx.UserID).Get(&source)
			if err != nil {
				return err
			}
			if !found {
				return &imap.Error{Type: imap.StatusResponseTypeNo, Text: "source message no longer exists"}
			}
			target := models.UserEmail{UserID: ctx.UserID, EmailID: source.EmailID, IsRead: source.IsRead, GroupId: groupID, Status: source.Status}
			if defaultBox {
				target.Status = status
			}
			if _, err = tx.Insert(&target); err != nil {
				return err
			}
			if move {
				if _, err = tx.Where("id=? and user_id=?", source.ID, ctx.UserID).Delete(&models.UserEmail{}); err != nil {
					return err
				}
			}
			data.SourceUIDs = append(data.SourceUIDs, imap.UIDRange{Start: imap.UID(source.ID), Stop: imap.UID(source.ID)})
			data.DestUIDs = append(data.DestUIDs, imap.UIDRange{Start: imap.UID(target.ID), Stop: imap.UID(target.ID)})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return data, nil
}
