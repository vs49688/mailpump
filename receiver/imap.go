/*
 * MailPump - Copyright (C) 2022 Zane van Iperen.
 *    Contact: zane@zanevaniperen.com
 *
 * This program is free software; you can redistribute it and/or modify
 * it under the terms of the GNU General Public License version 2, and only
 * version 2 as published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program; if not, write to the Free Software
 * Foundation, Inc., 59 Temple Place, Suite 330, Boston, MA  02111-1307  USA
 */

package receiver

import (
	"github.com/emersion/go-imap"
	log "github.com/sirupsen/logrus"
	imap2 "github.com/vs49688/mailpump/imap"
)

// buildCurrentSequence builds an imap.SeqSet instance containing
// the sequence numbers of all the current "known" messages.
func (mr *mailReceiver) buildCurrentSequence() imap.SeqSet {
	existing := imap.SeqSet{}
	for _, k := range mr.messages {
		existing.AddNum(k.SeqNum)
	}

	return existing
}

// buildSeqSet builds an imap.SeqSet instance containing
// the sequence numbers of all messages to fetch, excluding those already
// in existing, up do a maximum size.
func buildSeqSet(existing imap.SeqSet, mbStatus *imap.MailboxStatus, maxSize uint) imap.SeqSet {
	seq := imap.SeqSet{}

	i := uint(0)
	for next := uint32(1); i < maxSize && next <= mbStatus.Messages; next += 1 {
		if existing.Contains(next) {
			continue
		}

		seq.AddNum(next)
		i += 1
	}

	return seq
}

func doFetch(client imap2.Client, existing imap.SeqSet, maxSize uint, result chan<- interface{}, logger *log.Entry) bool {
	logger.Trace("receiver_fetching_messages")

	mbStatus := client.Mailbox()
	if mbStatus == nil {
		logger.Warn("receiver_no_mailbox")
		return false
	}

	logger.WithFields(log.Fields{
		"name":         mbStatus.Name,
		"num_messages": mbStatus.Messages,
		"recent":       mbStatus.Recent,
		"unseen":       mbStatus.Unseen,
		"unseen_seq":   mbStatus.UnseenSeqNum,
	}).Trace("receiver_mailbox_status")

	if mbStatus.Messages == 0 {
		return false
	}

	seqset := buildSeqSet(existing, mbStatus, maxSize)
	logger.WithField("set", seqset).Trace("receiver_fetch_set")
	if seqset.Empty() {
		return false
	}

	ch := make(chan *imap.Message)
	done := make(chan error)

	go func() {
		done <- client.Fetch(&seqset, []imap.FetchItem{imap.FetchUid, imap.FetchFlags, imap.FetchRFC822}, ch)
	}()

	uids, messages := readMessages(ch)

	if err := <-done; err != nil {
		logger.WithError(err).Warn("receiver_fetch_failed")
	} else {
		logger.WithFields(log.Fields{"uids": uids}).Trace("receiver_fetch_succeeded")
		result <- fetchResult{
			UIDs:     uids,
			Messages: messages,
		}
		logger.WithFields(log.Fields{"uids": uids}).Trace("receiver_fetch_succeeded_chanwrite")
	}

	return false
}

func doDelete(client imap2.Client, result chan<- interface{}, toProcess map[uint32]*messageState, logger *log.Entry) interface{} {
	toDelete := map[uint32]*imap.Message{}

	deleteSet := new(imap.SeqSet)

	for uid, msg := range toProcess {
		if msg.State == StateAcked {
			toDelete[uid] = msg.Message
			deleteSet.AddNum(uid)
		} else if msg.State == StateDeleted {
			// Message is already deleted, why are we receiving this?
			// Send it back and it should be removed.
			withMessageState(logger, msg).Warn("receiver_message_already_deleted")
			result <- deleteResult{UID: msg.UID, State: StateDeleted}
		}
	}

	done := make(chan error)

	if len(toDelete) > 0 {
		// Delete messages first
		ch := make(chan *imap.Message)
		go func() {
			done <- client.UidStore(deleteSet, imap.FormatFlagsOp(imap.AddFlags, false), []interface{}{imap.DeletedFlag}, ch)
		}()

		for msg := range ch {
			found := false
			for _, f := range msg.Flags {
				if f == imap.DeletedFlag {
					found = true
					break
				}
			}

			if found {
				result <- deleteResult{UID: msg.Uid, State: StateDeleted}
			} else {
				logger.WithFields(log.Fields{"uid": msg.Uid}).Warn("receiver_message_not_deleted_rescheduling")
				result <- deleteResult{UID: msg.Uid, State: StateAcked}
			}
		}

		if err := <-done; err != nil {
			logger.WithError(err).Warn("receiver_delete_failed")
		}
	}

	// Expunge. We don't use the returned sequence numbers as they
	// always seem inconsistent. If the mail server *really* doesn't want to
	// expunge a message, there's nothing we can do anyway...
	if err := client.Expunge(nil); err != nil {
		logger.WithError(err).Warn("receiver_expunge_failed")
	}

	return nil
}
