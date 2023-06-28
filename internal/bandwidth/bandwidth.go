package bandwidth

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/c0deaddict/waybar-widgets/pkg/waybar"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
)

type ifaceStats struct {
	iface  string
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

func linkSpeed(iface string) int32 {
	// Some interfaces don't support reading the speed. Loopback and WiFi
	// give an InvalidArgument error.
	if iface == "lo" || strings.HasPrefix(iface, "wl") {
		return -1
	}

	filename := fmt.Sprintf("/sys/class/net/%s/speed", iface)
	b, err := os.ReadFile(filename)
	if err != nil {
		log.Error().Err(err).Msgf("linkSpeed read %s", filename)
		return -1
	}

	line := strings.TrimSpace(string(b))
	speed, err := strconv.ParseInt(line, 10, 32)
	if err != nil {
		log.Error().Err(err).Msgf("linkSpeed parse %s", string(b))
		return -1
	}

	return int32(speed)
}

func stats(iface string) ifaceStats {
	res := ifaceStats{iface, 0, 0, "up"}
	file, err := os.Open("/proc/net/dev")
	if err != nil {
		log.Error().Err(err).Msg("open /proc/net/dev")
		return res
	}
	defer file.Close()

	s := bufio.NewScanner(file)
	for s.Scan() {
		fields := strings.Fields(s.Text())
		if fields[0] != iface+":" {
			continue
		}

		res.rx, err = strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			log.Error().Err(err).Msg("parse rx")
		}

		res.tx, err = strconv.ParseUint(fields[9], 10, 64)
		if err != nil {
			log.Error().Err(err).Msg("parse tx")
		}

		if linkState(iface) {
			res.state = "up"
		} else {
			res.state = "down"
		}

		return res
	}

	return res
}

var errNoDefaultRoute = errors.New("no default route found")

// Based on:
// https://github.com/tailscale/tailscale/blob/ab310a7f6086b38475a714afb6d69d92dc5e5af6/net/interfaces/interfaces_linux.go#L232
func defaultRouteInterface() (string, error) {
	file, err := os.Open("/proc/net/route")
	if err != nil {
		return "", err
	}
	defer file.Close()

	s := bufio.NewScanner(file)
	// Skip header.
	s.Scan()
	for s.Scan() {
		fields := strings.Fields(s.Text())
		ifc := fields[0]
		ip := fields[1]
		netmask := fields[7]

		if ip == "00000000" && netmask == "00000000" {
			// default route
			return ifc, nil // interface name
		}
	}

	return "", errNoDefaultRoute
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
	iface := w.iface
	if w.iface == "" {
		var err error
		iface, err = defaultRouteInterface()
		if err != nil {
			log.Error().Err(err).Msg("determine routing interface")
		}
	}

	prev := stats(iface)
	w.emit(iface, 0, prev.state)
	prevTime := time.Now()
	for {
		time.Sleep(w.interval)
		now := time.Now()
		window := now.Sub(prevTime).Seconds()
		prevTime = now

		// TODO: maybe do this once per X seconds?
		if w.iface == "" {
			var err error
			iface, err = defaultRouteInterface()
			if err != nil {
				log.Error().Err(err).Msg("determine routing interface")
			}
		}

		cur := stats(iface)
		var bytes uint64
		if w.rx {
			bytes = cur.rx - prev.rx
		} else {
			bytes = cur.tx - prev.tx
		}
		prev = cur

		rate := uint64(float64(bytes) / window)
		w.emit(iface, rate, cur.state)
	}
}

func (w widget) emit(iface string, rate uint64, state string) {
	message := waybar.Message{
		Class:   []string{state},
		Text:    format(float64(rate)),
		Tooltip: "",
		Alt:     fmt.Sprintf("iface-%s", iface),
	}

	if w.critical != 0 && rate >= w.critical {
		message.Class = []string{"critical"}
	} else if w.warning != 0 && rate >= w.warning {
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
