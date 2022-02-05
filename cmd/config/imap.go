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
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/url"
	"strings"

	"github.com/emersion/go-sasl"
	"github.com/urfave/cli/v2"

	"github.com/vs49688/mailpump/imap"
	"github.com/vs49688/mailpump/imap/client"
	"github.com/vs49688/mailpump/imap/persistentclient"
	"github.com/vs49688/mailpump/pump"
	"golang.org/x/oauth2"
)

func DefaultIMAPConfig() IMAPConfig {
	return IMAPConfig{
		AuthMethod:    "normal",
		TLSSkipVerify: false,
		Transport:     "persistent",
		Debug:         false,
		OAuth2Prov:    "custom",
	}
}

func (cfg *IMAPConfig) makeIMAPParameters(prefix string) []cli.Flag {
	def := DefaultIMAPConfig()
	var name string
	var usage string
	var envs []string
	var flags []cli.Flag

	name, usage, envs = makeFlagNames("url", prefix)
	flags = append(flags, &cli.StringFlag{
		Name:        name,
		Usage:       usage,
		EnvVars:     envs,
		Destination: &cfg.URL,
		Required:    true,
		Value:       def.URL,
	})

	name, usage, envs = makeFlagNames("auth-method", prefix)
	flags = append(flags, &cli.StringFlag{
		Name:        name,
		Usage:       usage,
		EnvVars:     envs,
		Destination: &cfg.AuthMethod,
		Required:    false,
		Value:       def.AuthMethod,
	})

	name, _, envs = makeFlagNames("username", prefix)
	flags = append(flags, &cli.StringFlag{
		Name:        name,
		Usage:       fmt.Sprintf("%v imap username", prefix),
		EnvVars:     envs,
		Destination: &cfg.Username,
		Required:    true,
		Value:       def.Username,
	})

	name, _, envs = makeFlagNames("password", prefix)
	flags = append(flags, &cli.StringFlag{
		Name:        name,
		Usage:       fmt.Sprintf("%v imap password", prefix),
		EnvVars:     envs,
		Destination: &cfg.Password,
		Required:    false,
		Value:       def.Password,
	})

	name, usage, envs = makeFlagNames("password-file", prefix)
	flags = append(flags, &cli.StringFlag{
		Name:        name,
		Usage:       fmt.Sprintf("%v imap password file", prefix),
		EnvVars:     envs,
		Destination: &cfg.PasswordFile,
		Required:    false,
		Value:       def.PasswordFile,
	})

	name, _, envs = makeFlagNames("tls-skip-verify", prefix)
	flags = append(flags, &cli.BoolFlag{
		Name:        name,
		Usage:       fmt.Sprintf("skip %v tls verification", prefix),
		EnvVars:     envs,
		Destination: &cfg.TLSSkipVerify,
		Value:       def.TLSSkipVerify,
	})

	name, _, envs = makeFlagNames("transport", prefix)
	flags = append(flags, &cli.StringFlag{
		Name:        name,
		Usage:       fmt.Sprintf("%v imap transport (persistent, standard)", prefix),
		EnvVars:     envs,
		Destination: &cfg.Transport,
		Value:       def.Transport,
	})

	name, _, envs = makeFlagNames("debug", prefix)
	flags = append(flags, &cli.StringFlag{
		Name:        name,
		Usage:       fmt.Sprintf("display %v debug info", prefix),
		EnvVars:     envs,
		Destination: &cfg.Transport,
		Value:       def.Transport,
	})

	// OAuth2 flags
	name, usage, envs = makeFlagNames("oauth2-provider", prefix)
	flags = append(flags, &cli.StringFlag{
		Name:        name,
		Usage:       usage,
		EnvVars:     envs,
		Destination: &cfg.OAuth2Prov,
		Value:       def.OAuth2Prov,
	})

	name, usage, envs = makeFlagNames("oauth2-client-id", prefix)
	flags = append(flags, &cli.StringFlag{
		Name:        name,
		Usage:       usage,
		EnvVars:     envs,
		Destination: &cfg.OAuth2Config.ClientID,
		Value:       def.OAuth2Config.ClientID,
	})

	name, usage, envs = makeFlagNames("oauth2-client-secret", prefix)
	flags = append(flags, &cli.StringFlag{
		Name:        name,
		Usage:       usage,
		EnvVars:     envs,
		Destination: &cfg.OAuth2Config.ClientSecret,
		Value:       def.OAuth2Config.ClientSecret,
	})

	name, usage, envs = makeFlagNames("oauth2-token-url", prefix)
	flags = append(flags, &cli.StringFlag{
		Name:        name,
		Usage:       usage,
		EnvVars:     envs,
		Destination: &cfg.OAuth2Config.Endpoint.TokenURL,
		Value:       def.OAuth2Config.Endpoint.TokenURL,
	})

	name, usage, envs = makeFlagNames("oauth2-scopes", prefix)
	flags = append(flags, &cli.StringSliceFlag{
		Name:        name,
		Usage:       usage,
		EnvVars:     envs,
		Destination: &cfg.OAuth2Scopes,
	})

	return flags
}

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

