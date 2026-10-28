//go:build linux

package agentworkspace

import (
	"bytes"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/warden-sandbox/warden/internal/policy"
)

func fixture(t *testing.T, format, content, field string) (*Session, string) {
	t.Helper()
	root := t.TempDir()
	path := filepath.Join(root, "config")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	p := policy.Policy{Filesystem: policy.Filesystem{Write: []string{root}}, Agent: &policy.Agent{Workspace: root, Redaction: policy.Redaction{Files: []policy.RedactedFile{{Path: "config", Format: format, Fields: []string{field}}}}}}
	s, err := Prepare(p)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	return s, path
}
func TestFormatsPreserveSecretsOnApply(t *testing.T) {
	for _, tc := range []struct{ format, content, field string }{
		{"dotenv", "KEY=hidden-value\nDEBUG=false\n", "KEY"},
		{"json", `{"auth":{"key":"hidden-value"},"debug":"false"}`, "/auth/key"},
		{"yaml", "auth:\n  key: hidden-value\ndebug: 'false'\n", "/auth/key"},
		{"toml", "debug='false'\n[auth]\nkey='hidden-value'\n", "/auth/key"},
	} {
		t.Run(tc.format, func(t *testing.T) {
			s, path := fixture(t, tc.format, tc.content, tc.field)
			projected := filepath.Join(s.Dir, "config")
			b, _ := os.ReadFile(projected)
			if bytes.Contains(b, []byte("hidden-value")) || !bytes.Contains(b, []byte("__WARDEN_REDACTED_")) {
				t.Fatal("projection failed to redact")
			}
			b = bytes.ReplaceAll(b, []byte("false"), []byte("true"))
			os.WriteFile(projected, b, 0600)
			changes, err := s.Changes()
			if err != nil {
				t.Fatal(err)
			}
			if len(changes) != 1 {
				t.Fatal("missing change")
			}
			var review bytes.Buffer
			if err = Review(&review, changes); err != nil {
				t.Fatal(err)
			}
			if strings.Contains(review.String(), "hidden-value") {
				t.Fatal("review disclosed secret")
			}
			if err = s.Apply(changes); err != nil {
				t.Fatal(err)
			}
			host, _ := os.ReadFile(path)
			d, err := parse(host, tc.format)
			if err != nil {
				t.Fatal(err)
			}
			v, err := d.replace(tc.field, nil)
			if err != nil || v != "hidden-value" {
				t.Fatal("secret was not preserved")
			}
			if !bytes.Contains(host, []byte("true")) {
				t.Fatal("safe edit missing")
			}
		})
	}
}
func TestTamperingFailsWithoutHostWrite(t *testing.T) {
	for _, kind := range []string{"replace", "delete", "duplicate", "move", "malformed", "symlink", "hardlink"} {
		t.Run(kind, func(t *testing.T) {
			s, path := fixture(t, "dotenv", "KEY=hidden-value\nDEBUG=false\n", "KEY")
			projected := filepath.Join(s.Dir, "config")
			b, _ := os.ReadFile(projected)
			switch kind {
			case "replace":
				b = []byte("KEY=attacker\n")
			case "delete":
				os.Remove(projected)
			case "duplicate":
				os.WriteFile(filepath.Join(s.Dir, "copy"), b, 0600)
			case "move":
				b = bytes.Replace(b, []byte("KEY="), []byte("OTHER="), 1)
			case "malformed":
				b = []byte("invalid\n")
			case "symlink":
				os.Remove(projected)
				os.Symlink(path, projected)
			case "hardlink":
				os.Remove(projected)
				os.Link(path, projected)
			}
			if kind == "replace" || kind == "move" || kind == "malformed" {
				os.WriteFile(projected, b, 0600)
			}
			if _, err := s.Changes(); err == nil {
				t.Fatal("tampering accepted")
			}
			host, _ := os.ReadFile(path)
			if string(host) != "KEY=hidden-value\nDEBUG=false\n" {
				t.Fatal("host modified")
			}
		})
	}
}
func TestConcurrentChangesRejectApply(t *testing.T) {
	for _, target := range []string{"host", "projection"} {
		t.Run(target, func(t *testing.T) {
			s, path := fixture(t, "dotenv", "KEY=hidden-value\nDEBUG=false\n", "KEY")
			projected := filepath.Join(s.Dir, "config")
			b, _ := os.ReadFile(projected)
			os.WriteFile(projected, bytes.ReplaceAll(b, []byte("false"), []byte("true")), 0600)
			changes, err := s.Changes()
			if err != nil {
				t.Fatal(err)
			}
			if target == "host" {
				os.WriteFile(path, []byte("KEY=rotated\n"), 0600)
			} else {
				os.WriteFile(projected, append(b, []byte("NEW=1\n")...), 0600)
			}
			if err = s.Apply(changes); err == nil {
				t.Fatal("concurrent changes accepted")
			}
		})
	}
}
func TestProjectionExclusionsAndEnvironment(t *testing.T) {
	root := t.TempDir()
	os.Mkdir(filepath.Join(root, ".git"), 0700)
	os.WriteFile(filepath.Join(root, ".git", "history"), []byte("old-secret"), 0600)
	os.WriteFile(filepath.Join(root, "denied"), []byte("secret"), 0600)
	p := policy.Policy{Filesystem: policy.Filesystem{Write: []string{root}, Deny: []string{filepath.Join(root, "denied")}}, Env: policy.Env{Allow: []string{"PATH", "KEY"}}, Agent: &policy.Agent{Workspace: root, Redaction: policy.Redaction{Env: []string{"KEY"}}}}
	s, err := Prepare(p)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	for _, name := range []string{".git", "denied"} {
		if _, e := os.Stat(filepath.Join(s.Dir, name)); !os.IsNotExist(e) {
			t.Fatal("excluded path copied")
		}
	}
	for _, name := range s.Policy.Env.Allow {
		if name == "KEY" {
			t.Fatal("secret env forwarded")
		}
	}
	if len(s.Policy.ReadOnlyPaths) != 3 {
		t.Fatal("notice not read-only")
	}
}
func TestAmbiguousFormatsRejected(t *testing.T) {
	for _, tc := range []struct{ format, data string }{
		{"json", `{"key":"a","key":"b"}`},
		{"yaml", "key: a\nkey: b\n"},
		{"yaml", "a: &x secret\nb: *x\n"},
		{"yaml", "a: one\n---\na: two\n"},
		{"toml", "key='a'\nkey='b'\n"},
		{"dotenv", "KEY=a\nKEY=b\n"},
		{"dotenv", "KEY=\"multiline\nsecret\"\n"},
	} {
		if _, err := parse([]byte(tc.data), tc.format); err == nil {
			t.Fatalf("accepted ambiguous %s", tc.format)
		}
	}
}
func TestNewFilesAndQuotedReview(t *testing.T) {
	s, _ := fixture(t, "dotenv", "KEY=hidden-value\n", "KEY")
	os.WriteFile(filepath.Join(s.Dir, "new.txt"), []byte("\x1b[2J"), 0600)
	changes, err := s.Changes()
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	Review(&out, changes)
	if bytes.Contains(out.Bytes(), []byte{27}) {
		t.Fatal("raw terminal escape in review")
	}
	if err = s.Apply(changes); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(s.original, "new.txt"))
	if !bytes.Equal(b, []byte("\x1b[2J")) {
		t.Fatal("new file not applied")
	}
}

