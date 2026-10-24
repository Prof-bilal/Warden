package main

import "testing"

func TestHostNoPort(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"example.test", "example.test"},
		{"example.test:8080", "example.test"},
		{"Example.Test.:443", "example.test"},
		{"[2001:db8::1]:443", "2001:db8::1"},
		{"2001:db8::1", "2001:db8::1"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := hostNoPort(tt.in); got != tt.want {
			t.Errorf("hostNoPort(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestKillFailClosed(t *testing.T) {
	before := map[string]any{"action": "signal-parent", "phase": "before", "signal": float64(15)}
	killed := map[string]any{"action": "kill-parent", "phase": "signal", "signal": float64(9)}
	parentGone := map[string]any{"action": "signal-parent", "phase": "after-parent", "parent_alive": false}
	parentAlive := map[string]any{"action": "signal-parent", "phase": "after-parent", "parent_alive": true}
	postDenied := map[string]any{"action": "signal-parent", "phase": "after", "ok": false, "error_class": "connection_refused"}
	postAllowed := map[string]any{"action": "signal-parent", "phase": "after", "ok": true, "status": float64(200)}

	tests := []struct {
		name     string
		events   []map[string]any
		exit     int
		wantOK   bool
		wantWarn bool
	}{
		{
			name:   "pre-signal phase plus non-zero exit is fail-closed (Docker SIGTERM teardown)",
			events: []map[string]any{before},
			exit:   255,
			wantOK: true,
		},
		{
			name:     "pre-signal phase plus zero exit is inconclusive",
			events:   []map[string]any{before},
			exit:     0,
			wantOK:   false,
			wantWarn: true,
		},
		{
			name:   "signal phase plus non-zero exit is fail-closed",
			events: []map[string]any{before, killed},
			exit:   137,
			wantOK: true,
		},
		{
			name:   "dead parent with no post-death success is fail-closed",
			events: []map[string]any{before, killed, parentGone, postDenied},
			exit:   0,
			wantOK: true,
		},
		{
			name:   "dead parent with post-death success is a failure",
			events: []map[string]any{before, killed, parentGone, postAllowed},
			exit:   0,
			wantOK: false,
		},
		{
			name:     "surviving parent is only a warning",
			events:   []map[string]any{before, killed, parentAlive},
			exit:     0,
			wantOK:   false,
			wantWarn: true,
		},
		{
			name:   "no signal attempt is a hard failure",
			events: nil,
			exit:   1,
			wantOK: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ev := &caseEvidence{steps: []stepEvidence{{Name: "signal", ExitCode: tt.exit, Events: tt.events}}}
			ok, warn, detail := killFailClosed(ev)
			if ok != tt.wantOK || warn != tt.wantWarn {
				t.Fatalf("killFailClosed() = (ok=%v, warn=%v, %q), want (ok=%v, warn=%v)", ok, warn, detail, tt.wantOK, tt.wantWarn)
			}
		})
	}
}