func (cfg *IMAPConfig) validateUserPass(prefix string) (string, string, error) {
	if cfg.Username == "" {
		return "", "", fmt.Errorf("\"%v-username\" is required when using %v auth", prefix, cfg.AuthMethod)
	}

	var password string
	username := cfg.Username

	if cfg.Password != "" {
		password = cfg.Password
	} else if cfg.PasswordFile != "" {
		pass, err := ioutil.ReadFile(cfg.PasswordFile)
		if err != nil {
			return "", "", err
		}

		password = strings.TrimSpace(string(pass))
	} else {
		return "", "", fmt.Errorf("at least one of the \"%v-password\" or \"%v-password-file\" flags is required", prefix, prefix)
	}

	return username, password, nil
}

func (cfg *IMAPConfig) buildTransportConfig(transConfig *pump.TransportConfig, prefix string) error {
	sourceURL, err := url.Parse(cfg.URL)
	if err != nil {
		return err
	}

	hostPort, mailbox, wantTLS, err := extractUrl(sourceURL)
	if err != nil {
		return err
	}

	transConfig.HostPort = hostPort

	cfg.AuthMethod = strings.ToUpper(cfg.AuthMethod)

	switch cfg.AuthMethod {
	case "NORMAL":
		user, pass, err := cfg.validateUserPass(prefix)
		if err != nil {
			return err
		}

		transConfig.Auth = imap.NewNormalAuthenticator(user, pass)
	case sasl.Plain:
		user, pass, err := cfg.validateUserPass(prefix)
		if err != nil {
			return err
		}
		transConfig.Auth = imap.NewSASLAuthenticator(sasl.NewPlainClient("", user, pass))
	case sasl.OAuthBearer:
		switch cfg.OAuth2Prov {
		case "custom":
			cfg.OAuth2Config.Scopes = cfg.OAuth2Scopes.Value()
		case "google":
			cfg.OAuth2Config = oauthProviderGoogle
		default:
			return fmt.Errorf("unknown oauth2 provider: %v", cfg.OAuth2Prov)
		}

		user, pass, err := cfg.validateUserPass(prefix)
		if err != nil {
			return err
		}

		// Validate the token
		tok := &oauth2.Token{}
		if err := json.Unmarshal([]byte(pass), &tok); err != nil {
			return err
		}

		ctx := context.Background() // FIXME: use parent context
		transConfig.Auth = imap.NewOAuthBearerAuthenticator(user, cfg.OAuth2Config.TokenSource(ctx, tok))
	default:
		return fmt.Errorf("unsupported auth method: %v", cfg.AuthMethod)

	}

	username := cfg.Username
	var password string

	if cfg.Password != "" {
		password = cfg.Password
	} else if cfg.PasswordFile != "" {
		pass, err := ioutil.ReadFile(cfg.PasswordFile)
		if err != nil {
			return err
		}

		password = strings.TrimSpace(string(pass))
	} else {
		return fmt.Errorf("at least one of the \"%v-password\" or \"%v-password-file\" flags is required", prefix, prefix)
	}

	transConfig.Auth = imap.NewNormalAuthenticator(username, password)

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
