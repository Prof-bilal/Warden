package mcpbridge

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/warden-sandbox/warden/internal/privatefile"
)

var pinServerName = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,127}$`)

const pinFileLimit = 1 << 20

// PinStore persists the first reviewed tool hash for each named server.
type PinStore struct {
	Dir string
}

type ToolPin struct {
	Accepted string `json:"accepted"`
	Observed string `json:"observed,omitempty"`
}

func DefaultPinStore() (PinStore, error) {
	config := os.Getenv("XDG_CONFIG_HOME")
	if config == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return PinStore{}, err
		}
		config = filepath.Join(home, ".config")
	}
	if !filepath.IsAbs(config) {
		return PinStore{}, fmt.Errorf("MCP pin configuration directory must be absolute")
	}
	return PinStore{Dir: filepath.Join(config, "warden", "pins")}, nil
}

// Observe pins a new tool and returns the stored hash. The caller must hide
// the tool when the stored and observed hashes differ or this call fails.
func (s PinStore) Observe(server, tool, digest string) (string, error) {
	if tool == "" || !validPinDigest(digest) {
		return "", fmt.Errorf("invalid tool name or definition hash")
	}
	var pinned string
	err := s.edit(server, func(pins map[string]ToolPin) (bool, error) {
		pin := pins[tool]
		if pin.Accepted == "" {
			pin.Accepted = digest
			pins[tool] = pin
			pinned = digest
			return true, nil
		}
		pinned = pin.Accepted
		if digest == pin.Accepted || digest == pin.Observed {
			return false, nil
		}
		pin.Observed = digest
		pins[tool] = pin
		return true, nil
	})
	if err != nil {
		return "", err
	}
	return pinned, nil
}

// Read returns the pinned hash without establishing trust.
func (s PinStore) Read(server, tool string) (string, error) {
	pin, err := s.Status(server, tool)
	return pin.Accepted, err
}

// Status returns the accepted hash and the latest observed change.
func (s PinStore) Status(server, tool string) (ToolPin, error) {
	path, err := s.FilePath(server)
	if err != nil {
		return ToolPin{}, err
	}
	pins, err := readToolPins(path)
	return pins[tool], err
}

// Accept replaces a pin only if it still matches the hash shown for review.
func (s PinStore) Accept(server, tool, previous, replacement string) error {
	if tool == "" || !validPinDigest(previous) || !validPinDigest(replacement) {
		return fmt.Errorf("invalid tool name or definition hash")
	}
	return s.edit(server, func(pins map[string]ToolPin) (bool, error) {
		pin := pins[tool]
		if pin.Accepted != previous || pin.Observed != replacement {
			return false, fmt.Errorf("tool pin changed since review")
		}
		pins[tool] = ToolPin{Accepted: replacement}
		return true, nil
	})
}

func (s PinStore) FilePath(server string) (string, error) {
	if !filepath.IsAbs(s.Dir) || !pinServerName.MatchString(server) || !filepath.IsLocal(server+".json") {
		return "", fmt.Errorf("invalid MCP pin directory or server name")
	}
	return filepath.Join(s.Dir, server+".json"), nil
}

func validPinDigest(digest string) bool {
	raw, err := hex.DecodeString(digest)
	return err == nil && len(raw) == sha256.Size && strings.ToLower(digest) == digest
}

func readToolPins(path string) (map[string]ToolPin, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return make(map[string]ToolPin), nil
	}
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > pinFileLimit {
		return nil, fmt.Errorf("MCP pin store must be a regular file no larger than %d bytes", pinFileLimit)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, pinFileLimit+1))
	if err != nil {
		return nil, err
	}
	var pins map[string]ToolPin
	if len(data) > pinFileLimit || json.Unmarshal(data, &pins) != nil || pins == nil {
		return nil, fmt.Errorf("invalid MCP pin store")
	}
	for name, pin := range pins {
		if name == "" || !validPinDigest(pin.Accepted) ||
			(pin.Observed != "" && (!validPinDigest(pin.Observed) || pin.Observed == pin.Accepted)) {
			return nil, fmt.Errorf("invalid entry in MCP pin store")
		}
	}
	return pins, nil
}

func (s PinStore) edit(server string, update func(map[string]ToolPin) (bool, error)) error {
	path, err := s.FilePath(server)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(s.Dir, 0700); err != nil {
		return err
	}
	info, err := os.Lstat(s.Dir)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("MCP pin store must be a directory")
	}
	if err := privatefile.Protect(s.Dir, true); err != nil {
		return err
	}
	lockPath := path + ".lock"
	// shortcut: a process crash can leave a lock, replace this with an OS-held lock when a portable file lock is available.
	for attempt := 0; ; attempt++ {
		lock, err := os.OpenFile(lockPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err == nil {
			defer os.Remove(lockPath)
			if err := lock.Close(); err != nil {
				return err
			}
			break
		}
		if !errors.Is(err, os.ErrExist) || attempt == 20 {
			return fmt.Errorf("lock MCP pin store: %w", err)
		}
		time.Sleep(10 * time.Millisecond)
	}
	pins, err := readToolPins(path)
	if err != nil {
		return err
	}
	changed, err := update(pins)
	if err != nil || !changed {
		return err
	}
	data, err := json.Marshal(pins)
	if err != nil {
		return err
	}
	if len(data) > pinFileLimit {
		return fmt.Errorf("MCP pin store exceeds %d bytes", pinFileLimit)
	}
	temp, err := os.CreateTemp(s.Dir, ".pins-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(temp.Name())
	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(temp.Name(), path)
}
