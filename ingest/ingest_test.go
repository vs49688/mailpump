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
	"bytes"
	"strings"
	"testing"

	"github.com/vs49688/mailpump/internal"

	imap2 "github.com/vs49688/mailpump/imap"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-message"
	"github.com/stretchr/testify/assert"
	"github.com/vs49688/mailpump/imap/client"
	"github.com/vs49688/mailpump/imap/persistentclient"
)

func makeTestMessage(t *testing.T, messageID string) (*imap.Message, []byte, int32) {
	rfc822Section, _ := imap.ParseBodySectionName(imap.FetchRFC822)

	hdr := message.Header{}
	hdr.Add("From", "from@example.com")
	hdr.Add("To", "to@example.com")
	hdr.Add("Subject", "Test Email")
	hdr.Add("Date", "Wed, 11 May 2016 14:31:59 +0000")
	hdr.Add("Content-Type", "text/plain")
	hdr.Add("Message-ID", messageID)

	msg, err := message.New(hdr, strings.NewReader("Привет!"))
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
	return imsg, bb.Bytes(), int32(bb.Len())
}

func runIngestTest(t *testing.T, f func(string) (Client, error)) {
	_, addr, mailbox := internal.BuildTestIMAPServer(t)

	ingest, err := f(addr)
	defer ingest.Close()

	assert.NoError(t, err)

	msg, data, _ := makeTestMessage(t, "test@example.com")
	msg.Uid = 1
	err = ingest.IngestMessageSync(msg)
	assert.NoError(t, err)

	assert.Len(t, mailbox.Messages, 1)
	assert.Equal(t, data, mailbox.Messages[0].Body)
}

func TestIngestStandard(t *testing.T) {
	runIngestTest(t, func(addr string) (Client, error) {
		return NewClient(&Config{
			HostPort:  addr,
			Auth:      imap2.NewNormalAuthenticator("username", "password"),
			Mailbox:   "INBOX",
			TLS:       false,
			TLSConfig: nil,
			Debug:     true,
		}, &client.Factory{})
	})
}

func TestIngestPersistent(t *testing.T) {
	runIngestTest(t, func(addr string) (Client, error) {
		return NewClient(&Config{
			HostPort:  addr,
			Auth:      imap2.NewNormalAuthenticator("username", "password"),
			Mailbox:   "INBOX",
			TLS:       false,
			TLSConfig: nil,
			Debug:     true,
		}, &persistentclient.Factory{Mailbox: "INBOX"})
	})
}
