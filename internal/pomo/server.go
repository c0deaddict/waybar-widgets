package pomo

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"
	"time"

	waybar "github.com/c0deaddict/waybar-widgets/pkg"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
)

type pomoServer struct {
	workTime       time.Duration
	breakTime      time.Duration
	updateInterval time.Duration

	mu      sync.Mutex
	listen  net.Listener
	clients []net.Conn

	workStart  time.Time
	breakStart *time.Time
	breakTotal time.Duration
}

func newServer(c *cli.Context) (*pomoServer, error) {
	socketPath := os.ExpandEnv(c.String("socket"))
	err := os.MkdirAll(path.Dir(socketPath), os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("ensure socket path parent dirs: %v", err)
	}

	err = os.RemoveAll(socketPath)
	if err != nil {
		return nil, fmt.Errorf("unlink socket: %v", err)
	}

	listen, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("listen error: %v", err)
	}

	s := pomoServer{
		workTime:       c.Duration("work-time"),
		breakTime:      c.Duration("break-time"),
		updateInterval: c.Duration("update-interval"),
		listen:         listen,
		workStart:      time.Now(),
	}

	return &s, nil
}

func (s *pomoServer) run() error {
	defer s.listen.Close()

	go s.loop()

	for {
		conn, err := s.listen.Accept()
		if err != nil {
			return fmt.Errorf("accept error: %v", err)
		}

		s.mu.Lock()
		s.clients = append(s.clients, conn)
		// TODO: send current time.
		s.mu.Unlock()

		go s.clientLoop(conn)
	}
}

func (s *pomoServer) removeClient(conn net.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, c := range s.clients {
		if c == conn {
			log.Info().Msgf("disconnecting client %v", conn)
			s.clients = append(s.clients[:i], s.clients[i+1:]...)
			return
		}
	}
}

func (s *pomoServer) clientLoop(conn net.Conn) {
	reader := bufio.NewReader(conn)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				log.Error().Err(err).Msg("client read error")
			}
			s.removeClient(conn)
			conn.Close()
			return
		}

		command := strings.TrimSpace(line)
		switch command {
		case "idle_start":
			s.idleStart()
		case "idle_stop":
			s.idleStop()
		default:
			fmt.Printf("unknown command received: %s\n", command)
		}
	}
}

func (s *pomoServer) loop() {
	ticker := time.NewTicker(1 * time.Second)
	for {
		<-ticker.C
		s.broadcast()
	}
}

func (s *pomoServer) idleStart() {
	fmt.Println("idle start")
}

func (s *pomoServer) idleStop() {
	fmt.Println("idle stop")
}

func genMessage(time time.Duration, max time.Duration, classes []string) waybar.Message {
	seconds := uint(time.Seconds())
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	seconds = seconds % 60

	text := fmt.Sprintf("%02d:%02d", minutes, seconds)
	if hours != 0 {
		text = fmt.Sprintf("%d:%s", hours, text)
	}

	percentage := uint((100 * time) / max)
	if percentage > 100 {
		percentage = 100
	}

	return waybar.Message{
		Class:      classes,
		Text:       text,
		Percentage: &percentage,
		Alt:        "",
		Tooltip:    "",
	}
}

func (s *pomoServer) message() waybar.Message {
	if s.breakStart != nil {
		return genMessage(time.Now().Sub(*s.breakStart), s.breakTime, []string{"break"})
	} else {
		workTotal := time.Now().Sub(s.workStart) - s.breakTotal
		classes := []string{"work"}
		if workTotal > s.workTime {
			classes = append(classes, "overtime")
		}
		return genMessage(workTotal, s.workTime, classes)
	}
}

func (s *pomoServer) broadcast() {
	s.mu.Lock()
	defer s.mu.Unlock()

	message, err := json.Marshal(s.message())
	if err != nil {
		log.Error().Err(err).Msg("marshal json message")
		return
	}

	for _, c := range s.clients {
		_, err := c.Write(append(message, '\n'))
		if err != nil {
			log.Error().Err(err).Msg("write to client failed")
		}
	}
}

func notify(message string, critical bool) {
	urgency := "normal"
	if critical {
		urgency = "critical"
	}
	cmd := exec.Command("notify-send", "-a", "pomo", "-u", urgency, message)
	if err := cmd.Run(); err != nil {
		log.Error().Err(err).Msg("notify")
	}
}
