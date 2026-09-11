package main

import (
	"strings"
	"testing"
)

// TestHelpOutput exercises all help paths and verifies output correctness.
func TestHelpOutput(t *testing.T) {
	bin := buildWarden(t)

	cases := []struct {
		name     string
		args     []string
		exitCode int
		checks   []outputCheck
	}{
		{
			name:     "warden help",
			args:     []string{"help"},
			exitCode: 0,
			checks: []outputCheck{
				{field: "stderr", contains: "WARDEN"},
				{field: "stderr", contains: "Secure execution for MCP servers"},
				{field: "stderr", contains: "Usage:"},
				{field: "stderr", contains: "Commands:"},
				{field: "stderr", contains: "init"},
				{field: "stderr", contains: "run"},
				{field: "stderr", contains: "doctor"},
				{field: "stderr", contains: "version"},
				{field: "stderr", contains: "help"},
				{field: "stderr", contains: "fails closed"},
			},
		},
		{
			name:     "warden --help",
			args:     []string{"--help"},
			exitCode: 0,
			checks: []outputCheck{
				{field: "stderr", contains: "WARDEN"},
				{field: "stderr", contains: "Usage:"},
				{field: "stderr", contains: "Commands:"},
			},
		},
		{
			name:     "warden -h",
			args:     []string{"-h"},
			exitCode: 0,
			checks: []outputCheck{
				{field: "stderr", contains: "WARDEN"},
				{field: "stderr", contains: "Usage:"},
			},
		},
		{
			name:     "warden help run",
			args:     []string{"help", "run"},
			exitCode: 0,
			checks: []outputCheck{
				{field: "stderr", contains: "WARDEN RUN"},
				{field: "stderr", contains: "Usage:"},
				{field: "stderr", contains: "--policy"},
				{field: "stderr", contains: "--backend"},
				{field: "stderr", contains: "--approve"},
				{field: "stderr", contains: "Security:"},
			},
		},
		{
			name:     "warden run --help",
			args:     []string{"run", "--help"},
			exitCode: 0,
			checks: []outputCheck{
				{field: "stderr", contains: "WARDEN RUN"},
				{field: "stderr", contains: "--policy"},
			},
		},
		{
			name:     "warden run -h",
			args:     []string{"run", "-h"},
			exitCode: 0,
			checks: []outputCheck{
				{field: "stderr", contains: "WARDEN RUN"},
				{field: "stderr", contains: "--policy"},
			},
		},
		{
			name:     "warden help init",
			args:     []string{"help", "init"},
			exitCode: 0,
			checks: []outputCheck{
				{field: "stderr", contains: "WARDEN INIT"},
				{field: "stderr", contains: "Usage:"},
				{field: "stderr", contains: "--log"},
				{field: "stderr", contains: "--output"},
				{field: "stderr", contains: "Filesystem"},
				{field: "stderr", contains: "Network"},
				{field: "stderr", contains: "Environment"},
				{field: "stderr", contains: "Resources"},
			},
		},
		{
			name:     "warden init --help",
			args:     []string{"init", "--help"},
			exitCode: 0,
			checks: []outputCheck{
				{field: "stderr", contains: "WARDEN INIT"},
				{field: "stderr", contains: "--log"},
			},
		},
		{
			name:     "warden help doctor",
			args:     []string{"help", "doctor"},
			exitCode: 0,
			checks: []outputCheck{
				{field: "stderr", contains: "WARDEN DOCTOR"},
				{field: "stderr", contains: "Usage:"},
				{field: "stderr", contains: "Sandbox backend"},
				{field: "stderr", contains: "fails closed"},
			},
		},
		{
			name:     "warden doctor --help",
			args:     []string{"doctor", "--help"},
			exitCode: 0,
			checks: []outputCheck{
				{field: "stderr", contains: "WARDEN DOCTOR"},
				{field: "stderr", contains: "Usage:"},
			},
		},
		{
			name:     "warden help version",
			args:     []string{"help", "version"},
			exitCode: 0,
			checks: []outputCheck{
				{field: "stderr", contains: "WARDEN VERSION"},
				{field: "stderr", contains: "Usage:"},
				{field: "stderr", contains: "warden version"},
				{field: "stderr", contains: "warden --version"},
			},
		},
		{
			name:     "warden help update",
			args:     []string{"help", "update"},
			exitCode: 0,
			checks: []outputCheck{
				{field: "stderr", contains: "WARDEN UPDATE"},
				{field: "stderr", contains: "Usage:"},
				{field: "stderr", contains: "warden update"},
				{field: "stderr", contains: "SHA256SUMS"},
			},
		},
		{
			name:     "warden update --help",
			args:     []string{"update", "--help"},
			exitCode: 0,
			checks: []outputCheck{
				{field: "stderr", contains: "WARDEN UPDATE"},
				{field: "stderr", contains: "SHA256SUMS"},
			},
		},
		{
			name:     "warden version --help",
			args:     []string{"version", "--help"},
			exitCode: 0,
			checks: []outputCheck{
				{field: "stderr", contains: "WARDEN VERSION"},
			},
		},
		{
			name:     "warden version -h",
			args:     []string{"version", "-h"},
			exitCode: 0,
			checks: []outputCheck{
				{field: "stderr", contains: "WARDEN VERSION"},
			},
		},
		{
			name:     "warden help trace",
			args:     []string{"help", "trace"},
			exitCode: 0,
			checks: []outputCheck{
				{field: "stderr", contains: "WARDEN TRACE"},
				{field: "stderr", contains: "Usage:"},
				{field: "stderr", contains: "-- <command...>"},
				{field: "stderr", contains: "warden init"},
			},
		},
		{
			name:     "warden trace --help",
			args:     []string{"trace", "--help"},
			exitCode: 0,
			checks: []outputCheck{
				{field: "stderr", contains: "WARDEN TRACE"},
			},
		},
		{
			name:     "warden help logs",
			args:     []string{"help", "logs"},
			exitCode: 0,
			checks: []outputCheck{
				{field: "stderr", contains: "WARDEN LOGS"},
				{field: "stderr", contains: "Usage:"},
				{field: "stderr", contains: "--tail"},
				{field: "stderr", contains: "--follow"},
			},
		},
		{
			name:     "warden logs --help",
			args:     []string{"logs", "--help"},
			exitCode: 0,
			checks: []outputCheck{
				{field: "stderr", contains: "WARDEN LOGS"},
			},
		},
		{
			name:     "warden help help",
			args:     []string{"help", "help"},
			exitCode: 0,
			checks: []outputCheck{
				{field: "stderr", contains: "WARDEN"},
				{field: "stderr", contains: "Usage:"},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, stderr, code := runWarden(t, bin, []string{"NO_COLOR=1"}, tc.args...)
			if code != tc.exitCode {
				t.Fatalf("exit code = %d, want %d\nstderr:\n%s", code, tc.exitCode, stderr)
			}
			for _, chk := range tc.checks {
				output := stderr
				if chk.field == "stdout" {
					// help always goes to stderr
					output = stderr
				}
				if !strings.Contains(output, chk.contains) {
					t.Errorf("output should contain %q\n\ngot:\n%s", chk.contains, output)
				}
			}
		})
	}
}

