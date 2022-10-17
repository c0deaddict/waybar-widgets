package online

import (
	"fmt"
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
	err := message.Emit()
	if err != nil {
		log.Error().Err(err).Msg("emit")
	}
}

func (w widget) run() error {
	ch := make(chan ping.Packet)
	go w.loop(ch)

	var pinger *ping.Pinger
	for {
		var err error
		pinger, err = ping.NewPinger(w.host)
		if err == nil {
			break
		} else {
			log.Error().Err(err).Msg("new pinger")
			time.Sleep(w.interval)
		}
	}

	pinger.Interval = w.interval
	pinger.OnRecv = func(pkg *ping.Packet) {
		ch <- *pkg
	}

	// Running pinger can fail if there is no network, retry until it succeeds.
	for {
		err := pinger.Run()
		if err == nil {
			return nil
		}
		log.Error().Err(err).Msg("pinger run")
		time.Sleep(w.interval)
	}
}

func (w widget) loop(ch chan ping.Packet) {
	missedPings := 0
	for {
		select {
		case pkg := <-ch:
			missedPings = 0
			text := fmt.Sprintf("%.1fms", pkg.Rtt.Seconds()*1000)
			emitUpdate(text, "online")

		case <-time.After(w.interval):
			missedPings += 1
			if missedPings >= w.warningThreshold {
				class := "warning"
				text := strconv.Itoa(missedPings)
				if missedPings >= w.offlineThreshold {
					class = "offline"
				}
				emitUpdate(text, class)
			}
		}
	}
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
