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
	"github.com/emersion/go-imap"
	log "github.com/sirupsen/logrus"
	"git.vs49688.net/zane/mailpump/ingest"
	"git.vs49688.net/zane/mailpump/receiver"
)

func NewMailPump(cfg *Config) (*MailPump, error) {
	ch := make(chan *imap.Message, 20)

	recv, err := receiver.NewReceiver(&receiver.Config{
		ConnectionConfig:     cfg.Source,
		Factory:              cfg.SourceFactory,
		IDLEFallbackInterval: cfg.IDLEFallbackInterval,
		BatchSize:            cfg.BatchSize,
		DisableDeletions:     cfg.DisableDeletions,
		FetchBufferSize:      cfg.FetchBufferSize,
		FetchMaxInterval:     cfg.FetchMaxInterval,
		Channel:              ch,
	})

	if err != nil {
		return nil, err
	}

	ing, err := ingest.NewClient(&ingest.Config{
		ConnectionConfig: cfg.Dest,
		Factory:          cfg.DestFactory,
	})
	if err != nil {
		recv.Close()
		return nil, err
	}

	pump := &MailPump{
		receiver:      recv,
		ingest:        ing,
		destMailbox:   cfg.Dest.Mailbox,
		incoming:      ch,
		ingestChannel: make(chan ingest.Response, 10),
	}

	go func() { cfg.DoneChan <- pump.tick(cfg.StopChan) }()

	return pump, nil
}

func (pump *MailPump) Close() {
	ch := make(chan struct{}, 2)
	go func() { pump.receiver.Close(); ch <- struct{}{} }()
	go func() { pump.ingest.Close(); ch <- struct{}{} }()
	<-ch
	<-ch
}

func (pump *MailPump) tick(ch <-chan struct{}) error {
	for {
		select {
		case msg := <-pump.incoming:
			log.WithFields(log.Fields{
				"uid": msg.Uid,
				"seq": msg.SeqNum,
			}).Trace("pump_handle_incoming")
			if err := pump.ingest.IngestMessage(pump.destMailbox, msg, pump.ingestChannel); err != nil {
				pump.receiver.Ack(msg.Uid, err)
			}

		case r := <-pump.ingestChannel:
			pump.receiver.Ack(r.UID, r.Error)
		case <-ch:
			log.Trace("exit_requested")
			return nil
		}
	}
}
