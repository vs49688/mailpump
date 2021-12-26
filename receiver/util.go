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

package receiver

import (
	"sort"

	"github.com/emersion/go-imap"
)

func readMessages(ch chan *imap.Message) ([]uint32, map[uint32]*imap.Message) {
	// Sometimes we have dups
	unique := map[uint32]*imap.Message{}
	for msg := range ch {
		unique[msg.Uid] = msg
	}

	var uids []uint32
	for uid, _ := range unique {
		uids = append(uids, uid)
	}

	sort.Slice(uids, func(i, j int) bool { return uids[i] < uids[j] })

	return uids, unique
}
