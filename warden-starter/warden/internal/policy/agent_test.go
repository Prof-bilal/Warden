package policy

import (
	"strings"
	"testing"
)

func TestAgentPolicyValidation(t *testing.T) {
	valid := "filesystem:\n  write: ['./project']\nnetwork:\n  allow: [api.example.com]\nagent:\n  workspace: ./project\n  redaction:\n    apply: review\n    files:\n      - path: config.json\n        format: json\n        fields: ['/auth/key']\n  secret_exec:\n    profiles:\n      check:\n        url: https://api.example.com/status\n        env: API_KEY\n"
	if _, err := Load(writeTempPolicy(t, valid)); err != nil {
		t.Fatal(err)
	}
	for _, pair := range [][2]string{
		{"apply: review", "apply: automatic"},
		{"path: config.json", "path: ../config.json"},
		{"format: json", "format: binary"},
		{"'/auth/key'", "'auth.key'"},
		{"https://api.example.com/status", "http://api.example.com/status"},
		{"https://api.example.com/status", "https://api.example.com/status?key=secret"},
		{"https://api.example.com/status", "https://other.example.com/status"},
		{"write: ['./project']", "write: ['./project', './other']"},
	} {
		p := strings.Replace(valid, pair[0], pair[1], 1)
		if _, err := Load(writeTempPolicy(t, p)); err == nil {
			t.Fatalf("invalid policy accepted: %s", pair[1])
		}
	}
}
