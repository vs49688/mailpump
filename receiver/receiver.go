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
	"time"

	client2 "github.com/emersion/go-imap/client"
	log "github.com/sirupsen/logrus"
	imap2 "github.com/vs49688/mailpump/imap"
)

func NewReceiver(cfg *Config, factory imap2.ClientFactory) (*MailReceiver, error) {
	updateChannel := make(chan client2.Update, 10)
	c, err := factory.NewClient(&imap2.ClientConfig{
		HostPort:  cfg.HostPort,
		Auth:      cfg.Auth,
		TLS:       cfg.TLS,
		TLSConfig: cfg.TLSConfig,
		Debug:     cfg.Debug,
		Updates:   updateChannel,
	})

	if err != nil {
		return nil, err
	}

	batchSize := cfg.BatchSize
	if batchSize == 0 {
		batchSize = 15
	}

	idleFallbackInterval := cfg.IDLEFallbackInterval
	if idleFallbackInterval == 0 {
		idleFallbackInterval = 1 * time.Minute
	}

	fetchBufferSize := cfg.FetchBufferSize
	if fetchBufferSize == 0 {
		fetchBufferSize = 20
	}

	fetchMaxInterval := cfg.FetchMaxInterval
	if fetchMaxInterval == 0 {
		fetchMaxInterval = 5 * time.Minute
	}

	mr := &MailReceiver{
		client:        c,
		updates:       updateChannel,
		imapChannel:   make(chan interface{}),
		ackChannel:    make(chan ackRequest, fetchBufferSize),
		updateChannel: make(chan *messageState, 10),
		outChannel:    cfg.Channel,

		messages: map[uint32]*messageState{},

		batchSize:            batchSize,
		idleFallbackInterval: idleFallbackInterval,
		fetchBufferSize:      fetchBufferSize,
		fetchMaxInterval:     fetchMaxInterval,
		disableDeletions:     cfg.DisableDeletions,

		hasQuit:  make(chan struct{}, 1),
		wantQuit: make(chan struct{}, 1),
	}

	go mr.run()
	return mr, nil
}

// Ack acknowledges the processing of a message. If error is nil, it is assumed that
// the message has fully processed and persisted, and thus is EXPUNGE'd from the server.
func (mr *MailReceiver) Ack(UID uint32, error error) {
	if error == nil {
		log.WithField("uid", UID).Trace("receiver_ack_called")
	} else {
		log.WithError(error).WithField("uid", UID).Trace("receiver_ack_called")
	}

	if UID == 0 {
		return
	}

	mr.ackChannel <- ackRequest{UID: UID, Error: error}
	log.WithField("uid", UID).Trace("receiver_ack_return")
}

func withMessageState(mstate *messageState) *log.Entry {
	return log.WithFields(log.Fields{
		"uid":   mstate.UID,
		"seq":   mstate.SeqNum,
		"state": mstate.State,
	})
}

func logMessageState(mstate *messageState) {
	withMessageState(mstate).Info("receiver_message_update")
}

func (mr *MailReceiver) handleFetch(r *fetchResult) uint {
	var num uint = 0
	log.WithField("uids", r.UIDs).Trace("receiver_got_fetch_result")
	for uid, msg := range r.Messages {
		if _, ok := mr.messages[uid]; !ok {
			mstate := &messageState{
				UID:     uid,
				SeqNum:  msg.SeqNum,
				Message: msg,
				State:   StateUnacked,
			}
			mr.messages[uid] = mstate
			logMessageState(mstate)
			mr.outChannel <- mstate.Message
			num += 1
		}
	}

	return num
}

func (mr *MailReceiver) handleDelete(r *deleteResult) *messageState {
	e := log.WithFields(log.Fields{"uid": r.UID, "state": r.State})
	if r.State == StateDeleted {
		e.Info("receiver_message_deleted")
		delete(mr.messages, r.UID)
		return nil
	}

	if msg, ok := mr.messages[r.UID]; ok {
		// Delete failed, try again
		e.Info("receiver_message_deletion_failed")
		msg.State = r.State
		logMessageState(msg)
		return msg
	}

	// Unknown message, do nothing
	e.Trace("receiver_message_deletion_unknown")
	return nil
}

