// Package compat_test enforces the M8 invariant: the manifest and the
// fixture directories agree exactly, so a future change cannot silently drop
// (or add) a server without updating the published matrix.
package compat_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/warden-sandbox/warden/internal/compat"
	"github.com/warden-sandbox/warden/internal/policy"
)

// compatDir locates testdata/compat relative to this package.
func compatDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join("..", "..", "testdata", "compat")
	abs, err := filepath.Abs(dir)
	if err != nil {
		t.Fatalf("resolve compat dir: %v", err)
	}
	if _, err := os.Stat(abs); err != nil {
		t.Fatalf("compat dir %s: %v (run go test from the repo root package ./internal/compat/)", abs, err)
	}
	return abs
}

func loadManifest(t *testing.T, dir string) compat.Manifest {
	t.Helper()
	m, err := compat.LoadManifest(dir)
	if err != nil {
		t.Fatalf("%v", err)
	}
	return m
}

// TestMatrixMatchesFixtures enforces the M8 invariant: the manifest and the
// fixture directories agree exactly, so a future change cannot silently drop
// (or add) a server without updating the published matrix.
func TestMatrixMatchesFixtures(t *testing.T) {
	dir := compatDir(t)
	m := loadManifest(t, dir)

	if len(m.Verdicts) != 18 {
		t.Fatalf("matrix has %d entries, want 18 (M8 scope is 15-20; keep docs/compatibility.md in sync)", len(m.Verdicts))
	}
	seen := map[string]bool{}
	for _, e := range m.Verdicts {
		if e.Name == "" || e.Upstream == "" || e.Policy == "" || e.Notes == "" {
			t.Errorf("entry %q: name/upstream/policy/notes must all be set", e.Name)
		}
		if seen[e.Name] {
			t.Errorf("duplicate matrix entry %q", e.Name)
		}
		seen[e.Name] = true
		switch e.Verdict {
		case "pass", "conditional", "fail":
		default:
			t.Errorf("entry %q: verdict %q (want pass|conditional|fail)", e.Name, e.Verdict)
		}
		switch e.FailureClass {
		case "none", "warden-bug", "schema-gap", "inherent":
		default:
			t.Errorf("entry %q: failure_class %q (want none|warden-bug|schema-gap|inherent)", e.Name, e.FailureClass)
		}
		// Verdict/class coherence: failures must be classified with a reason;
		// passes must not claim a live failure (a fixed warden-bug needs its fix note).
		if e.Verdict == "pass" && e.FailureClass != "none" && e.FailureClass != "warden-bug" {
			t.Errorf("entry %q: pass verdict with failure_class %q", e.Name, e.FailureClass)
		}
		if e.FailureClass == "warden-bug" && e.FixNote == "" {
			t.Errorf("entry %q: warden-bug without fix_note", e.Name)
		}
		if (e.Verdict == "fail" || e.Verdict == "conditional") && (e.FailureClass == "none" || e.Notes == "") {
			t.Errorf("entry %q: %s verdict needs a failure_class and notes", e.Name, e.Verdict)
		}
		p := filepath.Join(dir, filepath.FromSlash(e.Policy))
		if _, err := os.Stat(p); err != nil {
			t.Errorf("entry %q: policy %s: %v", e.Name, e.Policy, err)
		}
	}

	// No stray fixture directories outside the manifest.
	fis, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read compat dir: %v", err)
	}
	for _, fi := range fis {
		if !fi.IsDir() {
			continue
		}
		if !seen[fi.Name()] {
			t.Errorf("fixture dir %q has no matrix entry (add it to matrix.yaml or remove the dir)", fi.Name())
		}
	}
}

