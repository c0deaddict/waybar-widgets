package online

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/c0deaddict/waybar-widgets/pkg/waybar"
	"github.com/go-ping/ping"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
)

type widget struct {
	host             string
	interval         time.Duration
	warningThreshold int
	offlineThreshold int

	missedPings int
	lastSeq     int
}

func newWidget(c *cli.Context) widget {
	return widget{
		host:             c.String("host"),
		interval:         c.Duration("interval"),
		warningThreshold: c.Int("warning-threshold"),
		offlineThreshold: c.Int("offline-threshold"),
	}
}

func emitUpdate(text string, class string) {
	message := waybar.Message{
		Class:      []string{class},
		Text:       text,
		Percentage: nil,
		Tooltip:    "",
		Alt:        class,
	}
	data, err := json.Marshal(message)
	if err != nil {
		log.Error().Err(err).Msg("marshal json")
		return
	}
	data = append(data, '\n')
	os.Stdout.Write(data)
}

func (w widget) run() error {
	pinger, err := ping.NewPinger(w.host)
	if err != nil {
		return fmt.Errorf("pinger: %v", err)
	}

	pinger.Interval = w.interval
	pinger.OnSend = func(pkg *ping.Packet) {
		if w.lastSeq == pkg.Seq-1 {
			w.missedPings = 0
		} else {
			w.missedPings += 1
			if w.missedPings >= w.warningThreshold {
				class := "warning"
				text := strconv.Itoa(w.missedPings)
				if w.missedPings >= w.offlineThreshold {
					class = "offline"
				}
				emitUpdate(text, class)
			}
		}
	}
	pinger.OnRecv = func(pkg *ping.Packet) {
		w.lastSeq = pkg.Seq
		text := fmt.Sprintf("%.1fms", pkg.Rtt.Seconds()*1000)
		emitUpdate(text, "online")
	}

	return pinger.Run()
}

func OnlineCommand() *cli.Command {
	return &cli.Command{
		Name:  "online",
		Usage: "network online",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "host",
				Required: true,
				Aliases:  []string{"t"},
				EnvVars:  []string{"ONLINE_HOST"},
			},
			&cli.DurationFlag{
				Name:    "interval",
				Value:   5 * time.Second,
				Aliases: []string{"i"},
				EnvVars: []string{"ONLINE_INTERVAL"},
			},
			&cli.IntFlag{
				Name:    "warning-threshold",
				Usage:   "number of missed pings to go into warning state",
				Value:   3,
				Aliases: []string{"w"},
				EnvVars: []string{"ONLINE_WARNING_THRESHOLD"},
			},
			&cli.IntFlag{
				Name:    "offline-threshold",
				Usage:   "number of missed pings to go into offline state",
				Value:   5,
				Aliases: []string{"o"},
				EnvVars: []string{"ONLINE_OFFLINE_THRESHOLD"},
			},
		},
		Action: func(c *cli.Context) error {
			return newWidget(c).run()
		},
	}
}