func TestMissingHostParentRejectsBeforeAnyWrite(t *testing.T) {
	s, path := fixture(t, "dotenv", "KEY=hidden-value\nDEBUG=false\n", "KEY")
	projected := filepath.Join(s.Dir, "config")
	b, _ := os.ReadFile(projected)
	os.WriteFile(projected, bytes.ReplaceAll(b, []byte("false"), []byte("true")), 0600)
	os.Mkdir(filepath.Join(s.Dir, "newdir"), 0700)
	os.WriteFile(filepath.Join(s.Dir, "newdir", "new"), []byte("safe"), 0600)
	changes, err := s.Changes()
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Apply(changes); err == nil {
		t.Fatal("new parent should be rejected")
	}
	b, _ = os.ReadFile(path)
	if !bytes.Contains(b, []byte("DEBUG=false")) {
		t.Fatal("earlier file was changed before preflight")
	}
}
func TestPublicAddresses(t *testing.T) {
	for _, s := range []string{"127.0.0.1", "::1", "10.0.0.1", "169.254.169.254", "100.64.0.1", "2001:db8::1", "::ffff:192.168.1.1"} {
		if publicIP(net.ParseIP(s)) {
			t.Fatalf("allowed %s", s)
		}
	}
}

func TestApplyRollbackOnLaterWriteFailure(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("requires ordinary user directory permissions")
	}
	s, path := fixture(t, "dotenv", "KEY=hidden-value\nDEBUG=false\n", "KEY")
	// A host-only unreadable parent is detected before changes. To exercise
	// rollback after a completed replacement, use an existing non-writable root
	// subdirectory (the copied projection remains writable).
	locked := filepath.Join(s.original, "locked")
	os.Mkdir(locked, 0700)
	os.WriteFile(filepath.Join(locked, "file"), []byte("old"), 0600)
	// Add the newly created baseline file to this test session's snapshot.
	s.files["locked/file"] = &fileState{original: []byte("old"), sanitized: []byte("old"), mode: 0600}
	os.Mkdir(filepath.Join(s.Dir, "locked"), 0700)
	os.WriteFile(filepath.Join(s.Dir, "locked", "file"), []byte("new"), 0600)
	projected := filepath.Join(s.Dir, "config")
	b, _ := os.ReadFile(projected)
	os.WriteFile(projected, bytes.ReplaceAll(b, []byte("false"), []byte("true")), 0600)
	os.Chmod(locked, 0500)
	defer os.Chmod(locked, 0700)
	changes, err := s.Changes()
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Apply(changes); err == nil || !strings.Contains(err.Error(), "rolled back") {
		t.Fatalf("expected rollback: %v", err)
	}
	b, _ = os.ReadFile(path)
	if string(b) != "KEY=hidden-value\nDEBUG=false\n" {
		t.Fatal("first file was not restored")
	}
}

