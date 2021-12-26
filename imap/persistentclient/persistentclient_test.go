package persistentclient

import (
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/vs49688/mailpump/imap"
)

func TestIdleCancellation(t *testing.T) {
	log.SetLevel(log.TraceLevel)
	f := &Factory{}

	c, err := f.NewClient(&imap.ClientConfig{
		HostPort:  "0.0.0.0:993",
		Username:  "username",
		Password:  "password",
		TLS:       false,
		TLSConfig: nil,
		Debug:     false,
		Updates:   nil,
	})
	assert.NoError(t, err)
	ch := make(chan error)

	go func() { ch <- c.Idle(nil, nil) }()
	time.Sleep(5 * time.Second)
	err = c.Logout()
	assert.NoError(t, err)

	err = <-ch
	assert.NoError(t, err)
}

func TestIdleAfterLogout(t *testing.T) {
	log.SetLevel(log.TraceLevel)
	f := &Factory{}

	c, err := f.NewClient(&imap.ClientConfig{
		HostPort:  "0.0.0.0:993",
		Username:  "username",
		Password:  "password",
		TLS:       false,
		TLSConfig: nil,
		Debug:     false,
		Updates:   nil,
	})
	assert.NoError(t, err)

	err = c.Logout()
	assert.NoError(t, err)

	err = c.Idle(nil, nil)
	assert.Error(t, err)
}
