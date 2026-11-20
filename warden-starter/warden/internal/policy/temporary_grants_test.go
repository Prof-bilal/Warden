package policy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTemporaryGrantsUseInjectedClockAndKeepPermanentPolicy(t *testing.T) {
	now := time.Date(2026, 11, 19, 11, 59, 0, 0, time.UTC)
	path := filepath.Join(t.TempDir(), "warden.yaml")
	if err := os.WriteFile(path, []byte(`version: 2
network:
  allow: [permanent.example.org]
temporary_grants:
  - host: active.example.org
    expires: 2026-11-19T12:00:00Z
  - host: expired.example.org
    expires: 2026-11-19T11:59:00Z
  - path: ./sample.txt
    expires: 2026-11-19T12:01:00Z
`), 0600); err != nil {
		t.Fatal(err)
	}
	p, _, err := loadVerifiedAt(path, "", now)
	if err != nil || len(p.TemporaryGrants) != 2 || len(p.Network.Allow) != 1 {
		t.Fatalf("load = %+v, %v", p, err)
	}
	active := p.WithActiveGrants(now)
	if len(active.Network.Allow) != 2 || active.Network.Allow[1] != "active.example.org" ||
		!CoversFile(active, filepath.Join(filepath.Dir(path), "sample.txt")) ||
		!active.Limits.GrantDeadline.Equal(now.Add(time.Minute)) {
		t.Fatalf("active policy = %+v", active)
	}
	if _, err := active.Marshal(); err == nil {
		t.Fatal("runtime policy with temporary grants could be saved as permanent")
	}
	after := p.WithActiveGrants(now.Add(2 * time.Minute))
	if len(after.Network.Allow) != 1 || len(after.Filesystem.Read) != 0 || !after.Limits.GrantDeadline.IsZero() {
		t.Fatalf("expired privileges remain: %+v", after)
	}
	data, err := p.Marshal()
	if err != nil || strings.Contains(string(data), "expired.example.org") ||
		strings.Contains(string(data), "allow:\n        - active.example.org") {
		t.Fatalf("saved policy contains expired or promoted permissions: %s, %v", data, err)
	}
}
