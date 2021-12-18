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

package persistentclient

import "github.com/vs49688/mailpump/imap"

func (f *Factory) NewClient(cfg *imap.ClientConfig) (imap.Client, error) {
	c, err := NewClient(&Config{
		HostPort:  cfg.HostPort,
		Username:  cfg.Username,
		Password:  cfg.Password,
		Mailbox:   f.Mailbox,
		TLS:       cfg.TLS,
		TLSConfig: cfg.TLSConfig,
		Debug:     cfg.Debug,
		MaxDelay:  f.MaxDelay,
		Updates:   cfg.Updates,
	})

	return c, err
}
