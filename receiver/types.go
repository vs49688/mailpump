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
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	imap2 "mailpump/imap"
	"time"
)

type Config struct {
	HostPort  string
	Username  string
	Password  string
	Mailbox   string
	TLS       bool
	TLSConfig *tls.Config
	Debug     bool
	// TickInterval is the interval between "ticks" when no other
	// activity has occurred.
	TickInterval time.Duration
	BatchSize    uint
	Channel      chan<- *imap.Message
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

	messages     map[uint32]*messageState
	batchSize    uint
	tickInterval time.Duration

	hasQuit  chan struct{}
	wantQuit chan struct{}
}
