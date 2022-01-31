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
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net"
	"net/url"
	"strings"

	"github.com/urfave/cli/v2"
	"github.com/vs49688/mailpump/imap/client"
	"github.com/vs49688/mailpump/imap/persistentclient"
	"github.com/vs49688/mailpump/pump"
)

func DefaultIMAPConfig() IMAPConfig {
	return IMAPConfig{
		TLSSkipVerify: false,
		Transport:     "persistent",
		Debug:         false,
	}
}

func (cfg *IMAPConfig) makeIMAPParameters(lowerPrefix string) []cli.Flag {
	def := DefaultIMAPConfig()
	upperPrefix := strings.ToUpper(lowerPrefix)

	return []cli.Flag{
		&cli.StringFlag{
			Name:        fmt.Sprintf("%v-url", lowerPrefix),
			Usage:       fmt.Sprintf("%v imap url", lowerPrefix),
			EnvVars:     []string{fmt.Sprintf("MAILPUMP_%v_URL", upperPrefix)},
			Destination: &cfg.URL,
			Required:    true,
			Value:       def.URL,
		},
		&cli.StringFlag{
			Name:        fmt.Sprintf("%v-username", lowerPrefix),
			Usage:       fmt.Sprintf("%v imap username", lowerPrefix),
			EnvVars:     []string{fmt.Sprintf("MAILPUMP_%v_USERNAME", upperPrefix)},
			Destination: &cfg.Username,
			Required:    true,
			Value:       def.Username,
		},
		&cli.StringFlag{
			Name:        fmt.Sprintf("%v-password", lowerPrefix),
			Usage:       fmt.Sprintf("%v imap password", lowerPrefix),
			EnvVars:     []string{fmt.Sprintf("MAILPUMP_%v_PASSWORD", upperPrefix)},
			Destination: &cfg.Password,
			Required:    false,
			Value:       def.Password,
		},
		&cli.StringFlag{
			Name:        fmt.Sprintf("%v-password-file", lowerPrefix),
			Usage:       fmt.Sprintf("%v imap password file", lowerPrefix),
			EnvVars:     []string{fmt.Sprintf("MAILPUMP_%v_PASSWORD_FILE", upperPrefix)},
			Destination: &cfg.PasswordFile,
			Required:    false,
			Value:       def.PasswordFile,
		},
		&cli.BoolFlag{
			Name:        fmt.Sprintf("%v-tls-skip-verify", lowerPrefix),
			Usage:       fmt.Sprintf("skip %v tls verification", lowerPrefix),
			EnvVars:     []string{fmt.Sprintf("MAILPUMP_%v_TLS_SKIP_VERIFY", upperPrefix)},
			Destination: &cfg.TLSSkipVerify,
			Value:       def.TLSSkipVerify,
		},
		&cli.StringFlag{
			Name:        fmt.Sprintf("%v-transport", lowerPrefix),
			Usage:       fmt.Sprintf("%v imap transport (persistent, standard)", lowerPrefix),
			EnvVars:     []string{fmt.Sprintf("MAILPUMP_%v_TRANSPORT", upperPrefix)},
			Destination: &cfg.Transport,
			Value:       def.Transport,
		},
		&cli.BoolFlag{
			Name:        fmt.Sprintf("%v-debug", lowerPrefix),
			Usage:       fmt.Sprintf("display %v debug info", lowerPrefix),
			EnvVars:     []string{fmt.Sprintf("MAILPUMP_%v_DEBUG", upperPrefix)},
			Destination: &cfg.Debug,
			Value:       def.Debug,
		},
	}
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