func TestWorkspaceRootReplacementRejected(t *testing.T) {
	s, _ := fixture(t, "dotenv", "KEY=hidden-value\n", "KEY")
	os.WriteFile(filepath.Join(s.Dir, "new"), []byte("safe"), 0600)
	changes, err := s.Changes()
	if err != nil {
		t.Fatal(err)
	}
	moved := s.original + "-moved"
	if err = os.Rename(s.original, moved); err != nil {
		t.Fatal(err)
	}
	defer os.Rename(moved, s.original)
	os.Mkdir(s.original, 0700)
	defer os.Remove(s.original)
	if err = s.Apply(changes); err == nil {
		t.Fatal("replaced host root accepted")
	}
}

func TestAgentInstructionsStayInProjection(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("Existing project instructions.\n"), 0444)
	p := policy.Policy{Filesystem: policy.Filesystem{Write: []string{root}}, Agent: &policy.Agent{Workspace: root}}
	s, err := Prepare(p)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	for _, name := range []string{"AGENTS.md", "CLAUDE.md"} {
		b, _ := os.ReadFile(filepath.Join(s.Dir, name))
		if !bytes.Contains(b, []byte("hidden from you")) {
			t.Fatal("agent notice missing")
		}
	}
	changes, err := s.Changes()
	if err != nil || len(changes) != 0 {
		t.Fatalf("instruction overlays leaked into apply: %v", err)
	}
	host, _ := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if string(host) != "Existing project instructions.\n" {
		t.Fatal("host instructions modified")
	}
	os.Chmod(filepath.Join(s.Dir, "AGENTS.md"), 0600)
	os.WriteFile(filepath.Join(s.Dir, "AGENTS.md"), []byte("modified"), 0600)
	if _, err = s.Changes(); err == nil {
		t.Fatal("instruction tampering accepted")
	}
}

func TestJSONKeepsEmptyArraysAndLargeNumbers(t *testing.T) {
	s, path := fixture(t, "json", `{"key":"hidden-value","empty":[],"n":9007199254740993,"debug":false}`, "/key")
	projected := filepath.Join(s.Dir, "config")
	b, _ := os.ReadFile(projected)
	os.WriteFile(projected, bytes.Replace(b, []byte("false"), []byte("true"), 1), 0600)
	changes, err := s.Changes()
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Apply(changes); err != nil {
		t.Fatal(err)
	}
	host, _ := os.ReadFile(path)
	var doc map[string]json.RawMessage
	if err = json.Unmarshal(host, &doc); err != nil {
		t.Fatal(err)
	}
	if string(doc["empty"]) != "[]" || string(doc["n"]) != "9007199254740993" {
		t.Fatal("unprotected JSON values changed")
	}
}
