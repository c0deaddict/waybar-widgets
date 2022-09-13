package zpool

import (
	"bufio"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/c0deaddict/waybar-widgets/pkg/units"
	"github.com/c0deaddict/waybar-widgets/pkg/waybar"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
)

type widget struct {
	pool     string
	interval time.Duration
	column   int
	warning  uint64
	critical uint64
}

const (
	zpoolCapacityAlloc = iota
	zpoolCapacityFree
	zpoolIopsRead
	zpoolIopsWrite
	zpoolBandwidthRead
	zpoolBandwidthWrite
)

func newWidget(c *cli.Context, column int) widget {
	return widget{
		pool:     c.String("pool"),
		interval: c.Duration("interval"),
		column:   column,
		warning:  c.Uint64("warning"),
		critical: c.Uint64("critical"),
	}
}

func parseLine(line string) ([]uint64, error) {
	result := make([]uint64, 0)
	// Skip first col because that contains the pool name.
	for i, col := range strings.Split(line, "\t")[1:] {
		value, err := strconv.ParseUint(col, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse col %d '%s': %v", i, col, err)
		}
		result = append(result, value)

	}
	return result, nil
}

func (w widget) run() error {
	interval := int(w.interval.Seconds())
	if interval < 1 {
		interval = 1
	}
	// -p = display numbers in bytes
	// -H = skip header and separate columns with tab
	cmd := exec.Command("zpool", "iostat", "-pH", w.pool, strconv.Itoa(interval))
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err = cmd.Start(); err != nil {
		return err
	}
	s := bufio.NewScanner(stdout)
	for s.Scan() {
		columns, err := parseLine(s.Text())
		if err != nil {
			log.Error().Err(err).Msg("parse line")
			continue
		}
		w.emit(columns)
	}
	return nil
}

func (w widget) emit(columns []uint64) {
	value := columns[w.column]
	var text string

	switch w.column {
	case zpoolCapacityFree:
		text = units.HumanSize(value)
	case zpoolIopsRead, zpoolIopsWrite:
		text = fmt.Sprintf("%d/s", value)
	case zpoolBandwidthRead, zpoolBandwidthWrite:
		text = units.HumanSize(value) + "/s"
	}

	message := waybar.Message{
		Class:   []string{},
		Text:    text,
		Tooltip: "",
		Alt:     "",
	}

	if w.column != zpoolCapacityFree {
		// All other metrics are higher thresholds.
		if w.critical != 0 && value >= w.critical {
			message.Class = []string{"critical"}
		} else if w.warning != 0 && value >= w.warning {
			message.Class = []string{"warning"}
		}
	} else {
		// For free critical and warning are lower thresholds.
		if w.critical != 0 && value <= w.critical {
			message.Class = []string{"critical"}
		} else if w.warning != 0 && value <= w.warning {
			message.Class = []string{"warning"}
		}

		percentage := uint(float64(100*value) / float64(value+columns[zpoolCapacityAlloc]))
		message.Percentage = &percentage
	}

	if err := message.Emit(); err != nil {
		log.Error().Err(err).Msg("emit")
	}
}

func ZpoolCommand() *cli.Command {
	return &cli.Command{
		Name:  "zpool",
		Usage: "zpool",
		Flags: []cli.Flag{
			&cli.DurationFlag{
				Name:    "interval",
				Value:   3 * time.Second,
				Aliases: []string{"i"},
			},
			&cli.StringFlag{
				Name:     "pool",
				Required: true,
				Aliases:  []string{"p"},
			},
			&cli.Uint64Flag{
				Name:    "warning",
				Usage:   "warning in bytes",
				Value:   0,
				Aliases: []string{"w"},
			},
			&cli.Uint64Flag{
				Name:    "critical",
				Usage:   "critical in bytes",
				Value:   0,
				Aliases: []string{"c"},
			},
		},
		Subcommands: []*cli.Command{
			{
				Name:  "free",
				Usage: "free capacity",
				Action: func(c *cli.Context) error {
					return newWidget(c, zpoolCapacityFree).run()
				},
			},
			{
				Name:  "iops_read",
				Usage: "iops read",
				Action: func(c *cli.Context) error {
					return newWidget(c, zpoolIopsRead).run()
				},
			},
			{
				Name:  "iops_write",
				Usage: "iops write",
				Action: func(c *cli.Context) error {
					return newWidget(c, zpoolIopsWrite).run()
				},
			},
			{
				Name:  "bandwidth_read",
				Usage: "bandwidth read",
				Action: func(c *cli.Context) error {
					return newWidget(c, zpoolBandwidthRead).run()
				},
			},
			{
				Name:  "bandwidth_write",
				Usage: "bandwidth write",
				Action: func(c *cli.Context) error {
					return newWidget(c, zpoolBandwidthWrite).run()
				},
			},
		},
	}
}
