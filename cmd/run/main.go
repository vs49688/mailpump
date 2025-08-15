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

package run

import (
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"git.vs49688.net/zane/mailpump/cmd/config"
	"git.vs49688.net/zane/mailpump/pump"
)

func RegisterCommand(app *cli.App) *cli.App {
	cfg := &config.CliConfig{}
	app.Commands = append(app.Commands, &cli.Command{
		Name:   "run",
		Usage:  "Run the pump",
		Flags:  cfg.Parameters(),
		Action: func(context *cli.Context) error { return run(context, cfg) },
	})
	return app
}

func run(_ *cli.Context, cfg *config.CliConfig) error {
	logLevel, err := log.ParseLevel(cfg.LogLevel)
	if err == nil {
		log.SetLevel(logLevel)
	}

	if cfg.LogFormat == "json" {
		log.SetFormatter(&log.JSONFormatter{})
	}

	log.WithFields(log.Fields{
		"source_url":             cfg.Source.URL,
		"source_auth_method":     cfg.Source.AuthMethod,
		"source_username":        cfg.Source.Username,
		"source_password_file":   cfg.Source.PasswordFile,
		"source_tls_skip_verify": cfg.Source.TLSSkipVerify,
		"source_transport":       cfg.Source.Transport,
		"source_debug":           cfg.Source.Debug,
		"dest_url":               cfg.Dest.URL,
		"dest_username":          cfg.Dest.Username,
		"dest_auth_method":       cfg.Dest.AuthMethod,
		"dest_password_file":     cfg.Dest.PasswordFile,
		"dest_tls_skip_verify":   cfg.Dest.TLSSkipVerify,
		"dest_transport":         cfg.Dest.Transport,
		"dest_debug":             cfg.Dest.Debug,
		"log_level":              cfg.LogLevel,
		"log_format":             cfg.LogFormat,
		"idle_fallback_interval": cfg.IDLEFallbackInterval,
		"batch_size":             cfg.BatchSize,
		"fetch_buffer_size":      cfg.FetchBufferSize,
	}).Info("starting")

	pumpConfig := pump.Config{}
	if err := cfg.BuildPumpConfig(&pumpConfig); err != nil {
		return err
	}

	doneChan := make(chan error)
	stopChan := make(chan struct{})
	pumpConfig.DoneChan = doneChan
	pumpConfig.StopChan = stopChan

	p, err := pump.NewMailPump(&pumpConfig)
	if err != nil {
		log.Fatal(err)
	}

	defer p.Close()

	sigchan := make(chan os.Signal, 10)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	sigcount := 0
	for {
		select {
		case sig := <-sigchan:
			log.WithFields(log.Fields{"signal": sig, "count": sigcount}).Trace("caught_signal")

			sigcount += 1
			if sigcount > 1 {
				log.WithFields(log.Fields{"signal": sig}).Warn("received_interrupt_force_exit")
				os.Exit(1)
			}
			log.WithFields(log.Fields{"signal": sig}).Info("received_interrupt")

			close(stopChan)
		case <-doneChan:
			log.Info("pump_terminated")
			return nil
		}
	}
}