func (mr *MailReceiver) handleAck(r *ackRequest) *messageState {
	if r.Error != nil {
		log.WithError(r.Error).WithField("uid", r.UID).Warn("receiver_ack")
	} else {
		log.WithField("uid", r.UID).Info("receiver_ack")
	}

	if r.Error != nil {
		return nil
	}

	if msg, ok := mr.messages[r.UID]; ok {
		if msg.State == StateUnacked {
			msg.State = StateAcked
			logMessageState(msg)
			return msg
		}
	}

	return nil
}

func (mr *MailReceiver) handleMessageUpdate(upd client2.Update) bool {
	switch vv := upd.(type) {
	case *client2.StatusUpdate:
		// This is INFO because it often contains useful info to have in the logs
		log.WithFields(log.Fields{
			"tag":       vv.Status.Tag,
			"type":      vv.Status.Type,
			"code":      vv.Status.Code,
			"arguments": vv.Status.Arguments,
			"info":      vv.Status.Info,
		}).Info("receiver_got_status_update")
	case *client2.ExpungeUpdate:
		log.WithField("seq", vv.SeqNum).Trace("receiver_got_expunge_update")
	case *client2.MailboxUpdate:
		log.WithFields(log.Fields{
			"name":     vv.Mailbox.Name,
			"messages": vv.Mailbox.Messages,
		}).Trace("receiver_got_mailbox_update")
		return true
	}

	return false
}

