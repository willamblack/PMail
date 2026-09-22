package imap_server

import (
	"github.com/Jinnrry/pmail/utils/array"
	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver"
)

func (s *serverSession) Move(w *imapserver.MoveWriter, numSet imap.NumSet, dest string) error {
	if s.readOnly {
		return &imap.Error{Type: imap.StatusResponseTypeNo, Text: "mailbox is read-only"}
	}
	emails := s.messages(numSet)
	if len(emails) == 0 {
		return nil
	}
	data, err := transferMessages(s.ctx, emails, dest, true)
	if err != nil {
		return err
	}
	var pending []int
	for _, id := range s.deleteUidList {
		if !data.SourceUIDs.Contains(imap.UID(id)) {
			pending = append(pending, id)
		}
	}
	s.deleteUidList = array.Unique(pending)
	if w != nil {
		if err = w.WriteCopyData(data); err != nil {
			return err
		}
		for i := len(emails) - 1; i >= 0; i-- {
			if err = w.WriteExpunge(uint32(emails[i].SerialNumber)); err != nil {
				return err
			}
		}
	}
	return nil
}
