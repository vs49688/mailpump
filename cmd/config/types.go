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
	"errors"
	"time"
)

var (
	errInvalidScheme = errors.New("invalid uri scheme")
)

type IMAPConfig struct {
	URL           string `json:"url"`
	Username      string `json:"username"`
	Password      string `json:"-"`
	PasswordFile  string `json:"password_file"`
	TLSSkipVerify bool   `json:"tls_skip_verify"`
	Transport     string `json:"transport"`
	Debug         bool   `json:"debug"`
}

type CliConfig struct {
	Source               IMAPConfig    `json:"source"`
	Dest                 IMAPConfig    `json:"dest"`
	LogLevel             string        `json:"log_level"`
	LogFormat            string        `json:"log_format"`
	IDLEFallbackInterval time.Duration `json:"idle_fallback_interval"`
	BatchSize            uint          `json:"batch_size"`
	DisableDeletions     bool          `json:"disable_deletions"`
	FetchBufferSize      uint          `json:"fetch_buffer_size"`
	FetchMaxInterval     time.Duration `json:"fetch_max_interval"`
}
