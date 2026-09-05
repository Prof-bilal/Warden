package main

import (
	"reflect"
	"testing"
)

func TestParseRunArgs(t *testing.T) {
	cases := []struct {
		name       string
		args       []string
		wantPolicy string
		wantCmd    []string
		wantErr    bool
	}{
		{
			name:       "policy then command",
			args:       []string{"--policy", "p.yaml", "/usr/bin/node", "server.js"},
			wantPolicy: "p.yaml",
			wantCmd:    []string{"/usr/bin/node", "server.js"},
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
			name:    "missing policy",
			args:    []string{"/usr/bin/node"},
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
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gotPolicy, gotCmd, err := parseRunArgs(c.args)
			if c.wantErr {
				if err == nil {
					t.Fatalf("expected error, got policy=%q cmd=%v", gotPolicy, gotCmd)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotPolicy != c.wantPolicy {
				t.Errorf("policy = %q, want %q", gotPolicy, c.wantPolicy)
			}
			if !reflect.DeepEqual(gotCmd, c.wantCmd) {
				t.Errorf("cmd = %v, want %v", gotCmd, c.wantCmd)
			}
		})
	}
}
