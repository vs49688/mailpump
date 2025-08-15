package run_multi

import (
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/stretchr/testify/assert"
	"git.vs49688.net/zane/mailpump/cmd/config"
	"git.vs49688.net/zane/mailpump/ingest"
)

func TestConfigParse(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ConfigPath = "testdata/config.json"

	err := cfg.Resolve()
	assert.NoError(t, err)

	cfg.ResolvedDestination = ingest.Config{}
	cfg.ResolvedSources = nil

	assert.Equal(t, Configuration{
		ConfigPath: "testdata/config.json",
		Destination: config.IMAPConfig{
			URL:          "imaps://imap.example.com",
			Username:     "user1@example.com",
			AuthMethod:   "LOGIN",
			PasswordFile: "testdata/passfile1",
			Transport:    "persistent",
			OAuth2:       config.DefaultOAuth2Config(),
		},
		Sources: map[string]*Source{
			"yahoo-user-inbox": {
				Connection: config.IMAPConfig{
					URL:        "imaps://imap.mail.yahoo.com/INBOX",
					Username:   "user@yahoo.com.au",
					AuthMethod: "LOGIN",
					Password:   "direct_password",
					Transport:  "persistent",
					OAuth2:     config.DefaultOAuth2Config(),
				},
				TargetMailbox: "INBOX",
			},
			"yahoo-user-inbox-spam": {
				Connection: config.IMAPConfig{
					URL:          "imaps://imap.mail.yahoo.com/Bulk",
					Username:     "user@yahoo.com.au",
					AuthMethod:   "LOGIN",
					PasswordFile: "testdata/passfile2",
					Transport:    "persistent",
					OAuth2:       config.DefaultOAuth2Config(),
				},
				TargetMailbox: "Junk",
			},
		},
		LogLevel:  "info",
		LogFormat: "text",
		Logger:    logrus.StandardLogger(),
	}, cfg)
}
