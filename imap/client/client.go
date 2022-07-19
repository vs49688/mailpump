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

package client

import (
	"os"
	"time"

	"github.com/emersion/go-imap/client"
	"github.com/vs49688/mailpump/imap"
)

type Factory struct{}

func (f *Factory) NewClient(cfg *imap.Config) (imap.Client, error) {
	return NewClient(cfg)
}

func NewClient(cfg *imap.Config) (imap.Client, error) {
	var c *client.Client
	var err error
	if cfg.TLS {
		c, err = client.DialTLS(cfg.HostPort, cfg.TLSConfig)
	} else {
		c, err = client.Dial(cfg.HostPort)
	}

	if err != nil {
		return nil, err
	}

	c.Updates = cfg.Updates

	wantCleanup := true
	defer func() {
		if wantCleanup {
			_ = c.Logout()
		}
	}()

	if cfg.Debug {
		c.SetDebug(os.Stderr)
	}

	if err := cfg.Auth.Authenticate(c); err != nil {
		return nil, err
	}

	wantCleanup = false
	return &standardClient{c: c}, nil
}

type standardClient struct {
	c *client.Client
}

func (c *standardClient) Select(name string, readOnly bool) (*imap.MailboxStatus, error) {
	return c.c.Select(name, readOnly)
}

func (c *standardClient) Idle(stop <-chan struct{}, opts *client.IdleOptions) error {
	return c.c.Idle(stop, opts)
}

func (c *standardClient) Fetch(seqset *imap.SeqSet, items []imap.FetchItem, ch chan *imap.Message) error {
	return c.c.Fetch(seqset, items, ch)
}

func (c *standardClient) Expunge(ch chan uint32) error {
	return c.c.Expunge(ch)
}

func (c *standardClient) UidStore(seqset *imap.SeqSet, item imap.StoreItem, value interface{}, ch chan *imap.Message) error {
	return c.c.UidStore(seqset, item, value, ch)
}

func (c *standardClient) Append(mbox string, flags []string, date time.Time, msg imap.Literal) error {
	return c.c.Append(mbox, flags, date, msg)
}

func (c *standardClient) Mailbox() *imap.MailboxStatus {
	return c.c.Mailbox()
}

func (c *standardClient) Logout() error {
	return c.c.Logout()
}

func (c *standardClient) LoggedOut() <-chan struct{} {
	return c.c.LoggedOut()
}

func (c *standardClient) FlagQuit() {
	/* nop */
}
