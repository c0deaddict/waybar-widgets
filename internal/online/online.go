package online

import (
	"time"

	"github.com/urfave/cli/v2"
)

func OnlineCommand() *cli.Command {
	return &cli.Command{
		Name:  "online",
		Usage: "network online",
		Flags: []cli.Flag{
			&cli.DurationFlag{
				Name:    "update-interval",
				Value:   5 * time.Second,
				Aliases: []string{"u"},
				EnvVars: []string{"ONLINE_UPDATE_INTERVAL"},
			},
			&cli.Uint64Flag{
				Name:    "warning-threshold",
				Usage:   "number of missed pings",
				Value:   3,
				Aliases: []string{"w"},
				EnvVars: []string{"ONLINE_WARNING_THRESHOLD"},
			},
			&cli.Uint64Flag{
				Name:    "critical-threshold",
				Usage:   "number of missed pings",
				Value:   6,
				Aliases: []string{"c"},
				EnvVars: []string{"ONLINE_CRITICAL_THRESHOLD"},
			},
		},
		Action: func(c *cli.Context) error {
			// https://github.com/digineo/go-ping/blob/master/cmd/ping-test/main.go
			// TODO: ping 1.1.1.1 or/and ipv6 equivalent
			// TODO: show avg latency
			// TODO: warning and critical
			return nil
		},
	}
}
