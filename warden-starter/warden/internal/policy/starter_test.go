package policy

import (
	"runtime"
	"strings"
	"testing"

	"github.com/warden-sandbox/warden/internal/audit"
)

func TestStarterFromAuditIsConservative(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX paths not absolute on Windows — skip POSIX grant logic tests")
	}
	p := StarterFromAudit([]audit.Event{
		{Type: "file", Action: "openat", Resource: "/work/input.txt", Allowed: true},
		{Type: "file", Action: "creat", Resource: "/work/out/result.txt", Allowed: true},
		{Type: "file", Action: "openat", Resource: "/etc/shadow", Allowed: false},
		{Type: "file", Action: "openat", Resource: "/usr/lib/libc.so", Allowed: true},
		{Type: "network", Action: "connect", Resource: "api.example.test:443", Allowed: true},
		{Type: "network", Action: "connect", Resource: "blocked.example.test:443", Allowed: false},
	}, []string{"/usr/bin/server", "--stdio"})
	if got, want := strings.Join(p.Filesystem.Read, ","), "/work/input.txt"; got != want {
		t.Errorf("read = %q, want %q", got, want)
	}
	if got, want := strings.Join(p.Filesystem.Write, ","), "/work/out"; got != want {
		t.Errorf("write = %q, want %q", got, want)
	}
	if got, want := strings.Join(p.Network.Allow, ","), "api.example.test"; got != want {
		t.Errorf("network = %q, want %q", got, want)
	}
	data, err := p.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "command:") {
		t.Errorf("starter YAML omits command: %s", data)
	}
}
