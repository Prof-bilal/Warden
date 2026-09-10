package proxy

import (
	"testing"
)

func TestParseBridgeArgsValid(t *testing.T) {
	socket, listen, target, err := ParseBridgeArgs([]string{
		"--socket", "/tmp/warden-proxy-x/egress.sock",
		"--listen", "127.0.0.1:18080",
		"--", "/bin/sh", "-c", "echo hi",
	})
	if err != nil {
		t.Fatal(err)
	}
	if socket != "/tmp/warden-proxy-x/egress.sock" {
		t.Errorf("socket = %q", socket)
	}
	if listen != "127.0.0.1:18080" {
		t.Errorf("listen = %q", listen)
	}
	if len(target) != 3 || target[0] != "/bin/sh" || target[2] != "echo hi" {
		t.Errorf("target = %v", target)
	}
}

func TestParseBridgeArgsErrors(t *testing.T) {
	for _, args := range [][]string{
		nil,
		{},
		{"--socket", "/s", "--listen", "127.0.0.1:18080"},                        // no target
		{"--socket", "/s", "--listen", "127.0.0.1:18080", "--"},                  // empty target
		{"--listen", "127.0.0.1:18080", "--", "/bin/true"},                       // missing socket
		{"--socket", "/s", "--", "/bin/true"},                                    // missing listen
		{"--socket", "/s", "--listen"},                                           // dangling flag
		{"--bogus", "x", "--socket", "/s", "--listen", "1:2", "--", "/bin/true"}, // unknown flag
	} {
		if _, _, _, err := ParseBridgeArgs(args); err == nil {
			t.Errorf("ParseBridgeArgs(%q) = nil error, want error", args)
		}
	}
}