// TestFixturePoliciesGrantWhatManifestClaims loads every fixture policy and
// checks its probes: allowed files/hosts/env are granted, denied files are
// not covered, and blocked-host spot checks stay denied (deny-by-default).
func TestFixturePoliciesGrantWhatManifestClaims(t *testing.T) {
	dir := compatDir(t)
	m := loadManifest(t, dir)

	for _, e := range m.Verdicts {
		t.Run(e.Name, func(t *testing.T) {
			polPath := filepath.Join(dir, filepath.FromSlash(e.Policy))
			p, err := policy.Load(polPath)
			if err != nil {
				t.Fatalf("load fixture policy: %v", err)
			}
			fixtureDir := filepath.Dir(polPath)

			// Normalize (applied by Load) must be idempotent: re-running it
			// changes nothing, so fixtures are in canonical form.
			before := clonePolicy(p)
			p.Normalize()
			if !sameGrants(before, p) {
				t.Errorf("policy not in canonical form: re-Normalize changed grants (run Load/Normalize and save the result)")
			}

			// Command must be backend-ready: absolute (all fixtures are).
			if len(p.Command) == 0 {
				t.Errorf("fixture policy sets no command")
			} else if _, err := policy.ResolveExecutable(p.Command); err != nil {
				t.Errorf("command %q does not resolve: %v", p.Command, err)
			}

			for _, rel := range e.ProbeAllowFile {
				abs := rel
				if !filepath.IsAbs(rel) {
					abs = filepath.Join(fixtureDir, filepath.FromSlash(rel))
				}
				if !policy.CoversFile(p, abs) {
					t.Errorf("probe allow file %q (→ %s) is NOT covered by the fixture policy", rel, abs)
				}
			}
			for _, denied := range e.ProbeDenyFile {
				if policy.CoversFile(p, denied) {
					t.Errorf("probe deny file %q IS covered — sandbox leak in fixture policy", denied)
				}
			}
			allowSet := map[string]bool{}
			for _, h := range p.Network.Allow {
				allowSet[h] = true
			}
			for _, h := range e.ProbeAllowHost {
				if !allowSet[h] {
					t.Errorf("probe allow host %q is not in network.allow", h)
				}
			}
			// Deny-by-default spot checks: these must never be allowlisted.
			for _, blocked := range []string{"evil.example.com", "169.254.169.254"} {
				if allowSet[blocked] {
					t.Errorf("blocked host %q is allowlisted — deny-by-default violation", blocked)
				}
			}
			envSet := map[string]bool{}
			for _, n := range p.EnvAllowlist() {
				envSet[n] = true
			}
			for _, n := range e.ProbeAllowEnv {
				if !envSet[n] {
					t.Errorf("probe allow env %q is not in env.allow", n)
				}
			}
		})
	}
}

// TestDocumentedSchemaGaps locks in the M8 triage outcome: wildcard hosts
// are rejected by the schema (playwright's core gap). If wildcard support is
// ever added, this test forces the matrix, fixtures, and docs to be updated
// together instead of silently changing allowlist semantics.
func TestDocumentedSchemaGaps(t *testing.T) {
	var p policy.Policy
	if _, err := p.AddHost("*.example.com"); err == nil {
		t.Fatalf("AddHost accepted wildcard %q: schema now allows wildcards — update matrix.yaml, fixtures, and docs/compatibility.md", "*.example.com")
	}
}

// TestOverlapNormalizeIsWriteWins is the regression test for the M8 warden
// bug: the shipped Slack example listed the same dir in read and write and
// every backend rejected it as ambiguous. Load/Normalize must coalesce the
// overlap to a single write grant (write subsumes reads).
func TestOverlapNormalizeIsWriteWins(t *testing.T) {
	p := policy.Policy{
		Filesystem: policy.Filesystem{
			Read:  []string{"/srv/data/cache", "/srv/data/other"},
			Write: []string{"/srv/data/cache"},
		},
	}
	p.Normalize()
	if len(p.Filesystem.Write) != 1 || p.Filesystem.Write[0] != "/srv/data/cache" {
		t.Fatalf("Normalize mangled write grants: %v", p.Filesystem.Write)
	}
	if len(p.Filesystem.Read) != 1 || p.Filesystem.Read[0] != "/srv/data/other" {
		t.Fatalf("Normalize should drop only the shadowed read grant, got: %v", p.Filesystem.Read)
	}
	if !policy.CoversFile(p, "/srv/data/cache/token.json") {
		t.Fatalf("coalesced write grant no longer covers reads underneath it")
	}
}

// TestResolveExecutable mirrors the gateway behavior for `warden run`: bare
// launcher names resolve via PATH, absolute paths pass through, and unknown
// names fail closed (never a silent unsandboxed guess).
func TestResolveExecutable(t *testing.T) {
	abs := []string{"/usr/bin/node", "server.js"}
	got, err := policy.ResolveExecutable(abs)
	if err != nil || got[0] != "/usr/bin/node" {
		t.Fatalf("absolute command should pass through, got %v, %v", got, err)
	}
	if _, err := policy.ResolveExecutable(nil); err == nil {
		t.Fatalf("empty command should fail closed")
	}
	if _, err := policy.ResolveExecutable([]string{"definitely-not-a-warden-test-binary-xyz"}); err == nil {
		t.Fatalf("unresolvable bare name should fail closed")
	}
}

func clonePolicy(p policy.Policy) policy.Policy {
	out := p
	out.Filesystem.Read = append([]string(nil), p.Filesystem.Read...)
	out.Filesystem.Write = append([]string(nil), p.Filesystem.Write...)
	out.Network.Allow = append([]string(nil), p.Network.Allow...)
	out.Env.Allow = append([]string(nil), p.Env.Allow...)
	return out
}

func sameGrants(a, b policy.Policy) bool {
	return equalSets(a.Filesystem.Read, b.Filesystem.Read) &&
		equalSets(a.Filesystem.Write, b.Filesystem.Write)
}

func equalSets(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
