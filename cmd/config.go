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
	"github.com/vs49688/mailpump/imap/client"
	"github.com/vs49688/mailpump/imap/persistentclient"
	"github.com/vs49688/mailpump/pump"
	"io/ioutil"
	"net"
	"time"

	"github.com/urfave/cli/v2"
	"net/url"
	"strings"
)

type CliConfig struct {
	SourceURL            string        `json:"source_url"`
	SourceUsername       string        `json:"source_username"`
	SourcePassword       string        `json:"-"`
	SourcePasswordFile   string        `json:"source_password_file"`
	SourceTLSSkipVerify  bool          `json:"source_tls_skip_verify"`
	SourceTransport      string        `json:"source_transport"`
	SourceDebug          bool          `json:"source_debug"`
	DestURL              string        `json:"dest_url"`
	DestUsername         string        `json:"dest_username"`
	DestPassword         string        `json:"-"`
	DestPasswordFile     string        `json:"dest_password_file"`
	DestTLSSkipVerify    bool          `json:"dest_tls_skip_verify"`
	DestTransport        string        `json:"dest_transport"`
	DestDebug            bool          `json:"dest_debug"`
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
		SourceTLSSkipVerify:  false,
		SourceTransport:      "persistent",
		SourceDebug:          false,
		DestTLSSkipVerify:    false,
		DestTransport:        "persistent",
		DestDebug:            false,
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
			Destination: &cfg.SourceURL,
			Required:    true,
			Value:       def.SourceURL,
		},
		&cli.StringFlag{
			Name:        "source-username",
			Usage:       "destination imap username",
			EnvVars:     []string{"MAILPUMP_SOURCE_USERNAME"},
			Destination: &cfg.SourceUsername,
			Required:    true,
			Value:       def.SourceUsername,
		},
		&cli.StringFlag{
			Name:        "source-password",
			Usage:       "source imap password",
			EnvVars:     []string{"MAILPUMP_SOURCE_PASSWORD"},
			Destination: &cfg.SourcePassword,
			Required:    false,
			Value:       def.SourcePassword,
		},
		&cli.StringFlag{
			Name:        "source-password-file",
			Usage:       "source imap password file",
			EnvVars:     []string{"MAILPUMP_SOURCE_PASSWORD_FILE"},
			Destination: &cfg.SourcePasswordFile,
			Required:    false,
			Value:       def.SourcePasswordFile,
		},
		&cli.BoolFlag{
			Name:        "source-tls-skip-verify",
			Usage:       "skip source tls verification",
			EnvVars:     []string{"MAILPUMP_SOURCE_TLS_SKIP_VERIFY"},
			Destination: &cfg.SourceTLSSkipVerify,
			Value:       def.SourceTLSSkipVerify,
		},
		&cli.StringFlag{
			Name:        "source-transport",
			Usage:       "source imap transport (persistent, standard)",
			EnvVars:     []string{"MAILPUMP_SOURCE_TRANSPORT"},
			Destination: &cfg.SourceTransport,
			Value:       def.SourceTransport,
		},
		&cli.BoolFlag{
			Name:        "source-debug",
			Usage:       "display source debug info",
			EnvVars:     []string{"MAILPUMP_SOURCE_DEBUG"},
			Destination: &cfg.SourceDebug,
			Value:       def.SourceDebug,
		},
		&cli.StringFlag{
			Name:        "dest-url",
			Usage:       "destination imap url",
			EnvVars:     []string{"MAILPUMP_DEST_URL"},
			Destination: &cfg.DestURL,
			Required:    true,
			Value:       def.DestURL,
		},
		&cli.StringFlag{
			Name:        "dest-username",
			Usage:       "destination imap username",
			EnvVars:     []string{"MAILPUMP_DEST_USERNAME"},
			Destination: &cfg.DestUsername,
			Required:    true,
			Value:       def.DestUsername,
		},
		&cli.StringFlag{
			Name:        "dest-password",
			Usage:       "destination imap password",
			EnvVars:     []string{"MAILPUMP_DEST_PASSWORD"},
			Destination: &cfg.DestPassword,
			Required:    false,
			Value:       def.DestPassword,
		},
		&cli.StringFlag{
			Name:        "dest-password-file",
			Usage:       "destination imap password file",
			EnvVars:     []string{"MAILPUMP_DEST_PASSWORD_FILE"},
			Destination: &cfg.DestPasswordFile,
			Required:    false,
			Value:       def.DestPasswordFile,
		},
		&cli.BoolFlag{
			Name:        "dest-tls-skip-verify",
			Usage:       "skip destination tls Verification",
			EnvVars:     []string{"MAILPUMP_DEST_TLS_SKIP_VERIFY"},
			Destination: &cfg.DestTLSSkipVerify,
			Value:       def.DestTLSSkipVerify,
		},
		&cli.StringFlag{
			Name:        "dest-transport",
			Usage:       "destination imap transport (persistent, standard)",
			EnvVars:     []string{"MAILPUMP_DEST_TRANSPORT"},
			Destination: &cfg.DestTransport,
			Value:       def.DestTransport,
		},
		&cli.BoolFlag{
			Name:        "dest-debug",
			Usage:       "display destination debug info",
			EnvVars:     []string{"MAILPUMP_DEST_DEBUG"},
			Destination: &cfg.DestDebug,
			Value:       def.DestDebug,
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

func (cfg *CliConfig) BuildPumpConfig(pumpConfig *pump.Config) error {
	def := DefaultConfig()

	sourceURL, err := url.Parse(cfg.SourceURL)
	if err != nil {
		return err
	}

	sourceHostPort, sourceMailbox, sourceTLS, err := extractUrl(sourceURL)
	if err != nil {
		return err
	}

	pumpConfig.SourceHostPort = sourceHostPort
	pumpConfig.SourceUsername = cfg.SourceUsername

	if cfg.SourcePassword != "" {
		pumpConfig.SourcePassword = cfg.SourcePassword
	} else if cfg.SourcePasswordFile != "" {
		pass, err := ioutil.ReadFile(cfg.SourcePasswordFile)
		if err != nil {
			return err
		}

		pumpConfig.SourcePassword = strings.TrimSpace(string(pass))
	} else {
		return errors.New("at least one of the \"source-password\" or \"source-password-file\" flags is required")
	}

	pumpConfig.SourceMailbox = sourceMailbox
	pumpConfig.SourceTLS = sourceTLS
	pumpConfig.SourceTLSConfig = nil
	if cfg.SourceTLSSkipVerify {
		// #nosec G402
		pumpConfig.SourceTLSConfig = &tls.Config{InsecureSkipVerify: true}
	}

	if cfg.SourceTransport != "persistent" {
		pumpConfig.SourceFactory = &client.Factory{}
	} else {
		pumpConfig.SourceFactory = &persistentclient.Factory{
			Mailbox:  sourceMailbox,
			MaxDelay: 0,
		}
	}

	pumpConfig.SourceDebug = cfg.SourceDebug

	destURL, err := url.Parse(cfg.DestURL)
	if err != nil {
		return err
	}

	destHostPort, destMailbox, destTLS, err := extractUrl(destURL)
	if err != nil {
		return err
	}

	pumpConfig.DestHostPort = destHostPort
	pumpConfig.DestUsername = cfg.DestUsername

	if cfg.DestPassword != "" {
		pumpConfig.DestPassword = cfg.DestPassword
	} else if cfg.DestPasswordFile != "" {
		pass, err := ioutil.ReadFile(cfg.DestPasswordFile)
		if err != nil {
			return err
		}

		pumpConfig.DestPassword = strings.TrimSpace(string(pass))
	} else {
		return errors.New("at least one of the \"dest-password\" or \"dest-password-file\" flags is required")
	}

	pumpConfig.DestMailbox = destMailbox
	pumpConfig.DestTLS = destTLS
	pumpConfig.DestTLSConfig = nil
	if cfg.DestTLSSkipVerify {
		// #nosec G402
		pumpConfig.DestTLSConfig = &tls.Config{InsecureSkipVerify: true}
	}

	if cfg.DestTransport != "persistent" {
		pumpConfig.DestFactory = &client.Factory{}
	} else {
		pumpConfig.DestFactory = &persistentclient.Factory{
			Mailbox:  destMailbox,
			MaxDelay: 0,
		}
	}

	pumpConfig.DestDebug = cfg.DestDebug

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
