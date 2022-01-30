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

package cmd

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"time"

	"github.com/vs49688/mailpump/imap/client"
	"github.com/vs49688/mailpump/imap/persistentclient"
	"github.com/vs49688/mailpump/pump"

	"net/url"
	"strings"

	"github.com/urfave/cli/v2"
)

type CliIMAPConfig struct {
	URL           string `json:"url"`
	Username      string `json:"username"`
	Password      string `json:"-"`
	PasswordFile  string `json:"password_file"`
	TLSSkipVerify bool   `json:"tls_skip_verify"`
	Transport     string `json:"transport"`
	Debug         bool   `json:"debug"`
}

type CliConfig struct {
	Source               CliIMAPConfig `json:"source"`
	Dest                 CliIMAPConfig `json:"dest"`
	LogLevel             string        `json:"log_level"`
	LogFormat            string        `json:"log_format"`
	IDLEFallbackInterval time.Duration `json:"idle_fallback_interval"`
	BatchSize            uint          `json:"batch_size"`
	DisableDeletions     bool          `json:"disable_deletions"`
	FetchBufferSize      uint          `json:"fetch_buffer_size"`
	FetchMaxInterval     time.Duration `json:"fetch_max_interval"`
}

func DefaultConfig() CliConfig {
	return CliConfig{
		Source: CliIMAPConfig{
			TLSSkipVerify: false,
			Transport:     "persistent",
			Debug:         false,
		},
		Dest: CliIMAPConfig{
			TLSSkipVerify: false,
			Transport:     "persistent",
			Debug:         false,
		},
		LogLevel:             "info",
		LogFormat:            "text",
		IDLEFallbackInterval: time.Minute,
		BatchSize:            15,
		DisableDeletions:     false,
		FetchBufferSize:      20,
		FetchMaxInterval:     5 * time.Minute,
	}
}

