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
	"errors"
	"math/rand"
	"net/url"
	"sync/atomic"
	"time"

	goImapClient "github.com/emersion/go-imap/client"
	log "github.com/sirupsen/logrus"
	"github.com/vs49688/mailpump/imap"
	"github.com/vs49688/mailpump/imap/client"
)

var errConnectionClosed = errors.New("connection closed")

func (c *PersistentIMAPClient) isShutdown() bool {
	return atomic.LoadInt32(&c.shutdown) != 0
}

func (c *PersistentIMAPClient) Idle(stop <-chan struct{}, opts *goImapClient.IdleOptions) error {
	shutdown := c.isShutdown()
	c.log().WithField("shutdown", shutdown).Trace("pimap_idle_invoked")
	if shutdown {
		return errConnectionClosed
	}

	r := make(chan error)
	c.idleChannel <- idleRequest{
		r:    r,
		stop: stop,
		opts: opts,
	}
	return <-r
}

func (c *PersistentIMAPClient) Select(name string, readOnly bool) (*imap.MailboxStatus, error) {
	shutdown := c.isShutdown()
	c.log().WithField("shutdown", shutdown).Trace("pimap_select_invoked")
	if shutdown {
		return nil, errConnectionClosed
	}

	r := make(chan selectResponse)
	c.ch <- selectRequest{
		r:        r,
		name:     name,
		readOnly: readOnly,
	}
	sr := <-r
	return sr.status, sr.err
}

func (c *PersistentIMAPClient) Fetch(seqset *imap.SeqSet, items []imap.FetchItem, ch chan *imap.Message) error {
	shutdown := c.isShutdown()
	c.log().WithField("shutdown", shutdown).Trace("pimap_fetch_invoked")
	if shutdown {
		if ch != nil {
			close(ch)
		}
		return errConnectionClosed
	}

	r := make(chan error)
	c.ch <- fetchRequest{
		r:      r,
		seqset: seqset,
		items:  items,
		ch:     ch,
	}
	return <-r
}

func (c *PersistentIMAPClient) Expunge(ch chan uint32) error {
	shutdown := c.isShutdown()
	c.log().WithField("shutdown", shutdown).Trace("pimap_expunge_invoked")
	if shutdown {
		if ch != nil {
			close(ch)
		}
		return errConnectionClosed
	}

	r := make(chan error)
	c.ch <- expungeRequest{
		r:  r,
		ch: ch,
	}
	return <-r
}

func (c *PersistentIMAPClient) UidStore(seqset *imap.SeqSet, item imap.StoreItem, value interface{}, ch chan *imap.Message) error {
	shutdown := c.isShutdown()
	c.log().WithField("shutdown", shutdown).Trace("pimap_uidstore_invoked")
	if shutdown {
		if ch != nil {
			close(ch)
		}
		return errConnectionClosed
	}

	r := make(chan error)
	c.ch <- uidStoreRequest{
		r:      r,
		seqset: seqset,
		item:   item,
		value:  value,
		ch:     ch,
	}
	return <-r
}

func (c *PersistentIMAPClient) Append(mbox string, flags []string, date time.Time, msg imap.Literal) error {
	shutdown := c.isShutdown()
	c.log().WithField("shutdown", shutdown).Trace("pimap_append_invoked")
	if shutdown {
		return errConnectionClosed
	}

	r := make(chan error)
	c.ch <- appendRequest{
		r:     r,
		mbox:  mbox,
		flags: flags,
		date:  date,
		msg:   msg,
	}
	return <-r
}

func (c *PersistentIMAPClient) Mailbox() *imap.MailboxStatus {
	shutdown := c.isShutdown()
	c.log().WithField("shutdown", shutdown).Trace("pimap_mailbox_invoked")
	if shutdown {
		// TODO: track the selection state properly and return nil if neededq
		return &imap.MailboxStatus{Name: c.cfg.Mailbox}
	}

	r := make(chan *imap.MailboxStatus)
	c.ch <- mailboxRequest{r: r}
	return <-r
}

