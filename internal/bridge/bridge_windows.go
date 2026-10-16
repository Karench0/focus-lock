//go:build windows

// Package bridge serves only authenticated native-messaging hosts over a
// current-user ACL named pipe. It never exposes an HTTP control port.
package bridge

import (
	"bufio"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Karench0/focus-lock/internal/applications"
	"github.com/Karench0/focus-lock/internal/model"
	"github.com/Microsoft/go-winio"
	"golang.org/x/sys/windows"
)

const HostName = "com.focuslock.bridge"

// ExtensionID is derived from the public key in manifest.json (not a secret).

type Rules struct {
	SessionID string          `json:"sessionId"`
	Active    bool            `json:"active"`
	Mode      string          `json:"mode"`
	Websites  []model.Website `json:"websites"`
	Language  string          `json:"language"`
	LeaseMS   int             `json:"leaseMs"`
}
type Ack struct {
	Applied string `json:"applied"`
	OK      bool   `json:"ok"`
	Blocks  int    `json:"blocks"`
}
type Status struct {
	Browser   string    `json:"browser"`
	Path      string    `json:"path"`
	PID       uint32    `json:"pid"`
	Connected bool      `json:"connected"`
	Applied   string    `json:"applied"`
	LastSeen  time.Time `json:"-"`
	Created   int64     `json:"-"`
}
type Server struct {
	mu       sync.Mutex
	listener net.Listener
	clients  map[net.Conn]Status
	rules    func() Rules
	block    func(int)
	hostPath string
}

func PipeName() (string, error) {
	u, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return "", err
	}
	// Tests can isolate the bridge without touching a running desktop instance.
	suffix := os.Getenv("FOCUS_LOCK_PIPE_SUFFIX")
	for _, r := range suffix {
		if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-') {
			return "", errors.New("invalid test pipe suffix")
		}
	}
	return `\\.\pipe\FocusLock-` + u.User.Sid.String() + suffix, nil
}
func Start(hostPath string, rules func() Rules, block func(int)) (*Server, error) {
	name, err := PipeName()
	if err != nil {
		return nil, err
	}
	u, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return nil, err
	}
	l, err := winio.ListenPipe(name, &winio.PipeConfig{SecurityDescriptor: "D:P(A;;GA;;;" + u.User.Sid.String() + ")", InputBufferSize: 65536, OutputBufferSize: 65536})
	if err != nil {
		return nil, err
	}
	s := &Server{listener: l, clients: map[net.Conn]Status{}, rules: rules, block: block, hostPath: hostPath}
	go s.accept()
	return s, nil
}
func (s *Server) Close() {
	_ = s.listener.Close()
	s.mu.Lock()
	defer s.mu.Unlock()
	for c := range s.clients {
		_ = c.Close()
	}
	s.clients = map[net.Conn]Status{}
}
func (s *Server) accept() {
	for {
		c, err := s.listener.Accept()
		if err != nil {
			return
		}
		go s.serve(c)
	}
}
func (s *Server) authenticate(c net.Conn) (Status, error) {
	fd, ok := c.(interface{ Fd() uintptr })
	if !ok {
		return Status{}, errors.New("pipe handle unavailable")
	}
	var pid uint32
	if err := windows.GetNamedPipeClientProcessId(windows.Handle(fd.Fd()), &pid); err != nil {
		return Status{}, err
	}
	all, err := applications.Snapshot()
	if err != nil {
		return Status{}, err
	}
	var host applications.Process
	for _, p := range all {
		if p.PID == pid {
			host = p
			break
		}
	}
	if !strings.EqualFold(host.Path, s.hostPath) {
		return Status{}, errors.New("untrusted pipe client")
	}
	parent := host.Parent
	for depth := 0; depth < 8; depth++ {
		found := false
		for _, p := range all {
			if p.PID != parent {
				continue
			}
			found = true
			family := applications.Browser(p.Path)
			if family == "chrome" || family == "edge" {
				return Status{Browser: family, Path: p.Path, PID: p.PID, Created: p.Created}, nil
			}
			parent = p.Parent
			break
		}
		if !found {
			break
		}
	}
	return Status{}, errors.New("native host was not launched by Chrome or Edge")
}
func (s *Server) serve(c net.Conn) {
	defer c.Close()
	status, err := s.authenticate(c)
	if err != nil {
		return
	}
	s.mu.Lock()
	s.clients[c] = status
	s.mu.Unlock()
	defer func() { s.mu.Lock(); delete(s.clients, c); s.mu.Unlock() }()
	reader := bufio.NewScanner(c)
	reader.Buffer(make([]byte, 4096), 65536)
	encoder := json.NewEncoder(c)
	for {
		_ = c.SetReadDeadline(time.Now().Add(7 * time.Second))
		if !reader.Scan() {
			return
		}
		var ack Ack
		if json.Unmarshal(reader.Bytes(), &ack) != nil {
			return
		}
		rules := s.rules()
		s.mu.Lock()
		status.Connected = ack.OK
		status.Applied = ack.Applied
		status.LastSeen = time.Now()
		s.clients[c] = status
		s.mu.Unlock()
		if ack.OK && ack.Applied == rules.SessionID && rules.Active && ack.Blocks > 0 {
			if ack.Blocks > 100 {
				ack.Blocks = 100
			}
			s.block(ack.Blocks)
		}
		_ = c.SetWriteDeadline(time.Now().Add(2 * time.Second))
		if encoder.Encode(rules) != nil {
			return
		}
	}
}
func (s *Server) Status() []Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []Status{}
	for _, v := range s.clients {
		v.Connected = v.Connected && time.Since(v.LastSeen) < 6*time.Second
		out = append(out, v)
	}
	return out
}
func (s *Server) Healthy(p applications.Process, all []applications.Process, id string) bool {
	statuses := s.Status()
	for _, st := range statuses {
		if !st.Connected || st.Applied != id || !strings.EqualFold(p.Path, st.Path) {
			continue
		}
		pid := p.PID
		for depth := 0; depth < 32; depth++ {
			if pid == st.PID {
				for _, root := range all {
					if root.PID == pid && root.Created == st.Created {
						return true
					}
				}
			}
			found := false
			for _, parent := range all {
				if parent.PID == pid {
					pid = parent.Parent
					found = true
					break
				}
			}
			if !found {
				break
			}
		}
	}
	return false
}
func Register(hostPath string) error {
	hostPath, err := filepath.Abs(hostPath)
	if err != nil {
		return err
	}
	if _, err = os.Stat(hostPath); err != nil {
		return err
	}
	dataDir := filepath.Join(os.Getenv("LOCALAPPDATA"), "FocusLock")
	if err = os.MkdirAll(dataDir, 0700); err != nil {
		return err
	}
	manifest := map[string]any{"name": HostName, "description": "Focus Lock local bridge", "path": hostPath, "type": "stdio", "allowed_origins": []string{"chrome-extension://" + ExtensionID + "/"}}
	b, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(dataDir, "native-host.json")
	if err = os.WriteFile(path, b, 0600); err != nil {
		return err
	}
	for _, browser := range []string{`Google\Chrome`, `Microsoft\Edge`} {
		if err = registerKey(browser, path); err != nil {
			return err
		}
	}
	return nil
}