func (mr *MailReceiver) run() {
	state := StateNone
	nextToProcess := map[uint32]*messageState{}
	wantQuit := NewCounter()

	wantStopIdle := NewCounter()
	opChan := make(chan operation, 1)

	wantFetch := NewCounter()  // Do we need to fetch again
	wantDelete := NewCounter() // Do we need to delete

	setState := func(s sstate) {
		log.WithFields(log.Fields{
			"old": state,
			"new": s,
		}).Trace("receiver_state_change")
		state = s
	}

	for {
		log.WithFields(log.Fields{
			"state":          state,
			"want_quit":      wantQuit.IsFlagged(),
			"want_fetch":     wantFetch.IsFlagged(),
			"want_delete":    wantDelete.IsFlagged(),
			"want_stop_idle": wantStopIdle.IsFlagged(),
		}).Trace("receiver_loop_start")

		op := OperationNone

		select {
		case <-mr.wantQuit:
			wantQuit.Flag()
			mr.client.FlagQuit()
		case upd := <-mr.updates:
			if mr.handleMessageUpdate(upd) {
				wantFetch.Flag()
			}
		case _r := <-mr.imapChannel:
			switch r := _r.(type) {
			case fetchResult:
				if state != StateInFetch {
					log.WithField("state", state).Panic("receiver_fetch_outside_fetch")
				}

				// If we're quitting, just discard all new fetches
				if wantQuit.IsFlagged() {
					log.WithField("uids", r.UIDs).Trace("receiver_ignoring_fetch_quitting")
					break
				}

				// Only sends messages out
				_ = mr.handleFetch(&r)
			case deleteResult:
				if state != StateInDelete {
					log.WithField("state", state).Panic("receiver_delete_outside_delete")
				}

				if msg := mr.handleDelete(&r); msg != nil {
					// Flag if delete failed
					nextToProcess[msg.UID] = msg
					wantDelete.FlagIf(!mr.disableDeletions)
				}
			default:
				log.WithField("result", r).Panic("receiver_invalid_result")
			}
		case ack := <-mr.ackChannel:
			// ACKs should be handled in any state
			if msg := mr.handleAck(&ack); msg != nil {
				nextToProcess[msg.UID] = msg
				wantDelete.FlagIf(!mr.disableDeletions)
			}
		case <-time.After(mr.fetchMaxInterval):
			op = OperationTimeout
		case op = <-opChan:
			break
		}

		log.WithFields(log.Fields{
			"state":     state,
			"operation": op,
		}).Trace("receiver_tick")

		switch state {
		case StateNone:
			switch op {
			case OperationNone:
				break
			case OperationTimeout:
				wantFetch.Flag()
				wantDelete.FlagIf(!mr.disableDeletions)
			default:
				log.WithFields(log.Fields{"state": state, "operation": op}).Panic("invalid_operation_for_state")
			}

			log.WithFields(log.Fields{
				"state":            state,
				"operation":        op,
				"want_quit":        wantQuit.IsFlagged(),
				"fetch_flag":       wantFetch.IsFlagged(),
				"delete_flag":      wantDelete.IsFlagged(),
				"to_process_count": len(nextToProcess),
			}).Trace("receiver_processing_state_none")

			/*
				if wantDelete.IsFlagged() && !wantFetch.IsFlagged() && len(nextToProcess) == 1 && !wantQuit.IsFlagged() {
					panic("BUGBUGBUG1")
				}

				if !wantDelete.IsFlagged() && wantFetch.IsFlagged() && len(nextToProcess) == 1 && !wantQuit.IsFlagged() {
					panic("BUGBUGBUG2")
				}
			*/

			if wantQuit.IsFlagged() {
				// paranoia
				wantFetch.Reset()
			}

			if uint(len(nextToProcess)) >= mr.batchSize {
				wantDelete.FlagIf(!mr.disableDeletions)
			}

			if wantDelete.IsFlagged() {
				wantDelete.Reset()

				if len(nextToProcess) > 0 {
					log.Trace("receiver_delete_start")
					setState(StateInDelete)
					go func(toProcess map[uint32]*messageState) {
						_ = doDelete(mr.client, mr.imapChannel, toProcess)
						opChan <- OperationDeleteFinish
					}(nextToProcess)
					nextToProcess = map[uint32]*messageState{}
					continue
				}
			}

			if wantFetch.IsFlagged() {
				log.Trace("receiver_fetch_start")
				wantFetch.Reset()
				setState(StateInFetch)

				existing := mr.buildCurrentSequence()
				go func() {
					_ = doFetch(mr.client, existing, mr.fetchBufferSize, mr.imapChannel)
					opChan <- OperationFetchFinish
				}()
			} else if !wantQuit.IsFlagged() {
				log.Trace("receiver_idle_start")
				setState(StateInIDLE)
				go func(stop <-chan struct{}) {
					log.Trace("receiver_idle_go_start")
					err := mr.client.Idle(stop, &client2.IdleOptions{
						LogoutTimeout: 250 * time.Second, // Yahoo kills us after 5 mintues
						PollInterval:  mr.idleFallbackInterval,
					})
					if err != nil {
						log.WithError(err).Warn("receiver_idle_failed")
					}
					opChan <- OperationIDLEFinish
					log.Trace("receiver_idle_go_end")
				}(wantStopIdle.Channel())
			} else {
				goto done
			}

		case StateInIDLE:
			switch op {
			case OperationNone:
				fallthrough
			case OperationTimeout:
				wantFetch.Flag()
				wantStopIdle.Flag()
			case OperationIDLEFinish:
				log.Trace("receiver_idle_finish")
				wantStopIdle.Reset()
				setState(StateNone)
				opChan <- OperationNone
			default:
				log.WithFields(log.Fields{"state": state, "operation": op}).Panic("invalid_operation_for_state")
			}
		case StateInFetch:
			switch op {
			case OperationNone:
				break
			case OperationFetchFinish:
				log.Trace("receiver_fetch_finish")
				setState(StateNone)
				opChan <- OperationNone
			case OperationTimeout:
				break
			default:
				log.WithFields(log.Fields{"state": state, "operation": op}).Panic("invalid_operation_for_state")
			}
		case StateInDelete:
			switch op {
			case OperationNone:
				break
			case OperationDeleteFinish:
				log.Trace("receiver_delete_finish")
				setState(StateNone)
				opChan <- OperationNone
			case OperationTimeout:
				wantFetch.Flag()
				break
			default:
				log.WithFields(log.Fields{"state": state, "operation": op}).Panic("invalid_operation_for_state")
			}
		}
	}

done:
	log.WithField("state", state).Trace("receiver_loop_exit")

	mr.hasQuit <- struct{}{}
	log.Trace("receiver_proc_quit")
}

func (mr *MailReceiver) Close() {
	log.Trace("receiver_close_invoked")
	mr.wantQuit <- struct{}{}
	log.Trace("receiver_close_waiting_for_quit")
	<-mr.hasQuit
	log.Trace("receiver_close_have_quit")
	_ = mr.client.Logout()
	log.Trace("receiver_close_logout")
}
