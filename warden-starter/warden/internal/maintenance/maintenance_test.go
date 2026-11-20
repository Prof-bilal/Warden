package maintenance

import (
	"github.com/warden-sandbox/warden/internal/policy"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCompareInstallGuard(t *testing.T) {
	before := policy.Policy{Guard: &policy.InstallGuard{MinimumReleaseAge: 72 * time.Hour, Strict: true}}
	for _, tc := range []struct {
		next *policy.InstallGuard
		want bool
	}{
		{nil, true},
		{&policy.InstallGuard{MinimumReleaseAge: 48 * time.Hour, Strict: true}, true},
		{&policy.InstallGuard{MinimumReleaseAge: 72 * time.Hour}, true},
		{&policy.InstallGuard{MinimumReleaseAge: 96 * time.Hour, Strict: true}, false},
	} {
		d := Compare(before, policy.Policy{Guard: tc.next})
		if d.InstallGuardWeakened != tc.want || d.ReviewRequired != tc.want {
			t.Fatalf("guard %v produced %+v, want review %t", tc.next, d, tc.want)
		}
	}
}

func TestCompareDenialsAndAgent(t *testing.T) {
	before := policy.Policy{
		Filesystem: policy.Filesystem{Deny: []string{"**/.env"}, ProtectSecrets: true},
		Env:        policy.Env{Deny: []string{"*_TOKEN"}},
		Agent:      &policy.Agent{Workspace: "/before"},
	}
	for _, tc := range []struct {
		name string
		edit func(*policy.Policy)
		want bool
	}{
		{"remove path denial", func(p *policy.Policy) { p.Filesystem.Deny = nil }, true},
		{"remove environment denial", func(p *policy.Policy) { p.Env.Deny = nil }, true},
		{"add denial", func(p *policy.Policy) { p.Env.Deny = []string{"*_TOKEN", "*_KEY"} }, false},
		{"disable secret protection", func(p *policy.Policy) { p.Filesystem.ProtectSecrets = false }, true},
		{"change workspace", func(p *policy.Policy) { p.Agent = &policy.Agent{Workspace: "/after"} }, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			after := before
			tc.edit(&after)
			if got := Compare(before, after); got.ReviewRequired != tc.want {
				t.Fatalf("Compare = %+v, review required = %t", got, tc.want)
			}
		})
	}
}

func TestPermissionReviewAndLocalReport(t *testing.T) {
	a := policy.Policy{Filesystem: policy.Filesystem{Read: []string{"/selected"}}, Limits: policy.Limits{MemoryMB: 512, TimeoutS: 30}}
	b := a
	b.Filesystem.Write = []string{"/selected"}
	b.Env.Allow = []string{"API_TOKEN"}
	b.Limits.TimeoutS = 0
	d := Compare(a, b)
	if !d.ReviewRequired || !d.LimitsExpanded || len(d.Changes[1].Added) != 1 {
		t.Fatal("permission expansion not marked")
	}
	if Compare(a, a).ReviewRequired {
		t.Fatal("unchanged policy requires review")
	}
	path := filepath.Join(t.TempDir(), "pilot.jsonl")
	for _, r := range []Record{{time.Now().UTC(), "cursor", "setup-completed", 60}, {time.Now().UTC(), "cursor", "setup-completed", 120}, {time.Now().UTC(), "cursor", "setup-failed", 0}} {
		if e := Append(path, r); e != nil {
			t.Fatal(e)
		}
	}
	rows, e := Read(path)
	if e != nil {
		t.Fatal(e)
	}
	s := Summarize(rows)
	if s.MedianSetupSeconds != 90 || s.Records != 3 || s.SetupFailureFraction != float64(1)/3 {
		t.Fatalf("wrong report: %+v", s)
	}
	if Append(path, Record{time.Now(), "cursor", "tool-body-with-token", 0}) == nil {
		t.Fatal("arbitrary payload recorded")
	}
	_ = os.WriteFile(path, []byte("bad JSON\n"), 0600)
	if _, e := Read(path); e == nil {
		t.Fatal("malformed report ignored")
	}
}
