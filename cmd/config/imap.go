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
	"errors"
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
	"golang.org/x/oauth2"
)

var (
	ErrIMAPMissingUsername = errors.New("missing username")
	ErrIMAPMissingPassword = errors.New("missing password")
)

func DefaultIMAPConfig() IMAPConfig {
	return IMAPConfig{
		AuthMethod:    "LOGIN",
		TLSSkipVerify: false,
		Transport:     "persistent",
		Debug:         false,
		OAuth2:        DefaultOAuth2Config(),
	}
}

func (cfg *IMAPConfig) fillDefaults() {
	def := DefaultIMAPConfig()

	if cfg.AuthMethod == "" {
		cfg.AuthMethod = def.AuthMethod
	}

	if cfg.Transport == "" {
		cfg.Transport = def.Transport
	}

	cfg.OAuth2.fillDefaults()
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

	flags = append(flags, cfg.OAuth2.makeParameters(fmt.Sprintf("%v-oauth2", prefix))...)

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

func (cfg *IMAPConfig) validateUserPass() (string, string, error) {
	if cfg.Username == "" {
		return "", "", ErrIMAPMissingUsername
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
		return "", "", ErrIMAPMissingPassword
	}

	return username, password, nil
}

// Resolve will validate and resolve the configuration into an imap.ConnectionConfig, and
// an imap.Factory.
func (cfg *IMAPConfig) Resolve() (imap.ConnectionConfig, imap.Factory, error) {
	cfg.fillDefaults()

	connConfig := imap.ConnectionConfig{}

	sourceURL, err := url.Parse(cfg.URL)
	if err != nil {
		return imap.ConnectionConfig{}, nil, err
	}

	hostPort, mailbox, wantTLS, err := extractUrl(sourceURL)
	if err != nil {
		return imap.ConnectionConfig{}, nil, err
	}

	connConfig.HostPort = hostPort

	cfg.AuthMethod = strings.ToUpper(cfg.AuthMethod)

	switch cfg.AuthMethod {
	case "LOGIN":
		user, pass, err := cfg.validateUserPass()
		if err != nil {
			return imap.ConnectionConfig{}, nil, err
		}

		connConfig.Auth = imap.NewNormalAuthenticator(user, pass)
	case sasl.Plain:
		user, pass, err := cfg.validateUserPass()
		if err != nil {
			return imap.ConnectionConfig{}, nil, err
		}
		connConfig.Auth = imap.NewSASLAuthenticator(sasl.NewPlainClient("", user, pass))
	case sasl.OAuthBearer:
		if err := cfg.OAuth2.ResolveConfig(); err != nil {
			return imap.ConnectionConfig{}, nil, err
		}

		user, pass, err := cfg.validateUserPass()
		if err != nil {
			return imap.ConnectionConfig{}, nil, err
		}

		tok := &oauth2.Token{RefreshToken: pass}

		ctx := context.Background() // FIXME: use parent context
		connConfig.Auth = imap.NewOAuthBearerAuthenticator(user, cfg.OAuth2.Config.TokenSource(ctx, tok))
	default:
		return imap.ConnectionConfig{}, nil, fmt.Errorf("unsupported auth method: %v", cfg.AuthMethod)

	}

	connConfig.Mailbox = mailbox
	connConfig.TLS = wantTLS
	connConfig.TLSConfig = nil
	if cfg.TLSSkipVerify {
		// #nosec G402
		connConfig.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	}

	var factory imap.Factory
	if cfg.Transport != "persistent" {
		factory = client.Factory{}
	} else {
		factory = persistentclient.Factory{MaxDelay: 0}
	}

	connConfig.Debug = cfg.Debug
	return connConfig, factory, nil
}
