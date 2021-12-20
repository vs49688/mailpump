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

package persistentclient

import (
	"crypto/tls"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	imap2 "github.com/vs49688/mailpump/imap"
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
	MaxDelay  time.Duration
	Updates   chan<- client.Update
}

type idleRequest struct {
	r chan error

	stop <-chan struct{}
	opts *client.IdleOptions
}

type selectResponse struct {
	status *imap.MailboxStatus
	err    error
}

type selectRequest struct {
	r chan selectResponse

	name     string
	readOnly bool
}

type fetchRequest struct {
	r chan error

	seqset *imap.SeqSet
	items  []imap.FetchItem
	ch     chan *imap.Message
}

type expungeRequest struct {
	r chan error

	ch chan uint32
}

type uidStoreRequest struct {
	r chan error

	seqset *imap.SeqSet
	item   imap.StoreItem
	value  interface{}
	ch     chan *imap.Message
}

type appendRequest struct {
	r chan error

	mbox  string
	flags []string
	date  time.Time
	msg   imap.Literal
}

type mailboxRequest struct {
	r chan *imap.MailboxStatus
}

type logoutRequest struct {
	r chan error
}

type clientState int32

const (
	ClientStateDisconnected clientState = 0
	ClientStateConnected    clientState = 1
)

type PersistentIMAPClient struct {
	c             imap2.Client
	cfg           Config
	ch            chan interface{}
	logoutChannel chan logoutRequest
	shutdown      int32
	loggedOut     chan struct{}
	logURL        string
}

type Factory struct {
	Mailbox  string
	MaxDelay time.Duration
}
