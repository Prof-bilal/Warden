//go:build windows

package windows

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/warden-sandbox/warden/internal/policy"
)

func TestWindowsMaskProtectsFileBeforeParentGrant(t *testing.T) {
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, ".env")
	if err := os.WriteFile(file, []byte("disposable"), 0600); err != nil {
		t.Fatal(err)
	}
	before, err := snapshotDACL(longPath(file))
	if err != nil {
		t.Fatal(err)
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	p := policy.Policy{Filesystem: policy.Filesystem{Read: []string{dir}, ProtectSecrets: true}}
	plan, err := BuildPlan([]string{exe}, p, "mask-test", "127.0.0.1:54321")
	if err != nil {
		t.Fatal(err)
	}
	var fixtureGrants []pathGrant
	for _, grant := range plan.Grants {
		if strings.EqualFold(grant.Path, dir) || strings.EqualFold(grant.Path, file) {
			fixtureGrants = append(fixtureGrants, grant)
		}
	}
	plan.Grants = fixtureGrants
	restore, err := grantFilesystemAccess(plan)
	defer func() { restore() }()
	if err != nil {
		t.Fatal(err)
	}
	masked, err := snapshotDACL(longPath(file))
	if err != nil {
		t.Fatal(err)
	}
	acl := masked.sd.String()
	if !masked.protected || !strings.Contains(acl, plan.PackageSID) {
		t.Fatalf("mask at launch lacks the protected package ACL: %s", acl)
	}
	for _, ace := range strings.Split(acl, "(") {
		if strings.HasPrefix(ace, "A;") && strings.Contains(ace, plan.CapabilitySIDs[0]) {
			t.Fatalf("masked file retained a capability allow ACE: %s", acl)
		}
	}
	restore()
	restore = func() {}
	after, err := snapshotDACL(longPath(file))
	if err != nil {
		t.Fatal(err)
	}
	if before.sd.String() != after.sd.String() {
		t.Fatalf("mask cleanup changed the original DACL: before %s, after %s", before.sd.String(), after.sd.String())
	}
}
