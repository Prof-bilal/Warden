package main

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/warden-sandbox/warden/internal/policy"
	"github.com/warden-sandbox/warden/internal/sandbox"
)

const proofEnv = "WARDEN_DOCTOR_PROOF_SENTINEL"
const proofAddressEnv = "WARDEN_DOCTOR_PROOF_ADDRESS"

func doctorProve() error {
	var backend, fixtureParent string
	switch runtime.GOOS {
	case "linux":
		backend = sandbox.BackendLinux
	case "darwin":
		backend = sandbox.BackendSeatbelt
		// Seatbelt allows its scratch directories. Place negative fixtures outside them.
		var err error
		fixtureParent, err = os.UserHomeDir()
		if err != nil {
			return err
		}
	case "windows":
		backend = sandbox.BackendWindows
	default:
		return fmt.Errorf("containment proof requires a native Linux, macOS, or Windows backend")
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	dir, err := os.MkdirTemp(fixtureParent, "warden-doctor-proof-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	home := filepath.Join(dir, "home")
	shared := filepath.Join(home, "shared")
	write := filepath.Join(dir, "write")
	outside := filepath.Join(dir, "outside")
	for _, path := range []string{filepath.Join(home, ".ssh"), shared, write, outside} {
		if err := os.MkdirAll(path, 0700); err != nil {
			return err
		}
	}
	if err := os.WriteFile(filepath.Join(home, ".ssh", "id_decoy"), []byte("disposable decoy\n"), 0600); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(shared, "read-control"), []byte("ready\n"), 0600); err != nil {
		return err
	}

	saved, existed := os.LookupEnv(proofEnv)
	if err := os.Setenv(proofEnv, "disposable sentinel"); err != nil {
		return err
	}
	defer func() {
		if existed {
			_ = os.Setenv(proofEnv, saved)
		} else {
			_ = os.Unsetenv(proofEnv)
		}
	}()
	p := policy.Policy{
		Filesystem: policy.Filesystem{Read: []string{home}, Write: []string{write}, ProtectSecrets: true},
		Limits:     policy.Limits{TimeoutS: 15, MemoryMB: 256},
	}
	vars := map[string]string{"HOME": home}
	if runtime.GOOS == "windows" {
		// AppContainer grants the positive control directory without granting its secret sibling.
		p.Filesystem.Read = []string{shared}
		p.Filesystem.ProtectSecrets = false
		// A live host service tests AppContainer's offline and WFP modes without DNS.
		listener, err := net.Listen("tcp4", "127.0.0.1:0")
		if err != nil {
			return fmt.Errorf("start network control: %w", err)
		}
		defer listener.Close()
		address := listener.Addr().String()
		conn, err := net.DialTimeout("tcp4", address, time.Second)
		if err != nil {
			return fmt.Errorf("reach network control from the host: %w", err)
		}
		_ = conn.Close()
		vars[proofAddressEnv] = address
	} else if runtime.GOOS == "darwin" {
		// Seatbelt denies are unreliable beneath a broad read grant; keep the
		// positive control in shared and place the secret outside that grant.
		p.Filesystem.Read = []string{shared}
		p.Filesystem.ProtectSecrets = false
	}
	code, err := runSandboxWithEnv([]string{exe, "__prove-probe"}, p, backend, vars)
	if err != nil {
		return fmt.Errorf("containment probe failed: %w", err)
	}
	if code != 0 {
		return fmt.Errorf("containment probe failed with exit status %d", code)
	}
	if data, err := os.ReadFile(filepath.Join(write, "started")); err != nil || string(data) != "ready\n" {
		return fmt.Errorf("containment probe did not complete its write control")
	}
	if _, err := os.Lstat(filepath.Join(outside, "blocked-write")); !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("write escaped the granted directory or could not be verified")
	}
	return nil
}

func doctorProbe() int {
	home := os.Getenv("HOME")
	if filepath.Base(home) != "home" || !strings.HasPrefix(filepath.Base(filepath.Dir(home)), "warden-doctor-proof-") {
		fmt.Fprintln(os.Stderr, "FAIL disposable proof fixture missing")
		return 1
	}
	dir := filepath.Dir(home)
	data, err := os.ReadFile(filepath.Join(home, "shared", "read-control"))
	if err != nil || string(data) != "ready\n" {
		fmt.Fprintln(os.Stderr, "FAIL positive read control")
		return 1
	}
	if err := os.WriteFile(filepath.Join(dir, "write", "started"), data, 0600); err != nil {
		fmt.Fprintln(os.Stderr, "FAIL positive write control")
		return 1
	}
	_, readErr := os.ReadFile(filepath.Join(home, ".ssh", "id_decoy"))
	writeErr := os.WriteFile(filepath.Join(dir, "outside", "blocked-write"), []byte("probe\n"), 0600)
	_, hasSentinel := os.LookupEnv(proofEnv)
	ok := true
	for _, check := range []struct {
		label  string
		denied bool
	}{
		{"secret-file read", probeDenied(readErr)},
		{"write outside grant", probeDenied(writeErr)},
		{"blocked network host", proofNetworkDenied()},
		{"environment sentinel", !hasSentinel},
	} {
		if check.denied {
			fmt.Fprintln(os.Stdout, "PASS", check.label)
		} else {
			fmt.Fprintln(os.Stdout, "FAIL", check.label)
			ok = false
		}
	}
	if !ok {
		return 1
	}
	return 0
}

func proofNetworkDenied() bool {
	if runtime.GOOS == "windows" {
		address, err := netip.ParseAddrPort(os.Getenv(proofAddressEnv))
		if err != nil || address.Addr() != netip.AddrFrom4([4]byte{127, 0, 0, 1}) || address.Port() == 0 {
			return false
		}
		conn, err := net.DialTimeout("tcp4", address.String(), 3*time.Second)
		if err == nil {
			_ = conn.Close()
			return false
		}
		var op *net.OpError
		return errors.As(err, &op) && op.Op == "dial"
	}
	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("http://blocked.warden.invalid/")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 128))
	return err == nil && resp.StatusCode == http.StatusForbidden &&
		strings.TrimSpace(string(body)) == "Warden blocked network destination"
}

func probeDenied(err error) bool {
	return errors.Is(err, os.ErrNotExist) || errors.Is(err, os.ErrPermission) || errors.Is(err, syscall.EROFS)
}
