package bandwidth

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/c0deaddict/waybar-widgets/pkg/waybar"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
)

// source: https://github.com/cmprmsd/sway-netusage/blob/master/waybar-netusage.go
// stats fetches the cumulative rx/tx bytes for network interface iface.
func stats(iface string) (rx, tx uint64) {
	b, err := ioutil.ReadFile("/proc/net/dev")
	if err != nil {
		return 0, 0
	}
	buff := bytes.NewBuffer(b)
	for l, err := buff.ReadString('\n'); err == nil; {
		l = strings.Trim(l, " \n")
		if !strings.HasPrefix(l, iface) {
			l, err = buff.ReadString('\n')
			continue
		}
		re := regexp.MustCompile(" +")
		s := strings.Split(re.ReplaceAllString(l, " "), " ")
		rx, err := strconv.ParseUint(s[1], 10, 64)
		if err != nil {
			return 0, 0
		}
		tx, err := strconv.ParseUint(s[9], 10, 64)
		if err != nil {
			return 0, 0
		}
		return rx, tx
	}
	return 0, 0
}

func format(rate float64) string {
	if rate < 1024 {
		return fmt.Sprintf("%6.1f  B/s", rate)
	} else if rate < 1024*1024 {
		return fmt.Sprintf("%6.1f kB/s", rate/1024)
	} else if rate < 1024*1024*1024 {
		return fmt.Sprintf("%6.1f MB/s", rate/1024/1024)
	} else {
		return fmt.Sprintf("%6.1f GB/s", rate/1024/1024/1024)
	}
}

type widget struct {
	iface    string
	rx       bool
	interval time.Duration
	warning  uint64
	critical uint64
	maximum  uint64
}

func newWidget(c *cli.Context, iface string, rx bool) widget {
	return widget{
		iface:    iface,
		rx:       rx,
		interval: c.Duration("interval"),
		warning:  c.Uint64("warning"),
		critical: c.Uint64("critical"),
		maximum:  c.Uint64("maximum"),
	}
}

func (w widget) run() {
	w.emit(0)
	prevRx, prevTx := stats(w.iface)
	prev := time.Now()
	for {
		time.Sleep(w.interval)
		now := time.Now()
		window := now.Sub(prev).Seconds()
		prev = now
		rx, tx := stats(w.iface)
		var bytes uint64
		if w.rx {
			bytes = rx - prevRx
		} else {
			bytes = tx - prevTx
		}
		prevRx, prevTx = rx, tx

		rate := uint64(float64(bytes) / window)
		w.emit(rate)
	}
}

func (w widget) emit(rate uint64) {
	message := waybar.Message{
		Class:   []string{},
		Text:    format(float64(rate)),
		Tooltip: "",
		Alt:     "",
	}

	if w.critical != 0 && rate > w.critical {
		message.Class = []string{"critical"}
	} else if w.warning != 0 && rate > w.warning {
		message.Class = []string{"warning"}
	}

	if w.maximum != 0 {
		percentage := uint(100 * (float64(rate) / float64(w.maximum)))
		if percentage > 100 {
			percentage = 0
		}
		message.Percentage = &percentage
	}

	if err := message.Emit(); err != nil {
		log.Error().Err(err).Msg("emit")
	}
}

func BandwidthCommand() *cli.Command {
	return &cli.Command{
		Name:  "bandwidth",
		Usage: "network bandwidth",
		Flags: []cli.Flag{
			&cli.DurationFlag{
				Name:    "interval",
				Value:   3 * time.Second,
				Aliases: []string{"i"},
				EnvVars: []string{"BANDWIDTH_INTERVAL"},
			},
			&cli.Uint64Flag{
				Name:    "warning",
				Usage:   "warning rate in bytes/second",
				Value:   0,
				Aliases: []string{"w"},
				EnvVars: []string{"BANDWIDTH_WARNING"},
			},
			&cli.Uint64Flag{
				Name:    "critical",
				Usage:   "critical rate in bytes/second",
				Value:   0,
				Aliases: []string{"c"},
				EnvVars: []string{"BANDWIDTH_CRITICAL"},
			},
			&cli.Uint64Flag{
				Name:    "maximum",
				Usage:   "maximum rate on the interface",
				Value:   0,
				Aliases: []string{"m"},
				EnvVars: []string{"BANDWIDTH_MAXIMUM"},
			},
		},
		Subcommands: []*cli.Command{
			{
				Name:  "up",
				Usage: "bandwidth upload",
				Action: func(c *cli.Context) error {
					iface := c.Args().First()
					newWidget(c, iface, false).run()
					return nil
				},
			},
			{
				Name:  "down",
				Usage: "bandwidth download",
				Action: func(c *cli.Context) error {
					iface := c.Args().First()
					newWidget(c, iface, true).run()
					return nil
				},
			},
		},
	}
}
