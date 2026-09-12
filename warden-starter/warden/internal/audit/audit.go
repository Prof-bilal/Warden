// Package audit writes append-only, structured records about enforcement
// decisions.  The format is JSON Lines so it can be tailed while a server is
// running and processed without a Warden-specific parser.
package audit

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Event is one attempted access observed by Warden.
type Event struct {
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"`     // file or network
	Action    string    `json:"action"`   // open, connect, dns, ...
	Resource  string    `json:"resource"` // pathname or host[:port]
	Allowed   bool      `json:"allowed"`
	Reason    string    `json:"reason,omitempty"`
}

// Logger serializes Events to a JSON Lines stream.  It is safe to share
// between the proxy and the sandbox launcher.
type Logger struct {
	mu sync.Mutex
	w  io.Writer
}

func New(w io.Writer) *Logger { return &Logger{w: w} }

func (l *Logger) Log(e Event) error {
	if l == nil || l.w == nil {
		return nil
	}
	e.Timestamp = time.Now().UTC()
	l.mu.Lock()
	defer l.mu.Unlock()
	return json.NewEncoder(l.w).Encode(e)
}

// ReadEvents decodes a JSON Lines audit stream. Empty lines are ignored;
// malformed records are errors so policy generation never quietly drops
// evidence.
func ReadEvents(r io.Reader) ([]Event, error) {
	s := bufio.NewScanner(r)
	s.Buffer(make([]byte, 4*1024), 1024*1024)
	var events []Event
	line := 0
	for s.Scan() {
		line++
		if len(s.Bytes()) == 0 {
			continue
		}
		var e Event
		if err := json.Unmarshal(s.Bytes(), &e); err != nil {
			return nil, fmt.Errorf("parse audit event on line %d: %w", line, err)
		}
		events = append(events, e)
	}
	if err := s.Err(); err != nil {
		return nil, fmt.Errorf("read audit events: %w", err)
	}
	return events, nil
}

// OpenDefault opens Warden's persistent audit log with owner-only
// permissions.  A failure is returned to the caller: running without an
// audit trail would violate Warden's enforcement contract.
func OpenDefault() (*os.File, string, error) {
	path, err := DefaultPath()
	if err != nil {
		return nil, "", err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, "", fmt.Errorf("create audit directory: %w", err)
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, "", fmt.Errorf("open audit log: %w", err)
	}
	return f, path, nil
}

// DefaultPath returns the persistent log location without creating it.
func DefaultPath() (string, error) {
	dir, err := StateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "audit.jsonl"), nil
}

// StateDir returns Warden's owner-only state directory without creating it.
func StateDir() (string, error) {
	if dir := os.Getenv("XDG_STATE_HOME"); dir != "" {
		return filepath.Join(dir, "warden"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("locate user state directory: %w", err)
	}
	return filepath.Join(home, ".local", "state", "warden"), nil
}

// OpenTrace creates an isolated trace-session log. Unlike the continuous
// enforcement audit log, a trace session is safe to hand directly to init
// without accidentally collecting grants from another server.
func OpenTrace() (*os.File, string, error) {
	return openTraceFile(".jsonl")
}

// OpenRawTrace creates an owner-only temporary strace destination under
// Warden state, rather than in the shared system temporary directory.
func OpenRawTrace() (*os.File, string, error) {
	return openTraceFile(".strace")
}

func openTraceFile(extension string) (*os.File, string, error) {
	dir, err := StateDir()
	if err != nil {
		return nil, "", err
	}
	dir = filepath.Join(dir, "traces")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, "", fmt.Errorf("create trace directory: %w", err)
	}
	path := filepath.Join(dir, time.Now().UTC().Format("20060102T150405.000000000Z")+extension)
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return nil, "", fmt.Errorf("create trace log: %w", err)
	}
	return f, path, nil
}

// LatestTracePath returns the most recently written trace session.
func LatestTracePath() (string, error) {
	dir, err := StateDir()
	if err != nil {
		return "", err
	}
	dir = filepath.Join(dir, "traces")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("no trace logs recorded yet; run `warden trace -- <command...>` first")
		}
		return "", fmt.Errorf("read trace directory: %w", err)
	}
	var newest string
	var newestTime time.Time
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".jsonl" {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return "", fmt.Errorf("inspect trace log %q: %w", entry.Name(), err)
		}
		if newest == "" || info.ModTime().After(newestTime) {
			newest, newestTime = filepath.Join(dir, entry.Name()), info.ModTime()
		}
	}
	if newest == "" {
		return "", fmt.Errorf("no trace logs recorded yet; run `warden trace -- <command...>` first")
	}
	return newest, nil
}
