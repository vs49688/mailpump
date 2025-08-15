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
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"git.vs49688.net/zane/mailpump/imap"
)

func TestIdleCancellation(t *testing.T) {
	log.SetLevel(log.TraceLevel)
	f := Factory{}

	c, err := f.NewClient(&imap.ClientConfig{
		ConnectionConfig: imap.ConnectionConfig{
			HostPort:  "0.0.0.0:993",
			Auth:      imap.NewNormalAuthenticator("username", "password"),
			TLS:       false,
			TLSConfig: nil,
			Debug:     false,
		},
		Updates: nil,
	})
	assert.NoError(t, err)
	ch := make(chan error)

	go func() { ch <- c.Idle(nil, nil) }()
	time.Sleep(5 * time.Second)
	err = c.Logout()
	assert.NoError(t, err)

	err = <-ch
	assert.NoError(t, err)
}

func TestIdleAfterLogout(t *testing.T) {
	log.SetLevel(log.TraceLevel)
	f := Factory{}

	c, err := f.NewClient(&imap.ClientConfig{
		ConnectionConfig: imap.ConnectionConfig{
			HostPort:  "0.0.0.0:993",
			Auth:      imap.NewNormalAuthenticator("username", "password"),
			TLS:       false,
			TLSConfig: nil,
			Debug:     false,
		},
		Updates: nil,
	})
	assert.NoError(t, err)

	err = c.Logout()
	assert.NoError(t, err)

	err = c.Idle(nil, nil)
	assert.Error(t, err)
}
