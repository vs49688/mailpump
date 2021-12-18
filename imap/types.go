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

package imap

import (
	"crypto/tls"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

type Client interface {
	Select(name string, readOnly bool) (*imap.MailboxStatus, error)

	Idle(stop <-chan struct{}, opts *client.IdleOptions) error

	Fetch(seqset *imap.SeqSet, items []imap.FetchItem, ch chan *imap.Message) error

	Expunge(ch chan uint32) error

	UidStore(seqset *imap.SeqSet, item imap.StoreItem, value interface{}, ch chan *imap.Message) error

	Append(mbox string, flags []string, date time.Time, msg imap.Literal) error

	Mailbox() *imap.MailboxStatus

	Logout() error

	LoggedOut() <-chan struct{}
}

type ClientConfig struct {
	HostPort  string
	Username  string
	Password  string
	TLS       bool
	TLSConfig *tls.Config
	Debug     bool
	Updates   chan<- client.Update
}

type ClientFactory interface {
	NewClient(cfg *ClientConfig) (Client, error)
}

type Message = imap.Message
type SeqSet = imap.SeqSet
type StoreItem = imap.StoreItem
type MailboxStatus = imap.MailboxStatus
type FetchItem = imap.FetchItem
type Literal = imap.Literal