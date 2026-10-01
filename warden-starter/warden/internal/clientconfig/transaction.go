package clientconfig

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/warden-sandbox/warden/internal/privatefile"
)

type Record struct {
	Config     string   `json:"config"`
	Server     string   `json:"server"`
	Format     string   `json:"format"`
	Original   Launcher `json:"original"`
	Wrapped    Launcher `json:"wrapped"`
	BeforeHash string   `json:"before_hash"`
	AfterHash  string   `json:"after_hash"`
	Policy     string   `json:"policy"`
	PolicyHash string   `json:"policy_hash"`
	Backup     string   `json:"backup"`
	Created    string   `json:"created"`
	Active     bool     `json:"active"`
}

type Plan struct {
	Config, Server, Format, Policy, PolicyHash string
	Before, After                              []byte
	Original, Wrapped                          Launcher
	Unchanged                                  bool
}

func Hash(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }

func readRegular(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("refusing non-regular file or symlink")
	}
	if info.Size() > 8*1024*1024 {
		return nil, fmt.Errorf("file exceeds 8 MiB")
	}
	return os.ReadFile(path)
}

// StateDirectory keeps historical config secrets outside project/data grants.
// An explicit override is useful for isolated test/deployment environments.
func StateDirectory(config string) string {
	base := os.Getenv("WARDEN_SETUP_STATE_DIR")
	if base == "" {
		dir, e := os.UserConfigDir()
		if e != nil {
			return ""
		}
		base = filepath.Join(dir, "warden", "setup")
	}
	if !filepath.IsAbs(base) {
		return ""
	}
	// Resolve existing ancestors too, so a state-directory symlink cannot place
	// backup secrets underneath an application grant unnoticed.
	probe := base
	suffix := []string{}
	for {
		_, e := os.Lstat(probe)
		if e == nil {
			break
		}
		if !os.IsNotExist(e) || filepath.Dir(probe) == probe {
			return ""
		}
		suffix = append([]string{filepath.Base(probe)}, suffix...)
		probe = filepath.Dir(probe)
	}
	resolved, e := filepath.EvalSymlinks(probe)
	if e != nil {
		return ""
	}
	base = filepath.Join(append([]string{resolved}, suffix...)...)
	return filepath.Join(base, Hash([]byte(config)))
}
func stateDir(config string) string { return StateDirectory(config) }
func statePath(config, server string) string {
	return filepath.Join(stateDir(config), Hash([]byte(filepath.Base(config)+"\x00"+server))+".json")
}

func Prepare(config, format, server, policy, warden, backend string) (Plan, error) {
	return PrepareWithCommand(config, format, server, policy, warden, backend, nil)
}

// PrepareWithCommand is an explicit opt-in for replacing a downloader launcher
// with the command from a reviewed prepared policy. Original argv is retained
// for undo; all other server fields remain unchanged.
func PrepareWithCommand(config, format, server, policy, warden, backend string, upstream []string) (Plan, error) {
	config, err := filepath.Abs(config)
	if err != nil {
		return Plan{}, err
	}
	// Resolve parent symlinks, but never a symlink for the config itself.
	parent, err := filepath.EvalSymlinks(filepath.Dir(config))
	if err != nil {
		return Plan{}, err
	}
	config = filepath.Join(parent, filepath.Base(config))
	if stateDir(config) == "" {
		return Plan{}, fmt.Errorf("private setup storage unavailable; set an absolute WARDEN_SETUP_STATE_DIR")
	}
	policy, err = filepath.Abs(policy)
	if err != nil {
		return Plan{}, err
	}
	pb, err := readRegular(policy)
	if err != nil {
		return Plan{}, err
	}
	b, err := readRegular(config)
	if err != nil {
		return Plan{}, err
	}
	d, err := Parse(b, format)
	if err != nil {
		return Plan{}, err
	}
	l, err := d.Launcher(server)
	if err != nil {
		return Plan{}, err
	}
	p := Plan{Config: config, Server: server, Format: format, Policy: policy, PolicyHash: Hash(pb), Before: b, Original: l}
	r, err := loadRecord(config, server)
	if err == nil && r.Active && Equal(l, r.Wrapped) {
		if r.Policy != policy || r.PolicyHash != p.PolicyHash {
			return Plan{}, fmt.Errorf("policy changed; undo the existing wrapper before changing its grants")
		}
		p.After = b
		p.Wrapped = l
		p.Unchanged = true
		return p, nil
	}
	if err != nil && !os.IsNotExist(err) {
		return Plan{}, err
	}
	if err == nil && r.Active && !Equal(l, r.Original) {
		return Plan{}, fmt.Errorf("managed launcher changed; resolve configuration drift before wrapping")
	}
	base := strings.ToLower(filepath.Base(l.Command))
	if base == "warden" || base == "warden.exe" || l.Command == warden {
		return Plan{}, fmt.Errorf("server already launches Warden; refusing nested wrappers")
	}
	if len(upstream) == 0 {
		upstream = append([]string{l.Command}, l.Args...)
	}
	args := []string{"run", "--policy", policy, "--policy-sha256", p.PolicyHash, "--backend", backend, "--"}
	args = append(args, upstream...)
	p.Wrapped = Launcher{Command: warden, Args: args, HadArgs: true}
	p.After, err = d.Patch(server, p.Wrapped)
	return p, err
}

