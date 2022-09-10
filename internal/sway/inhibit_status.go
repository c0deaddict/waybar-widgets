package sway

import (
	"context"
	"fmt"

	"github.com/joshuarubin/go-sway"
	gosway "github.com/joshuarubin/go-sway"
	"github.com/urfave/cli/v2"
)

type handler struct {
	sway.EventHandler
}

func (h handler) Window(ctx context.Context, e sway.WindowEvent) {
	fmt.Printf("%v %v\n", e.Container.Name, *e.Container.InhibitIdle)
}

// TODO add widget which shows current sway inhibit_idle state.
// if any window has inhibit_idle = true then show "idle inhibitted"
// listen to stream of swaymsg changes?
func main() {
	h := handler{
		EventHandler: gosway.NoOpEventHandler(),
	}

	ctx := context.Background()

	client, err := gosway.New(ctx)
	if err != nil {
		panic(err)
	}
	node, err := client.GetTree(ctx)
	if err != nil {
		panic(err)
	}
	fmt.Println(node)

	gosway.Subscribe(ctx, h, sway.EventTypeWindow)
}

func SwayCommand() *cli.Command {
	return &cli.Command{
		Name:  "sway",
		Usage: "sway",
		Flags: []cli.Flag{},
		Action: func(c *cli.Context) error {
			return nil
		},
	}
}
