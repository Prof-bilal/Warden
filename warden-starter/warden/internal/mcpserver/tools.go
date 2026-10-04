package mcpserver

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/warden-sandbox/warden/internal/audit"
	"github.com/warden-sandbox/warden/internal/policy"
)

// Bounded-output constants. Every tool returns bounded results so a hostile
// or runaway workload cannot flood the conversation.
const (
	maxOutputBytes  = 16 * 1024
	maxEventsInList = 200
	maxRunSeconds   = 300
)

// ToolDefinitions returns the MCP tool descriptors. The surface is narrow by
// design: inspection is read-only; only run_sandbox and generate_policy have
// side effects, and generate_policy never applies anything.
func ToolDefinitions() []map[string]any {
	inputSchema := func(required []string, props map[string]any) map[string]any {
		s := map[string]any{"type": "object", "properties": props}
		if len(required) > 0 {
			s["required"] = required
		}
		return s
	}
	strArr := func(desc string) map[string]any {
		return map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": desc}
	}
	strField := func(desc string) map[string]any {
		return map[string]any{"type": "string", "description": desc}
	}
	return []map[string]any{
		{
			"name": "inspect_policy",
			"description": "Show what a Warden policy allows and denies: filesystem paths, network hosts, and environment variables. " +
				"Read-only. Warden is deny-by-default — anything not listed is blocked.",
			"inputSchema": inputSchema([]string{"policy_path"}, map[string]any{
				"policy_path": strField("Path to the Warden policy YAML file to inspect"),
			}),
		},
		{
			"name": "run_sandbox",
			"description": "Run a command (for example an MCP server) inside the Warden sandbox under an explicit policy. " +
				"Execution is enforced by the real Warden runtime; nothing outside the policy is reachable. " +
				"Side effect: executes the given command sandboxed.",
			"inputSchema": inputSchema([]string{"policy_path"}, map[string]any{
				"policy_path": strField("Path to the Warden policy YAML enforcing this run"),
				"command":     strArr("Command to run, e.g. [\"node\", \"server.js\"] or [\"npx\", \"-y\", \"@modelcontextprotocol/server-everything\"]"),
				"backend":     strField("Optional sandbox backend: auto (default), linux, seatbelt, windows, docker"),
			}),
		},
		{
			"name": "trace_execution",
			"description": "Summarize what a Warden execution attempted: filesystem and network operations, allowed vs denied counts, and the specific denied resources. " +
				"Use the run_id returned by run_sandbox, or the latest audit/trace log. Read-only.",
			"inputSchema": inputSchema(nil, map[string]any{
				"run_id": strField("Run identifier returned by run_sandbox"),
				"log":    strField("Path to a Warden audit JSONL log (default: Warden's audit log)"),
			}),
		},
		{
			"name": "explain_denial",
			"description": "Explain why Warden blocked an access: which resource was denied, which policy rule (or deny-by-default) applies, and what granting it would mean. " +
				"Read-only; it never grants permission.",
			"inputSchema": inputSchema(nil, map[string]any{
				"resource":    strField("Resource that was denied, e.g. a file path or network host"),
				"policy_path": strField("Policy to check against (optional)"),
				"run_id":      strField("Run identifier from run_sandbox to look up the exact denied event (optional)"),
			}),
		},
		{
			"name": "generate_policy",
			"description": "Propose a restrictive Warden policy from explicit requirements, e.g. \"read ./data and call api.example.com\". " +
				"Returns the proposed YAML for review; it does NOT write or apply any policy.",
			"inputSchema": inputSchema(nil, map[string]any{
				"filesystem_read":  strArr("Paths the workload may read"),
				"filesystem_write": strArr("Paths the workload may write"),
				"network_allow":    strArr("Network hosts the workload may connect to"),
				"env_allow":        strArr("Environment variable names to pass through"),
				"command":          strArr("Command the policy will run"),
				"memory_mb":        map[string]any{"type": "integer", "description": "Memory limit in MB"},
				"timeout_s":        map[string]any{"type": "integer", "description": "Wall-clock timeout in seconds"},
			}),
		},
	}
}

// ---------------------------------------------------------------------------
// inspect_policy
// ---------------------------------------------------------------------------

