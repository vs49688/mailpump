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
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"github.com/vs49688/mailpump/pump"
	"os"
	"os/signal"
	"syscall"
)

func Main() {
	cfg := &CliConfig{}
	app := cli.App{
		Name:  "mailpump",
		Usage: os.Args[0],
		Description: `MailPump monitors a mailbox via IMAP and will "pump" mail
to another mailbox on a different server, deleting the originals. 
`,
		Flags:  cfg.Parameters(),
		Action: func(context *cli.Context) error { return runPump(cfg) },
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func runPump(cfg *CliConfig) error {
	logLevel, err := log.ParseLevel(cfg.LogLevel)
	if err == nil {
		log.SetLevel(logLevel)
	}

	if cfg.LogFormat == "json" {
		log.SetFormatter(&log.JSONFormatter{})
	}

	log.WithFields(log.Fields{
		"source_url":             cfg.SourceURL,
		"source_username":        cfg.SourceUsername,
		"source_tls_skip_verify": cfg.SourceTLSSkipVerify,
		"source_transport":       cfg.SourceTransport,
		"source_debug":           cfg.SourceDebug,
		"dest_url":               cfg.DestURL,
		"dest_username":          cfg.DestUsername,
		"dest_tls_skip_verify":   cfg.DestTLSSkipVerify,
		"dest_transport":         cfg.DestTransport,
		"dest_debug":             cfg.DestDebug,
		"log_level":              cfg.LogLevel,
		"log_format":             cfg.LogFormat,
		"tick_interval":          cfg.TickInterval,
		"batch_size":             cfg.BatchSize,
	}).Info("starting")

	pumpConfig := pump.Config{}
	if err := cfg.BuildPumpConfig(&pumpConfig); err != nil {
		log.WithError(err).Error("invalid_arguments")
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
