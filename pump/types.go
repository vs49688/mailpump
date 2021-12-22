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

package pump

import (
	"crypto/tls"
	"github.com/vs49688/mailpump/imap"
	"github.com/vs49688/mailpump/ingest"
	"github.com/vs49688/mailpump/receiver"
	"time"
)

type Config struct {
	SourceHostPort  string
	SourceUsername  string
	SourcePassword  string
	SourceMailbox   string
	SourceTLS       bool
	SourceTLSConfig *tls.Config
	SourceFactory   imap.ClientFactory
	SourceDebug     bool

	DestHostPort  string
	DestUsername  string
	DestPassword  string
	DestMailbox   string
	DestTLS       bool
	DestTLSConfig *tls.Config
	DestTransport string
	DestFactory   imap.ClientFactory
	DestDebug     bool

	TickInterval time.Duration
	BatchSize    uint

	DoneChan chan<- error
	StopChan <-chan struct{}
}

type MailPump struct {
	receiver      *receiver.MailReceiver
	ingest        *ingest.Client
	incoming      chan *imap.Message
	eventChannel  chan interface{}
	ingestChannel chan ingest.Response
}
