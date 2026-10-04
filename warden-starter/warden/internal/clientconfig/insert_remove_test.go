package clientconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const insertJSONFixture = `{
  "mcpServers": {
    "existing": {"command": "node", "args": ["old.js"], "env": {"TOKEN": "${TOKEN}"}}
  },
  "preferences": {"theme": "dark"}
}`

const insertTOMLFixture = "# retain comment\nmodel = 'example'\n[mcp_servers.existing]\ncommand = 'node' # launcher\nargs = ['old.js'] # retain argv\nstartup_timeout_sec = 30\n[mcp_servers.existing.env]\nTOKEN = '${TOKEN}'\n"

const insertJSONCFixture = "{\n// retain this comment\n\"servers\": {\"existing\": {\"command\": \"node\" /* launcher */, \"args\": [\"old.js\"],}},\n}"

func insertedLauncher() Launcher {
	return Launcher{Command: "/absolute/warden", Args: []string{"run", "--policy", "/p.yaml", "--policy-sha256", "abc", "--backend", "auto", "--", "/usr/bin/node", "/r/dist/index.js"}, HadArgs: true}
}

func TestInsertAndRemoveJSON(t *testing.T) {
	d, err := Parse([]byte(insertJSONFixture), "json")
	if err != nil {
		t.Fatal(err)
	}
	l := insertedLauncher()
	after, err := d.Insert("added", l)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(after), `"theme": "dark"`) {
		t.Fatal("unrelated field lost")
	}
	if !strings.Contains(string(after), `"TOKEN": "${TOKEN}"`) {
		t.Fatal("existing entry settings lost")
	}
	parsed, err := Parse(after, "json")
	if err != nil {
		t.Fatal(err)
	}
	got, err := parsed.Launcher("added")
	if err != nil || !Equal(got, l) {
		t.Fatalf("inserted launcher mismatch: %v %v", got, err)
	}
	if _, err := parsed.Launcher("existing"); err != nil {
		t.Fatalf("existing entry lost: %v", err)
	}
	// Duplicate insert is refused.
	if _, err := d.Insert("existing", l); err == nil {
		t.Fatal("duplicate insert accepted")
	}
	// Remove restores a parseable config without the entry.
	afterRemove, err := parsed.Remove("added")
	if err != nil {
		t.Fatal(err)
	}
	final, err := Parse(afterRemove, "json")
	if err != nil {
		t.Fatalf("remove produced invalid JSON: %v", err)
	}
	if _, err := final.Launcher("added"); err == nil {
		t.Fatal("removed entry still present")
	}
	if _, err := final.Launcher("existing"); err != nil {
		t.Fatal("existing entry lost after remove")
	}
	if !strings.Contains(string(afterRemove), `"theme": "dark"`) {
		t.Fatal("unrelated field lost after remove")
	}
}

func TestInsertAndRemoveJSONC(t *testing.T) {
	d, err := Parse([]byte(insertJSONCFixture), "jsonc")
	if err != nil {
		t.Fatal(err)
	}
	after, err := d.Insert("added", insertedLauncher())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(after), "retain this comment") || !strings.Contains(string(after), "/* launcher */") {
		t.Fatal("JSONC comments lost")
	}
	parsed, err := Parse(after, "jsonc")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := parsed.Launcher("added"); err != nil {
		t.Fatalf("inserted launcher missing: %v", err)
	}
	afterRemove, err := parsed.Remove("added")
	if err != nil {
		t.Fatal(err)
	}
	final, err := Parse(afterRemove, "jsonc")
	if err != nil {
		t.Fatalf("remove produced invalid JSONC: %v", err)
	}
	if !strings.Contains(string(afterRemove), "retain this comment") {
		t.Fatal("JSONC comment lost after remove")
	}
	if _, err := final.Launcher("existing"); err != nil {
		t.Fatal("existing entry lost after remove")
	}
}

func TestInsertAndRemoveTOML(t *testing.T) {
	d, err := Parse([]byte(insertTOMLFixture), "toml")
	if err != nil {
		t.Fatal(err)
	}
	after, err := d.Insert("added", insertedLauncher())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(after), "# retain comment") || !strings.Contains(string(after), "# retain argv") {
		t.Fatal("TOML comments lost")
	}
	if !strings.Contains(string(after), "TOKEN = '${TOKEN}'") {
		t.Fatal("existing env table lost")
	}
	parsed, err := Parse(after, "toml")
	if err != nil {
		t.Fatal(err)
	}
	got, err := parsed.Launcher("added")
	if err != nil || !Equal(got, insertedLauncher()) {
		t.Fatalf("inserted launcher mismatch: %v %v", got, err)
	}
	afterRemove, err := parsed.Remove("added")
	if err != nil {
		t.Fatal(err)
	}
	final, err := Parse(afterRemove, "toml")
	if err != nil {
		t.Fatalf("remove produced invalid TOML: %v", err)
	}
	if _, err := final.Launcher("added"); err == nil {
		t.Fatal("removed entry still present")
	}
	if _, err := final.Launcher("existing"); err != nil {
		t.Fatal("existing entry lost after remove")
	}
	if !strings.Contains(string(afterRemove), "# retain comment") || !strings.Contains(string(afterRemove), "TOKEN = '${TOKEN}'") {
		t.Fatal("unrelated content lost after remove")
	}
}

