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

	"github.com/emersion/go-sasl"

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

	FlagQuit()
}

// Authenticatable is the minimal set of functions that are
// used by an Authenticator. This is mainly to assist with mocking.
type Authenticatable interface {
	Login(username, password string) error
	Authenticate(auth sasl.Client) error
}

// Authenticator provides a common interface to authenticate via
// username/password and SASL.
type Authenticator interface {
	Authenticate(a Authenticatable) error
}

// ConnectionConfig contains the base configuration required
// for an IMAP connection
type ConnectionConfig struct {
	// HostPort is the combined HOSTNAME:PORT of the server
	HostPort string

	// Auth is the authenticator to use when connecting.
	Auth Authenticator

	// Mailbox is the default mailbox to be used when connecting.
	// This should only populated if the connection was parsed from an
	// imaps:// URL; In this case, it should set to the URL path without
	// the leading /.
	Mailbox string

	// TLS, if set, indicates that connection should use TLS.
	TLS bool

	// TLSConfig contains the options to use if connecting via TLS.
	// Ignored if TLS is false.
	TLSConfig *tls.Config

	// Debug, if set, enables verbose IMAP debug output
	Debug bool
}

type ClientConfig struct {
	ConnectionConfig
	Updates chan<- client.Update
}

type Factory interface {
	NewClient(cfg *ClientConfig) (Client, error)
}

type Message = imap.Message
type SeqSet = imap.SeqSet
type StoreItem = imap.StoreItem
type MailboxStatus = imap.MailboxStatus
type FetchItem = imap.FetchItem
type Literal = imap.Literal