func (s *Server) toolInspectPolicy(args map[string]any) (*toolResult, error) {
	path, err := argString(args, "policy_path")
	if err != nil || path == "" {
		return nil, fmt.Errorf("policy_path is required")
	}
	p, err := policy.Load(path)
	if err != nil {
		return nil, err
	}
	fsDenied, netDenied, envDenied := denySummary(p)
	structured := map[string]any{
		"policy_path": path,
		"filesystem": map[string]any{
			"read":        p.Filesystem.Read,
			"write":       p.Filesystem.Write,
			"denied_note": fsDenied,
		},
		"network": map[string]any{
			"allow":       p.Network.Allow,
			"denied_note": netDenied,
		},
		"env": map[string]any{
			"allow":       p.EnvAllowlist(),
			"denied_note": envDenied,
		},
		"limits": map[string]any{
			"memory_mb": p.Limits.MemoryMB,
			"timeout_s": p.Limits.TimeoutS,
		},
	}
	if p.MCP != nil {
		structured["mcp"] = map[string]any{
			"upstream":       p.MCP.Upstream,
			"allow_tools":    p.MCP.AllowTools,
			"deny_patterns":  p.MCP.DenyPatterns,
			"max_payload_kb": p.MCP.MaxPayloadKB,
		}
	}
	return &toolResult{
		Text:       renderPolicyText(path, p),
		Structured: structured,
		Widget:     "ui://warden/policy",
	}, nil
}

// denySummary returns the deny-by-default wording per section.
func denySummary(p policy.Policy) (fs, net, env string) {
	fs = "everything else is invisible (deny by default)"
	if len(p.Filesystem.Read) == 0 && len(p.Filesystem.Write) == 0 {
		fs = "no paths granted; everything is denied by default"
	}
	net = "all other hosts are blocked before DNS"
	if len(p.Network.Allow) == 0 {
		net = "no hosts granted; all network access is denied by default"
	}
	env = "all other variables are withheld"
	if len(p.Env.Allow) == 0 {
		env = "no variables granted; the process gets an empty environment"
	}
	return
}

