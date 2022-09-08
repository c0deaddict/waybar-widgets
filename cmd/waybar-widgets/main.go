package main

import (
	"time"

	"os"

	"github.com/c0deaddict/waybar-widgets/internal/bandwidth"
	"github.com/c0deaddict/waybar-widgets/internal/online"
	"github.com/c0deaddict/waybar-widgets/internal/pomo"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	app := &cli.App{
		Name:  "waybar-widgets",
		Usage: "My custom waybar widgets and services",
		Flags: []cli.Flag{},
		Commands: []*cli.Command{
			pomo.PomoCommand(),
			bandwidth.BandwidthCommand(),
			online.OnlineCommand(),
			// TODO add widget which shows current sway inhibit_idle state.
			// if any window has inhibit_idle = true then show "idle inhibitted"
			// listen to stream of swaymsg changes?
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal().Err(err).Msg("app")
	}
}
