package run_multi

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vs49688/mailpump/cmd/config"
	"github.com/vs49688/mailpump/ingest"
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
		Sources: []Source{
			{
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
			{
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
	}, cfg)
}
