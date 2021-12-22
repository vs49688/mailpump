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

// FlagCounter is a hybrid between a counter and a flag.
// When FlagCounter.Counter goes above zero, an empty struct
// is written to FlagCounter.Channel, if non-nil.
type FlagCounter struct {
	Counter uint
	Channel chan<- struct{}
}

func (c *FlagCounter) Flag() {
	c.Counter++

	if c.Counter == 1 && c.Channel != nil {
		c.Channel <- struct{}{}
	}
}

func (c *FlagCounter) FlagIf(b bool) {
	if b {
		c.Flag()
	}
}

func (c *FlagCounter) FlagMany(count uint) {
	old := c.Counter
	c.Counter += count

	if old == 0 && count != 0 && c.Channel != nil {
		c.Channel <- struct{}{}
	}
}

func (c *FlagCounter) IsFlagged() bool {
	return c.Counter > 0
}

func (c *FlagCounter) Reset() {
	c.Counter = 0
}
