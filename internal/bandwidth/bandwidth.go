package bandwidth

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/c0deaddict/waybar-widgets/pkg/waybar"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
)

type ifaceStats struct {
	rx, tx uint64
	state  string
}

func linkState(iface string) bool {
	filename := fmt.Sprintf("/sys/class/net/%s/operstate", iface)
	b, err := os.ReadFile(filename)
	if err != nil {
		log.Error().Err(err).Msgf("read %s", filename)
		return false
	}
	return strings.HasPrefix(string(b), "up")
}

func stats(iface string) ifaceStats {
	res := ifaceStats{0, 0, "disabled"}
	file, err := os.Open("/proc/net/dev")
	if err != nil {
		log.Error().Err(err).Msg("open /proc/net/dev")
		return res
	}
	defer file.Close()

	s := bufio.NewScanner(file)
	re := regexp.MustCompile(" +")
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if !strings.HasPrefix(line, iface+":") {
			continue
		}
		if linkState(iface) {
			res.state = "up"
		} else {
			res.state = "down"
		}
		parts := strings.Split(re.ReplaceAllString(line, " "), " ")
		res.rx, err = strconv.ParseUint(parts[1], 10, 64)
		if err != nil {
			log.Error().Err(err).Msg("parse rx")
		}
		res.tx, err = strconv.ParseUint(parts[9], 10, 64)
		if err != nil {
			log.Error().Err(err).Msg("parse tx")
		}
	}
	return res
}

func format(rate float64) string {
	if rate < 1024 {
		return fmt.Sprintf("%.0f B/s", rate)
	} else if rate < 1024*1024 {
		return fmt.Sprintf("%.1f kB/s", rate/1024)
	} else if rate < 1024*1024*1024 {
		return fmt.Sprintf("%.1f MB/s", rate/1024/1024)
	} else {
		return fmt.Sprintf("%.1f GB/s", rate/1024/1024/1024)
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
	prev := stats(w.iface)
	w.emit(0, prev.state)
	prevTime := time.Now()
	for {
		time.Sleep(w.interval)
		now := time.Now()
		window := now.Sub(prevTime).Seconds()
		prevTime = now

		cur := stats(w.iface)
		var bytes uint64
		if w.rx {
			bytes = cur.rx - prev.rx
		} else {
			bytes = cur.tx - prev.tx
		}
		prev = cur

		rate := uint64(float64(bytes) / window)
		w.emit(rate, cur.state)
	}
}

func (w widget) emit(rate uint64, state string) {
	message := waybar.Message{
		Class:   []string{state},
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
			percentage = 100
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
