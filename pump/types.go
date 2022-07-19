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
	"time"

	"github.com/vs49688/mailpump/imap"
	"github.com/vs49688/mailpump/ingest"
	"github.com/vs49688/mailpump/receiver"
)

type Config struct {
	Source imap.ConnectionConfig
	Dest   imap.ConnectionConfig

	SourceFactory imap.Factory
	DestFactory   imap.Factory

	IDLEFallbackInterval time.Duration
	BatchSize            uint
	DisableDeletions     bool
	FetchBufferSize      uint
	FetchMaxInterval     time.Duration

	DoneChan chan<- error
	StopChan <-chan struct{}
}

type MailPump struct {
	receiver      receiver.Client
	ingest        ingest.Client
	destMailbox   string
	incoming      chan *imap.Message
	ingestChannel chan ingest.Response
}
