package guard

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const npmGuardFixture = `{
	"name": "@example/tool",
	"dist-tags": {"latest": "1.2.0"},
	"time": {
		"1.1.0": "2026-11-10T10:00:00.000Z",
		"1.2.0": "2026-11-19T10:00:00.000Z"
	},
	"versions": {
		"1.1.0": {
			"name": "@example/tool", "version": "1.1.0",
			"maintainers": [{"name":"first"}]
		},
		"1.2.0": {
			"name": "@example/tool", "version": "1.2.0",
			"scripts": {"postinstall":"node setup", "test":"node test"},
			"gypfile": true,
			"maintainers": [{"name":"second"}],
			"dist": {"tarball":"https://registry.example/tool.tgz", "integrity":"sha512-test"}
		}
	}
}`

func TestReadNPMRelease(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/@example/tool" || !strings.Contains(r.URL.EscapedPath(), "%2F") {
			t.Errorf("package request path = %q", r.URL.EscapedPath())
		}
		_, _ = io.WriteString(w, npmGuardFixture)
	}))
	defer server.Close()

	now := time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC)
	fresh, err := ReadNPMRelease(context.Background(), server.Client(), server.URL, "@example/tool", "")
	if err != nil {
		t.Fatal(err)
	}
	if fresh.Facts.Version != "1.2.0" || fresh.Tarball != "https://registry.example/tool.tgz" || fresh.Integrity != "sha512-test" {
		t.Fatalf("latest release = %+v", fresh)
	}
	if !fresh.Facts.HasBindingGYP || len(fresh.Facts.InstallScripts) != 1 || fresh.Facts.InstallScripts[0] != "postinstall" {
		t.Fatalf("install facts = %+v", fresh.Facts)
	}
	decision := DecideRelease(now, fresh.Facts, ReleaseRules{})
	if len(decision.Blocks) != 2 || len(decision.Warnings) != 1 {
		t.Fatalf("fresh release decision = %+v", decision)
	}

	old, err := ReadNPMRelease(context.Background(), server.Client(), server.URL, "@example/tool", "1.1.0")
	if err != nil {
		t.Fatal(err)
	}
	if old.Facts.Version != "1.1.0" || len(DecideRelease(now, old.Facts, ReleaseRules{}).Blocks) != 0 {
		t.Fatalf("older release = %+v", old)
	}
}

func TestParseNPMReleaseRejectsMismatchedIdentity(t *testing.T) {
	bad := strings.Replace(npmGuardFixture, `"name": "@example/tool", "version": "1.2.0"`, `"name": "@example/other", "version": "1.2.0"`, 1)
	if _, err := parseNPMRelease([]byte(bad), "@example/tool", "1.2.0"); err == nil {
		t.Fatal("expected mismatched package name to be rejected")
	}
}