// Apply serializes Warden writers, checks the snapshot again, makes a private
// backup, persists undo data first, and atomically replaces the configuration.
// Hosts/editors do not honor our lock: the final comparison detects observed
// edits, but an external writer can still race the last compare/rename.
func Apply(p Plan) (string, error) {
	unlock, err := lock(p.Config)
	if err != nil {
		return "", err
	}
	defer unlock()
	if p.Unchanged {
		return "", nil
	}
	current, err := readRegular(p.Config)
	if err != nil {
		return "", err
	}
	if !bytes.Equal(current, p.Before) {
		return "", fmt.Errorf("config changed since preview; no changes applied")
	}
	pb, err := readRegular(p.Policy)
	if err != nil || Hash(pb) != p.PolicyHash {
		return "", fmt.Errorf("policy changed since preview; no changes applied")
	}
	dir := stateDir(p.Config)
	if err := privateDir(dir); err != nil {
		return "", err
	}
	backup := filepath.Join(dir, "backup-"+Hash([]byte(p.Server+time.Now().UTC().Format(time.RFC3339Nano)))+".config")
	if err := writeNew(backup, current); err != nil {
		return "", err
	}
	r := Record{Config: p.Config, Server: p.Server, Format: p.Format, Original: p.Original, Wrapped: p.Wrapped, BeforeHash: Hash(p.Before), AfterHash: Hash(p.After), Policy: p.Policy, PolicyHash: p.PolicyHash, Backup: backup, Created: time.Now().UTC().Format(time.RFC3339), Active: true}
	rb, _ := json.MarshalIndent(r, "", "  ")
	if err := AtomicWrite(statePath(p.Config, p.Server), rb); err != nil {
		return "", err
	}
	current, err = readRegular(p.Config)
	if err != nil || !bytes.Equal(current, p.Before) {
		r.Active = false
		rb, _ = json.MarshalIndent(r, "", "  ")
		_ = AtomicWrite(statePath(p.Config, p.Server), rb)
		return "", fmt.Errorf("config changed during transaction; backup retained, no config changes applied")
	}
	if err := AtomicWrite(p.Config, p.After); err != nil {
		r.Active = false
		rb, _ = json.MarshalIndent(r, "", "  ")
		_ = AtomicWrite(statePath(p.Config, p.Server), rb)
		return "", err
	}
	return backup, nil
}

