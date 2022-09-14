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

	"github.com/coreos/go-systemd/activation"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"

	"github.com/c0deaddict/waybar-widgets/pkg/waybar"
)

type pomoServer struct {
	workTime              time.Duration
	breakTime             time.Duration
	updateInterval        time.Duration
	idleTimeout           time.Duration
	overtimeInterval      time.Duration
	overtimeNotifications uint

	mu           sync.Mutex
	listener     net.Listener
	clients      []net.Conn
	clientStates map[net.Conn]string

	workStart         time.Time
	breakStart        *time.Time
	breakTotal        time.Duration
	notificationsSent map[uint]bool
}

type pomoUpdate struct {
	class      string
	time       time.Duration
	percentage uint
}

func newServer(c *cli.Context) (*pomoServer, error) {
	s := pomoServer{
		workTime:              c.Duration("work-time"),
		breakTime:             c.Duration("break-time"),
		updateInterval:        c.Duration("update-interval"),
		idleTimeout:           c.Duration("idle-timeout"),
		overtimeInterval:      c.Duration("overtime-interval"),
		overtimeNotifications: c.Uint("overtime-notifications"),
	}

	listeners, err := activation.Listeners()
	if err != nil {
		return nil, fmt.Errorf("activation listeners: %v", err)
	}

	if len(listeners) != 0 {
		// Use listener from SystemD socket activation.
		log.Info().Msg("using socket activation")
		s.listener = listeners[0]
	} else {
		socketPath := os.ExpandEnv(c.String("socket"))
		err := os.MkdirAll(path.Dir(socketPath), os.ModePerm)
		if err != nil {
			return nil, fmt.Errorf("ensure socket path parent dirs: %v", err)
		}

		err = os.RemoveAll(socketPath)
		if err != nil {
			return nil, fmt.Errorf("unlink socket: %v", err)
		}

		s.listener, err = net.Listen("unix", socketPath)
		if err != nil {
			return nil, fmt.Errorf("listen error: %v", err)
		}
	}

	s.reset()

	return &s, nil
}

func (s *pomoServer) run() error {
	defer s.listener.Close()

	go s.loop()

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return fmt.Errorf("accept error: %v", err)
		}

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
			delete(s.clientStates, conn)
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
		case "restart":
			s.restart()
		case "register":
			s.register(conn)
		default:
			fmt.Printf("unknown command received: %s\n", command)
		}
	}
}

func (s *pomoServer) loop() {
	ticker := time.NewTicker(1 * time.Second)
	for {
		<-ticker.C
		s.mu.Lock()
		s.sendUpdates()
		s.mu.Unlock()
	}
}

// Needs s.mu locked.
func (s *pomoServer) reset() {
	s.workStart = time.Now()
	s.breakStart = nil
	s.breakTotal = time.Duration(0)
	s.clientStates = make(map[net.Conn]string)
	s.notificationsSent = make(map[uint]bool)
}

func (s *pomoServer) idleStart() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.breakStart != nil {
		log.Warn().Msg("idle_start: break already started")
		return
	}
	now := time.Now().Add(time.Duration(-1) * s.idleTimeout)
	s.breakStart = &now
}

func (s *pomoServer) idleStop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.breakStart == nil {
		log.Warn().Msg("idle_stop: break not started")
		return
	}
	breakTime := time.Now().Sub(*s.breakStart)
	if breakTime >= s.breakTime {
		notify("Welcome back! Start new work cycle.", false)
		s.reset()
	} else {
		s.breakTotal += breakTime
		s.breakStart = nil
	}
}

func (s *pomoServer) restart() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reset()
}

func (s *pomoServer) register(conn net.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients = append(s.clients, conn)
	s.sendUpdates()
}

func percentage(time time.Duration, max time.Duration) uint {
	result := uint((100 * time) / max)
	if result > 100 {
		result = 100
	}
	return result
}

func genMessage(update pomoUpdate) waybar.Message {
	seconds := uint(update.time.Seconds())
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	seconds = seconds % 60

	text := fmt.Sprintf("%02d:%02d", minutes, seconds)
	if hours != 0 {
		text = fmt.Sprintf("%d:%s", hours, text)
	}

	return waybar.Message{
		Class:      []string{update.class},
		Text:       text,
		Percentage: &update.percentage,
		Alt:        update.class,
		Tooltip:    "",
	}
}

// Requires s.mu locked.
func (s *pomoServer) sendUpdates() {
	update := pomoUpdate{}
	if s.breakStart != nil {
		update.class = "break"
		update.time = time.Now().Sub(*s.breakStart)
		update.percentage = percentage(update.time, s.breakTime)
	} else {
		update.time = time.Now().Sub(s.workStart) - s.breakTotal
		update.class = "work"
		update.percentage = percentage(update.time, s.workTime)
		if update.time > s.workTime {
			update.class = "overtime"
			s.sendOvertimeNotifications(update.time - s.workTime)
		}
	}

	message, err := json.Marshal(genMessage(update))
	if err != nil {
		log.Error().Err(err).Msg("marshal json message")
		return
	}
	message = append(message, '\n')

	for _, c := range s.clients {
		if s.shouldSendUpdate(c, update) {
			_, err := c.Write(message)
			if err != nil {
				log.Error().Err(err).Msg("write to client failed")
			}
			s.clientStates[c] = update.class
		}
	}
}

func (s *pomoServer) shouldSendUpdate(conn net.Conn, update pomoUpdate) bool {
	if update.onInterval(s.updateInterval) {
		return true
	}
	if state, ok := s.clientStates[conn]; ok {
		return state != update.class
	}
	return true
}

func (s *pomoServer) sendOvertimeNotifications(overtime time.Duration) {
	if overtime < s.overtimeInterval {
		s.notifyOnce(0, "End of work period. Take a break now", false)
	} else if overtime < time.Duration(1+s.overtimeNotifications)*s.overtimeInterval {
		id := uint(overtime / s.overtimeInterval)
		s.notifyOnce(id, "You are on overtime. Please take a break.", true)
	}
}

func (s *pomoServer) notifyOnce(id uint, message string, critical bool) {
	if _, ok := s.notificationsSent[id]; !ok {
		notify(message, critical)
		s.notificationsSent[id] = true
	}
}

func equal[T comparable](a, b []T) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
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

func (u pomoUpdate) onInterval(interval time.Duration) bool {
	return uint(u.time.Seconds())%uint(interval.Seconds()) == 0
}
