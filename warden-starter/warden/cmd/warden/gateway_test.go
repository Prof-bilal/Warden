package main

import (
	"os"
	"testing"

	"github.com/warden-sandbox/warden/internal/gateway"
	"github.com/warden-sandbox/warden/internal/policy"
)

func TestExpandEnvTemplate(t *testing.T) {
	os.Setenv("GW_TEST_SET", "yes")
	defer os.Unsetenv("GW_TEST_SET")
	os.Unsetenv("GW_TEST_UNSET")

	if got, err := expandEnvTemplate("plain"); err != nil || got != "plain" {
		t.Fatalf("plain = %q, %v", got, err)
	}
	if got, err := expandEnvTemplate("${GW_TEST_SET}"); err != nil || got != "yes" {
		t.Fatalf("set = %q, %v", got, err)
	}
	if got, err := expandEnvTemplate("${GW_TEST_UNSET:-fallback}"); err != nil || got != "fallback" {
		t.Fatalf("default = %q, %v", got, err)
	}
	if _, err := expandEnvTemplate("${GW_TEST_UNSET}"); err == nil {
		t.Fatal("expected error for unset required var")
	}
	if _, err := expandEnvTemplate("${UNTERMINATED"); err == nil {
		t.Fatal("expected error for unterminated reference")
	}
}

func TestInjectGatewayEnvRespectsAllowlist(t *testing.T) {
	os.Unsetenv("GW_A")
	os.Unsetenv("GW_B")
	defer os.Unsetenv("GW_A")
	defer os.Unsetenv("GW_B")
	entry := gateway.ServerEntry{
		Name:    "s",
		Command: []string{"/bin/true"},
		Env:     map[string]string{"GW_A": "from-gateway", "GW_B": "blocked"},
	}
	p := policy.Policy{Env: policy.Env{Allow: []string{"GW_A"}}}
	if err := injectGatewayEnv(entry, p); err != nil {
		t.Fatalf("inject: %v", err)
	}
	if got := os.Getenv("GW_A"); got != "from-gateway" {
		t.Fatalf("GW_A = %q, want from-gateway", got)
	}
	if _, ok := os.LookupEnv("GW_B"); ok {
		t.Fatal("GW_B must not be exported (not in allowlist)")
	}
}

func TestInjectGatewayEnvParentWins(t *testing.T) {
	os.Setenv("GW_C", "parent")
	defer os.Unsetenv("GW_C")
	entry := gateway.ServerEntry{
		Name:    "s",
		Command: []string{"/bin/true"},
		Env:     map[string]string{"GW_C": "gateway-default"},
	}
	p := policy.Policy{Env: policy.Env{Allow: []string{"GW_C"}}}
	if err := injectGatewayEnv(entry, p); err != nil {
		t.Fatalf("inject: %v", err)
	}
	if got := os.Getenv("GW_C"); got != "parent" {
		t.Fatalf("GW_C = %q, want parent (parent env wins)", got)
	}
}

func TestRemoveArgsNoAlias(t *testing.T) {
	args := []string{"--config", "c.json", "--policies", "p", "--server", "s"}
	_, rest, err := gatewayFlag(args, "config")
	if err != nil {
		t.Fatal(err)
	}
	// The original slice must be intact so sequential extraction works.
	if len(args) != 6 || args[2] != "--policies" {
		t.Fatalf("gatewayFlag aliased input: %v", args)
	}
	if len(rest) != 4 {
		t.Fatalf("rest = %v", rest)
	}
	_, rest2, err := gatewayFlag(rest, "policies")
	if err != nil {
		t.Fatal(err)
	}
	if len(rest2) != 2 || rest2[1] != "s" {
		t.Fatalf("rest2 = %v", rest2)
	}
}