func (c *PersistentIMAPClient) Logout() error {
	shutdown := !atomic.CompareAndSwapInt32(&c.shutdown, 0, 1)
	c.log().WithField("shutdown", shutdown).Trace("pimap_logout_invoked")
	if shutdown {
		return nil
	}

	r := make(chan error)
	c.logoutChannel <- logoutRequest{r: r}
	return <-r
}

func (c *PersistentIMAPClient) LoggedOut() <-chan struct{} {
	return c.loggedOut
}

func (c *PersistentIMAPClient) FlagQuit() {
	shutdown := c.isShutdown()
	c.log().WithField("shutdown", shutdown).Trace("pimap_flagquit_invoked")
	if shutdown {
		return
	}

	go c.Logout()
}

func (c *PersistentIMAPClient) log() *log.Entry {
	e := log.WithField("url", c.logURL)
	if log.IsLevelEnabled(log.TraceLevel) {
		e = e.WithField("now", time.Now().UnixNano())
	}
	return e
}

func makeAndInitClient(cfg *Config, readOnly bool) (imap.Client, error) {
	c, err := client.NewClient(&imap.ClientConfig{
		ConnectionConfig: imap.ConnectionConfig{
			HostPort:  cfg.HostPort,
			Auth:      cfg.Auth,
			TLS:       cfg.TLS,
			TLSConfig: cfg.TLSConfig,
			Debug:     cfg.Debug,
		},
		Updates: cfg.Updates,
	})

	if err != nil {
		return nil, err
	}

	if cfg.Mailbox != "" {
		if _, err = c.Select(cfg.Mailbox, readOnly); err != nil {
			_ = c.Logout()
			return nil, err
		}
	}

	return c, err
}

func (c *PersistentIMAPClient) run() {
	var nextDelay time.Duration = 0
	var logout logoutRequest
	state := ClientStateDisconnected
	for {
		c.log().WithFields(log.Fields{
			"state":     state,
			"fake_idle": c.idle != nil,
		}).Trace("pimap_loop_enter")
		if state == ClientStateDisconnected {
			select {
			case <-c.stopIdle:
				if c.idle == nil {
					panic("not in idle")
				}

				// Stop IDLE during disconnect
				c.log().Trace("pimap_fake_idle_stop")
				c.idle.r <- nil
				c.idle = nil
				c.stopIdle = nil
			case r := <-c.idleChannel:
				// We're disconnected, special IDLE handling
				if c.idle != nil {
					panic("already in idle")
				}
				c.log().Trace("pimap_fake_idle_start")
				c.idle = &r
				c.stopIdle = r.stop
			case req := <-c.logoutChannel:
				c.log().WithField("fake_idle", c.idle != nil).Trace("pimap_logout_request")
				logout = req
				if c.idle != nil {
					c.log().Trace("pimap_fake_idle_stop")
					c.idle.r <- nil
					c.idle = nil
					c.stopIdle = nil
				}
				goto done
			case <-time.After(nextDelay):
				break
			}

			cli, err := makeAndInitClient(&c.cfg, false)
			if err != nil {
				if nextDelay == 0 {
					nextDelay = time.Second
				} else {
					nextDelay = 2 * (nextDelay - (nextDelay % (1000 * time.Millisecond)))
				}

				// #nosec G404 -- Not used for crypto
				nextDelay += time.Duration(rand.Intn(1000)) * time.Millisecond
				if nextDelay > c.cfg.MaxDelay {
					nextDelay = c.cfg.MaxDelay
				}

				c.log().WithError(err).WithFields(log.Fields{
					"new_delay": nextDelay,
				}).Error("pimap_connection_failed")
				continue
			}

			c.c = cli
			state = ClientStateConnected
			nextDelay = time.Second
		}

		if state == ClientStateConnected {
			c.log().WithField("state", state).Trace("pimap_entering_connected_select")

			// Upgrade to a "real" IDLE
			if c.idle != nil {
				c.log().Trace("pimap_fake_idle_upgrade")
				stop := c.stopIdle
				c.stopIdle = nil
				c.log().Trace("pimap_fake_idle_upgrade_enter")
				c.idle.r <- c.c.Idle(stop, c.idle.opts)
				c.log().Trace("pimap_fake_idle_upgrade_exit")
				c.idle = nil
				continue
			}

			select {
			case <-c.c.LoggedOut():
				c.log().Trace("pimap_disconnected")
				c.c = nil
				state = ClientStateDisconnected
			case req := <-c.logoutChannel:
				c.log().Trace("pimap_logout_request")
				logout = req
				goto done
			case req := <-c.idleChannel:
				// We're connected, no special IDLE handling
				c.log().Trace("pimap_idle_request_before")
				req.r <- c.c.Idle(req.stop, req.opts)
				c.log().Trace("pimap_idle_request_after")
			case _req := <-c.ch:
				switch req := _req.(type) {
				case selectRequest:
					c.log().Trace("pimap_select_request")
					s, err := c.c.Select(req.name, req.readOnly)
					req.r <- selectResponse{status: s, err: err}
				case fetchRequest:
					c.log().Trace("pimap_fetch_request")
					req.r <- c.c.Fetch(req.seqset, req.items, req.ch)
				case expungeRequest:
					c.log().Trace("pimap_expunge_request")
					req.r <- c.c.Expunge(req.ch)
				case uidStoreRequest:
					c.log().Trace("pimap_uidstore_request")
					req.r <- c.c.UidStore(req.seqset, req.item, req.value, req.ch)
				case appendRequest:
					c.log().Trace("pimap_append_request")
					req.r <- c.c.Append(req.mbox, req.flags, req.date, req.msg)
				case mailboxRequest:
					c.log().Trace("pimap_mailbox_request")
					req.r <- c.c.Mailbox()
				}
			}
		}
	}
done:
	atomic.StoreInt32(&c.shutdown, 1)
	// NB: At this point, we've shut down any new requests, but
	// there may be ones queued up.

	logout.r <- nil
	if c.c != nil {
		err := c.c.Logout()
		if err != nil {
			log.WithError(err).Info("logout_failed")
		}
		c.c = nil
	}
	c.drainRequests()
	close(c.ch)
	close(c.idleChannel)
	close(c.logoutChannel)
	close(c.loggedOut)
	c.log().Trace("pimap_proc_exit")
}