func renderPolicyText(path string, p policy.Policy) string {
	var b strings.Builder
	b.WriteString("WARDEN POLICY — " + path + "\n\n")
	b.WriteString("FILESYSTEM\n")
	denFS, denNet, denEnv := denySummary(p)
	for _, r := range p.Filesystem.Read {
		b.WriteString("  ✓ " + r + "  (read)\n")
	}
	for _, w := range p.Filesystem.Write {
		b.WriteString("  ✓ " + w + "  (read-write)\n")
	}
	b.WriteString("  ✗ " + denFS + "\n\n")
	b.WriteString("NETWORK\n")
	for _, h := range p.Network.Allow {
		b.WriteString("  ✓ " + h + "\n")
	}
	b.WriteString("  ✗ " + denNet + "\n\n")
	b.WriteString("ENVIRONMENT\n")
	for _, e := range p.EnvAllowlist() {
		b.WriteString("  ✓ " + e + "\n")
	}
	b.WriteString("  ✗ " + denEnv + "\n")
	if p.Limits.MemoryMB > 0 || p.Limits.TimeoutS > 0 {
		b.WriteString("\nLIMITS\n")
		if p.Limits.MemoryMB > 0 {
			fmt.Fprintf(&b, "  memory_mb: %d\n", p.Limits.MemoryMB)
		}
		if p.Limits.TimeoutS > 0 {
			fmt.Fprintf(&b, "  timeout_s: %d\n", p.Limits.TimeoutS)
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

// ---------------------------------------------------------------------------
// run_sandbox
// ---------------------------------------------------------------------------

type runOutput struct {
	Status       string   `json:"status"` // completed | failed
	ExitCode     int      `json:"exit_code"`
	Backend      string   `json:"backend,omitempty"`
	PolicyPath   string   `json:"policy_path"`
	Command      []string `json:"command"`
	AllowedOps   int      `json:"allowed_ops"`
	DeniedOps    int      `json:"denied_ops"`
	DeniedSample []string `json:"denied_sample,omitempty"`
	RunID        string   `json:"run_id"`
	AuditLog     string   `json:"audit_log"`
	Output       string   `json:"output,omitempty"`
}

func (s *Server) toolRunSandbox(args map[string]any) (*toolResult, error) {
	policyPath, err := argString(args, "policy_path")
	if err != nil || policyPath == "" {
		return nil, fmt.Errorf("policy_path is required; Warden never runs without an explicit policy")
	}
	cmd, err := argStrings(args, "command")
	if err != nil {
		return nil, err
	}
	backend, _ := argString(args, "backend")
	if backend != "" {
		switch backend {
		case "auto", "linux", "seatbelt", "windows", "docker":
		default:
			return nil, fmt.Errorf("backend %q is not valid (want auto, linux, seatbelt, windows, or docker)", backend)
		}
	}

	// Load and validate the policy through Warden's own engine so malformed
	// or unsafe policies fail exactly as the CLI would.
	p, err := policy.Load(policyPath)
	if err != nil {
		return nil, err
	}
	resolved, err := p.ResolveCommand(cmd)
	if err != nil {
		return nil, err
	}

	bin, err := wardenBin()
	if err != nil {
		return nil, err
	}
	runArgs := []string{"run", "--policy", policyPath}
	if backend != "" && backend != "auto" {
		runArgs = append(runArgs, "--backend", backend)
	}
	runArgs = append(runArgs, "--")
	runArgs = append(runArgs, resolved...)

	// The audit log records this run's events from its current end offset.
	logPath, err := audit.DefaultPath()
	if err != nil {
		return nil, err
	}
	startOffset := fileSize(logPath)

	runID := fmt.Sprintf("run-%d", timeNow().UnixNano())
	s.rememberRun(runID, runRecord{
		LogPath:    logPath,
		StartOffet: startOffset,
		PolicyPath: policyPath,
		Command:    resolved,
		StartedAt:  timeNow(),
	})

	execCmd := exec.Command(bin, runArgs...)
	buf := &limitedBuffer{max: maxOutputBytes}
	execCmd.Stdout = buf
	execCmd.Stderr = buf
	execErr := execCmd.Run()

	status := "completed"
	if execErr != nil {
		status = "failed"
	}
	events, _ := readEventsFrom(logPath, startOffset)
	allowed, denied, deniedSample := summarizeEvents(events)

	res := &runOutput{
		Status:       status,
		ExitCode:     execCmd.ProcessState.ExitCode(),
		PolicyPath:   policyPath,
		Command:      resolved,
		AllowedOps:   allowed,
		DeniedOps:    denied,
		DeniedSample: deniedSample,
		RunID:        runID,
		AuditLog:     logPath,
		Output:       buf.String(),
	}
	if beName := resolveBackendName(backend); beName != "" {
		res.Backend = beName
	}

	text := renderRunText(res)
	if execErr != nil && !isExitError(execErr) {
		// The run could not even start (e.g. no sandbox backend). Surface
		// the runtime failure clearly; Warden fails closed, never sandboxed-
		// less.
		text += "\n\nRuntime error: " + execErr.Error()
	}
	return &toolResult{
		Text:       text,
		Structured: map[string]any{"run": res},
		Widget:     "ui://warden/execution",
	}, nil
}

func isExitError(err error) bool {
	var ee *exec.ExitError
	return errors.As(err, &ee)
}

func resolveBackendName(backend string) string {
	// Cheap introspection: warden run prints the backend in its summary, but
	// in non-TTY mode it may not; report what was requested instead of
	// guessing. "auto" stays empty so we do not fabricate a backend claim.
	if backend == "" || backend == "auto" {
		return ""
	}
	return backend
}

func renderRunText(r *runOutput) string {
	var b strings.Builder
	b.WriteString("WARDEN EXECUTION\n\n")
	if r.Status == "completed" {
		b.WriteString("Status  ✓ completed\n")
	} else {
		b.WriteString(fmt.Sprintf("Status  ✗ failed (exit code %d)\n", r.ExitCode))
	}
	if r.Backend != "" {
		fmt.Fprintf(&b, "Backend %s\n", r.Backend)
	}
	fmt.Fprintf(&b, "Policy  %s\n", r.PolicyPath)
	fmt.Fprintf(&b, "Command %s\n", strings.Join(r.Command, " "))
	fmt.Fprintf(&b, "\nAllowed operations: %d\n", r.AllowedOps)
	fmt.Fprintf(&b, "Blocked operations: %d\n", r.DeniedOps)
	if len(r.DeniedSample) > 0 {
		b.WriteString("\nDenied:\n")
		for _, d := range r.DeniedSample {
			b.WriteString("  ✗ " + d + "\n")
		}
	}
	if r.Output != "" {
		b.WriteString("\nOutput (bounded):\n" + r.Output)
	}
	fmt.Fprintf(&b, "\nTrace: run_id=%s (use trace_execution to inspect)\n", r.RunID)
	return strings.TrimRight(b.String(), "\n")
}

// ---------------------------------------------------------------------------
// trace_execution
// ---------------------------------------------------------------------------

func (s *Server) toolTraceExecution(args map[string]any) (*toolResult, error) {
	runID, _ := argString(args, "run_id")
	logPath, _ := argString(args, "log")
	startOffset := int64(0)
	if runID != "" {
		rec, ok := s.getRun(runID)
		if !ok {
			return nil, fmt.Errorf("unknown run_id %q (it may have expired from the bounded registry; pass a log path instead)", runID)
		}
		logPath = rec.LogPath
		startOffset = rec.StartOffet
	}
	if logPath == "" {
		var err error
		logPath, err = audit.DefaultPath()
		if err != nil {
			return nil, err
		}
	}
	if _, err := os.Stat(logPath); err != nil {
		return nil, fmt.Errorf("audit log %q is not readable: %w", logPath, err)
	}
	events, err := readEventsFrom(logPath, startOffset)
	if err != nil {
		return nil, err
	}
	allowed, denied, deniedSample := summarizeEvents(events)
	fsCount, netCount := 0, 0
	for _, e := range events {
		switch e.Type {
		case "file":
			fsCount++
		case "network":
			netCount++
		}
	}
	structured := map[string]any{
		"log_path":       logPath,
		"total_events":   len(events),
		"filesystem_ops": fsCount,
		"network_ops":    netCount,
		"allowed":        allowed,
		"denied":         denied,
		"denied_sample":  deniedSample,
	}
	text := renderTraceText(logPath, events, allowed, denied, deniedSample)
	return &toolResult{
		Text:       text,
		Structured: structured,
		Widget:     "ui://warden/trace",
	}, nil
}

func renderTraceText(logPath string, events []audit.Event, allowed, denied int, deniedSample []string) string {
	var b strings.Builder
	b.WriteString("WARDEN TRACE — " + logPath + "\n\n")
	fmt.Fprintf(&b, "%d filesystem operations\n", countType(events, "file"))
	fmt.Fprintf(&b, "%d network requests\n", countType(events, "network"))
	fmt.Fprintf(&b, "%d allowed\n", allowed)
	fmt.Fprintf(&b, "%d denied\n", denied)
	if len(deniedSample) > 0 {
		b.WriteString("\nDenied:\n")
		for _, d := range deniedSample {
			b.WriteString("  ✗ " + d + "\n")
		}
	}
	if len(events) == 0 {
		b.WriteString("\nNo events recorded in this window.\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func countType(events []audit.Event, typ string) int {
	n := 0
	for _, e := range events {
		if e.Type == typ {
			n++
		}
	}
	return n
}

// ---------------------------------------------------------------------------
// explain_denial
// ---------------------------------------------------------------------------

func (s *Server) toolExplainDenial(args map[string]any) (*toolResult, error) {
	resource, _ := argString(args, "resource")
	policyPath, _ := argString(args, "policy_path")
	runID, _ := argString(args, "run_id")

	// If a run is given, find the actual denied event so the explanation is
	// grounded in Warden's evidence, not a guess.
	var matched []audit.Event
	if runID != "" {
		rec, ok := s.getRun(runID)
		if !ok {
			return nil, fmt.Errorf("unknown run_id %q", runID)
		}
		events, err := readEventsFrom(rec.LogPath, rec.StartOffet)
		if err != nil {
			return nil, err
		}
		for _, e := range events {
			if e.Allowed {
				continue
			}
			if resource == "" || strings.Contains(e.Resource, resource) || strings.Contains(resource, e.Resource) {
				matched = append(matched, e)
			}
		}
		if len(matched) == 0 {
			return nil, fmt.Errorf("no denied events matching %q in run %s", resource, runID)
		}
		if resource == "" {
			resource = matched[0].Resource
		}
		if policyPath == "" {
			policyPath = rec.PolicyPath
		}
	}
	if resource == "" {
		return nil, fmt.Errorf("resource is required (or pass run_id to explain a specific run's denials)")
	}

	reason, rule := classifyDenial(resource, policyPath)
	if len(matched) > 0 && matched[0].Reason != "" {
		reason = matched[0].Reason
	}

	structured := map[string]any{
		"resource":    resource,
		"reason":      reason,
		"rule":        rule,
		"policy_path": policyPath,
		"suggestion":  suggestionFor(resource),
	}
	return &toolResult{
		Text:       renderDenialText(resource, reason, rule, policyPath),
		Structured: structured,
		Widget:     "ui://warden/trace",
	}, nil
}

// classifyDenial explains a denial from the policy engine's rules. It is a
// description of Warden's existing deny-by-default semantics, not a new
// decision procedure.
func classifyDenial(resource, policyPath string) (reason, rule string) {
	rule = "deny-by-default"
	switch {
	case isSensitivePath(resource):
		reason = "No filesystem permission exists for this path, and Warden's sensitive-path guard blocks SSH keys, cloud credentials, and .env files."
	case strings.HasPrefix(resource, "/"):
		reason = "No filesystem.read or filesystem.write permission covers this path. Unlisted paths are invisible inside the sandbox."
	case isHostLike(resource):
		reason = "No network.allow entry matches this host. Connections to non-allowlisted hosts are blocked before DNS resolution."
	case looksLikeEnvVar(resource):
		reason = "This environment variable is not in env.allow, so it is withheld from the sandboxed process."
	default:
		reason = "This resource is not explicitly granted by the policy. Warden denies anything unspecified."
	}
	if policyPath == "" {
		rule += " (no policy inspected; pass policy_path for a rule-level check)"
	}
	return
}

func isSensitivePath(p string) bool {
	lp := strings.ToLower(p)
	for _, pat := range []string{".ssh", ".aws/credentials", ".env", "id_rsa", ".pem", ".kube/config", ".netrc", ".docker/config"} {
		if strings.Contains(lp, pat) {
			return true
		}
	}
	return false
}

func isHostLike(r string) bool {
	if strings.ContainsAny(r, "/\\") || strings.Contains(r, " ") {
		return false
	}
	return strings.Contains(r, ".") && !looksLikeEnvVar(r)
}

func looksLikeEnvVar(r string) bool {
	if strings.Contains(r, "/") {
		return false
	}
	for _, c := range r {
		if !(c == '_' || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')) {
			return false
		}
	}
	return len(r) > 0 && !strings.ContainsAny(r, "0123456789") // exclude pure numbers
}

func suggestionFor(resource string) string {
	switch {
	case isSensitivePath(resource):
		return "Keep blocked unless the workload genuinely requires these credentials. If it truly needs them, add the narrowest possible grant to the policy yourself and re-run."
	case strings.HasPrefix(resource, "/"):
		return "If the workload genuinely needs this path, add it under filesystem.read (or filesystem.write) in the policy and re-run. Grant the narrowest directory that works."
	case isHostLike(resource):
		return "If the workload genuinely needs this host, add it to network.allow in the policy and re-run."
	case looksLikeEnvVar(resource):
		return "If the workload genuinely needs this variable, add its name to env.allow. Values come from your environment; they are never stored in the policy."
	default:
		return "Grant it explicitly in the policy only if the workload genuinely needs it."
	}
}

func renderDenialText(resource, reason, rule, policyPath string) string {
	var b strings.Builder
	b.WriteString("ACCESS DENIED\n\n")
	b.WriteString("Resource:\n  " + resource + "\n\n")
	b.WriteString("Reason:\n  " + reason + "\n\n")
	b.WriteString("Policy:\n  " + rule + "\n")
	if policyPath != "" {
		b.WriteString("  (" + policyPath + ")\n")
	}
	b.WriteString("\nSuggested action:\n  " + suggestionFor(resource) + "\n")
	b.WriteString("\nWarden does not grant permission from this explanation.\n")
	return strings.TrimRight(b.String(), "\n")
}

// ---------------------------------------------------------------------------
// generate_policy (propose-only)
// ---------------------------------------------------------------------------

func (s *Server) toolGeneratePolicy(args map[string]any) (*toolResult, error) {
	read, err := argStrings(args, "filesystem_read")
	if err != nil {
		return nil, err
	}
	write, err := argStrings(args, "filesystem_write")
	if err != nil {
		return nil, err
	}
	hosts, err := argStrings(args, "network_allow")
	if err != nil {
		return nil, err
	}
	env, err := argStrings(args, "env_allow")
	if err != nil {
		return nil, err
	}
	command, err := argStrings(args, "command")
	if err != nil {
		return nil, err
	}
	if len(read) == 0 && len(write) == 0 && len(hosts) == 0 && len(env) == 0 {
		return nil, fmt.Errorf("specify at least one grant (filesystem_read, filesystem_write, network_allow, or env_allow); a fully empty policy would block everything")
	}
	memoryMB, timeoutS := 0, 0
	if v, ok := args["memory_mb"].(float64); ok {
		memoryMB = int(v)
	}
	if v, ok := args["timeout_s"].(float64); ok {
		timeoutS = int(v)
	}

	// Validate through the real policy engine: the proposal must be a policy
	// Warden would actually accept.
	proposed := policy.Policy{
		Command:    command,
		Filesystem: policy.Filesystem{Read: read, Write: write},
		Network:    policy.Network{Allow: hosts},
		Env:        policy.Env{Allow: env},
		Limits:     policy.Limits{MemoryMB: memoryMB, TimeoutS: timeoutS},
	}
	if err := proposed.Validate(); err != nil {
		return nil, fmt.Errorf("proposed policy is invalid: %w", err)
	}
	data, err := proposed.Marshal()
	if err != nil {
		return nil, err
	}

	yamlText := string(data)
	structured := map[string]any{
		"proposed_yaml": yamlText,
		"applied":       false,
		"note":          "Proposal only. Review it, save it to a file, and run it with run_sandbox. Nothing was written or applied.",
	}
	text := "PROPOSED WARDEN POLICY (not applied)\n\n" + yamlText +
		"\nReview this proposal. To use it, save it to a file (e.g. policy.yaml) and run:\n" +
		"  run_sandbox with policy_path set to that file.\n" +
		"generate_policy never writes or applies a policy."
	return &toolResult{
		Text:       text,
		Structured: structured,
		Widget:     "ui://warden/policy",
	}, nil
}

// ---------------------------------------------------------------------------
// shared helpers
// ---------------------------------------------------------------------------

func argString(args map[string]any, key string) (string, error) {
	v, ok := args[key]
	if !ok || v == nil {
		return "", nil
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("%s must be a string", key)
	}
	return s, nil
}

func argStrings(args map[string]any, key string) ([]string, error) {
	v, ok := args[key]
	if !ok || v == nil {
		return nil, nil
	}
	arr, ok := v.([]any)
	if !ok {
		return nil, fmt.Errorf("%s must be an array of strings", key)
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		s, ok := item.(string)
		if !ok {
			return nil, fmt.Errorf("%s must contain only strings", key)
		}
		out = append(out, s)
	}
	return out, nil
}

// limitedBuffer caps captured process output so a chatty or hostile server
// cannot flood the conversation.
type limitedBuffer struct {
	max   int
	buf   bytes.Buffer
	total int
}

func (l *limitedBuffer) Write(p []byte) (int, error) {
	l.total += len(p)
	if l.buf.Len() < l.max {
		room := l.max - l.buf.Len()
		if len(p) > room {
			p = p[:room]
		}
		l.buf.Write(p)
	}
	return len(p), nil
}

func (l *limitedBuffer) String() string {
	s := l.buf.String()
	if l.total > l.max {
		s += "\n... (output truncated; full output in the sandboxed process's own logs)"
	}
	return s
}

// readEventsFrom reads audit events from a JSONL log starting at byte
// offset, bounded to maxEventsInList. Offset 0 means the whole file.
func readEventsFrom(path string, offset int64) ([]audit.Event, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read audit log %q: %w", path, err)
	}
	if offset < 0 || offset > int64(len(data)) {
		offset = 0 // log rotated/truncated; fall back to whole file
	}
	window := data[offset:]
	if i := bytes.IndexByte(window, '\n'); offset > 0 && i >= 0 {
		window = window[i+1:] // drop a possibly partial first line
	}
	events, err := audit.ReadEvents(bytes.NewReader(window))
	if err != nil {
		return nil, fmt.Errorf("parse audit log %q: %w", path, err)
	}
	if len(events) > maxEventsInList {
		events = events[len(events)-maxEventsInList:]
	}
	return events, nil
}

func fileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}

// summarizeEvents returns allowed/denied counts plus a deduplicated sample
// of denied resources (no contents, ever — only paths/hosts).
func summarizeEvents(events []audit.Event) (allowed, denied int, deniedSample []string) {
	seen := map[string]bool{}
	for _, e := range events {
		if e.Allowed {
			allowed++
			continue
		}
		denied++
		label := e.Resource
		if len(label) > 120 {
			label = label[:117] + "..."
		}
		if !seen[label] && len(deniedSample) < 10 {
			seen[label] = true
			deniedSample = append(deniedSample, label)
		}
	}
	sort.Strings(deniedSample)
	return
}
