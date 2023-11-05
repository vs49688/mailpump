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

package internal

import (
	"testing"

	"golang.org/x/net/nettest"

	"github.com/emersion/go-imap/backend/memory"
	"github.com/emersion/go-imap/server"
	"github.com/stretchr/testify/assert"
)

func BuildTestIMAPServer(t *testing.T) (*server.Server, string, *memory.Mailbox) {
	be := memory.New()
	user, err := be.Login(nil, "username", "password")
	assert.NoError(t, err)
	if err != nil {
		t.FailNow()
	}

	mb, err := user.GetMailbox("INBOX")
	assert.NoError(t, err)
	if err != nil {
		t.FailNow()
	}

	mailbox := mb.(*memory.Mailbox)
	mailbox.Messages = nil

	s := server.New(be)
	t.Cleanup(func() { _ = s.Close() })

	s.AllowInsecureAuth = true

	l, err := nettest.NewLocalListener("tcp")
	if err != nil {
		t.Logf("%v", err)
		t.Skip()
	}

	go func() { err = s.Serve(l) }()

	return s, l.Addr().String(), mailbox
}
