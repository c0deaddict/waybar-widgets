package main

import (
	"log"

	"os"

	"github.com/c0deaddict/waybar-widgets/internal/network"
	"github.com/c0deaddict/waybar-widgets/internal/pomo"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "waybar-widgets",
		Usage: "My custom waybar widgets and services",
		Flags: []cli.Flag{},
		Commands: []*cli.Command{
			pomo.PomoCommand(),
			network.NetworkCommand(),
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatalln(err)
	}
}
