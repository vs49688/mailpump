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

package run_multi

import (
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"github.com/vs49688/mailpump/multipump"
)

func RegisterCommand(app *cli.App) *cli.App {
	cfg := DefaultConfig()
	app.Commands = append(app.Commands, &cli.Command{
		Name:                   "run-multi",
		Usage:                  "Run the experimental many-to-one pump",
		Flags:                  cfg.Parameters(),
		UseShortOptionHandling: true,
		Before: func(context *cli.Context) error {
			return cfg.Resolve()
		},
		Action: func(context *cli.Context) error {
			return run(context, &cfg)
		},
	})
	return app
}

func run(_ *cli.Context, cfg *Configuration) error {
	logLevel, err := log.ParseLevel(cfg.LogLevel)
	if err == nil {
		cfg.Logger.SetLevel(logLevel)
	}

	if cfg.LogFormat == "json" {
		cfg.Logger.SetFormatter(&log.JSONFormatter{})
	}

	doneChan := make(chan error)
	stopChan := make(chan struct{})

	targetMailboxes := make([]string, 0, len(cfg.Sources))
	for _, src := range cfg.Sources {
		targetMailboxes = append(targetMailboxes, src.TargetMailbox)
	}

	pumpConfig := multipump.Config{
		Destination:     cfg.ResolvedDestination,
		Sources:         cfg.ResolvedSources,
		TargetMailboxes: targetMailboxes,
		DoneChan:        doneChan,
		StopChan:        stopChan,
	}

	p, err := multipump.NewPump(&pumpConfig)
	if err != nil {
		cfg.Logger.Fatal(err)
	}
	defer p.Close()

	sigchan := make(chan os.Signal, 10)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	sigcount := 0
	for {
		select {
		case sig := <-sigchan:
			cfg.Logger.WithFields(log.Fields{"signal": sig, "count": sigcount}).Trace("caught_signal")

			sigcount += 1
			if sigcount > 1 {
				cfg.Logger.WithFields(log.Fields{"signal": sig}).Warn("received_interrupt_force_exit")
				os.Exit(1)
			}
			cfg.Logger.WithFields(log.Fields{"signal": sig}).Info("received_interrupt")

			close(stopChan)
		case <-doneChan:
			cfg.Logger.Info("pump_terminated")
			return nil
		}
	}
}
