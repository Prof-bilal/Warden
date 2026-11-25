//go:build darwin || windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/warden-sandbox/warden/internal/autopilot"
)

func TestAutopilotNativeReviewedEdits(t *testing.T) {
	for _, test := range []struct {
		name, want        string
		confirm, conflict bool
	}{
		{name: "decline", want: "disposable project"},
		{name: "approve", want: "reviewed change", confirm: true},
		{name: "conflict", want: "host change", confirm: true, conflict: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			readme, stateDir, program := nativeAutopilotFixture(t)
			enrollment, err := autopilot.Enroll(filepath.Dir(readme), program, autopilot.ReviewedEdits)
			if err != nil {
				t.Fatal(err)
			}
			p, err := enrollment.Policy(enrollment.Project)
			if err != nil {
				t.Fatal(err)
			}

			prompts := 0
			confirm := func(prompt, answer string) bool {
				prompts++
				if !strings.Contains(prompt, "reviewed changes") || answer != "apply" {
					t.Fatalf("unexpected review prompt %q (answer %q)", prompt, answer)
				}
				got, err := os.ReadFile(readme)
				if err != nil || string(got) != "disposable project" {
					t.Fatalf("host changed before review: %q, %v", got, err)
				}
				if test.conflict {
					if err := os.WriteFile(readme, []byte("host change"), 0600); err != nil {
						t.Fatal(err)
					}
				}
				return test.confirm
			}
			code, err := runAgentWithReview(p, []string{program, "-test.run=^TestAutopilotNativeEditProbe$"}, confirm)
			if test.conflict {
				if code == 0 || err == nil || !strings.Contains(err.Error(), "host workspace changed") {
					t.Fatalf("expected a host conflict: exit %d, %v", code, err)
				}
			} else if code != 0 || err != nil {
				t.Fatalf("native reviewed run: exit %d, %v", code, err)
			}
			if prompts != 1 {
				t.Fatalf("expected one review prompt, got %d", prompts)
			}
			got, err := os.ReadFile(readme)
			if err != nil || string(got) != test.want {
				t.Fatalf("host contents: %q, %v, want %q", got, err, test.want)
			}
			logs, err := os.ReadFile(filepath.Join(stateDir, "audit.jsonl"))
			wantAudit := fmt.Sprintf(`"action":"apply_review","resource":"workspace","allowed":%t`, test.confirm)
			if err != nil || !strings.Contains(string(logs), wantAudit) {
				t.Fatalf("audit log lacks review decision: %v", err)
			}
		})
	}
}

func TestAutopilotNativeEditProbe(t *testing.T) {
	if len(os.Args) < 2 || os.Args[1] != "-test.run=^TestAutopilotNativeEditProbe$" {
		return
	}
	if err := os.WriteFile("README.md", []byte("reviewed change"), 0600); err != nil {
		t.Fatal(err)
	}
}
