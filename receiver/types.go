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
	"crypto/tls"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	imap2 "github.com/vs49688/mailpump/imap"
)

type Config struct {
	HostPort  string
	Username  string
	Password  string
	Mailbox   string
	TLS       bool
	TLSConfig *tls.Config
	Debug     bool

	IDLEFallbackInterval time.Duration
	BatchSize            uint
	FetchBufferSize      uint
	FetchMaxInterval     time.Duration
	Channel              chan<- *imap.Message

	// DisableDeletions if set, will cause deletion requests to be ignored.
	// This is intended solely as a data-loss prevention measure when debugging
	// against live accounts.
	DisableDeletions bool
}

type ackRequest struct {
	UID   uint32
	Error error
}

type state int

const (
	StateUnacked state = 0
	StateAcked   state = 1
	StateDeleted state = 2
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

type operation int

const (
	OperationNone         operation = 0
	OperationIDLEFinish   operation = 1
	OperationFetchFinish  operation = 2
	OperationDeleteFinish operation = 3
	OperationTimeout      operation = 4
)

func (op operation) String() string {
	switch op {
	case OperationNone:
		return "none"
	case OperationIDLEFinish:
		return "idle_finish"
	case OperationFetchFinish:
		return "fetch_finish"
	case OperationDeleteFinish:
		return "delete_finish"
	case OperationTimeout:
		return "timeout"
	default:
		panic("invalid state")
	}
}

type messageState struct {
	UID     uint32
	SeqNum  uint32
	Message *imap.Message
	State   state
}

type fetchResult struct {
	UIDs     []uint32
	Messages map[uint32]*imap.Message
}

type deleteResult struct {
	UID   uint32
	State state
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

type MailReceiver struct {
	client imap2.Client

	// client -> imap handler, state updates
	updates chan client.Update

	// imap handler -> receiver, fetch & delete updates
	imapChannel chan interface{}

	// external -> receiver, incoming acks
	ackChannel chan ackRequest

	// receiver -> imap handler, message state updates
	updateChannel chan *messageState

	// receiver -> external, message notifications
	outChannel chan<- *imap.Message

	messages             map[uint32]*messageState
	batchSize            uint
	idleFallbackInterval time.Duration
	fetchBufferSize      uint
	fetchMaxInterval     time.Duration
	disableDeletions     bool

	hasQuit  chan struct{}
	wantQuit chan struct{}
}
