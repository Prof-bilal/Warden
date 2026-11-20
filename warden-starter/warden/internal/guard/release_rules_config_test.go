package guard

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

func TestReadReleaseRules(t *testing.T) {
	path := filepath.Join(t.TempDir(), "guard.yaml")
	setup := func(body string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	rules, err := ReadReleaseRules(path)
	if err != nil || rules.MinimumAge != 48*time.Hour {
		t.Fatalf("default release rules = %+v, %v", rules, err)
	}
	setup("minimum_release_age: 72h\nstrict: true\nscript_allowlist: ['@example/tool@1.2.0']\n")
	rules, err = ReadReleaseRules(path)
	if err != nil || rules.MinimumAge != 72*time.Hour || !rules.StrictMetadata ||
		!slices.Equal(rules.ScriptAllowlist, []string{"@example/tool@1.2.0"}) {
		t.Fatalf("configured release rules = %+v, %v", rules, err)
	}
	for _, invalid := range []string{
		"minimum_release_age: 0s\n",
		"override_age: true\n",
		"script_allowlist: ['@example/tool@*']\n",
		"script_allowlist: ['@example/tool@latest']\n",
		"strict: true\n---\nstrict: false\n",
	} {
		setup(invalid)
		if _, err := ReadReleaseRules(path); err == nil {
			t.Errorf("expected invalid guard config to fail: %q", invalid)
		}
	}
}
