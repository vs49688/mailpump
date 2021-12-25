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

// FlagCounter is a hybrid between a counter and a flag, with an
// internal channel for signalling. When its count goes above zero,
// the channel is closed.
type FlagCounter struct {
	counter uint
	channel chan struct{}
}

func NewCounter() FlagCounter {
	fc := FlagCounter{}
	fc.Reset()
	return fc
}

func (c *FlagCounter) Flag() {
	c.counter++

	if c.counter == 1 {
		close(c.channel)
	}
}

func (c *FlagCounter) FlagIf(b bool) {
	if b {
		c.Flag()
	}
}

func (c *FlagCounter) FlagMany(count uint) {
	old := c.counter
	c.counter += count

	if old == 0 && count != 0 {
		close(c.channel)
	}
}

func (c *FlagCounter) IsFlagged() bool {
	return c.counter > 0
}

func (c *FlagCounter) Reset() {
	c.counter = 0
	c.channel = make(chan struct{})
}

func (c *FlagCounter) Channel() <-chan struct{} {
	return c.channel
}

func (c *FlagCounter) Count() uint {
	return c.counter
}