func (c *PersistentIMAPClient) drainRequests() {
	count := 0
	for {
		select {
		case req := <-c.logoutChannel:
			count += 1
			req.r <- nil
		case req, ok := <-c.idleChannel:
			if !ok {
				continue
			}
			count += 1
			req.r <- errConnectionClosed
		case _req := <-c.ch:
			count += 1
			switch req := _req.(type) {
			case idleRequest:
				req.r <- errConnectionClosed
			case fetchRequest:
				req.r <- errConnectionClosed
			case expungeRequest:
				req.r <- errConnectionClosed
			case uidStoreRequest:
				req.r <- errConnectionClosed
			case appendRequest:
				req.r <- errConnectionClosed
			case mailboxRequest:
				req.r <- &imap.MailboxStatus{Name: c.cfg.Mailbox}
			}
		default:
			goto done
		}
	}
done:
}

func NewClient(cfg *Config) (imap.Client, error) {
	ourCfg := *cfg
	if ourCfg.MaxDelay == 0 {
		ourCfg.MaxDelay = 64 * time.Second
	} else if ourCfg.MaxDelay < time.Second {
		ourCfg.MaxDelay = time.Second
	}

	u := url.URL{
		Host: ourCfg.HostPort,
		Path: ourCfg.Mailbox,
	}

	if ourCfg.TLS {
		u.Scheme = "imaps"
	} else {
		u.Scheme = "imap"
	}

	c := &PersistentIMAPClient{
		cfg:           ourCfg,
		ch:            make(chan interface{}),
		logoutChannel: make(chan logoutRequest),
		shutdown:      0,
		loggedOut:     make(chan struct{}),
		logURL:        u.String(),
		idle:          nil,
		idleChannel:   make(chan idleRequest),
		stopIdle:      nil,
	}
	go c.run()
	return c, nil
}
