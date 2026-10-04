package mcpserver

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/warden-sandbox/warden/internal/audit"
	"github.com/warden-sandbox/warden/internal/policy"
)

// writeTestPolicy writes a valid policy file and returns its path.
func writeTestPolicy(t *testing.T, dir string, p policy.Policy) string {
	t.Helper()
	path := filepath.Join(dir, "policy.yaml")
	if err := p.Save(path); err != nil {
		t.Fatalf("save policy: %v", err)
	}
	return path
}

func TestToolInspectPolicyShowsGrantsAndDenials(t *testing.T) {
	dir := t.TempDir()
	path := writeTestPolicy(t, dir, policy.Policy{
		Filesystem: policy.Filesystem{Read: []string{filepath.Join(dir, "data")}},
		Network:    policy.Network{Allow: []string{"api.example.com"}},
		Env:        policy.Env{Allow: []string{"PATH"}},
	})
	res, err := NewServer().callTool("inspect_policy", mustJSON(map[string]any{"policy_path": path}))
	if err != nil {
		t.Fatalf("inspect_policy: %v", err)
	}
	for _, want := range []string{"✓", "✗", "api.example.com", "deny"} {
		if !strings.Contains(res.Text, want) {
			t.Errorf("policy text missing %q:\n%s", want, res.Text)
		}
	}
	if res.Widget != "ui://warden/policy" {
		t.Errorf("widget = %q, want ui://warden/policy", res.Widget)
	}
}

func TestToolInspectPolicyRejectsMissingPath(t *testing.T) {
	if _, err := NewServer().callTool("inspect_policy", mustJSON(map[string]any{})); err == nil {
		t.Fatal("expected error for missing policy_path")
	}
}

func TestToolRunSandboxRejectsMissingPolicy(t *testing.T) {
	// Warden never runs without an explicit policy.
	_, err := NewServer().callTool("run_sandbox", mustJSON(map[string]any{"command": []string{"echo"}}))
	if err == nil || !strings.Contains(err.Error(), "policy") {
		t.Fatalf("expected explicit-policy error, got %v", err)
	}
}

func TestToolRunSandboxRejectsBadBackend(t *testing.T) {
	dir := t.TempDir()
	path := writeTestPolicy(t, dir, policy.Policy{})
	_, err := NewServer().callTool("run_sandbox", mustJSON(map[string]any{
		"policy_path": path, "backend": "unsandboxed",
	}))
	if err == nil || !strings.Contains(err.Error(), "backend") {
		t.Fatalf("expected backend validation error, got %v", err)
	}
}

