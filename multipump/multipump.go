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
	"errors"
	"reflect"

	"github.com/emersion/go-imap"
	log "github.com/sirupsen/logrus"
	"git.vs49688.net/zane/mailpump/ingest"
	"git.vs49688.net/zane/mailpump/receiver"
)

func makeReceivers(sources []receiver.Config) ([]receiver.Client, error) {
	var savedErr error

	recvs := make([]receiver.Client, 0, len(sources))
	for i := range sources {
		recv, err := receiver.NewReceiver(&sources[i])
		if err != nil {
			log.WithError(err).Error("error_closing_receiver")
			savedErr = err
			goto failed
		}
		recvs = append(recvs, recv)
	}

	return recvs, nil

failed:
	closeAndWait(recvs...)

	return nil, savedErr
}

func NewPump(cfg *Config) (MultiPump, error) {
	var err error

	if len(cfg.Sources) == 0 {
		return nil, errors.New("no sources configured")
	}

	if len(cfg.Sources) != len(cfg.TargetMailboxes) {
		return nil, errors.New("mismatching source configuration/mailbox pairs")
	}

	pump := &multiPump{}

	pump.targetMailboxes = cfg.TargetMailboxes

	// Our config comes from the user, don't trust their channels
	pump.recvChannels = make([]chan *imap.Message, len(cfg.Sources))
	for i := 0; i < len(cfg.Sources); i++ {
		pump.recvChannels[i] = make(chan *imap.Message, 20)
		cfg.Sources[i].Channel = pump.recvChannels[i]
	}

	// Make one ingest channel per source so we know which one to ack
	pump.ingestChannels = make([]chan ingest.Response, len(cfg.Sources))
	for i := 0; i < len(cfg.Sources); i++ {
		pump.ingestChannels[i] = make(chan ingest.Response, 10)
	}

	// Build the switch cases
	pump.cases = make([]reflect.SelectCase, 2*len(cfg.Sources)+1)

	pump.recvBaseOffset = 0
	for i := 0; i < len(cfg.Sources); i++ {
		pump.cases[pump.recvBaseOffset+i] = reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(pump.recvChannels[i]),
		}
	}

	pump.ingestBaseOffset = pump.recvBaseOffset + len(cfg.Sources)
	for i := 0; i < len(cfg.Sources); i++ {
		pump.cases[pump.ingestBaseOffset+i] = reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(pump.ingestChannels[i]),
		}
	}

	pump.exitOffset = pump.ingestBaseOffset + len(cfg.Sources)
	pump.cases[pump.exitOffset] = reflect.SelectCase{
		Dir:  reflect.SelectRecv,
		Chan: reflect.ValueOf(cfg.StopChan),
	}

	if pump.ingestClient, err = ingest.NewClient(&cfg.Destination); err != nil {
		return nil, err
	}

	if pump.receivers, err = makeReceivers(cfg.Sources); err != nil {
		closeAndWait(pump.ingestClient)
		return nil, err
	}

	go func() { cfg.DoneChan <- pump.tick() }()

	return pump, nil
}

func (pump *multiPump) Close() {
	closeAndWait(pump.receivers...)
	closeAndWait(pump.ingestClient)
}

func (pump *multiPump) tick() error {
	for {
		chosen, val, ok := reflect.Select(pump.cases)

		if chosen >= pump.recvBaseOffset && chosen < pump.ingestBaseOffset {
			msg := val.Interface().(*imap.Message)
			receiverIndex := chosen - pump.recvBaseOffset
			log.WithFields(log.Fields{
				"receiver": receiverIndex,
				"uid":      msg.Uid,
				"seq":      msg.SeqNum,
			}).Trace("pump_handle_incoming")
			if err := pump.ingestClient.IngestMessage(pump.targetMailboxes[receiverIndex], msg, pump.ingestChannels[receiverIndex]); err != nil {
				pump.receivers[chosen].Ack(msg.Uid, err)
			}
		} else if chosen >= pump.ingestBaseOffset && chosen < pump.exitOffset {
			r := val.Interface().(ingest.Response)
			receiverIndex := chosen - pump.ingestBaseOffset
			pump.receivers[receiverIndex].Ack(r.UID, r.Error)
		} else if chosen == pump.exitOffset || !ok {
			log.Trace("exit_requested")
			break
		} else {
			panic("unhandled select case")
		}
	}

	return nil
}
