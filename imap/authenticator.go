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
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-sasl"
)

type plainAuthenticator struct {
	username string
	password string
}

func NewNormalAuthenticator(username string, password string) *plainAuthenticator {
	return &plainAuthenticator{username: username, password: password}
}

func (a *plainAuthenticator) Authenticate(c *client.Client) error {
	return c.Login(a.username, a.password)
}

type saslAuthenticator struct {
	client sasl.Client
}

func NewSASLAuthenticator(client sasl.Client) *saslAuthenticator {
	return &saslAuthenticator{client: client}
}

func (a *saslAuthenticator) Authenticate(c *client.Client) error {
	return c.Authenticate(a.client)
}
