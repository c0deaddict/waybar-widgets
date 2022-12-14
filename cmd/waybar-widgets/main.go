package main

import (
	"time"

	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"

	"github.com/c0deaddict/waybar-widgets/internal/bandwidth"
	"github.com/c0deaddict/waybar-widgets/internal/online"
	"github.com/c0deaddict/waybar-widgets/internal/pomo"
	"github.com/c0deaddict/waybar-widgets/internal/sway"
	"github.com/c0deaddict/waybar-widgets/internal/zpool"
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
			sway.SwayCommand(),
			zpool.ZpoolCommand(),
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal().Err(err).Msg("app")
	}
}