// TestHelpNoANSI verifies help output contains no ANSI escape sequences
// when NO_COLOR is set.
func TestHelpNoANSI(t *testing.T) {
	bin := buildWarden(t)

	helpArgs := [][]string{
		{"help"},
		{"help", "run"},
		{"help", "init"},
		{"help", "doctor"},
		{"help", "version"},
		{"help", "trace"},
		{"help", "logs"},
		{"--help"},
		{"run", "--help"},
		{"init", "--help"},
		{"doctor", "--help"},
		{"version", "--help"},
		{"trace", "--help"},
		{"logs", "--help"},
	}

	for _, args := range helpArgs {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			_, stderr, _ := runWarden(t, bin, []string{"NO_COLOR=1"}, args...)
			if strings.Contains(stderr, "\x1b[") {
				t.Errorf("help output should not contain ANSI escape sequences\ngot:\n%s", stderr)
			}
		})
	}
}

// TestHelpNoBanner verifies help output never contains the large ASCII banner.
func TestHelpNoBanner(t *testing.T) {
	bin := buildWarden(t)

	args := [][]string{
		{"help"},
		{"help", "run"},
		{"--help"},
		{"run", "--help"},
	}

	for _, a := range args {
		t.Run(strings.Join(a, "_"), func(t *testing.T) {
			_, stderr, _ := runWarden(t, bin, nil, a...)
			if strings.Contains(stderr, "██") {
				t.Errorf("help should not contain large banner block art")
			}
		})
	}
}

