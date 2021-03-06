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
	"errors"
	"sync/atomic"

	"github.com/emersion/go-imap"
	log "github.com/sirupsen/logrus"
	imap2 "github.com/vs49688/mailpump/imap"
)

func NewClient(cfg *Config) (Client, error) {
	rfc822Section, err := imap.ParseBodySectionName(imap.FetchRFC822)
	if err != nil {
		panic(err)
	}

	imapClient, err := cfg.Factory.NewClient(&imap2.ClientConfig{
		ConnectionConfig: cfg.ConnectionConfig,
		Updates:          nil,
	})

	if err != nil {
		return nil, err
	}

	ingest := &ingestClient{
		client:        imapClient,
		rfc822Section: rfc822Section,
		incoming:      make(chan request),
		hasQuit:       make(chan struct{}),
		wantQuit:      make(chan struct{}),
		shutdown:      0,
	}

	go ingest.run()
	return ingest, nil
}

var (
	errInvalidUID       = errors.New("invalid uid")
	errConnectionClosed = errors.New("connection closed")
)

func (ingest *ingestClient) isShutdown() bool {
	return atomic.LoadInt32(&ingest.shutdown) != 0
}

func (ingest *ingestClient) IngestMessage(mailbox string, msg *imap.Message, ch chan<- Response) error {
	log.WithFields(log.Fields{"mailbox": mailbox, "uid": msg.Uid, "seq": msg.SeqNum}).Trace("ingest_message")
	if msg.Uid == 0 {
		return errInvalidUID
	}

	if ingest.isShutdown() {
		return errConnectionClosed
	}

	ingest.incoming <- request{Mailbox: mailbox, UID: msg.Uid, Message: msg, ch: ch}
	return nil
}

func IngestMessageSync(mailbox string, ingestClient Client, msg *imap.Message) error {
	ch := make(chan Response)
	if err := ingestClient.IngestMessage(mailbox, msg, ch); err != nil {
		return err
	}

	res := <-ch
	if res.Error != nil {
		return res.Error
	}

	return nil
}

func (ingest *ingestClient) run() {
	for {
		select {
		case <-ingest.wantQuit:
			goto done
		case req := <-ingest.incoming:
			log.WithFields(log.Fields{
				"uid": req.UID,
				"seq": req.Message.SeqNum,
			}).Trace("ingest_start")
			err := ingest.client.Append(req.Mailbox, req.Message.Flags, req.Message.InternalDate, req.Message.GetBody(ingest.rfc822Section))
			if err != nil {
				log.WithError(err).WithFields(log.Fields{
					"uid": req.UID,
					"seq": req.Message.SeqNum,
				}).Error("ingest_failed")
			} else {
				log.WithFields(log.Fields{
					"uid": req.UID,
					"seq": req.Message.SeqNum,
				}).Info("ingest_success")
			}
			req.ch <- Response{UID: req.UID, Error: err}
		}
	}
done:
	atomic.StoreInt32(&ingest.shutdown, 1)
	drain(ingest.incoming)
	if err := ingest.client.Logout(); err != nil {
		log.WithError(err).Error("ingest_client_close_failed")
	}

	close(ingest.hasQuit)
}

func drain(ch chan request) {
	count := 0
	for {
		select {
		case req := <-ch:
			req.ch <- Response{UID: req.UID, Error: errConnectionClosed}
		default:
			goto done
		}
	}
done:
	close(ch)
	log.WithField("count", count).Trace("ingest_drained_requests")
}

func (ingest *ingestClient) Close() {
	close(ingest.wantQuit)
	_ = <-ingest.hasQuit
}
