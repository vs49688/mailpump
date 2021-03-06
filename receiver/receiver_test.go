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

package receiver

import (
	"bytes"
	"crypto/tls"
	"strings"
	"testing"
	"time"

	"github.com/vs49688/mailpump/internal"

	imap2 "github.com/vs49688/mailpump/imap"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-message"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/vs49688/mailpump/imap/client"
	"github.com/vs49688/mailpump/imap/persistentclient"
	"github.com/vs49688/mailpump/ingest"
)

func makeTestMessage(t *testing.T, messageID string) (*imap.Message, int32) {
	rfc822Section, _ := imap.ParseBodySectionName(imap.FetchRFC822)

	hdr := message.Header{}
	hdr.Add("From", "from@example.com")
	hdr.Add("To", "to@example.com")
	hdr.Add("Subject", "Test Email")
	hdr.Add("Date", "Wed, 11 May 2016 14:31:59 +0000")
	hdr.Add("Content-Type", "text/plain")
	hdr.Add("Message-ID", messageID)

	msg, err := message.New(hdr, strings.NewReader("Привет!"))
	//msg, err := message.NewMultipart(hdr, []*message.Entity{})
	assert.NoError(t, err)
	if err != nil {
		t.FailNow()
	}

	bb := new(bytes.Buffer)
	_ = msg.WriteTo(bb)
	assert.NoError(t, err)
	if err != nil {
		t.FailNow()
	}

	imsg := imap.NewMessage(1, []imap.FetchItem{imap.FetchRFC822})
	imsg.Body[rfc822Section] = imap.Literal(bb)
	return imsg, int32(bb.Len())
}

func TestReceiver(t *testing.T) {
	log.SetLevel(log.TraceLevel)

	_, addr, _ := internal.BuildTestIMAPServer(t)

	ing, err := ingest.NewClient(&ingest.Config{
		ConnectionConfig: imap2.ConnectionConfig{
			HostPort:  addr,
			Auth:      imap2.NewNormalAuthenticator("username", "password"),
			TLS:       false,
			TLSConfig: &tls.Config{InsecureSkipVerify: true},
			Debug:     false,
		},
		Factory: client.Factory{},
	})

	// Add an initial message, the receiver should check this
	testMsg, _ := makeTestMessage(t, "<01@localhost>")
	testMsg.Uid = 1
	err = ingest.IngestMessageSync("INBOX", ing, testMsg)
	assert.NoError(t, err)

	ch := make(chan *imap.Message, 1)
	receiver, err := NewReceiver(&Config{
		ConnectionConfig: imap2.ConnectionConfig{
			HostPort:  addr,
			Auth:      imap2.NewNormalAuthenticator("username", "password"),
			Mailbox:   "INBOX",
			TLS:       false,
			TLSConfig: &tls.Config{InsecureSkipVerify: true},
			Debug:     false,
		},
		Factory:              persistentclient.Factory{},
		Channel:              ch,
		IDLEFallbackInterval: 1 * time.Second,

		// The go-imap server doesn't always send mailbox updates,
		// so depending on which state the receiver's in when this is
		// ingested, we may need a force-fetch.
		FetchMaxInterval: 5 * time.Second,
	})
	assert.NoError(t, err)
	defer receiver.Close()

	// Get our initial message and Ack it
	msg := <-ch
	assert.Equal(t, uint32(1), msg.Uid)
	receiver.Ack(msg.Uid, nil)

	t.Log("Ingesting Message 2")
	// Add another message, the receiver should receive it via IDLE
	// or a force-fetch via timeout
	testMsg, _ = makeTestMessage(t, "<02@localhost>")
	testMsg.Uid = 2
	err = ingest.IngestMessageSync("INBOX", ing, testMsg)
	assert.NoError(t, err)

	t.Log("Waiting for message 2")
	msg = <-ch
	assert.Equal(t, uint32(2), msg.Uid)
	close(ch)
	t.Log("Got message 2, ACK'ing...")
	receiver.Ack(msg.Uid, nil)
	t.Log("ACK'ed message 2")
}

func TestLogoutWhenDisconnected(t *testing.T) {
	log.SetLevel(log.TraceLevel)
	ch := make(chan *imap.Message, 1)
	receiver, err := NewReceiver(&Config{
		ConnectionConfig: imap2.ConnectionConfig{
			HostPort:  "0.0.0.0:993",
			Auth:      imap2.NewNormalAuthenticator("username", "password"),
			Mailbox:   "INBOX",
			TLS:       false,
			TLSConfig: nil,
			Debug:     true,
		},
		Factory:              persistentclient.Factory{},
		Channel:              ch,
		IDLEFallbackInterval: 1 * time.Second,
	})
	assert.NoError(t, err)
	time.Sleep(500 * time.Millisecond)
	receiver.Close()
}

// TestImmediateLogout tests the case where Logout()
// is called immediately after creation. This can sometimes
// cause a race.
func TestImmediateLogout(t *testing.T) {
	log.SetLevel(log.TraceLevel)

	_, addr, _ := internal.BuildTestIMAPServer(t)

	ch := make(chan *imap.Message, 1)
	receiver, err := NewReceiver(&Config{
		ConnectionConfig: imap2.ConnectionConfig{
			HostPort:  addr,
			Auth:      imap2.NewNormalAuthenticator("username", "password"),
			Mailbox:   "INBOX",
			TLS:       false,
			TLSConfig: nil,
			Debug:     true,
		},
		Factory:              client.Factory{},
		Channel:              ch,
		IDLEFallbackInterval: 1 * time.Second,
	})
	assert.NoError(t, err)
	defer receiver.Close()
}

func TestSequenceGeneration(t *testing.T) {
	mbStatus := imap.MailboxStatus{
		Name:     "INBOX",
		Messages: 53,
	}

	mr := mailReceiver{
		messages: map[uint32]*messageState{
			1:  {UID: 1, SeqNum: 1},
			2:  {UID: 2, SeqNum: 2},
			10: {UID: 10, SeqNum: 10},
		},
		fetchBufferSize: 20,
	}

	existing := imap.SeqSet{}
	for _, k := range mr.messages {
		existing.AddNum(k.SeqNum)
	}

	toFetch := buildSeqSet(existing, &mbStatus, mr.fetchBufferSize)

	expected := imap.SeqSet{}
	expected.AddRange(3, 9)
	expected.AddRange(11, 23)
	assert.Equal(t, expected, toFetch)

	for i := 3; i <= 9; i++ {
		mr.messages[uint32(i)] = &messageState{UID: uint32(i), SeqNum: uint32(i)}
	}

	for i := 11; i <= 23; i++ {
		mr.messages[uint32(i)] = &messageState{UID: uint32(i), SeqNum: uint32(i)}
	}

	existing = imap.SeqSet{}
	for _, k := range mr.messages {
		existing.AddNum(k.SeqNum)
	}

	expected = imap.SeqSet{}
	expected.AddRange(24, 43)
	toFetch = buildSeqSet(existing, &mbStatus, mr.fetchBufferSize)
	assert.Equal(t, expected, toFetch)
}