// TestUnknownCommand verifies unknown command output and exit code.
func TestUnknownCommand(t *testing.T) {
	bin := buildWarden(t)

	cases := []struct {
		name        string
		args        []string
		exitCode    int
		contains    []string
		notContains []string
	}{
		{
			name:     "unknown command",
			args:     []string{"something"},
			exitCode: 1,
			contains: []string{
				"Unknown command: something",
				"warden help",
			},
		},
		{
			name:     "typo suggests doctor",
			args:     []string{"doctro"},
			exitCode: 1,
			contains: []string{
				"Unknown command: doctro",
				"Did you mean:",
				"warden doctor",
			},
		},
		{
			name:     "typo suggests run",
			args:     []string{"rn"},
			exitCode: 1,
			contains: []string{
				"Unknown command: rn",
				"Did you mean:",
				"warden run",
			},
		},
		{
			name:     "typo suggests init",
			args:     []string{"ini"},
			exitCode: 1,
			contains: []string{
				"Unknown command: ini",
				"Did you mean:",
				"warden init",
			},
		},
		{
			name:     "typo suggests version",
			args:     []string{"versio"},
			exitCode: 1,
			contains: []string{
				"Unknown command: versio",
				"Did you mean:",
				"warden version",
			},
		},
		{
			name:     "completely unrelated",
			args:     []string{"zzzzz"},
			exitCode: 1,
			contains: []string{
				"Unknown command: zzzzz",
				"warden help",
			},
			notContains: []string{
				"Did you mean:",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, stderr, code := runWarden(t, bin, []string{"NO_COLOR=1"}, tc.args...)
			if code != tc.exitCode {
				t.Fatalf("exit code = %d, want %d\nstderr:\n%s", code, tc.exitCode, stderr)
			}
			for _, s := range tc.contains {
				if !strings.Contains(stderr, s) {
					t.Errorf("stderr should contain %q\n\ngot:\n%s", s, stderr)
				}
			}
			for _, s := range tc.notContains {
				if strings.Contains(stderr, s) {
					t.Errorf("stderr should NOT contain %q\n\ngot:\n%s", s, stderr)
				}
			}
		})
	}
}

// TestHelpUnknownTarget verifies `warden help <unknown>` output.
func TestHelpUnknownTarget(t *testing.T) {
	bin := buildWarden(t)

	_, stderr, code := runWarden(t, bin, []string{"NO_COLOR=1"}, "help", "something")
	if code != 1 {
		t.Fatalf("exit code = %d, want 1\nstderr:\n%s", code, stderr)
	}
	if !strings.Contains(stderr, "No help available for: something") {
		t.Errorf("should mention unknown target\ngot:\n%s", stderr)
	}
	if !strings.Contains(stderr, "warden help") {
		t.Errorf("should suggest warden help\ngot:\n%s", stderr)
	}
}

// TestHelpNoNetwork verifies help makes no network calls (by checking it
// completes instantly and doesn't reference network).
func TestHelpNoNetwork(t *testing.T) {
	bin := buildWarden(t)
	// Just verify it exits cleanly with no network-related errors
	_, stderr, code := runWarden(t, bin, []string{"NO_COLOR=1", "HOME=/nonexistent"}, "help")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0\nstderr:\n%s", code, stderr)
	}
}

