package pomo

import (
	"github.com/urfave/cli/v2"
)

func PomoCommand() *cli.Command {
	return &cli.Command{
		Name:  "pomo",
		Usage: "pomodoro timer",
		Subcommands: []*cli.Command{
			{
				Name:  "server",
				Usage: "background server",
				Action: func(c *cli.Context) error {
					return nil
				},
			},
			{
				Name:  "widget",
				Usage: "widget client",
				Action: func(c *cli.Context) error {
					return nil
				},
			},
			{
				Name:  "idle_start",
				Usage: "signal idle start",
				Action: func(c *cli.Context) error {
					return nil
				},
			},
			{
				Name:  "idle_stop",
				Usage: "signal idle stop",
				Action: func(c *cli.Context) error {
					return nil
				},
			},
		},
	}
}
