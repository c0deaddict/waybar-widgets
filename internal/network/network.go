package network

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/urfave/cli/v2"
)

// https://github.com/cmprmsd/sway-netusage/blob/master/waybar-netusage.go
// stats fetches the cumulative rx/tx bytes for network interface iface
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

// format converts a number of bytes in KiB or MiB.
func format(counter, prevCounter uint64, window float64) string {
	if prevCounter == 0 {
		return "B"
	}
	r := float64(counter-prevCounter) / window
	if r < 1024 {
		return fmt.Sprintf("%.0f B", r)
	}
	if r < 1024*1024 {
		return fmt.Sprintf("%.0f KiB", r/1024)
	}
	return fmt.Sprintf("%.1f MiB", r/1024/1024)
}

func monitor(iface string, output func(string, string)) {
	prevRx, prevTx := stats(iface)
	prev := time.Now()
	for {
		time.Sleep(1 * time.Second)
		now := time.Now()
		window := now.Sub(prev).Seconds()
		prev = now
		rx, tx := stats(iface)
		rxRate := format(rx, prevRx, window)
		txRate := format(tx, prevTx, window)
		prevRx, prevTx = rx, tx
		output(rxRate, txRate)
	}
}

func NetworkCommand() *cli.Command {
	return &cli.Command{
		Name:  "net",
		Usage: "network widgets",
		Subcommands: []*cli.Command{
			{
				Name:  "bandwidth_up",
				Usage: "bandwidth upload",
				Action: func(c *cli.Context) error {
					iface := c.Args().First()
					monitor(iface, func(rxRate string, txRate string) {
						fmt.Println(txRate)
					})
					return nil
				},
			},
			{
				Name:  "bandwidth_down",
				Usage: "bandwidth download",
				Action: func(c *cli.Context) error {
					iface := c.Args().First()
					monitor(iface, func(rxRate string, txRate string) {
						fmt.Println(rxRate)
					})
					return nil
				},
			},
		},
	}
}
