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

package run_multi

import (
	"encoding/json"
	"io/ioutil"
	"net/url"
	"os"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/urfave/cli/v2"
	"github.com/vs49688/mailpump/cmd/config"
	"github.com/vs49688/mailpump/imap"
	"github.com/vs49688/mailpump/ingest"
	"github.com/vs49688/mailpump/receiver"
)

const (
	DefaultLogLevel  = "info"
	DefaultLogFormat = "text"

	DefaultIDLEFallbackInterval = time.Minute
	DefaultBatchSize            = 10
	DefaultFetchBufferSize      = 20
	DefaultFetchMaxInterval     = 5 * time.Minute
)

type Source struct {
	Connection           config.IMAPConfig `json:"connection"`
	TargetMailbox        string            `json:"target_mailbox"`
	IDLEFallbackInterval time.Duration     `json:"idle_fallback_interval"`
	BatchSize            uint              `json:"batch_size"`
	DisableDeletions     bool              `json:"disable_deletions"`
	FetchBufferSize      uint              `json:"fetch_buffer_size"`
	FetchMaxInterval     time.Duration     `json:"fetch_max_interval"`
}

func makeSourceName(username string, cfg *imap.ConnectionConfig) string {
	u := url.URL{
		User: url.User(username),
		Host: cfg.HostPort,
		Path: cfg.Mailbox,
	}

	if cfg.TLS {
		u.Scheme = "imaps"
	} else {
		u.Scheme = "imap"
	}

	return u.String()
}

func (src *Source) Resolve(logger *log.Entry) (receiver.Config, error) {
	connConfig, factory, err := src.Connection.Resolve()
	if err != nil {
		return receiver.Config{}, err
	}

	cfg := receiver.Config{
		ConnectionConfig:     connConfig,
		Factory:              factory,
		Logger:               logger,
		IDLEFallbackInterval: src.IDLEFallbackInterval,
		BatchSize:            src.BatchSize,
		FetchBufferSize:      src.FetchBufferSize,
		FetchMaxInterval:     src.FetchMaxInterval,
		Channel:              nil, // Not our problem yet
		DisableDeletions:     src.DisableDeletions,
	}

	if cfg.IDLEFallbackInterval == 0 {
		cfg.IDLEFallbackInterval = DefaultIDLEFallbackInterval
	}

	if cfg.BatchSize == 0 {
		cfg.BatchSize = DefaultBatchSize
	}

	if cfg.FetchBufferSize == 0 {
		cfg.FetchBufferSize = DefaultFetchBufferSize
	}

	if cfg.FetchMaxInterval == 0 {
		cfg.FetchMaxInterval = DefaultFetchMaxInterval
	}

	return cfg, nil
}

type Configuration struct {
	ConfigPath string `json:"-"`

	Destination config.IMAPConfig  `json:"destination,omitempty"`
	Sources     map[string]*Source `json:"sources,omitempty"`
	LogLevel    string             `json:"log_level,omitempty"`
	LogFormat   string             `json:"log_format,omitempty"`

	ResolvedDestination ingest.Config     `json:"-"`
	ResolvedSources     []receiver.Config `json:"-"`
	Logger              *log.Logger       `json:"-"`
}

func DefaultConfig() Configuration {
	return Configuration{
		Destination: config.DefaultIMAPConfig(),
		ConfigPath:  "config.json",
		LogLevel:    DefaultLogLevel,
		LogFormat:   DefaultLogFormat,
		Logger:      log.StandardLogger(),
	}
}

func (cfg *Configuration) Parameters() []cli.Flag {
	def := DefaultConfig()
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "config",
			Aliases:     []string{"c"},
			Usage:       "path to configuration file, or '-' to read from stdin",
			Value:       def.ConfigPath,
			Destination: &cfg.ConfigPath,
		},
	}
}

func (cfg *Configuration) Resolve() error {
	var err error
	var raw []byte

	if cfg.ConfigPath == "" {
		raw, err = ioutil.ReadAll(os.Stdin)
	} else {
		raw, err = ioutil.ReadFile(cfg.ConfigPath)
	}

	if err != nil {
		return err
	}

	if err := json.Unmarshal(raw, cfg); err != nil {
		return err
	}

	destConfig, factory, err := cfg.Destination.Resolve()
	if err != nil {
		return err
	}
	cfg.ResolvedDestination = ingest.Config{
		ConnectionConfig: destConfig,
		Factory:          factory,
	}

	cfg.ResolvedSources = make([]receiver.Config, 0, len(cfg.Sources))
	for name, src := range cfg.Sources {
		rs, err := src.Resolve(cfg.Logger.WithField("source", name))
		if err != nil {
			return err
		}

		cfg.ResolvedSources = append(cfg.ResolvedSources, rs)
	}

	return nil
}
