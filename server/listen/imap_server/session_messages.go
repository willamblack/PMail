package imap_server

import (
	"github.com/Jinnrry/pmail/dto/response"
	"github.com/Jinnrry/pmail/services/list"
	"github.com/emersion/go-imap/v2"
)

// Resolve against actual mailbox rows: this handles disjoint/reversed/star
// ranges without enumerating untrusted 32-bit UID ranges or narrowing to int.
func (s *serverSession) messages(set imap.NumSet) []*response.EmailResponseData {
	rows := list.GetUEListByUID(s.ctx, s.currentMailbox, 0, 0, nil)
	if len(rows) == 0 {
		return nil
	}
	var selected []*response.UserEmailUIDData
	maximumSeq, maximumUID := uint32(len(rows)), uint32(rows[len(rows)-1].ID)
	contains := func(number, start, end, maximum uint32) bool {
		if start == 0 {
			start = maximum
		}
		if end == 0 {
			end = maximum
		}
		if start > end {
			start, end = end, start
		}
		return number >= start && number <= end
	}
	for _, row := range rows {
		match := false
		switch ranges := set.(type) {
		case imap.SeqSet:
			for _, r := range ranges {
				if contains(uint32(row.SerialNumber), r.Start, r.Stop, maximumSeq) {
					match = true
					break
				}
			}
		case imap.UIDSet:
			for _, r := range ranges {
				if contains(uint32(row.ID), uint32(r.Start), uint32(r.Stop), maximumUID) {
					match = true
					break
				}
			}
		}
		if match {
			selected = append(selected, row)
		}
	}
	return list.GetEmailsForUIDRows(s.ctx, selected)
}
