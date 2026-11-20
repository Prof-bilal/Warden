package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/warden-sandbox/warden/internal/mcpbridge"
)

func TestMCPPinDiffShowsObservedChange(t *testing.T) {
	bin := buildWarden(t)
	config := t.TempDir()
	store := mcpbridge.PinStore{Dir: filepath.Join(config, "warden", "pins")}
	accepted, observed := strings.Repeat("a", 64), strings.Repeat("b", 64)
	for _, digest := range []string{accepted, observed} {
		if _, err := store.Observe("source", "read", digest); err != nil {
			t.Fatal(err)
		}
	}
	stdout, stderr, code := runWarden(t, bin, []string{"XDG_CONFIG_HOME=" + config, "XDG_STATE_HOME=" + config},
		"mcp", "pin", "--server", "source", "--diff", "read")
	if code != 0 || !strings.Contains(stdout, "Accepted SHA-256: "+accepted) ||
		!strings.Contains(stdout, "Observed SHA-256: "+observed) {
		t.Fatalf("pin diff exited %d, stdout %q, stderr %q", code, stdout, stderr)
	}
	if current, err := store.Read("source", "read"); current != accepted || err != nil {
		t.Fatalf("diff modified the pin: %q, %v", current, err)
	}
}
