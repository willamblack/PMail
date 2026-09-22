package imap_server

import (
	"github.com/Jinnrry/pmail/services/del_email"
	"github.com/Jinnrry/pmail/services/list"
	"github.com/Jinnrry/pmail/utils/array"
	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver"
)

func (s *serverSession) Expunge(w *imapserver.ExpungeWriter, uids *imap.UIDSet) error {
	if s.readOnly {
		return &imap.Error{Type: imap.StatusResponseTypeNo, Text: "mailbox is read-only"}
	}
	// Iterate actual mailbox rows, never a client-controlled numeric range.
	rows := list.GetUEListByUID(s.ctx, s.currentMailbox, 0, 0, nil)
	var ids []int
	var sequences []uint32
	for _, row := range rows {
		if !array.InArray(row.ID, s.deleteUidList) {
			continue
		}
		if uids != nil && !uids.Contains(imap.UID(row.ID)) {
			continue
		}
		ids = append(ids, row.ID)
		sequences = append(sequences, uint32(row.SerialNumber))
	}
	if len(ids) == 0 {
		return nil
	}
	if err := del_email.DelByUID(s.ctx, ids); err != nil {
		return err
	}
	var pending []int
	for _, id := range s.deleteUidList {
		if !array.InArray(id, ids) {
			pending = append(pending, id)
		}
	}
	s.deleteUidList = pending
	if w != nil {
		// Highest sequence first prevents removals shifting later numbers.
		for i := len(sequences) - 1; i >= 0; i-- {
			if err := w.WriteExpunge(sequences[i]); err != nil {
				return err
			}
		}
	}
	return nil
}
