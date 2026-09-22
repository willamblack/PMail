package imap_server

import (
	"github.com/Jinnrry/pmail/db"
	"github.com/Jinnrry/pmail/models"
	"github.com/Jinnrry/pmail/utils/array"
	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver"
)

// Deleted flags are session-local. Only EXPUNGE removes stored mail.
func (s *serverSession) Store(w *imapserver.FetchWriter, numSet imap.NumSet, flags *imap.StoreFlags, options *imap.StoreOptions) error {
	if s.readOnly {
		return &imap.Error{Type: imap.StatusResponseTypeNo, Text: "mailbox is read-only"}
	}
	emails := s.messages(numSet)
	for _, email := range emails {
		seen := email.IsRead == 1
		deleted := array.InArray(email.UeId, s.deleteUidList)
		apply := func(old, requested bool) bool {
			switch flags.Op {
			case imap.StoreFlagsSet:
				return requested
			case imap.StoreFlagsAdd:
				return old || requested
			case imap.StoreFlagsDel:
				return old && !requested
			}
			return old
		}
		seen = apply(seen, array.InArray(imap.FlagSeen, flags.Flags))
		deleted = apply(deleted, array.InArray(imap.FlagDeleted, flags.Flags))
		read := int8(0)
		if seen {
			read = 1
		}
		if _, err := db.Instance.Table(&models.UserEmail{}).Where("id=? and user_id=?", email.UeId, s.ctx.UserID).Update(map[string]interface{}{"is_read": read}); err != nil {
			return err
		}
		email.IsRead = read
		var pending []int
		for _, id := range s.deleteUidList {
			if id != email.UeId {
				pending = append(pending, id)
			}
		}
		if deleted {
			pending = append(pending, email.UeId)
		}
		s.deleteUidList = pending
	}
	if !flags.Silent && w != nil {
		write(s.ctx, w, emails, &imap.FetchOptions{Flags: true, UID: true}, s.deleteUidList, false)
	}
	return nil
}
