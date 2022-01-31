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

package config

import (
	"time"

	"github.com/urfave/cli/v2"
	"github.com/vs49688/mailpump/pump"
)

func DefaultConfig() CliConfig {
	return CliConfig{
		Source:               DefaultIMAPConfig(),
		Dest:                 DefaultIMAPConfig(),
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

	var flags []cli.Flag
	flags = append(flags, cfg.Source.makeIMAPParameters("source")...)
	flags = append(flags, cfg.Dest.makeIMAPParameters("dest")...)
	flags = append(flags, []cli.Flag{
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
	}...)

	return flags
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
