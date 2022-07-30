package config

import (
	"crypto/tls"
	"os"
	"path"
	"testing"

	"github.com/emersion/go-sasl"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/vs49688/mailpump/imap"
	"github.com/vs49688/mailpump/imap/client"
	mock_imap "github.com/vs49688/mailpump/imap/mocks"
	"github.com/vs49688/mailpump/imap/persistentclient"
)

func getTestIMAPConfig() IMAPConfig {
	cfg := DefaultIMAPConfig()
	cfg.URL = "imaps://imap.hostname.com:1234/INBOX"
	cfg.Username = "username"
	cfg.Password = "password"

	return cfg
}

func TestIMAPConfig_Resolve(t *testing.T) {
	t.Run("factories", func(t *testing.T) {
		t.Run("persistent", func(t *testing.T) {
			cfg := getTestIMAPConfig()
			cfg.Transport = "persistent"

			_, factory, err := cfg.Resolve()
			assert.NoError(t, err)
			assert.Equal(t, persistentclient.Factory{MaxDelay: 0}, factory)
		})

		t.Run("standard", func(t *testing.T) {
			cfg := getTestIMAPConfig()
			cfg.Transport = "standard"

			_, factory, err := cfg.Resolve()
			assert.NoError(t, err)
			assert.Equal(t, client.Factory{}, factory)
		})

		t.Run("anything_else", func(t *testing.T) {
			cfg := getTestIMAPConfig()
			cfg.Transport = "anything_else"

			_, factory, err := cfg.Resolve()
			assert.NoError(t, err)
			assert.Equal(t, client.Factory{}, factory)
		})
	})

	t.Run("passwords", func(t *testing.T) {
		t.Run("password", func(t *testing.T) {
			cfg := getTestIMAPConfig()

			connConfig, _, err := cfg.Resolve()
			assert.NoError(t, err)
			assert.Equal(t, imap.ConnectionConfig{
				HostPort:  "imap.hostname.com:1234",
				Auth:      imap.NewNormalAuthenticator("username", "password"),
				Mailbox:   "INBOX",
				TLS:       true,
				TLSConfig: nil,
				Debug:     false,
			}, connConfig)
		})

		t.Run("password_file", func(t *testing.T) {
			cfg := getTestIMAPConfig()
			cfg.Password = ""
			cfg.PasswordFile = "testdata/testpass.txt"

			connConfig, _, err := cfg.Resolve()
			assert.NoError(t, err)
			assert.Equal(t, imap.ConnectionConfig{
				HostPort:  "imap.hostname.com:1234",
				Auth:      imap.NewNormalAuthenticator("username", "password"),
				Mailbox:   "INBOX",
				TLS:       true,
				TLSConfig: nil,
				Debug:     false,
			}, connConfig)
		})

		t.Run("systemd_credential", func(t *testing.T) {
			t.Setenv("CREDENTIALS_DIRECTORY", "testdata")

			cfg := getTestIMAPConfig()
			cfg.Password = ""
			cfg.SystemdCredential = "testpass.txt"

			connConfig, _, err := cfg.Resolve()
			assert.NoError(t, err)
			assert.Equal(t, imap.ConnectionConfig{
				HostPort:  "imap.hostname.com:1234",
				Auth:      imap.NewNormalAuthenticator("username", "password"),
				Mailbox:   "INBOX",
				TLS:       true,
				TLSConfig: nil,
				Debug:     false,
			}, connConfig)
		})

		t.Run("systemd_credential_invalid", func(t *testing.T) {
			cwd, err := os.Getwd()
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			t.Setenv("CREDENTIALS_DIRECTORY", path.Join(cwd, "testdata"))

			cfg := getTestIMAPConfig()
			cfg.Password = ""
			cfg.SystemdCredential = "../testpass.txt"

			_, _, err = cfg.Resolve()
			assert.Error(t, err)
		})
	})

	t.Run("tls", func(t *testing.T) {
		cfg := getTestIMAPConfig()
		cfg.TLSSkipVerify = true

		connConfig, _, err := cfg.Resolve()
		assert.NoError(t, err)
		assert.Equal(t, imap.ConnectionConfig{
			HostPort:  "imap.hostname.com:1234",
			Auth:      imap.NewNormalAuthenticator("username", "password"),
			Mailbox:   "INBOX",
			TLS:       true,
			TLSConfig: &tls.Config{InsecureSkipVerify: true},
			Debug:     false,
		}, connConfig)
	})

	t.Run("auth", func(t *testing.T) {
		t.Run("login", func(t *testing.T) {
			ctrl := gomock.NewController(t)

			mockAuth := mock_imap.NewMockAuthenticatable(ctrl)
			mockAuth.EXPECT().Login("username", "password")

			cfg := getTestIMAPConfig()
			cfg.AuthMethod = "LOGIN"

			connConfig, _, err := cfg.Resolve()
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			assert.NoError(t, connConfig.Auth.Authenticate(mockAuth))
		})

		t.Run("plain", func(t *testing.T) {
			ctrl := gomock.NewController(t)

			mockAuth := mock_imap.NewMockAuthenticatable(ctrl)
			mockAuth.EXPECT().Authenticate(gomock.Any()).DoAndReturn(func(c sasl.Client) error {
				mech, ir, err := c.Start()
				if err != nil {
					return err
				}

				assert.Equal(t, "PLAIN", mech)
				assert.Equal(t, []byte("\x00username\x00password"), ir)
				return nil
			})

			cfg := getTestIMAPConfig()
			cfg.AuthMethod = "PLAIN"

			connConfig, _, err := cfg.Resolve()
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			assert.NoError(t, connConfig.Auth.Authenticate(mockAuth))
		})

		// TODO: figure out how to test oauth
	})
}
