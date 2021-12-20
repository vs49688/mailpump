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
	client2 "github.com/emersion/go-imap/client"
	log "github.com/sirupsen/logrus"
	imap2 "github.com/vs49688/mailpump/imap"
	"time"
)

func (s state) String() string {
	switch s {
	case StateUnacked:
		return "StateUnacked"
	case StateAcked:
		return "StateAcked"
	case StateDeleted:
		return "StateDeleted"
	default:
		panic("invalid state")
	}
}

func NewReceiver(cfg *Config, factory imap2.ClientFactory) (*MailReceiver, error) {
	updateChannel := make(chan client2.Update, 10)
	c, err := factory.NewClient(&imap2.ClientConfig{
		HostPort:  cfg.HostPort,
		Username:  cfg.Username,
		Password:  cfg.Password,
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

	tickInterval := cfg.TickInterval
	if tickInterval == 0 {
		tickInterval = 1 * time.Minute
	}

	mr := &MailReceiver{
		client:        c,
		updates:       updateChannel,
		imapChannel:   make(chan interface{}),
		ackChannel:    make(chan ackRequest, 10),
		updateChannel: make(chan *messageState, 10),
		outChannel:    cfg.Channel,

		messages: map[uint32]*messageState{},

		batchSize:    batchSize,
		tickInterval: tickInterval,

		hasQuit:  make(chan struct{}, 1),
		wantQuit: make(chan struct{}, 1),
	}

	go mr.run()
	return mr, nil
}

// Ack acknowledges the processing of a message. If error is nil, it is assumed that
// the message has fully processed and persisted, and thus is EXPUNGE'd from the server.
func (mr *MailReceiver) Ack(UID uint32, error error) {
	if error != nil {
		log.WithField("uid", UID).Trace("ack_called")
	} else {
		log.WithError(error).WithField("uid", UID).Trace("ack_called")
	}

	if UID == 0 {
		return
	}

	mr.ackChannel <- ackRequest{UID: UID, Error: error}
}

func withMessageState(mstate *messageState) *log.Entry {
	return log.WithFields(log.Fields{
		"uid":   mstate.UID,
		"seq":   mstate.SeqNum,
		"state": mstate.State,
	})
}

func logMessageState(mstate *messageState) {
	withMessageState(mstate).Trace("message_update")
}

func (mr *MailReceiver) handleFetch(r *fetchResult) {
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
		}
	}
}

func (mr *MailReceiver) handleDelete(r *deleteResult) *messageState {
	log.WithFields(log.Fields{
		"uid":   r.UID,
		"state": r.State,
	}).Trace("message_deleted")
	if r.State == StateDeleted {
		delete(mr.messages, r.UID)
		return nil
	}

	if msg, ok := mr.messages[r.UID]; ok {
		// Delete failed, try again
		msg.State = r.State
		logMessageState(msg)
		return msg
	}

	// Unknown message, do nothing
	return nil
}

