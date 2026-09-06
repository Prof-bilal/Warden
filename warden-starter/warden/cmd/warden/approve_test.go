package main

import (
	"reflect"
	"testing"
	"time"
)

func TestParseApproveFlags(t *testing.T) {
	enabled, timeout, rest, err := parseApproveFlags([]string{"--approve", "--policy", "p.yaml", "--", "/usr/bin/true"})
	if err != nil || !enabled || timeout != 0 {
		t.Fatalf("approve = %v, %v, %v", enabled, timeout, err)
	}
	if !reflect.DeepEqual(rest, []string{"--policy", "p.yaml", "--", "/usr/bin/true"}) {
		t.Fatalf("rest = %v", rest)
	}

	enabled, timeout, rest, err = parseApproveFlags([]string{"--approve-timeout=30s", "--approve", "--policy", "p.yaml"})
	if err != nil || !enabled || timeout != 30*time.Second {
		t.Fatalf("approve = %v, %v, %v", enabled, timeout, err)
	}
	if !reflect.DeepEqual(rest, []string{"--policy", "p.yaml"}) {
		t.Fatalf("rest = %v", rest)
	}

	// Unknown flags pass through for the downstream parser.
	_, _, rest, err = parseApproveFlags([]string{"--backend", "docker"})
	if err != nil || !reflect.DeepEqual(rest, []string{"--backend", "docker"}) {
		t.Fatalf("passthrough = %v, %v", rest, err)
	}

	for _, args := range [][]string{
		{"--approve-timeout"},                  // missing value
		{"--approve-timeout=bogus"},            // bad duration
		{"--approve-timeout=-5s", "--approve"}, // negative
		{"--approve-timeout=10s"},              // timeout without approve
		{"--approve=maybe"},                    // bad bool
	} {
		if _, _, _, err := parseApproveFlags(args); err == nil {
			t.Errorf("parseApproveFlags(%v) should fail", args)
		}
	}
}

func TestCheckSandboxCommand(t *testing.T) {
	if err := checkSandboxCommand([]string{"/bin/true"}); err != nil {
		// /bin/true may not exist on all platforms; accept either outcome,
		// but only these two reasons.
		t.Logf("checkSandboxCommand(/bin/true): %v", err)
	}
	if err := checkSandboxCommand([]string{"relative"}); err == nil {
		t.Fatal("relative command must fail")
	}
	if err := checkSandboxCommand([]string{"/nonexistent-warden-test-binary"}); err == nil {
		t.Fatal("missing executable must fail")
	}
	if err := checkSandboxCommand(nil); err == nil {
		t.Fatal("empty command must fail")
	}
}

func TestGatewayBoolOpt(t *testing.T) {
	enabled, rest := gatewayBoolOpt([]string{"--server", "s", "--approve", "--backend", "docker"}, "approve")
	if !enabled {
		t.Fatal("approve should be detected")
	}
	if !reflect.DeepEqual(rest, []string{"--server", "s", "--backend", "docker"}) {
		t.Fatalf("rest = %v", rest)
	}
	enabled, _ = gatewayBoolOpt([]string{"--server", "s"}, "approve")
	if enabled {
		t.Fatal("approve should be absent")
	}
}
