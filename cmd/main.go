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
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"git.vs49688.net/zane/mailpump/cmd/oauthlogin"
	"git.vs49688.net/zane/mailpump/cmd/run"
	run_multi "git.vs49688.net/zane/mailpump/cmd/run-multi"
)

func Main() {
	app := cli.App{
		Name:  "mailpump",
		Usage: os.Args[0],
		Description: `MailPump monitors a mailbox via IMAP and will "pump" mail
to another mailbox on a different server, deleting the originals. 
`,
	}

	run.RegisterCommand(&app)
	run_multi.RegisterCommand(&app)
	oauthlogin.RegisterCommand(&app)

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
