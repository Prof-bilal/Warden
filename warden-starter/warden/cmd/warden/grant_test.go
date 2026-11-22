package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/warden-sandbox/warden/internal/approve"
	"github.com/warden-sandbox/warden/internal/audit"
	"github.com/warden-sandbox/warden/internal/policy"
)

func TestClassifyGrant(t *testing.T) {
	for _, tc := range []struct {
		kind, value string
		hosts       []string
		review      bool
	}{
		{"write", "report.txt", nil, true},
		{"read", filepath.Join("project", ".ssh", "config"), nil, true},
		{"read", "report.txt", nil, false},
		{"env", "my_token", nil, true},
		{"env", "LANG", nil, false},
		{"host", "login.example.org", nil, true},
		{"host", "*.example.org", nil, true},
		{"host", "api.github.com", nil, true},
		{"host", "docs.example.org", []string{}, true},
		{"host", "docs.example.org", []string{"docs.example.org"}, false},
		{"unconfined", "", nil, true},
		{"disable_protect_secrets", "", nil, true},
		{"widen_upgrade", "", nil, true},
	} {
		if reason := classifyGrant(tc.kind, tc.value, tc.hosts); (reason != "") != tc.review {
			t.Errorf("classify %s %q = %q, requires review = %t", tc.kind, tc.value, reason, tc.review)
		}
	}
}

func TestReviewedGrantRefusesChangedPolicy(t *testing.T) {
	path := filepath.Join(t.TempDir(), "policy.yaml")
	if err := os.WriteFile(path, []byte("filesystem: {}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	_, digest, err := policy.LoadVerifiedWithDigest(path, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := approve.SaveReviewedGrant(path, digest, "docs.example.org", true); err != nil {
		t.Fatal(err)
	}
	if err := approve.SaveReviewedGrant(path, digest, "other.example.org", true); err == nil {
		t.Fatal("policy changed since review but grant was accepted")
	}
	p, err := policy.Load(path)
	if err != nil || len(p.Network.Allow) != 1 || p.Network.Allow[0] != "docs.example.org" {
		t.Fatalf("unexpected grants %v, error %v", p.Network.Allow, err)
	}
}

func TestExplainDeniedEvents(t *testing.T) {
	var out bytes.Buffer
	err := explainDeniedEvents(&out, []audit.Event{
		{Type: "network", Resource: "docs.example.org:443", Reason: "blocked by policy"},
		{Type: "network", Resource: "login.example.org:443"},
		{Type: "network", Resource: "ignored.example.org", Allowed: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	output := out.String()
	if strings.Count(output, "warden grant host") != 1 || !strings.Contains(output, "blocked by policy") ||
		!strings.Contains(output, "Review network.allow directly") || strings.Contains(output, "ignored.example.org") {
		t.Fatalf("unexpected explanation: %s", output)
	}
}