func (mr *MailReceiver) handleAck(r *ackRequest) *messageState {
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

type sstate int

var (
	StateNone     sstate = 0
	StateInIDLE   sstate = 1
	StateInFetch  sstate = 2
	StateInDelete sstate = 3
)

func (s sstate) String() string {
	switch s {
	case StateNone:
		return "none"
	case StateInIDLE:
		return "in_idle"
	case StateInFetch:
		return "in_fetch"
	case StateInDelete:
		return "in_delete"
	default:
		panic("invalid_state")
	}
}

func (mr *MailReceiver) run() {
	nextToProcess := map[uint32]*messageState{}
	timeout := false

	state := StateNone
	// For when we're done fetching or deleting"
	opChan := make(chan interface{}, 1)

	fetchFlag := FlagCounter{}

	stopIdleChannel := make(chan struct{}, 1) // NB: needs buffer of 1, as we write to it to trigger ourselves
	stopIdleFlag := FlagCounter{
		Counter: 0,
		Channel: stopIdleChannel,
	}

	quitFlag := FlagCounter{}

	canProcess := func() bool {
		r := fetchFlag.IsFlagged() || ((timeout || quitFlag.IsFlagged()) && len(nextToProcess) > 0) || uint(len(nextToProcess)) >= mr.batchSize
		//log.WithFields(log.Fields{
		//	"fetch_flag":     fetchFlag.IsFlagged(),
		//	"quit_flag":      quitFlag.IsFlagged(),
		//	"timeout":        timeout,
		//	"num_to_process": len(nextToProcess),
		//	"result":         r,
		//}).Trace("can_process")
		return r
	}

	addToProcess := func(msg *messageState) {
		nextToProcess[msg.UID] = msg
		if canProcess() {
			stopIdleFlag.Flag()
		}
	}

	// Wake up the first iteration
	opChan <- nil

	for {
		log.Trace("receiver_loop_start")
		select {
		case <-mr.wantQuit:
			log.Trace("receiver_xx_want_quit")
			quitFlag.Flag()
			switch state {
			case StateNone:
				break
			case StateInIDLE:
				stopIdleFlag.Flag()
				continue
			case StateInDelete:
				continue
			case StateInFetch:
				continue
			}
		case v := <-opChan:
			log.Trace("receiver_xx_opchan")
			if state == StateNone {
				// nop, self-wakeup
			} else if state == StateInDelete {
				// nop, delete finished
			} else if state == StateInFetch {
				// fetch finished
				more, _ := v.(bool)
				fetchFlag.FlagIf(more && !quitFlag.IsFlagged())
				state = StateNone
			} else if state == StateInIDLE {
				// idle finished
				stopIdleFlag.Reset()
			}

			state = StateNone
		case upd := <-mr.updates:
			log.Trace("receiver_xx_update")
			switch vv := upd.(type) {
			case *client2.StatusUpdate:
				log.WithFields(log.Fields{
					"tag":       vv.Status.Tag,
					"type":      vv.Status.Type,
					"code":      vv.Status.Code,
					"arguments": vv.Status.Arguments,
					"info":      vv.Status.Info,
				}).Trace("received_status_update")
			case *client2.ExpungeUpdate:
				log.WithFields(log.Fields{"seq": vv.SeqNum}).Trace("received_expunge_update")
			case *client2.MailboxUpdate:
				log.Trace("received_mailbox_update")
				fetchFlag.FlagIf(!quitFlag.IsFlagged())
				stopIdleFlag.Flag()
			}
			continue
		case <-time.After(5 * time.Second):
			log.Trace("receiver_xx_timeout")
			timeout = true
			if state == StateInIDLE {
				fetchFlag.Flag()
				stopIdleFlag.Flag()
				continue
			} else if state == StateInDelete {
				continue
			} else if state == StateInFetch {
				continue
			}
			//if canProcess() {
			//	stopIdleFlag.Flag()
			//}
		case _r := <-mr.imapChannel:
			log.Trace("receiver_xx_imapchan")
			// Message updates should be run in any state
			switch r := _r.(type) {
			case fetchResult:
				// If we're quitting, just discard all new fetches
				if quitFlag.IsFlagged() {
					break
				}

				// Only sends messages out
				mr.handleFetch(&r)
			case deleteResult:
				if msg := mr.handleDelete(&r); msg != nil {
					addToProcess(msg)
				}
			}
			continue
		case ack := <-mr.ackChannel:
			log.Trace("receiver_xx_ackchan")
			// ACKs should be handled in any state
			if msg := mr.handleAck(&ack); msg != nil {
				addToProcess(msg)
			}
			continue
		}

		// We can't really do anything if we're IDLE'ing.
		if state == StateInIDLE {
			continue
		}

		if state != StateNone {
			log.WithField("state", state).Panicf("invalid_state")
		}

		wantProc := canProcess()
		if wantProc {
			if fetchFlag.IsFlagged() && !quitFlag.IsFlagged() {
				state = StateInFetch
				fetchFlag.Reset()
				go func() { opChan <- doFetch(mr.client, mr.imapChannel) }()
			} else {
				state = StateInDelete
				go func(toProcess map[uint32]*messageState) { opChan <- doDelete(mr.client, mr.imapChannel, toProcess) }(nextToProcess)
				nextToProcess = map[uint32]*messageState{}
			}
			continue
		}

		if quitFlag.IsFlagged() {
			if len(nextToProcess) > 0 || len(mr.messages) > 0 {
				opChan <- struct{}{}
				continue
			}

			break
		}

		state = StateInIDLE
		go func() {
			opChan <- mr.client.Idle(stopIdleChannel, &client2.IdleOptions{
				LogoutTimeout: 250 * time.Second, // Yahoo kills us after 5 mintues
				PollInterval:  mr.tickInterval,
			})
		}()
	}

	mr.hasQuit <- struct{}{}
}

func (mr *MailReceiver) Close() {
	mr.wantQuit <- struct{}{}
	<-mr.hasQuit
	_ = mr.client.Logout()
}
