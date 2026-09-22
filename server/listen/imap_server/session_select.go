package imap_server

import (
	"github.com/Jinnrry/pmail/services/group"
	"github.com/emersion/go-imap/v2"
	"github.com/spf13/cast"
	"strings"
)

func (s *serverSession) Select(mailbox string, options *imap.SelectOptions) (*imap.SelectData, error) {
	if "" == mailbox {
		return nil, &imap.Error{
			Type: imap.StatusResponseTypeBad,
			Text: "mailbox not found",
		}
	}

	mailbox = strings.Trim(mailbox, `"`)
	if !group.IsDefaultBox(mailbox) {
		mailboxInfo, err := group.GetGroupByFullPath(s.ctx, mailbox)
		if err != nil || mailboxInfo.ID == 0 {
			return nil, &imap.Error{Type: imap.StatusResponseTypeNo, Text: "mailbox not found"}
		}
	}
	s.currentMailbox = mailbox
	s.deleteUidList = nil
	s.readOnly = options != nil && options.ReadOnly
	_, data := group.GetGroupStatus(s.ctx, s.currentMailbox, []string{"MESSAGES", "UNSEEN", "UIDNEXT", "UIDVALIDITY"})

	ret := &imap.SelectData{
		Flags:          []imap.Flag{imap.FlagSeen, imap.FlagDeleted},
		PermanentFlags: []imap.Flag{imap.FlagSeen},
		NumMessages:    cast.ToUint32(data["MESSAGES"]),
		UIDNext:        imap.UID(data["UIDNEXT"]),
		UIDValidity:    cast.ToUint32(data["UIDVALIDITY"]),
	}

	return ret, nil

}
