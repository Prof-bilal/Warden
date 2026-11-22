package main

import (
	"reflect"
	"testing"
)

func TestSplitRunCommand(t *testing.T) {
	flags := []string{"--policy", "policy.yaml", "--approve-timeout", "5s"}
	command := []string{"launcher", "--policy-sha256", "launcher-value", "--approve", "--dry-run"}
	for _, separator := range [][]string{nil, {"--"}} {
		args := append(append(append([]string{}, flags...), separator...), command...)
		gotFlags, gotCommand := splitRunCommand(args)
		if !reflect.DeepEqual(gotFlags, flags) || !reflect.DeepEqual(gotCommand, command) {
			t.Fatalf("split %v into %v and %v", args, gotFlags, gotCommand)
		}
	}
}

func TestParseRunDryRun(t *testing.T) {
	path, _, _, dryRun, err := parseRunArgs([]string{"--policy", "policy.yaml", "--dry-run"})
	if err != nil || !dryRun || path != "policy.yaml" {
		t.Fatalf("parse dry-run = %q, %t, %v", path, dryRun, err)
	}
}

func TestParseRunArgs(t *testing.T) {
	cases := []struct {
		name        string
		args        []string
		wantPolicy  string
		wantBackend string
		wantCmd     []string
		wantErr     bool
	}{
		{
			name:       "policy then command",
			args:       []string{"--policy", "p.yaml", "/usr/bin/node", "server.js"},
			wantPolicy: "p.yaml",
			wantCmd:    []string{"/usr/bin/node", "server.js"},
		},
		{
			name:        "policy and backend",
			args:        []string{"--policy", "p.yaml", "--backend", "docker", "/usr/bin/true"},
			wantPolicy:  "p.yaml",
			wantBackend: "docker",
			wantCmd:     []string{"/usr/bin/true"},
		},
		{
			name:        "backend equals form",
			args:        []string{"--policy=p.yaml", "--backend=seatbelt", "--", "/usr/bin/true"},
			wantPolicy:  "p.yaml",
			wantBackend: "seatbelt",
			wantCmd:     []string{"/usr/bin/true"},
		},
		{
			name:       "policy with equals then command",
			args:       []string{"--policy=p.yaml", "/usr/bin/node"},
			wantPolicy: "p.yaml",
			wantCmd:    []string{"/usr/bin/node"},
		},
		{
			name:       "command after double dash",
			args:       []string{"--policy", "p.yaml", "--", "/usr/bin/node", "--inspect"},
			wantPolicy: "p.yaml",
			wantCmd:    []string{"/usr/bin/node", "--inspect"},
		},
		{
			name:       "no command at all",
			args:       []string{"--policy", "p.yaml"},
			wantPolicy: "p.yaml",
			wantCmd:    nil,
		},
		{
			name:    "implicit policy",
			args:    []string{"/usr/bin/node"},
			wantCmd: []string{"/usr/bin/node"},
		},
		{
			name:    "empty policy",
			args:    []string{"--policy="},
			wantErr: true,
		},
		{
			name:    "missing policy value",
			args:    []string{"--policy"},
			wantErr: true,
		},
		{
			name:    "unknown flag",
			args:    []string{"--bogus", "p.yaml", "/usr/bin/node"},
			wantErr: true,
		},
		{
			name:    "duplicate policy",
			args:    []string{"--policy", "a.yaml", "--policy", "b.yaml"},
			wantErr: true,
		},
		{
			name:    "duplicate backend",
			args:    []string{"--policy", "a.yaml", "--backend", "linux", "--backend", "docker"},
			wantErr: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gotPolicy, gotBackend, gotCmd, _, err := parseRunArgs(c.args)
			if c.wantErr {
				if err == nil {
					t.Fatalf("expected error, got policy=%q backend=%q cmd=%v", gotPolicy, gotBackend, gotCmd)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotPolicy != c.wantPolicy {
				t.Errorf("policy = %q, want %q", gotPolicy, c.wantPolicy)
			}
			if gotBackend != c.wantBackend {
				t.Errorf("backend = %q, want %q", gotBackend, c.wantBackend)
			}
			if !reflect.DeepEqual(gotCmd, c.wantCmd) {
				t.Errorf("cmd = %v, want %v", gotCmd, c.wantCmd)
			}
		})
	}
}

func TestParseM3Args(t *testing.T) {
	log, output, cmd, err := parseInitArgs([]string{"--log", "trace.jsonl", "--output", "starter.yaml", "--", "/usr/bin/node", "server.js"})
	if err != nil || log != "trace.jsonl" || output != "starter.yaml" || !reflect.DeepEqual(cmd, []string{"/usr/bin/node", "server.js"}) {
		t.Fatalf("parseInitArgs = %q, %q, %v, %v", log, output, cmd, err)
	}
	log, tail, follow, _, err := parseLogsArgs([]string{"--log", "audit.jsonl", "--tail", "10", "--follow"})
	if err != nil || log != "audit.jsonl" || tail != 10 || !follow {
		t.Fatalf("parseLogsArgs = %q, %d, %v, %v", log, tail, follow, err)
	}
	if _, _, _, _, err := parseLogsArgs([]string{"--tail", "nope"}); err == nil {
		t.Fatal("expected invalid tail error")
	}
}

func TestLastLines(t *testing.T) {
	if got, want := string(lastLines([]byte("one\ntwo\nthree\n"), 2)), "two\nthree\n"; got != want {
		t.Errorf("lastLines = %q, want %q", got, want)
	}
}