func TestInsertLastEntryJSON(t *testing.T) {
	// Removing the only member must leave valid JSON with no dangling comma.
	d, err := Parse([]byte(`{"mcpServers":{"only":{"command":"node"}}}`), "json")
	if err != nil {
		t.Fatal(err)
	}
	after, err := d.Remove("only")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Parse(after, "json"); err != nil {
		t.Fatalf("empty section invalid: %v (%s)", err, after)
	}
}

func TestValidEntryName(t *testing.T) {
	for _, ok := range []string{"slack", "My-Server_2", "a"} {
		if err := ValidEntryName(ok); err != nil {
			t.Fatalf("%q rejected: %v", ok, err)
		}
	}
	for _, bad := range []string{"", "with space", "dot.name", strings.Repeat("x", 65)} {
		if err := ValidEntryName(bad); err == nil {
			t.Fatalf("%q accepted", bad)
		}
	}
}

func TestPrepareCreateApplyUndo(t *testing.T) {
	t.Setenv("WARDEN_SETUP_STATE_DIR", filepath.Join(t.TempDir(), "state"))
	dir := t.TempDir()
	config := filepath.Join(dir, "config")
	policyPath := filepath.Join(dir, "policy.yaml")
	if err := os.WriteFile(config, []byte(insertJSONFixture), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(policyPath, []byte("filesystem: {}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	plan, err := PrepareCreate(config, "json", "added", policyPath, "/absolute/warden", "auto", []string{"/usr/bin/node", "/r/dist/index.js"})
	if err != nil {
		t.Fatal(err)
	}
	if !plan.EntryCreated || plan.ConfigMissing {
		t.Fatal("unexpected plan flags for existing config")
	}
	if _, err := PrepareCreate(config, "json", "existing", policyPath, "/absolute/warden", "auto", nil); err == nil {
		t.Fatal("existing entry not refused")
	}
	if _, err := Apply(plan); err != nil {
		t.Fatal(err)
	}
	current, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	d, err := Parse(current, "json")
	if err != nil {
		t.Fatal(err)
	}
	got, err := d.Launcher("added")
	if err != nil || got.Command != "/absolute/warden" {
		t.Fatalf("applied launcher wrong: %v %v", got, err)
	}
	changed, err := Undo(config, "added", false)
	if err != nil || !changed {
		t.Fatalf("undo: %v %v", changed, err)
	}
	after, _ := os.ReadFile(config)
	if string(after) != insertJSONFixture {
		t.Fatalf("undo did not restore config:\n%s", after)
	}
	again, err := Undo(config, "added", false)
	if err != nil || again {
		t.Fatalf("second undo should be a no-op: %v %v", again, err)
	}
}

func TestPrepareCreateMissingConfig(t *testing.T) {
	t.Setenv("WARDEN_SETUP_STATE_DIR", filepath.Join(t.TempDir(), "state"))
	for _, format := range []string{"json", "toml"} {
		t.Run(format, func(t *testing.T) {
			dir := t.TempDir()
			config := filepath.Join(dir, "nested", "config")
			policyPath := filepath.Join(dir, "policy.yaml")
			if err := os.WriteFile(policyPath, []byte("filesystem: {}\n"), 0600); err != nil {
				t.Fatal(err)
			}
			plan, err := PrepareCreate(config, format, "added", policyPath, "/absolute/warden", "auto", []string{"/usr/bin/node", "/r/index.js"})
			if err != nil {
				t.Fatal(err)
			}
			if !plan.ConfigMissing {
				t.Fatal("ConfigMissing not set")
			}
			if _, err := Apply(plan); err != nil {
				t.Fatal(err)
			}
			current, err := os.ReadFile(config)
			if err != nil {
				t.Fatal(err)
			}
			d, err := Parse(current, format)
			if err != nil {
				t.Fatalf("created config invalid: %v", err)
			}
			if _, err := d.Launcher("added"); err != nil {
				t.Fatalf("created entry missing: %v", err)
			}
			changed, err := Undo(config, "added", false)
			if err != nil || !changed {
				t.Fatalf("undo: %v %v", changed, err)
			}
			if _, err := os.Stat(config); !os.IsNotExist(err) {
				t.Fatal("created config was not removed on undo")
			}
		})
	}
}
