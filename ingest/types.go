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

package ingest

import (
	"crypto/tls"

	"github.com/emersion/go-imap"
	imap2 "github.com/vs49688/mailpump/imap"
)

type Config struct {
	HostPort  string
	Auth      imap2.Authenticator
	Mailbox   string
	TLS       bool
	TLSConfig *tls.Config
	Debug     bool
	DoneChan  chan<- error
}

type Response struct {
	UID   uint32
	Error error
}

type Client interface {
	IngestMessage(msg *imap.Message, ch chan<- Response) error

	Close()
}

type request struct {
	UID     uint32
	Message *imap.Message
	ch      chan<- Response
}

type ingestClient struct {
	client        imap2.Client
	rfc822Section *imap.BodySectionName
	incoming      chan request
	mbox          string
	hasQuit       chan struct{}
	wantQuit      chan struct{}
	shutdown      int32
}
