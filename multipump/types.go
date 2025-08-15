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

package multipump

import (
	"reflect"

	"git.vs49688.net/zane/mailpump/imap"
	"git.vs49688.net/zane/mailpump/ingest"
	"git.vs49688.net/zane/mailpump/receiver"
)

type Config struct {
	Destination     ingest.Config
	Sources         []receiver.Config
	TargetMailboxes []string

	DoneChan chan<- error
	StopChan <-chan struct{}
}

type MultiPump interface {
	Close()
}

type multiPump struct {
	ingestClient ingest.Client
	receivers    []receiver.Client

	recvChannels    []chan *imap.Message
	ingestChannels  []chan ingest.Response
	targetMailboxes []string

	cases            []reflect.SelectCase
	recvBaseOffset   int
	ingestBaseOffset int
	exitOffset       int
}
