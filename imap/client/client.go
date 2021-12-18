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
	"github.com/emersion/go-imap/client"
	"github.com/vs49688/mailpump/imap"
	"os"
)

type Factory struct{}

func (f *Factory) NewClient(cfg *imap.ClientConfig) (imap.Client, error) {
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

	if err := c.Login(cfg.Username, cfg.Password); err != nil {
		return nil, err
	}

	wantCleanup = false
	return c, nil
}