func (cfg *CliConfig) Parameters() []cli.Flag {
	def := DefaultConfig()

	return []cli.Flag{
		&cli.StringFlag{
			Name:        "source-url",
			Usage:       "source imap url",
			EnvVars:     []string{"MAILPUMP_SOURCE_URL"},
			Destination: &cfg.Source.URL,
			Required:    true,
			Value:       def.Source.URL,
		},
		&cli.StringFlag{
			Name:        "source-username",
			Usage:       "destination imap username",
			EnvVars:     []string{"MAILPUMP_SOURCE_USERNAME"},
			Destination: &cfg.Source.Username,
			Required:    true,
			Value:       def.Source.Username,
		},
		&cli.StringFlag{
			Name:        "source-password",
			Usage:       "source imap password",
			EnvVars:     []string{"MAILPUMP_SOURCE_PASSWORD"},
			Destination: &cfg.Source.Password,
			Required:    false,
			Value:       def.Source.Password,
		},
		&cli.StringFlag{
			Name:        "source-password-file",
			Usage:       "source imap password file",
			EnvVars:     []string{"MAILPUMP_SOURCE_PASSWORD_FILE"},
			Destination: &cfg.Source.PasswordFile,
			Required:    false,
			Value:       def.Source.PasswordFile,
		},
		&cli.BoolFlag{
			Name:        "source-tls-skip-verify",
			Usage:       "skip source tls verification",
			EnvVars:     []string{"MAILPUMP_SOURCE_TLS_SKIP_VERIFY"},
			Destination: &cfg.Source.TLSSkipVerify,
			Value:       def.Source.TLSSkipVerify,
		},
		&cli.StringFlag{
			Name:        "source-transport",
			Usage:       "source imap transport (persistent, standard)",
			EnvVars:     []string{"MAILPUMP_SOURCE_TRANSPORT"},
			Destination: &cfg.Source.Transport,
			Value:       def.Source.Transport,
		},
		&cli.BoolFlag{
			Name:        "source-debug",
			Usage:       "display source debug info",
			EnvVars:     []string{"MAILPUMP_SOURCE_DEBUG"},
			Destination: &cfg.Source.Debug,
			Value:       def.Source.Debug,
		},
		&cli.StringFlag{
			Name:        "dest-url",
			Usage:       "destination imap url",
			EnvVars:     []string{"MAILPUMP_DEST_URL"},
			Destination: &cfg.Dest.URL,
			Required:    true,
			Value:       def.Dest.URL,
		},
		&cli.StringFlag{
			Name:        "dest-username",
			Usage:       "destination imap username",
			EnvVars:     []string{"MAILPUMP_DEST_USERNAME"},
			Destination: &cfg.Dest.Username,
			Required:    true,
			Value:       def.Dest.Username,
		},
		&cli.StringFlag{
			Name:        "dest-password",
			Usage:       "destination imap password",
			EnvVars:     []string{"MAILPUMP_DEST_PASSWORD"},
			Destination: &cfg.Dest.Password,
			Required:    false,
			Value:       def.Dest.Password,
		},
		&cli.StringFlag{
			Name:        "dest-password-file",
			Usage:       "destination imap password file",
			EnvVars:     []string{"MAILPUMP_DEST_PASSWORD_FILE"},
			Destination: &cfg.Dest.PasswordFile,
			Required:    false,
			Value:       def.Dest.PasswordFile,
		},
		&cli.BoolFlag{
			Name:        "dest-tls-skip-verify",
			Usage:       "skip destination tls Verification",
			EnvVars:     []string{"MAILPUMP_DEST_TLS_SKIP_VERIFY"},
			Destination: &cfg.Dest.TLSSkipVerify,
			Value:       def.Dest.TLSSkipVerify,
		},
		&cli.StringFlag{
			Name:        "dest-transport",
			Usage:       "destination imap transport (persistent, standard)",
			EnvVars:     []string{"MAILPUMP_DEST_TRANSPORT"},
			Destination: &cfg.Dest.Transport,
			Value:       def.Dest.Transport,
		},
		&cli.BoolFlag{
			Name:        "dest-debug",
			Usage:       "display destination debug info",
			EnvVars:     []string{"MAILPUMP_DEST_DEBUG"},
			Destination: &cfg.Dest.Debug,
			Value:       def.Dest.Debug,
		},
		&cli.StringFlag{
			Name:        "log-level",
			Usage:       "logging level",
			EnvVars:     []string{"MAILPUMP_LOG_LEVEL"},
			Destination: &cfg.LogLevel,
			Value:       def.LogLevel,
		},
		&cli.StringFlag{
			Name:        "log-format",
			Usage:       "logging format (text/json)",
			EnvVars:     []string{"MAILPUMP_LOG_FORMAT"},
			Destination: &cfg.LogFormat,
			Value:       def.LogFormat,
		},
		&cli.DurationFlag{
			Name:        "idle-fallback-interval",
			Usage:       "fallback poll interval for servers that don't support IDLE",
			EnvVars:     []string{"MAILPUMP_IDLE_FALLBACK_INTERVAL"},
			Destination: &cfg.IDLEFallbackInterval,
			Value:       def.IDLEFallbackInterval,
		},
		&cli.UintFlag{
			Name:        "batch-size",
			Usage:       "deletion batch size",
			EnvVars:     []string{"MAILPUMP_BATCH_SIZE"},
			Destination: &cfg.BatchSize,
			Value:       def.BatchSize,
		},
		&cli.BoolFlag{
			Name:        "disable-deletions",
			Usage:       "disable deletions. for debugging only",
			EnvVars:     []string{"MAILPUMP_DISABLE_DELETIONS"},
			Destination: &cfg.DisableDeletions,
			Value:       def.DisableDeletions,
			Hidden:      true,
		},
		&cli.UintFlag{
			Name:        "fetch-buffer-size",
			Usage:       "fetch buffer size",
			EnvVars:     []string{"MAILPUMP_FETCH_BUFFER_SIZE"},
			Destination: &cfg.FetchBufferSize,
			Value:       def.FetchBufferSize,
		},
		&cli.DurationFlag{
			Name:        "fetch-max-interval",
			Usage:       "maximum interval between fetches. can abort IDLE",
			EnvVars:     []string{"MAILPUMP_FETCH_MAX_INTERVAL"},
			Destination: &cfg.FetchMaxInterval,
			Value:       def.FetchMaxInterval,
		},
	}
}

