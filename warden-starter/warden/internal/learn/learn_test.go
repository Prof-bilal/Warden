package learn

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/warden-sandbox/warden/internal/audit"
)

func TestProposeReadsFromObservedWorkspace(t *testing.T) {
	workspace, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	data := filepath.Join(workspace, "data.txt")
	secret := filepath.Join(workspace, ".env")
	for _, file := range []string{data, secret} {
		if err := os.WriteFile(file, []byte("fixture"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	event := func(action, path string, allowed bool) audit.Event {
		return audit.Event{Type: "file", Action: action, Resource: path, Allowed: allowed}
	}
	p, report := Propose([]audit.Event{
		event("openat", data, true),
		event("openat", secret, true),
		event("openat", "relative.txt", true),
		event("stat", workspace, true),
		event("unlink", data, false),
		{Type: "network", Kind: "learn-egress", Resource: "example.org"},
		{Type: "network", Kind: "learn-egress", Resource: "example.org"},
	}, []string{filepath.Join(workspace, "launcher")}, workspace)
	if !slices.Equal(p.Filesystem.Read, []string{data}) || !p.Filesystem.ProtectSecrets ||
		len(p.Filesystem.Write) != 0 || len(p.Network.Allow) != 0 {
		t.Fatalf("candidate policy = %+v", p)
	}
	if !slices.Equal(report.ProposedReads, []string{data}) || !slices.Equal(report.RequestedHosts, []string{"example.org"}) {
		t.Fatalf("report = %+v", report)
	}
}