func TestToolRunSandboxRejectsMalformedPolicy(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(path, []byte("filesystem:\n  read: [\"relative/path\"]\nunknown_key: 1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := NewServer().callTool("run_sandbox", mustJSON(map[string]any{"policy_path": path}))
	if err == nil {
		t.Fatal("expected malformed policy to fail through Warden's own engine")
	}
}

func TestToolRunSandboxExecutesRealWardenRun(t *testing.T) {
	if _, err := os.Stat("/bin/sh"); err != nil {
		t.Skip("/bin/sh not available")
	}
	dir := t.TempDir()
	// Policy grants nothing; the workload only echoes. Must still be blocked
	// from running without a backend if none is available, but with a real
	// backend present the run completes. Either way, the result must come
	// from actual execution, never fabricated.
	path := writeTestPolicy(t, dir, policy.Policy{
		Command: []string{"/bin/sh", "-c", "echo wardensandboxok"},
	})
	res, err := NewServer().callTool("run_sandbox", mustJSON(map[string]any{"policy_path": path}))
	if err != nil {
		t.Fatalf("run_sandbox: %v", err)
	}
	var payload struct {
		Run runOutput `json:"run"`
	}
	if err := json.Unmarshal([]byte(structuredJSON(res)), &payload); err != nil {
		t.Fatalf("structured payload: %v", err)
	}
	if payload.Run.RunID == "" {
		t.Error("run_id missing")
	}
	if payload.Run.Status != "completed" && payload.Run.Status != "failed" {
		t.Errorf("status %q not from real execution", payload.Run.Status)
	}
	if payload.Run.Status == "failed" && payload.Run.Backend == "" {
		t.Logf("run failed (likely no sandbox backend on this host): %s", res.Text)
	}
	// The command string appears in the echoed command line even when the
	// run fails to start; only the *Output* section would indicate a real
	// execution. Fabrication check: a failed run must not contain output that
	// could only come from the process having run.
	if payload.Run.Status == "failed" && strings.Contains(payload.Run.Output, "wardensandboxok\n") {
		t.Error("fabricated output on failed run")
	}
}

func TestToolTraceExecutionReadsAuditLog(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "audit.jsonl")
	events := []audit.Event{
		{Type: "file", Action: "open", Resource: "/home/user/.ssh/id_rsa", Allowed: false, Reason: "no grant"},
		{Type: "file", Action: "open", Resource: "/project/config.json", Allowed: true},
		{Type: "network", Action: "connect", Resource: "evil.example.com:443", Allowed: false, Reason: "not allowlisted"},
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	for _, e := range events {
		if err := enc.Encode(e); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(logPath, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	res, err := NewServer().callTool("trace_execution", mustJSON(map[string]any{"log": logPath}))
	if err != nil {
		t.Fatalf("trace_execution: %v", err)
	}
	for _, want := range []string{"2 filesystem operations", "1 network requests", "✗ /home/user/.ssh/id_rsa", "✗ evil.example.com"} {
		if !strings.Contains(res.Text, want) {
			t.Errorf("trace text missing %q:\n%s", want, res.Text)
		}
	}
	if res.Widget != "ui://warden/trace" {
		t.Errorf("widget = %q", res.Widget)
	}
}

func TestToolExplainDenialGroundedInPolicyRules(t *testing.T) {
	dir := t.TempDir()
	path := writeTestPolicy(t, dir, policy.Policy{})
	res, err := NewServer().callTool("explain_denial", mustJSON(map[string]any{
		"resource":    "/home/user/.ssh/id_rsa",
		"policy_path": path,
	}))
	if err != nil {
		t.Fatalf("explain_denial: %v", err)
	}
	for _, want := range []string{"ACCESS DENIED", "/home/user/.ssh/id_rsa", "deny-by-default", "Keep blocked"} {
		if !strings.Contains(res.Text, want) {
			t.Errorf("denial text missing %q:\n%s", want, res.Text)
		}
	}
	if strings.Contains(strings.ToLower(res.Text), "granted permission") {
		t.Error("explanation must not offer to grant permission")
	}
}

func TestToolExplainDenialFromRunEvents(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "audit.jsonl")
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	_ = enc.Encode(audit.Event{Type: "file", Action: "open", Resource: "/home/user/.aws/credentials", Allowed: false, Reason: "no filesystem.read grant"})
	if err := os.WriteFile(logPath, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	s := NewServer()
	s.rememberRun("run-42", runRecord{LogPath: logPath, PolicyPath: filepath.Join(dir, "p.yaml")})
	res, err := s.callTool("explain_denial", mustJSON(map[string]any{"run_id": "run-42"}))
	if err != nil {
		t.Fatalf("explain_denial: %v", err)
	}
	if !strings.Contains(res.Text, "/home/user/.aws/credentials") {
		t.Errorf("denial text missing event resource:\n%s", res.Text)
	}
	// The reason must come from the audit event, not invented.
	if !strings.Contains(res.Text, "no filesystem.read grant") {
		t.Errorf("denial text missing audited reason:\n%s", res.Text)
	}
}

func TestToolExplainDenialUnknownRun(t *testing.T) {
	if _, err := NewServer().callTool("explain_denial", mustJSON(map[string]any{"run_id": "nope"})); err == nil {
		t.Fatal("expected error for unknown run_id")
	}
}

func TestToolGeneratePolicyProposesNeverApplies(t *testing.T) {
	dir := t.TempDir()
	before := dirEntries(t, dir)
	res, err := NewServer().callTool("generate_policy", mustJSON(map[string]any{
		"filesystem_read": []string{filepath.Join(dir, "data")},
		"network_allow":   []string{"api.example.com"},
		"env_allow":       []string{"NODE_ENV"},
	}))
	if err != nil {
		t.Fatalf("generate_policy: %v", err)
	}
	if !strings.Contains(res.Text, "PROPOSED") || !strings.Contains(res.Text, "not applied") {
		t.Errorf("proposal must be clearly marked as not applied:\n%s", res.Text)
	}
	if after := dirEntries(t, dir); len(after) != len(before) {
		t.Error("generate_policy must not write any files")
	}
	if sc, ok := res.Structured["applied"].(bool); !ok || sc {
		t.Error("structured payload must record applied=false")
	}
}

func TestToolGeneratePolicyValidatesThroughRealEngine(t *testing.T) {
	// An invalid host must fail through Warden's own validation.
	if _, err := NewServer().callTool("generate_policy", mustJSON(map[string]any{
		"network_allow": []string{"https://api.example.com/path"},
	})); err == nil {
		t.Fatal("expected Warden policy validation to reject URL as host")
	}
}

func TestToolGeneratePolicyRequiresAtLeastOneGrant(t *testing.T) {
	if _, err := NewServer().callTool("generate_policy", mustJSON(map[string]any{})); err == nil {
		t.Fatal("expected error for fully empty proposal")
	}
}

func TestUnknownToolRejected(t *testing.T) {
	if _, err := NewServer().callTool("delete_everything", mustJSON(map[string]any{})); err == nil {
		t.Fatal("unknown tool must be rejected")
	}
}

func TestJSONRPCEndToEnd(t *testing.T) {
	dir := t.TempDir()
	path := writeTestPolicy(t, dir, policy.Policy{Network: policy.Network{Allow: []string{"api.example.com"}}})
	stdin := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"inspect_policy","arguments":{"policy_path":"` + path + `"}}}`,
		`{not json`,
		`{"jsonrpc":"2.0","id":4,"method":"resources/list"}`,
		`{"jsonrpc":"2.0","id":5,"method":"resources/read","params":{"uri":"ui://warden/policy"}}`,
		`{"jsonrpc":"2.0","id":6,"method":"nope"}`,
	}, "\n") + "\n"
	var out bytes.Buffer
	if err := Serve(strings.NewReader(stdin), &out); err != nil {
		t.Fatalf("Serve: %v", err)
	}
	output := out.String()
	for _, want := range []string{
		`"protocolVersion":"` + protocolVersion + `"`,
		`"name":"inspect_policy"`,
		`"name":"run_sandbox"`,
		`"name":"trace_execution"`,
		`"name":"explain_denial"`,
		`"name":"generate_policy"`,
		`api.example.com`,
		`"parse error`,
		`ui://warden/policy`,
		`unknown method`,
	} {
		if !strings.Contains(output, want) {
			t.Errorf("MCP response missing %q", want)
		}
	}
	// 8 lines in: 7 requests (initialize, tools/list, tools/call, the parse
	// error, resources/list, resources/read, unknown method) each answered,
	// plus 1 notification that must never be answered.
	if got := strings.Count(output, "\"id\":"); got != 7 {
		t.Errorf("expected 7 responses for 7 requests, got %d", got)
	}
	if !strings.Contains(output, "\"id\":null") {
		t.Error("parse failure must produce a null-id JSON-RPC error")
	}
}

func TestLimitedBufferBoundsOutput(t *testing.T) {
	buf := &limitedBuffer{max: 10}
	buf.Write(bytes.Repeat([]byte("a"), 100))
	s := buf.String()
	if len(strings.TrimSuffix(s, "\n... (output truncated; full output in the sandboxed process's own logs)")) > 10 {
		t.Errorf("buffer not bounded: %d bytes", len(s))
	}
	if !strings.Contains(s, "truncated") {
		t.Error("truncation must be disclosed")
	}
}

func TestWidgetsRender(t *testing.T) {
	for _, uri := range []string{"ui://warden/policy", "ui://warden/execution", "ui://warden/trace"} {
		html, ok := widgetHTML(uri)
		if !ok || !strings.Contains(html, "WARDEN") {
			t.Errorf("widget %s missing content", uri)
		}
	}
	if _, ok := widgetHTML("ui://warden/dashboard"); ok {
		t.Error("dashboard must not exist")
	}
}

func TestDenySummariesAlwaysDenyByDefault(t *testing.T) {
	fs, net, env := denySummary(policy.Policy{})
	for _, s := range []string{fs, net} {
		if !strings.Contains(s, "denied by default") {
			t.Errorf("empty policy must read as denied by default, got %q", s)
		}
	}
	if env != "no variables granted; the process gets an empty environment" {
		t.Errorf("empty env must read as fully withheld, got %q", env)
	}
}

func mustJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

func structuredJSON(res *toolResult) string {
	b, _ := json.Marshal(res.Structured)
	return string(b)
}

func dirEntries(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}