func Undo(config, server string, dryRun bool) (bool, error) {
	config, err := filepath.Abs(config)
	if err != nil {
		return false, err
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(config))
	if err != nil {
		return false, err
	}
	config = filepath.Join(parent, filepath.Base(config))
	var unlock func()
	if !dryRun {
		unlock, err = lock(config)
		if err != nil {
			return false, err
		}
		defer unlock()
	}
	r, err := loadRecord(config, server)
	if err != nil {
		return false, err
	}
	if !r.Active {
		return false, nil
	}
	current, err := readRegular(config)
	if err != nil {
		return false, err
	}
	d, err := Parse(current, r.Format)
	if err != nil {
		return false, err
	}
	l, err := d.Launcher(server)
	if err != nil {
		return false, err
	}
	if Hash(current) == r.BeforeHash || Equal(l, r.Original) { // interrupted apply or undo before state update
		if !dryRun {
			r.Active = false
			rb, _ := json.MarshalIndent(r, "", "  ")
			err = AtomicWrite(statePath(config, server), rb)
		}
		return false, err
	}
	if !Equal(l, r.Wrapped) {
		return false, fmt.Errorf("selected launcher changed since wrapping; refusing to overwrite it")
	}
	var out []byte
	if Hash(current) == r.AfterHash {
		out, err = readRegular(r.Backup)
		if err != nil || Hash(out) != r.BeforeHash {
			return false, fmt.Errorf("backup missing or modified")
		}
	} else {
		out, err = d.Patch(server, r.Original)
		if err != nil {
			return false, err
		}
	}
	if dryRun {
		return true, nil
	}
	latest, err := readRegular(config)
	if err != nil || !bytes.Equal(latest, current) {
		return false, fmt.Errorf("config changed during undo")
	}
	if err = AtomicWrite(config, out); err != nil {
		return false, err
	}
	r.Active = false
	rb, _ := json.MarshalIndent(r, "", "  ")
	if err = AtomicWrite(statePath(config, server), rb); err != nil {
		return true, err
	}
	return true, nil
}

func loadRecord(config, server string) (Record, error) {
	path := statePath(config, server)
	if info, err := os.Lstat(stateDir(config)); err == nil && (!info.IsDir() || info.Mode()&os.ModeSymlink != 0) {
		return Record{}, fmt.Errorf("invalid Warden state directory")
	}
	b, err := readRegular(path)
	if err != nil {
		return Record{}, err
	}
	var r Record
	if json.Unmarshal(b, &r) != nil || r.Config != config || r.Server != server {
		return Record{}, fmt.Errorf("invalid undo record")
	}
	return r, nil
}

func privateDir(dir string) error {
	if dir == "" {
		return fmt.Errorf("private storage unavailable")
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	info, err := os.Lstat(dir)
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing state directory symlink")
	}
	return privatefile.Protect(dir, true)
}

func lock(config string) (func(), error) {
	path := config + ".warden-lock"
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("configuration is locked or not writable; no changes applied")
	}
	_ = f.Close()
	return func() { _ = os.Remove(path) }, nil
}

func writeNew(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := privatefile.Protect(path, false); err != nil {
		return err
	}
	if _, err = f.Write(data); err != nil {
		return err
	}
	return f.Sync()
}

// AtomicWrite uses a same-directory temporary file and never follows a
// destination symlink. Callers must hold the configuration lock for updates.
func AtomicWrite(path string, data []byte) error {
	if info, err := os.Lstat(path); err == nil && !info.Mode().IsRegular() {
		return fmt.Errorf("refusing non-regular destination")
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".warden-tmp-")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if err = privatefile.Protect(name, false); err != nil {
		f.Close()
		return err
	}
	if _, err = f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}

type InventoryEntry struct {
	Name  string `json:"name"`
	State string `json:"state"`
}

func Inventory(config, format string) ([]InventoryEntry, error) {
	config, err := filepath.Abs(config)
	if err != nil {
		return nil, err
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(config))
	if err != nil {
		return nil, err
	}
	config = filepath.Join(parent, filepath.Base(config))
	b, err := readRegular(config)
	if err != nil {
		return nil, err
	}
	d, err := Parse(b, format)
	if err != nil {
		return nil, err
	}
	entries := []InventoryEntry{}
	for _, name := range d.Names() {
		status := "direct / unverified"
		l, _ := d.Launcher(name)
		r, e := loadRecord(config, name)
		if e == nil && r.Active {
			if Equal(l, r.Wrapped) {
				pb, e := readRegular(r.Policy)
				if e != nil || Hash(pb) != r.PolicyHash {
					status = "policy drift"
				} else {
					status = "wrapped / workflow unverified"
				}
			} else {
				status = "launcher drift"
			}
		} else if e != nil && !os.IsNotExist(e) {
			status = "invalid management record"
		}
		entries = append(entries, InventoryEntry{name, status})
	}
	return entries, nil
}
