package clientconfig

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestAdaptersRoundTripAndPreserveSettings(t *testing.T) {
	t.Setenv("WARDEN_SETUP_STATE_DIR", filepath.Join(t.TempDir(), "state"))
	fixtures := []struct{ id, format, config string }{
		{"cline", "json", `{"mcpServers":{"files":{"command":"node","args":[],"autoApprove":[],"disabled":false,"env":{"TOKEN":"${TOKEN}"}}}}`},
		{"cascade", "json", `{"mcpServers":{"files":{"command":"node","args":[],"disabledTools":["write"],"env":{"TOKEN":"${TOKEN}"}}}}`},
		{"claude-desktop", "json", `{"mcpServers":{"files":{"command":"/path with space/node","args":["a b.js"],"env":{"TOKEN":"do-not-print"},"disabled":false}},"preferences":{"theme":"dark"}}`},
		{"claude-code", "json", `{"mcpServers":{"files":{"command":"node","args":[],"env":{"TOKEN":"${TOKEN}"}}},"projects":{"/a":{"allowedTools":[]}}}`},
		{"cursor", "json", `{"mcpServers":{"files":{"command":"node","args":["${workspaceFolder}/s.js"],"envFile":".env"},"remote":{"url":"https://example.com/mcp"}}}`},
		{"gemini", "json", `{"mcpServers":{"files":{"command":"node","args":[],"trust":false,"timeout":30000,"cwd":"./app"}},"mcp":{"excluded":["other"]}}`},
		{"vscode", "jsonc", "{\n// retain this comment\n\"inputs\": [{\"id\": \"secret\", \"type\": \"promptString\"}],\n\"servers\": {\"files\": {\"command\": \"node\", /* launcher */ \"args\": [\"${input:secret}\",], \"sandbox\": {\"enabled\": true},},},\n}"},
		{"codex", "toml", "# retain comment\nmodel = 'example'\n[mcp_servers.files]\ncommand = '/path with space/node' # launcher\nargs = [\n  'a b.js', # retain argv\n]\nstartup_timeout_sec = 30\nenabled_tools = ['read']\n[mcp_servers.files.env]\nTOKEN = '${TOKEN}'\n"},
	}
	for _, f := range fixtures {
		t.Run(f.id, func(t *testing.T) {
			dir := t.TempDir()
			config := filepath.Join(dir, "config")
			policy := filepath.Join(dir, "policy.yaml")
			if err := os.WriteFile(config, []byte(f.config), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(policy, []byte("filesystem: {}\n"), 0600); err != nil {
				t.Fatal(err)
			}
			plan, err := Prepare(config, f.format, "files", policy, "/absolute/warden", "auto")
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(f.config, "TOKEN") && !bytes.Contains(plan.After, []byte("TOKEN")) {
				t.Fatal("unknown settings lost")
			}
			if f.id == "vscode" && !bytes.Contains(plan.After, []byte("retain this comment")) {
				t.Fatal("JSONC comment lost")
			}
			if f.id == "codex" && !bytes.Contains(plan.After, []byte("# retain comment")) {
				t.Fatal("TOML comment lost")
			}
			backup, err := Apply(plan)
			if err != nil {
				t.Fatal(err)
			}
			info, _ := os.Stat(backup)
			if runtime.GOOS != "windows" && info.Mode().Perm() != 0600 {
				t.Fatal("backup not private")
			}
			again, err := Prepare(config, f.format, "files", policy, "/absolute/warden", "auto")
			if err != nil || !again.Unchanged {
				t.Fatalf("second wrap: %v", err)
			}
			inventory, err := Inventory(config, f.format)
			if err != nil || len(inventory) != 1 || inventory[0].State != "wrapped / workflow unverified" {
				t.Fatalf("inventory: %v %v", inventory, err)
			}
			if _, err = Undo(config, "files", false); err != nil {
				t.Fatal(err)
			}
			got, _ := os.ReadFile(config)
			if string(got) != f.config {
				t.Fatalf("undo did not restore exact bytes: %s", got)
			}
		})
	}
}

func TestUndoPreservesInterveningEditsAndRejectsConflicts(t *testing.T) {
	t.Setenv("WARDEN_SETUP_STATE_DIR", filepath.Join(t.TempDir(), "state"))
	for _, format := range []string{"json", "toml"} {
		t.Run(format, func(t *testing.T) {
			dir := t.TempDir()
			config := filepath.Join(dir, "config")
			policy := filepath.Join(dir, "policy")
			original := `{"mcpServers":{"s":{"command":"node","enabled":true}},"theme":"dark"}`
			if format == "toml" {
				original = "theme = 'dark'\n[mcp_servers.s]\ncommand = 'node'\nenabled = true\n"
			}
			_ = os.WriteFile(config, []byte(original), 0600)
			_ = os.WriteFile(policy, []byte("filesystem: {}"), 0600)
			plan, e := Prepare(config, format, "s", policy, "/absolute/warden", "auto")
			if e != nil {
				t.Fatal(e)
			}
			if _, e = Apply(plan); e != nil {
				t.Fatal(e)
			}
			changed := bytes.ReplaceAll(plan.After, []byte("dark"), []byte("light"))
			changed = bytes.ReplaceAll(changed, []byte("true"), []byte("false"))
			_ = os.WriteFile(config, changed, 0600)
			if _, e = Undo(config, "s", false); e != nil {
				t.Fatal(e)
			}
			got, _ := os.ReadFile(config)
			want := strings.ReplaceAll(strings.ReplaceAll(original, "dark", "light"), "true", "false")
			if format == "toml" {
				want = strings.ReplaceAll(want, "command = 'node'", `command = "node"`)
			}
			if string(got) != want {
				t.Fatalf("unexpected undo:\n%s\nwant:\n%s", got, want)
			}
			plan, e = Prepare(config, format, "s", policy, "/absolute/warden", "auto")
			if e != nil {
				t.Fatal(e)
			}
			if _, e = Apply(plan); e != nil {
				t.Fatal(e)
			}
			conflict := bytes.ReplaceAll(plan.After, []byte("/absolute/warden"), []byte("/another/launcher"))
			_ = os.WriteFile(config, conflict, 0600)
			if _, e = Undo(config, "s", false); e == nil {
				t.Fatal("launcher conflict was overwritten")
			}
			got, _ = os.ReadFile(config)
			if !bytes.Equal(got, conflict) {
				t.Fatal("conflicted config mutated")
			}
		})
	}
}

func TestSafeFailuresAndDrift(t *testing.T) {
	t.Setenv("WARDEN_SETUP_STATE_DIR", filepath.Join(t.TempDir(), "state"))
	for _, config := range []string{`{bad}`, `{"mcpServers":{"s":{"command":"a","command":"b"}}}`, `{"mcpServers":{"s":{"url":"https://example.com"}}}`, `{"mcpServers":{"s":{"command":"node","args":null}}}`} {
		d, e := Parse([]byte(config), "json")
		if e == nil {
			_, e = d.Launcher("s")
		}
		if e == nil {
			t.Fatalf("accepted invalid/remote launcher %s", config)
		}
	}
	dir := t.TempDir()
	config := filepath.Join(dir, "config")
	policy := filepath.Join(dir, "policy")
	_ = os.WriteFile(config, []byte(`{"mcpServers":{"s":{"command":"node","args":[]}}}`), 0600)
	_ = os.WriteFile(policy, []byte("filesystem: {}"), 0600)
	plan, e := Prepare(config, "json", "s", policy, "/absolute/warden", "auto")
	if e != nil {
		t.Fatal(e)
	}
	_ = os.WriteFile(config, append(plan.Before, ' '), 0600)
	if _, e = Apply(plan); e == nil {
		t.Fatal("concurrent edit not detected")
	}
	_ = os.WriteFile(config, plan.Before, 0600)
	if _, e = Apply(plan); e != nil {
		t.Fatal(e)
	}
	_ = os.WriteFile(policy, []byte("filesystem: {read: ['/data']}"), 0600)
	if _, e = Prepare(config, "json", "s", policy, "/absolute/warden", "auto"); e == nil {
		t.Fatal("policy expansion silently accepted")
	}
	entries, e := Inventory(config, "json")
	if e != nil || entries[0].State != "policy drift" {
		t.Fatalf("drift: %v %v", entries, e)
	}
}

func TestSymlinkAndLockRefusal(t *testing.T) {
	t.Setenv("WARDEN_SETUP_STATE_DIR", filepath.Join(t.TempDir(), "state"))
	dir := t.TempDir()
	config := filepath.Join(dir, "config")
	policy := filepath.Join(dir, "policy")
	_ = os.WriteFile(config, []byte(`{"servers":{"s":{"command":"node"}}}`), 0600)
	_ = os.WriteFile(policy, []byte("filesystem: {}"), 0600)
	link := filepath.Join(dir, "link")
	if e := os.Symlink(config, link); e != nil {
		t.Skip(e)
	}
	if _, e := Prepare(link, "json", "s", policy, "/warden", "auto"); e == nil {
		t.Fatal("followed config symlink")
	}
	plan, e := Prepare(config, "json", "s", policy, "/warden", "auto")
	if e != nil {
		t.Fatal(e)
	}
	_ = os.WriteFile(config+".warden-lock", []byte{}, 0600)
	if _, e = Apply(plan); e == nil {
		t.Fatal("ignored lock")
	}
}

func TestOptInPreparedCommandAndUndo(t *testing.T) {
	t.Setenv("WARDEN_SETUP_STATE_DIR", filepath.Join(t.TempDir(), "state"))
	dir := t.TempDir()
	config := filepath.Join(dir, "mcp.json")
	policy := filepath.Join(dir, "policy.yaml")
	original := `{"mcpServers":{"s":{"command":"npx","args":["old-package"],"env":{"KEY":"${KEY}"}}}}`
	_ = os.WriteFile(config, []byte(original), 0600)
	_ = os.WriteFile(policy, []byte("filesystem: {}"), 0600)
	upstream := []string{"/prepared/node", "/prepared/server.js", "/selected/data"}
	plan, e := PrepareWithCommand(config, "json", "s", policy, "/warden", "auto", upstream)
	if e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(string(plan.After), "/prepared/server.js") {
		t.Fatal("prepared argv not used")
	}
	if _, e = Apply(plan); e != nil {
		t.Fatal(e)
	}
	if _, e = Undo(config, "s", false); e != nil {
		t.Fatal(e)
	}
	b, _ := os.ReadFile(config)
	if string(b) != original {
		t.Fatal("original downloader argv not restored")
	}
}

func TestStateDirectoryResolvesAncestorSymlinks(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	_ = os.Mkdir(target, 0700)
	link := filepath.Join(dir, "alias")
	if e := os.Symlink(target, link); e != nil {
		t.Skip(e)
	}
	t.Setenv("WARDEN_SETUP_STATE_DIR", filepath.Join(link, "new-state"))
	got := StateDirectory(filepath.Join(dir, "config"))
	// StateDirectory resolves ancestor symlinks, so the expected prefix must
	// be canonical too: on macOS the temp root itself resolves
	// /var -> /private/var and the literal target path would never match.
	want := filepath.Join(target, "new-state")
	if real, e := filepath.EvalSymlinks(target); e == nil {
		want = filepath.Join(real, "new-state")
	}
	if !strings.HasPrefix(got, want) {
		t.Fatalf("state alias not canonicalized: %s (want prefix %s)", got, want)
	}
	// The alias itself must be gone from the resolved path.
	if strings.Contains(got, string(filepath.Separator)+"alias"+string(filepath.Separator)) {
		t.Fatalf("state directory still routes through the symlink alias: %s", got)
	}
}
