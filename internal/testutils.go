package internal

import (
	"net"
	"testing"

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

	l, err := net.Listen("tcp", "localhost:0")
	assert.NoError(t, err)
	if err != nil {
		t.FailNow()
	}

	go func() { err = s.Serve(l) }()

	return s, l.Addr().String(), mailbox
}
