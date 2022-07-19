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
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/urfave/cli/v2"
	"github.com/vs49688/mailpump/imap"
	"github.com/vs49688/mailpump/pump"
)

func makeFlagNames(name string, prefix string) (string, string, []string) {
	name = strings.ToLower(name)
	prefix = strings.ToLower(prefix)

	desc := strings.ReplaceAll(name, "-", " ")
	env := strings.ToUpper(strings.ReplaceAll(name, "-", "_"))

	if prefix == "" {
		return name, desc, []string{"MAILPUMP_" + env}
	}

	desc = strings.ReplaceAll(prefix, "-", " ") + " " + desc
	env = "MAILPUMP_" + strings.ToUpper(strings.ReplaceAll(prefix, "-", "_")) + "_" + env

	return prefix + "-" + name, desc, []string{env}
}

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
	var name string
	var usage string
	var envs []string
	var flags []cli.Flag

	flags = append(flags, cfg.Source.makeIMAPParameters("source")...)
	flags = append(flags, cfg.Dest.makeIMAPParameters("dest")...)

	name, usage, envs = makeFlagNames("log-level", "")
	flags = append(flags, &cli.StringFlag{
		Name:        name,
		Usage:       usage,
		EnvVars:     envs,
		Destination: &cfg.LogLevel,
		Value:       def.LogLevel,
	})

	name, _, envs = makeFlagNames("log-format", "")
	flags = append(flags, &cli.StringFlag{
		Name:        name,
		Usage:       "log format (text/json)",
		EnvVars:     envs,
		Destination: &cfg.LogFormat,
		Value:       def.LogFormat,
	})

	name, _, envs = makeFlagNames("idle-fallback-interval", "")
	flags = append(flags, &cli.DurationFlag{
		Name:        name,
		Usage:       "fallback poll interval for servers that don't support IDLE",
		EnvVars:     envs,
		Destination: &cfg.IDLEFallbackInterval,
		Value:       def.IDLEFallbackInterval,
	})

	name, _, envs = makeFlagNames("batch-size", "")
	flags = append(flags, &cli.UintFlag{
		Name:        name,
		Usage:       "deletion batch size",
		EnvVars:     envs,
		Destination: &cfg.BatchSize,
		Value:       def.BatchSize,
	})

	name, _, envs = makeFlagNames("disable-deletions", "")
	flags = append(flags, &cli.BoolFlag{
		Name:        name,
		Usage:       "disable deletions. for debugging only",
		EnvVars:     envs,
		Destination: &cfg.DisableDeletions,
		Value:       def.DisableDeletions,
		Hidden:      true,
	})

	name, usage, envs = makeFlagNames("fetch-buffer-size", "")
	flags = append(flags, &cli.UintFlag{
		Name:        name,
		Usage:       usage,
		EnvVars:     envs,
		Destination: &cfg.FetchBufferSize,
		Value:       def.FetchBufferSize,
	})

	name, _, envs = makeFlagNames("fetch-max-interval", "")
	flags = append(flags, &cli.DurationFlag{
		Name:        name,
		Usage:       "maximum interval between fetches. can abort IDLE",
		EnvVars:     envs,
		Destination: &cfg.FetchMaxInterval,
		Value:       def.FetchMaxInterval,
	})

	return flags
}

func prettifyError(err error, prefix string, authMethod string) error {
	if errors.Is(err, ErrIMAPMissingUsername) {
		return fmt.Errorf("\"%v-username\" is required when using %v auth", prefix, authMethod)
	} else if errors.Is(err, ErrIMAPMissingPassword) {
		return fmt.Errorf("at least one of the \"%v-password\" or \"%v-password-file\" flags is required", prefix, prefix)
	}
	return err
}

func (cfg *CliConfig) BuildPumpConfig(pumpConfig *pump.Config) error {
	var connConfig imap.ConnectionConfig
	var factory imap.Factory
	var err error

	def := DefaultConfig()

	if connConfig, factory, err = cfg.Source.Resolve(); err != nil {
		return prettifyError(err, "source", cfg.Source.AuthMethod)
	}
	pumpConfig.Source = connConfig
	pumpConfig.SourceFactory = factory

	if connConfig, factory, err = cfg.Dest.Resolve(); err != nil {
		return prettifyError(err, "dest", cfg.Dest.AuthMethod)
	}
	pumpConfig.Dest = connConfig
	pumpConfig.DestFactory = factory

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