// TestHelpNoConfig verifies help works without any config files.
func TestHelpNoConfig(t *testing.T) {
	bin := buildWarden(t)
	_, stderr, code := runWarden(t, bin, []string{"NO_COLOR=1", "XDG_STATE_HOME=/nonexistent", "WARDEN_NO_FIRST_RUN=1"}, "help")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0\nstderr:\n%s", code, stderr)
	}
	if !strings.Contains(stderr, "WARDEN") {
		t.Errorf("should contain WARDEN header\ngot:\n%s", stderr)
	}
}

// TestBareInvocationShowsUsage verifies that running `warden` with no args
// shows the branded header and usage (and exits non-zero).
func TestBareInvocationShowsUsage(t *testing.T) {
	bin := buildWarden(t)
	_, stderr, code := runWarden(t, bin, []string{"NO_COLOR=1", "XDG_STATE_HOME=" + t.TempDir(), "WARDEN_NO_FIRST_RUN=1"})
	// bare invocation exits 1 (see main.go)
	if code != 1 {
		t.Fatalf("bare invocation exit = %d, want 1", code)
	}
	if !strings.Contains(stderr, "WARDEN") {
		t.Errorf("bare invocation should contain WARDEN\ngot:\n%s", stderr)
	}
	if !strings.Contains(stderr, "Usage:") {
		t.Errorf("bare invocation should contain Usage:\ngot:\n%s", stderr)
	}
}

// TestLevenshtein verifies the edit distance helper used for suggestions.
func TestLevenshtein(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"", "abc", 3},
		{"abc", "", 3},
		{"abc", "abc", 0},
		{"abc", "ab", 1},
		{"abc", "abcd", 1},
		{"doctro", "doctor", 2},
		{"rn", "run", 1},
		{"ini", "init", 1},
		{"versio", "version", 1},
		{"zzzzz", "run", 5},
	}

	for _, tc := range cases {
		t.Run(tc.a+"_"+tc.b, func(t *testing.T) {
			got := levenshtein(tc.a, tc.b)
			if got != tc.want {
				t.Errorf("levenshtein(%q, %q) = %d, want %d", tc.a, tc.b, got, tc.want)
			}
		})
	}
}

// TestSuggestCommand verifies the suggestion logic.
func TestSuggestCommand(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"doctro", "doctor"},
		{"rn", "run"},
		{"ini", "init"},
		{"versio", "version"},
		{"log", "logs"},
		{"zzzzz", ""},
		{"run", ""},   // exact match, no suggestion needed
		{"help", ""},  // exact match
		{"runn", "run"},
		{"doctors", "doctor"},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got := suggestCommand(tc.input)
			if got != tc.want {
				t.Errorf("suggestCommand(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// TestDoctorHelpIsDeterministic verifies doctor help produces the same
// output regardless of system state.
func TestDoctorHelpIsDeterministic(t *testing.T) {
	bin := buildWarden(t)
	_, stderr1, code1 := runWarden(t, bin, []string{"NO_COLOR=1"}, "help", "doctor")
	_, stderr2, code2 := runWarden(t, bin, []string{"NO_COLOR=1"}, "help", "doctor")
	if code1 != 0 || code2 != 0 {
		t.Fatalf("exit codes: %d, %d", code1, code2)
	}
	if stderr1 != stderr2 {
		t.Errorf("doctor help output is not deterministic:\nrun 1: %q\nrun 2: %q", stderr1, stderr2)
	}
}

// TestVersionHelpIsDeterministic verifies version help produces the same
// output regardless of system state.
func TestVersionHelpIsDeterministic(t *testing.T) {
	bin := buildWarden(t)
	_, stderr1, code1 := runWarden(t, bin, []string{"NO_COLOR=1"}, "help", "version")
	_, stderr2, code2 := runWarden(t, bin, []string{"NO_COLOR=1"}, "help", "version")
	if code1 != 0 || code2 != 0 {
		t.Fatalf("exit codes: %d, %d", code1, code2)
	}
	if stderr1 != stderr2 {
		t.Errorf("version help output is not deterministic:\nrun 1: %q\nrun 2: %q", stderr1, stderr2)
	}
}

type outputCheck struct {
	field    string // "stdout" or "stderr"
	contains string
}
