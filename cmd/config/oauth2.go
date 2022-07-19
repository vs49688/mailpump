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

	"github.com/urfave/cli/v2"
)

func DefaultOAuth2Config() OAuth2Config {
	return OAuth2Config{Provider: "custom"}
}

func (cfg *OAuth2Config) fillDefaults() {
	def := DefaultOAuth2Config()

	if cfg.Provider == "" {
		cfg.Provider = def.Provider
	}
}

func (cfg *OAuth2Config) makeParameters(prefix string) []cli.Flag {
	def := DefaultOAuth2Config()
	var name string
	var usage string
	var envs []string
	var flags []cli.Flag

	name, usage, envs = makeFlagNames("provider", prefix)
	flags = append(flags, &cli.StringFlag{
		Name:        name,
		Usage:       usage + " (custom)",
		EnvVars:     envs,
		Destination: &cfg.Provider,
		Value:       def.Provider,
	})

	name, usage, envs = makeFlagNames("client-id", prefix)
	flags = append(flags, &cli.StringFlag{
		Name:        name,
		Usage:       usage,
		EnvVars:     envs,
		Destination: &cfg.Config.ClientID,
		Value:       def.Config.ClientID,
	})

	name, usage, envs = makeFlagNames("client-secret", prefix)
	flags = append(flags, &cli.StringFlag{
		Name:        name,
		Usage:       usage,
		EnvVars:     envs,
		Destination: &cfg.Config.ClientSecret,
		Value:       def.Config.ClientSecret,
	})

	name, usage, envs = makeFlagNames("token-url", prefix)
	flags = append(flags, &cli.StringFlag{
		Name:        name,
		Usage:       usage,
		EnvVars:     envs,
		Destination: &cfg.Config.Endpoint.TokenURL,
		Value:       def.Config.Endpoint.TokenURL,
	})

	name, usage, envs = makeFlagNames("scopes", prefix)
	flags = append(flags, &cli.StringSliceFlag{
		Name:        name,
		Usage:       usage,
		EnvVars:     envs,
		Destination: &cfg.Scopes,
	})
	return flags
}

func (cfg *OAuth2Config) Parameters() []cli.Flag {
	return cfg.makeParameters("")
}

func (cfg *OAuth2Config) Resolve() error {
	switch cfg.Provider {
	case "custom":
		cfg.Config.Scopes = append(cfg.Config.Scopes, cfg.Scopes.Value()...)
	default:
		return fmt.Errorf("unknown oauth2 provider: %v", cfg.Provider)
	}

	if cfg.Config.ClientID == "" {
		return errors.New("oauth2 client id not set")
	}

	if cfg.Config.ClientSecret == "" {
		return errors.New("oauth2 client secret not set")
	}

	if cfg.Config.Endpoint.AuthURL == "" {
		return errors.New("oauth2 auth url not set")
	}

	if cfg.Config.Endpoint.TokenURL == "" {
		return errors.New("oauth2 token url not set")
	}

	if len(cfg.Config.Scopes) == 0 {
		return errors.New("oauth2 scopes not set")
	}

	return nil
}
