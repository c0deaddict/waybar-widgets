package pomo

import (
	"time"

	"github.com/urfave/cli/v2"
)

func PomoCommand() *cli.Command {
	return &cli.Command{
		Name:  "pomo",
		Usage: "pomodoro timer",
		Flags: []cli.Flag{
			&cli.PathFlag{
				Name:    "socket",
				Value:   "$XDG_RUNTIME_DIR/waybar-widgets/pomo.sock",
				EnvVars: []string{"POMO_SOCKET"},
			},
			&cli.DurationFlag{
				Name:    "work-time",
				Value:   30 * time.Minute,
				EnvVars: []string{"POMO_WORK_TIME"},
			},
			&cli.DurationFlag{
				Name:    "break-time",
				Value:   5 * time.Minute,
				EnvVars: []string{"POMO_BREAK_TIME"},
			},
			&cli.DurationFlag{
				Name:    "update-interval",
				Value:   5 * time.Second,
				EnvVars: []string{"POMO_UPDATE_INTERVAL"},
			},
			&cli.DurationFlag{
				Name:    "idle-timeout",
				Value:   30 * time.Second,
				EnvVars: []string{"POMO_IDLE_TIMEOUT"},
			},
			&cli.DurationFlag{
				Name:    "overtime-interval",
				Value:   5 * time.Minute,
				EnvVars: []string{"POMO_OVERTIME_INTERVAL"},
			},
			&cli.UintFlag{
				Name:    "overtime-notifications",
				Value:   3,
				EnvVars: []string{"POMO_OVERTIME_NOTIFICATIONS"},
			},
		},
		Subcommands: []*cli.Command{
			{
				Name:  "server",
				Usage: "background server",
				Action: func(c *cli.Context) error {
					s, err := newServer(c)
					if err != nil {
						return err
					}
					return s.run()
				},
			},
			{
				Name:  "widget",
				Usage: "widget client",
				Action: func(c *cli.Context) error {
					return widgetClient(c)
				},
			},
			{
				Name:  "idle_start",
				Usage: "signal idle start",
				Action: func(c *cli.Context) error {
					return sendCommand(c, "idle_start")
				},
			},
			{
				Name:  "idle_stop",
				Usage: "signal idle stop",
				Action: func(c *cli.Context) error {
					return sendCommand(c, "idle_stop")
				},
			},
			{
				Name:  "restart",
				Usage: "restart timer",
				Action: func(c *cli.Context) error {
					return sendCommand(c, "restart")
				},
			},
		},
	}
}
