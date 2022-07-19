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

package oauthlogin

import (
	"github.com/emersion/go-oauthdialog"
	"github.com/emersion/go-sasl"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"github.com/vs49688/mailpump/cmd/config"
	"golang.org/x/oauth2"
)

func RegisterCommand(app *cli.App) *cli.App {
	cfg := &config.OAuth2Config{}
	app.Commands = append(app.Commands, &cli.Command{
		Name:   "oauthlogin",
		Usage:  "Generate an OAuth2 Token",
		Flags:  cfg.Parameters(),
		Action: func(context *cli.Context) error { return oauthlogin(context, cfg) },
	})
	return app
}

func oauthlogin(ctx *cli.Context, cfg *config.OAuth2Config) error {
	if err := cfg.Resolve(); err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"auth_url":  cfg.Config.Endpoint.AuthURL,
		"token_url": cfg.Config.Endpoint.TokenURL,
		"client_id": cfg.Config.ClientID,
		"scopes":    cfg.Config.Scopes,
	}).Info("using_provider")

	code, err := oauthdialog.Open(&cfg.Config)
	if err != nil {
		return err
	}

	tok, err := cfg.Config.Exchange(ctx.Context, code, oauth2.AccessTypeOffline)
	if err != nil {
		return err
	}

	log.Infof("Your OAuth2 token is:\n")
	log.Info()
	log.Infof("  %v\n", tok.RefreshToken)
	log.Info()
	log.Infof("You may now pass this via:\n")
	log.Infof("  --{source,dest}-auth-method=%v (MAILPUMP_{SOURCE,DEST}_AUTH_METHOD=%v), and\n", sasl.OAuthBearer, sasl.OAuthBearer)
	log.Infof("  --{source,dest}-password=<token> (MAILPUMP_{SOURCE,DEST}_PASSWORD=<token>)\n")
	log.Info()
	log.Infof("> Keep It Secret, Keep It Safe\n")
	log.Infof(">   - Gandalf\n")

	return nil
}