var (
	errInvalidScheme = errors.New("invalid uri scheme")
)

func extractUrl(u *url.URL) (string, string, bool, error) {
	var defaultPort string
	var useTLS bool
	switch strings.ToLower(u.Scheme) {
	case "imap":
		defaultPort = "143"
		useTLS = false
	case "imaps":
		defaultPort = "993"
		useTLS = true
	default:
		return "", "", false, errInvalidScheme
	}

	host := u.Hostname()
	port := u.Port()

	if port == "" {
		port = defaultPort
	}

	return net.JoinHostPort(host, port), strings.TrimPrefix(u.Path, "/"), useTLS, nil
}

func (cfg *CliIMAPConfig) buildTransportConfig(transConfig *pump.TransportConfig, prefix string) error {
	sourceURL, err := url.Parse(cfg.URL)
	if err != nil {
		return err
	}

	hostPort, mailbox, wantTLS, err := extractUrl(sourceURL)
	if err != nil {
		return err
	}

	transConfig.HostPort = hostPort
	transConfig.Username = cfg.Username

	if cfg.Password != "" {
		transConfig.Password = cfg.Password
	} else if cfg.PasswordFile != "" {
		pass, err := ioutil.ReadFile(cfg.PasswordFile)
		if err != nil {
			return err
		}

		transConfig.Password = strings.TrimSpace(string(pass))
	} else {
		return fmt.Errorf("at least one of the \"%v-password\" or \"%v-password-file\" flags is required", prefix, prefix)
	}

	transConfig.Mailbox = mailbox
	transConfig.TLS = wantTLS
	transConfig.TLSConfig = nil
	if cfg.TLSSkipVerify {
		// #nosec G402
		transConfig.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	}

	if cfg.Transport != "persistent" {
		transConfig.Factory = &client.Factory{}
	} else {
		transConfig.Factory = &persistentclient.Factory{
			Mailbox:  mailbox,
			MaxDelay: 0,
		}
	}

	transConfig.Debug = cfg.Debug
	return nil
}

func (cfg *CliConfig) BuildPumpConfig(pumpConfig *pump.Config) error {
	def := DefaultConfig()

	if err := cfg.Source.buildTransportConfig(&pumpConfig.Source, "source"); err != nil {
		return err
	}

	if err := cfg.Dest.buildTransportConfig(&pumpConfig.Dest, "dest"); err != nil {
		return err
	}

	pumpConfig.IDLEFallbackInterval = cfg.IDLEFallbackInterval
	if pumpConfig.IDLEFallbackInterval == 0 {
		pumpConfig.IDLEFallbackInterval = def.IDLEFallbackInterval
	}

	pumpConfig.BatchSize = cfg.BatchSize
	if pumpConfig.BatchSize == 0 {
		pumpConfig.BatchSize = def.BatchSize
	}

	pumpConfig.DisableDeletions = cfg.DisableDeletions

	pumpConfig.FetchBufferSize = cfg.FetchBufferSize
	if pumpConfig.FetchBufferSize == 0 {
		pumpConfig.FetchBufferSize = def.FetchBufferSize
	}

	pumpConfig.FetchMaxInterval = cfg.FetchMaxInterval
	if pumpConfig.FetchMaxInterval == 0 {
		pumpConfig.FetchMaxInterval = def.FetchMaxInterval
	}

	return nil
}
