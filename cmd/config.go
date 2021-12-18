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
	"github.com/vs49688/mailpump/pump"
	"net"
	"time"

	"github.com/urfave/cli/v2"
	"net/url"
	"strings"
)

type CliConfig struct {
	SourceURL           string        `json:"source_url"`
	SourceUsername      string        `json:"source_username"`
	SourcePassword      string        `json:"-"`
	SourceTLSSkipVerify bool          `json:"source_tls_skip_verify"`
	SourceDebug         bool          `json:"source_debug"`
	DestURL             string        `json:"dest_url"`
	DestUsername        string        `json:"dest_username"`
	DestPassword        string        `json:"-"`
	DestTLSSkipVerify   bool          `json:"dest_tls_skip_verify"`
	DestDebug           bool          `json:"dest_debug"`
	LogLevel            string        `json:"log_level"`
	LogFormat           string        `json:"log_format"`
	TickInterval        time.Duration `json:"tick_interval"`
	BatchSize           uint          `json:"batch_size"`
}

func DefaultConfig() CliConfig {
	return CliConfig{
		SourceTLSSkipVerify: false,
		SourceDebug:         false,
		DestTLSSkipVerify:   false,
		DestDebug:           false,
		LogLevel:            "info",
		LogFormat:           "text",
		TickInterval:        time.Minute,
		BatchSize:           15,
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
			Required:    true,
			Value:       def.SourcePassword,
		},
		&cli.BoolFlag{
			Name:        "source-tls-skip-verify",
			Usage:       "skip source tls verification",
			EnvVars:     []string{"MAILPUMP_SOURCE_TLS_SKIP_VERIFY"},
			Destination: &cfg.SourceTLSSkipVerify,
			Value:       def.SourceTLSSkipVerify,
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
			Required:    true,
			Value:       def.DestPassword,
		},
		&cli.BoolFlag{
			Name:        "dest-tls-skip-verify",
			Usage:       "skip destination tls Verification",
			EnvVars:     []string{"MAILPUMP_DEST_TLS_SKIP_VERIFY"},
			Destination: &cfg.DestTLSSkipVerify,
			Value:       def.DestTLSSkipVerify,
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
			Name:        "tick-interval",
			Usage:       "tick interval",
			EnvVars:     []string{"MAILPUMP_TICK_INTERVAL"},
			Destination: &cfg.TickInterval,
			Value:       def.TickInterval,
		},
		&cli.UintFlag{
			Name:        "batch-size",
			Usage:       "deletion batch size",
			EnvVars:     []string{"MAILPUMP_BATCH_SIZE"},
			Destination: &cfg.BatchSize,
			Value:       def.BatchSize,
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
	pumpConfig.SourcePassword = cfg.SourcePassword
	pumpConfig.SourceMailbox = sourceMailbox
	pumpConfig.SourceTLS = sourceTLS
	pumpConfig.SourceTLSConfig = nil
	if cfg.SourceTLSSkipVerify {
		pumpConfig.SourceTLSConfig = &tls.Config{InsecureSkipVerify: true}
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
	pumpConfig.DestPassword = cfg.DestPassword
	pumpConfig.DestMailbox = destMailbox
	pumpConfig.DestTLS = destTLS
	pumpConfig.DestTLSConfig = nil
	if cfg.SourceTLSSkipVerify {
		pumpConfig.DestTLSConfig = &tls.Config{InsecureSkipVerify: true}
	}
	pumpConfig.DestDebug = cfg.DestDebug

	if cfg.TickInterval == 0 {
		pumpConfig.TickInterval = def.TickInterval
	}

	if cfg.BatchSize == 0 {
		pumpConfig.BatchSize = def.BatchSize
	}
	return nil
}
