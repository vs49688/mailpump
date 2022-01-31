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
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/endpoints"

	"github.com/vs49688/mailpump/cmd/config/obscure"
)

var oauthProviderGoogle = oauth2.Config{
	ClientID:     "684151813510-c11bifk1po8voa90cgr28gob7dldv6ou.apps.googleusercontent.com",
	ClientSecret: obscure.MustReveal("G4zsjbGQZrWaPkkMu_czWh4-ulp9wj0JC8I8WpP-EUg0vHJvO5STPzBpz5Dc0HvNOGne"),
	Endpoint:     endpoints.Google,
	Scopes:       []string{"https://mail.google.com/"},
}
