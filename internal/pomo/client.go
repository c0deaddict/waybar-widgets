package pomo

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"

	"github.com/urfave/cli/v2"
)

type pomoClient struct {
	conn net.Conn
}

func newClient(c *cli.Context) (*pomoClient, error) {
	socketPath := os.ExpandEnv(c.String("socket"))
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("connect to server: %v", err)
	}

	return &pomoClient{conn}, nil
}

func sendCommand(c *cli.Context, command string) error {
	client, err := newClient(c)
	if err != nil {
		return err
	}
	err = client.send(command)
	client.close()
	return err
}

func widgetClient(c *cli.Context) error {
	client, err := newClient(c)
	if err != nil {
		return err
	}
	err = client.send("register")
	if err != nil {
		return err
	}
	client.stream()
	return nil
}

func (c *pomoClient) send(command string) error {
	_, err := c.conn.Write([]byte(command + "\n"))
	return err
}

func (c *pomoClient) stream() {
	reader := bufio.NewReader(c.conn)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				fmt.Printf("client got disconnected: %v", err)
			}
			break
		}

		os.Stdout.Write([]byte(line))
	}
}

func (c *pomoClient) close() {
	c.conn.Close()
}
