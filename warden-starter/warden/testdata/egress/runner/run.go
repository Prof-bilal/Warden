package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Runner executes matrix cases for one backend.
type Runner struct {
	cfg           *runConfig
	backend       string
	mode          string
	zone          string
	runID         string
	dnsIP         string
	moduleRoot    string
	harnessDir    string
	wardenPath    string
	probePath     string
	caFile        string
	collectors    *Collectors
	allowedIP     string
	deniedIP      string
	metaIP        string
	privateIP     string
	loopIP        string
	lanIP         string
	image         string
	blockedReason string
}

type stepEvidence struct {
	Name        string           `json:"name"`
	Args        []string         `json:"args"`
	CommandLine string           `json:"command_line"`
	ExitCode    int              `json:"exit_code"`
	TimedOut    bool             `json:"timed_out"`
	Stdout      string           `json:"stdout,omitempty"`
	Stderr      string           `json:"stderr,omitempty"`
	Events      []map[string]any `json:"events,omitempty"`
	Positive    bool             `json:"positive_marker"`
}

type caseEvidence struct {
	c                 *Case
	vars              map[string]string
	token             string
	caseID            string
	dir               string
	steps             []stepEvidence
	audit             []map[string]any
	hits              []Hit
	dns               []DNSHit
	collectors        *Collectors
	notes             []string
	policyDigest      string
	policyDigestAfter string
}

func (ev *caseEvidence) allEvents() []map[string]any {
	var out []map[string]any
	for _, s := range ev.steps {
		out = append(out, s.Events...)
	}
	return out
}

func (ev *caseEvidence) events(action string) []map[string]any {
	var out []map[string]any
	for _, e := range ev.allEvents() {
		if isMarker(e) {
			continue
		}
		if a, _ := e["action"].(string); a == action {
			out = append(out, e)
		}
	}
	return out
}

// isMarker reports whether an event is the probe's liveness marker rather
// than an observation. Markers prove the workload started; they are never
// evidence that a network operation succeeded.
func isMarker(e map[string]any) bool {
	m, _ := e["marker"].(string)
	return m == "PROBE_UP"
}

func (ev *caseEvidence) anyPositive() bool {
	for _, s := range ev.steps {
		if s.Positive {
			return true
		}
	}
	return false
}

func (r *Runner) forbiddenNames(c *Case) []string {
	if len(c.Forbidden) > 0 {
		return c.Forbidden
	}
	return []string{
		"denied-http", "denied-https", "denied-tcp", "denied-udp",
		"metadata-tcp", "metadata-udp", "private-tcp", "loopback-tcp",
		"v6-denied-tcp", "v6-denied-udp", "fake-proxy", "unix-fake",
	}
}

func (ev *caseEvidence) forbiddenHits(names []string) []Hit {
	var out []Hit
	for _, h := range ev.hits {
		for _, n := range names {
			if h.Collector == n {
				out = append(out, h)
			}
		}
	}
	return out
}

// fileDigest returns a stable digest of the active policy file. A missing or
// unreadable file yields an error string, which never matches a real digest.
func fileDigest(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return "error:" + err.Error()
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func (r *Runner) runCase(c Case) CaseResult {
	start := time.Now()
	dir := filepath.Join(r.cfg.OutDir, "cases", c.ID)
	res := CaseResult{
		ID: c.ID, Group: c.Group, Title: c.Title, Kind: c.Kind,
		Backend: r.backend, Mode: r.mode, Expected: c.Expect,
		StartedAt: start, Artifacts: map[string]string{},
	}
	finish := func(class, reason string) CaseResult {
		res.Classification = class
		res.Reason = reason
		res.FinishedAt = time.Now()
		res.DurationMS = time.Since(start).Milliseconds()
		return res
	}
	for _, sub := range []string{"sandbox-out", "state", "tmp"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
			return finish(ClassBlocked, "prepare case dir: "+err.Error())
		}
	}
	token := "tok-" + c.ID + "-" + r.runID
	vars := r.buildVars(c.ID, dir, token)
	res.Command = r.probeCommandLine(c, vars)

	if len(c.Backends) > 0 && !containsStr(c.Backends, r.backend) {
		return finish(ClassNotRun, fmt.Sprintf("case is not applicable to backend %s (applicable: %s)", r.backend, strings.Join(c.Backends, ",")))
	}
	if r.blockedReason != "" {
		return finish(ClassBlocked, r.blockedReason)
	}
	for _, req := range c.Requires {
		if why := r.requireUnsupported(req); why != "" {
			return finish(ClassUnsupported, why)
		}
	}
	switch c.Kind {
	case "unsupported":
		return finish(ClassUnsupported, c.Reason)
	case "not_run":
		return finish(ClassNotRun, c.Reason)
	}
	if why := c.Extra["blocked_reason"]; why != "" && r.backend == "docker" && r.mode == "netns" {
		return finish(ClassBlocked, why)
	}

	policyPath := ""
	policyDigest := ""
	if c.Policy != "" {
		allow := r.policyAllow(&c, vars)
		data := policyData{
			ProbeDir: r.harnessDir,
			OutDir:   filepath.Join(dir, "sandbox-out"),
			Allow:    allow,
			Timeout:  int(r.cfg.Timeout.Seconds()),
		}
		// The active policy lives outside the per-case tree that the sandbox
		// can see. Rendering it under cases/<id>/ would let a sandbox-local
		// tmpfs shadow the host file and make a tamper attempt look denied (or
		// worse, look successful) for the wrong reason.
		policyDir := filepath.Join(r.cfg.OutDir, "policies")
		if err := os.MkdirAll(policyDir, 0o755); err != nil {
			return finish(ClassBlocked, "prepare policy dir: "+err.Error())
		}
		policyPath = filepath.Join(policyDir, c.ID+".yaml")
		if err := renderPolicy(r.cfg.PolicyDir, c.Policy, data, policyPath); err != nil {
			return finish(ClassBlocked, "render policy: "+err.Error())
		}
		policyDigest = fileDigest(policyPath)
		vars["policyPath"] = policyPath
		res.Policy = c.Policy
		res.Artifacts["policy"] = policyPath
	}
	r.collectors.RegisterCaseDNS(strings.ToLower(c.ID), vars, r.mode == "netns")

	ev := &caseEvidence{c: &c, vars: vars, token: token, caseID: strings.ToLower(c.ID), dir: dir, collectors: r.collectors,
		policyDigest: policyDigest}

	if c.Extra["unix_echo"] == "true" {
		if err := r.collectors.StartUnixEcho(filepath.Join(dir, "sandbox-out", "control.sock")); err != nil {
			ev.notes = append(ev.notes, "could not start the granted-path Unix echo socket: "+err.Error())
		}
	}

	switch c.Kind {
	case "mcp_stdio":
		r.runMCP(c, ev, policyPath)
	case "kill_warden":
		r.runKillWarden(c, ev, policyPath)
	case "stop_dns":
		r.runStopDNS(c, ev, policyPath)
	case "stop_collector":
		r.runStopCollector(c, ev, policyPath)
	case "multi":
		r.runMulti(c, ev, policyPath)
	default:
		step := r.execProbe(c, dir, policyPath, substituteAll(c.Probe, vars), "", 0)
		ev.steps = append(ev.steps, step)
	}

	if policyPath != "" {
		ev.policyDigestAfter = fileDigest(policyPath)
		// Keep a copy next to the case evidence without making it the active
		// file that the sandbox could reach.
		if b, err := os.ReadFile(policyPath); err == nil {
			copyPath := filepath.Join(dir, "policy.yaml")
			if err := os.WriteFile(copyPath, b, 0o644); err == nil {
				res.Artifacts["policy"] = copyPath
			}
		}
		ev.audit = readAudit(filepath.Join(dir, "state", "warden", "audit.jsonl"))
	}
	ev.hits = r.collectors.HitsForToken(token)
	ev.dns = filterDNS(r.collectors.DNSQueries(), ev.caseID)

	class, reason, checks := r.evaluate(&c, ev)
	var classOverride, reasonOverride string
	switch c.Kind {
	case "schema_observe":
		classOverride = ClassUnsupported
		reasonOverride = strings.TrimSpace(c.Reason + " observation: " + summarizeObserved(ev))
	}
	if classOverride != "" && ev.anyPositive() {
		class, reason = classOverride, reasonOverride
	}
	res.Checks = checks
	res.ProbeEvents = ev.allEvents()
	res.CollectorHits = ev.hits
	res.DNSHits = ev.dns
	res.AuditEvents = ev.audit
	res.PositiveMarker = ev.anyPositive()
	res.ExitCode = 0
	if len(ev.steps) > 0 {
		res.ExitCode = ev.steps[len(ev.steps)-1].ExitCode
		res.TimedOut = ev.steps[len(ev.steps)-1].TimedOut
	}
	res.Notes = c.Notes

	// Persist per-case evidence artifacts.
	_ = writeJSON(filepath.Join(dir, "probe-events.json"), ev.allEvents())
	_ = writeJSON(filepath.Join(dir, "collector-hits.json"), ev.hits)
	_ = writeJSON(filepath.Join(dir, "dns-hits.json"), ev.dns)
	_ = writeJSON(filepath.Join(dir, "audit-events.json"), ev.audit)
	_ = writeJSON(filepath.Join(dir, "checks.json"), checks)
	_ = writeJSON(filepath.Join(dir, "steps.json"), ev.steps)
	res.Artifacts["steps"] = filepath.Join(dir, "steps.json")
	res.Artifacts["probe_events"] = filepath.Join(dir, "probe-events.json")
	res.Artifacts["collector_hits"] = filepath.Join(dir, "collector-hits.json")
	res.Artifacts["dns_hits"] = filepath.Join(dir, "dns-hits.json")
	res.Artifacts["audit_events"] = filepath.Join(dir, "audit-events.json")
	res.Artifacts["checks"] = filepath.Join(dir, "checks.json")
	return finish(class, reason)
}

func (r *Runner) buildVars(caseID, dir, token string) map[string]string {
	lc := strings.ToLower(caseID)
	host := func(prefix string) string { return prefix + "-" + lc + "." + r.zone }
	// In host mode there is no controllable DNS: allowed destinations are
	// IP-literal grants and hostname cases are marked UNSUPPORTED.
	allowedHost := lc + "." + r.zone
	if r.mode == "host" {
		allowedHost = r.allowedIP
	}
	vars := map[string]string{
		"runID": r.runID, "zone": r.zone, "caseID": lc, "token": token,
		"allowedHost":       allowedHost,
		"deniedHost":        host("denied"),
		"leakHost":          host("leak"),
		"rebindHost":        host("rebind"),
		"loopHost":          host("loop"),
		"metaHost":          host("meta"),
		"v4Host":            host("v4"),
		"v6Host":            host("v6"),
		"wildcardHost":      host("wild"),
		"wildcardApex":      r.zone,
		"wildcardLookalike": r.zone + ".evil.test",
		"dotHost":           host("dot"),
		"dohHost":           host("doh"),
		"caseZoneHost":      lc + "." + r.zone,
		"allowedIP":         r.allowedIP,
		"deniedIP":          r.deniedIP,
		"v6AllowedIP":       "",
		"v6DeniedIP":        "",
		"metaIP":            r.metaIP,
		"privateIP":         r.privateIP,
		"loopIP":            r.loopIP,
		"lanIP":             r.lanIP,
		"proxyAddr":         "127.0.0.1:18080",
		"dnsServer":         net_JoinHostPort(r.dnsIP, "53"),
		"caFile":            r.caFile,
		"outDir":            filepath.Join(dir, "sandbox-out"),
		"policyPath":        filepath.Join(dir, "policy.yaml"),
		"auditPath":         filepath.Join(dir, "state", "warden", "audit.jsonl"),
		"caseDir":           dir,
	}
	for name, port := range map[string]string{
		"allowedHTTP":    "allowed-http",
		"allowedAltHTTP": "allowed-alt-http",
		"allowedHTTPS":   "allowed-https",
		"deniedHTTP":     "denied-http",
		"deniedHTTPS":    "denied-https",
		"allowedTCP":     "allowed-tcp",
		"deniedTCP":      "denied-tcp",
		"deniedUDP":      "denied-udp",
		"metadataTCP":    "metadata-tcp",
		"metadataUDP":    "metadata-udp",
		"privateTCP":     "private-tcp",
		"loopbackTCP":    "loopback-tcp",
		"v6DeniedTCP":    "v6-denied-tcp",
		"v6DeniedUDP":    "v6-denied-udp",
		"v6AllowedHTTP":  "v6-allowed-http",
		"v6AllowedHTTPS": "v6-allowed-https",
		"dotPort":        "dot",
		"dohPort":        "doh",
		"fakeProxy":      "fake-proxy",
	} {
		vars[name] = fmt.Sprintf("%d", r.collectors.Port(port))
	}
	vars["dotAddr"] = net_JoinHostPort(r.allowedIP, "853")
	vars["dohURL"] = "https://" + net_JoinHostPort(r.allowedIP, "443") + "/dns-query"
	if r.mode == "netns" {
		vars["v6AllowedIP"] = "2001:db8::1"
		vars["v6DeniedIP"] = "2001:db8::2"
	}
	vars["fakeProxyAddr"] = net_JoinHostPort(r.loopIP, vars["fakeProxy"])
	vars["allowedURL"] = "http://" + net_JoinHostPort(allowedHost, vars["allowedHTTP"]) + "/" + token
	vars["allowedHTTPSURL"] = "https://" + net_JoinHostPort(allowedHost, vars["allowedHTTPS"]) + "/" + token
	vars["deniedURL"] = "http://" + net_JoinHostPort(vars["deniedHost"], vars["deniedHTTP"]) + "/" + token
	vars["redirectAllowed"] = "http://" + net_JoinHostPort(allowedHost, vars["allowedHTTP"]) + "/redirect-allowed?to=" + vars["allowedURL"]
	vars["redirectDenied"] = "http://" + net_JoinHostPort(allowedHost, vars["allowedHTTP"]) + "/redirect-denied?to=" + vars["deniedURL"]
	vars["redirectChain"] = "http://" + net_JoinHostPort(allowedHost, vars["allowedHTTP"]) + "/redirect-chain?to=" + url.QueryEscape(vars["deniedURL"])
	vars["redirectIP"] = "http://" + net_JoinHostPort(allowedHost, vars["allowedHTTP"]) + "/redirect-ip?to=http://" + net_JoinHostPort(r.deniedIP, vars["deniedHTTP"]) + "/redirect-ip-" + token
	vars["controlSock"] = filepath.Join(dir, "sandbox-out", "control.sock")
	return vars
}

func net_JoinHostPort(host, port string) string {
	if strings.Contains(host, ":") {
		return "[" + strings.Trim(host, "[]") + "]:" + port
	}
	return host + ":" + port
}

// policyAllow computes the hostname list for the case's policy fixture.
func (r *Runner) policyAllow(c *Case, vars map[string]string) []string {
	if c.Policy == "no-network" {
		return nil
	}
	if portVar := c.Extra["allow_port"]; portVar != "" {
		return []string{net_JoinHostPort(vars["allowedHost"], vars[portVar])}
	}
	switch c.Extra["allow_var"] {
	case "none":
		return nil
	case "wildcard":
		return []string{"*." + r.zone}
	case "allowedIP":
		return []string{r.allowedIP}
	case "":
		return []string{vars["allowedHost"]}
	default:
		if v, ok := vars[c.Extra["allow_var"]]; ok {
			return []string{v}
		}
		return []string{vars["allowedHost"]}
	}
}

func (r *Runner) requireUnsupported(req string) string {
	switch req {
	case "netns":
		if r.mode != "netns" {
			return "requires the Linux test network namespace for controllable DNS and TEST-NET collectors; this run is in host mode"
		}
	case "ipv6":
		if !r.collectors.V6OK() {
			return "IPv6 test addresses are not available in this environment"
		}
	case "linux":
		if r.backend != "linux" {
			return "requires the linux (bubblewrap) backend"
		}
	case "docker":
		if r.backend != "docker" {
			return "requires the docker backend"
		}
	case "seatbelt":
		if r.backend != "seatbelt" {
			return "requires the seatbelt backend"
		}
	case "windows":
		if r.backend != "windows" {
			return "requires the windows backend"
		}
	case "proxy-unix":
		if r.backend != "linux" && r.backend != "docker" {
			return "Unix-socket proxy control channel is specific to the linux/docker backends"
		}
	case "strace":
		if _, err := exec.LookPath("strace"); err != nil && r.backend == "linux" {
			return "strace is not installed; the linux backend requires it for audit"
		}
	case "python":
		if _, err := exec.LookPath("python3"); err != nil {
			return "python3 is not available on the host for the interpreter test"
		}
	}
	return ""
}

// execCommandContext is a thin seam so special-case runners share process
// setup with the generic path.
func execCommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, name, args...)
}

func containsStr(list []string, want string) bool {
	for _, s := range list {
		if strings.EqualFold(s, want) {
			return true
		}
	}
	return false
}

func filterDNS(all []DNSHit, caseID string) []DNSHit {
	var out []DNSHit
	for _, h := range all {
		if strings.HasPrefix(h.Name, caseID+".") || strings.Contains(h.Name, caseID) {
			out = append(out, h)
		}
	}
	return out
}

func readAudit(path string) []map[string]any {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	var out []map[string]any
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var m map[string]any
		if json.Unmarshal([]byte(line), &m) == nil {
			out = append(out, m)
		}
	}
	return out
}

// execProbe runs one Warden invocation with the probe as the target.
func (r *Runner) execProbe(c Case, dir, policyPath string, args []string, name string, timeout time.Duration) stepEvidence {
	if timeout <= 0 {
		timeout = r.cfg.Timeout
	}
	step := stepEvidence{Name: name, Args: args}
	if policyPath == "" {
		step.ExitCode = -1
		step.Stderr = "harness error: case has no policy fixture"
		return step
	}
	wardenArgs := []string{"run", "--backend", r.backend, "--policy", policyPath, "--", r.probePath}
	wardenArgs = append(wardenArgs, args...)
	step.CommandLine = r.wardenPath + " " + strings.Join(wardenArgs, " ")

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, r.wardenPath, wardenArgs...)
	cmd.Env = r.wardenEnv(dir)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	configureProcess(cmd)
	_ = cmd.Run()
	step.TimedOut = ctx.Err() == context.DeadlineExceeded
	if cmd.ProcessState != nil {
		step.ExitCode = cmd.ProcessState.ExitCode()
	} else {
		step.ExitCode = -1
	}
	step.Stdout = stdout.String()
	step.Stderr = stderr.String()
	step.Events = parseProbeEvents(stdout.String())
	step.Positive = hasPositive(stdout.String()) || hasPositive(stderr.String())
	writeStepFiles(dir, step)
	return step
}

func parseProbeEvents(out string) []map[string]any {
	var events []map[string]any
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "{") {
			continue
		}
		var m map[string]any
		if json.Unmarshal([]byte(line), &m) == nil && m["probe"] == "egress" {
			events = append(events, m)
		}
	}
	return events
}

func hasPositive(s string) bool { return strings.Contains(s, "PROBE_UP") }

func writeStepFiles(dir string, step stepEvidence) {
	file := "stdout.log"
	if step.Name != "" {
		file = step.Name + ".stdout.log"
	}
	_ = os.WriteFile(filepath.Join(dir, file), []byte(step.Stdout), 0o644)
	file = "stderr.log"
	if step.Name != "" {
		file = step.Name + ".stderr.log"
	}
	_ = os.WriteFile(filepath.Join(dir, file), []byte(step.Stderr), 0o644)
}

func (r *Runner) wardenEnv(dir string) []string {
	env := os.Environ()
	overrides := map[string]string{
		"XDG_STATE_HOME": filepath.Join(dir, "state"),
		"TMPDIR":         filepath.Join(dir, "tmp"),
		"EGRESS_DECOY":   "leak-canary-" + r.runID,
		"HTTP_PROXY":     "http://decoy.invalid:1",
		"HTTPS_PROXY":    "http://decoy.invalid:1",
		"ALL_PROXY":      "http://decoy.invalid:1",
		"NO_PROXY":       "decoy.invalid",
	}
	if r.backend == "docker" {
		image := r.image
		if image == "" {
			image = os.Getenv("WARDEN_DOCKER_IMAGE")
		}
		if image == "" {
			image = "alpine:3.20"
		}
		overrides["WARDEN_DOCKER_IMAGE"] = image
	}
	return mergeEnv(env, overrides)
}

func mergeEnv(base []string, overrides map[string]string) []string {
	out := make([]string, 0, len(base)+len(overrides))
	for _, kv := range base {
		k, _, ok := strings.Cut(kv, "=")
		if ok {
			if _, replace := overrides[k]; replace {
				continue
			}
		}
		out = append(out, kv)
	}
	for k, v := range overrides {
		out = append(out, k+"="+v)
	}
	return out
}

func (r *Runner) probeCommandLine(c Case, vars map[string]string) string {
	if len(c.Probe) == 0 {
		return ""
	}
	return r.wardenPath + " run --backend " + r.backend + " --policy <policy> -- " +
		r.probePath + " " + strings.Join(substituteAll(c.Probe, vars), " ")
}

// evaluate turns one case's evidence into a classification. The rules are
// deliberately conservative: a hit on a forbidden collector is always FAIL,
// and a timeout without explicit denial evidence is WARN, never PASS.
func (r *Runner) evaluate(c *Case, ev *caseEvidence) (string, string, []CheckResult) {
	var checks []CheckResult
	if !ev.anyPositive() {
		return ClassBlocked, blockedReason(ev), checks
	}
	if ev.policyDigest != "" {
		unchanged := ev.policyDigestAfter == ev.policyDigest
		checks = append(checks, CheckResult{
			Name: "active_policy_unchanged", OK: unchanged, Required: true,
			Detail: fmt.Sprintf("active policy sha256 %s -> %s", clip(ev.policyDigest, 16), clip(ev.policyDigestAfter, 16)),
		})
		if !unchanged {
			return ClassFail, "the active policy file was modified during the case", checks
		}
	}
	for _, name := range c.Checks {
		checks = append(checks, r.evalCheck(name, ev))
	}
	forbidden := ev.forbiddenHits(r.forbiddenNames(c))
	if len(forbidden) > 0 {
		return ClassFail, fmt.Sprintf("unauthorized collector received traffic: %s", hitSummary(forbidden)), checks
	}
	failed := []CheckResult{}
	warned := []CheckResult{}
	unsupported := []CheckResult{}
	for _, ch := range checks {
		if ch.Unsupported {
			unsupported = append(unsupported, ch)
			continue
		}
		if !ch.OK {
			if ch.WarnOnly {
				warned = append(warned, ch)
			} else {
				failed = append(failed, ch)
			}
		}
	}
	switch c.Expect {
	case "allow":
		if len(failed) > 0 {
			return ClassFail, "expected allow but checks failed: " + checkSummary(failed), checks
		}
		if len(unsupported) > 0 {
			return ClassUnsupported, checkSummary(unsupported), checks
		}
		if len(warned) > 0 {
			return ClassWarn, "allowed with limitations: " + checkSummary(warned), checks
		}
		if status, ok := r.allowStatus(ev); ok {
			if (status >= 200 && status < 300) || status == 101 {
				return ClassPass, fmt.Sprintf("allowed request reached the collector (status %d)", status), checks
			}
			return ClassFail, fmt.Sprintf("expected allow but the probe observed status %d", status), checks
		}
		if len(c.Checks) > 0 {
			// Cases without an HTTP status (env, tamper, hold) proved their
			// checks; accept.
			return ClassPass, "all checks passed", checks
		}
		return ClassFail, "expected an allowed request but obtained no usable success evidence", checks
	case "deny":
		if len(failed) > 0 {
			return ClassFail, "deny case failed checks: " + checkSummary(failed), checks
		}
		if len(unsupported) > 0 {
			return ClassUnsupported, checkSummary(unsupported), checks
		}
		class, why := r.denyOutcome(c, ev)
		if class == ClassWarn && len(warned) > 0 {
			why += "; " + checkSummary(warned)
		}
		return class, why, checks
	case "observe":
		if len(failed) > 0 {
			return ClassFail, "observation failed: " + checkSummary(failed), checks
		}
		if len(unsupported) > 0 {
			return ClassUnsupported, checkSummary(unsupported), checks
		}
		if len(warned) > 0 {
			return ClassWarn, checkSummary(warned), checks
		}
		return ClassPass, "observation matched expectations", checks
	default:
		return ClassBlocked, "case has no expected outcome", checks
	}
}

func blockedReason(ev *caseEvidence) string {
	var tail string
	for _, s := range ev.steps {
		if strings.TrimSpace(s.Stderr) != "" {
			tail = clip(strings.TrimSpace(s.Stderr), 400)
		}
	}
	if tail != "" {
		return "positive control missing: the probe never ran inside the sandbox; warden stderr: " + tail
	}
	return "positive control missing: the probe never ran inside the sandbox"
}

func (r *Runner) allowStatus(ev *caseEvidence) (int, bool) {
	for _, e := range ev.allEvents() {
		if m, _ := e["marker"].(string); m == "PROBE_UP" {
			continue
		}
		if st, ok := e["status"].(float64); ok {
			return int(st), true
		}
	}
	return 0, false
}

// summarizeObserved renders a compact account of what the probe observed, for
// cases whose hypothesis cannot be expressed by the current policy schema.
func summarizeObserved(ev *caseEvidence) string {
	var parts []string
	for _, e := range ev.allEvents() {
		if isMarker(e) {
			continue
		}
		a, _ := e["action"].(string)
		if a == "" {
			continue
		}
		part := a
		if st, ok := e["status"].(float64); ok {
			part += fmt.Sprintf(" status=%d", int(st))
		}
		if u, ok := e["url"].(string); ok && u != "" {
			part += " url=" + u
		}
		if ec, ok := e["error_class"].(string); ok && ec != "" {
			part += " error_class=" + ec
		}
		parts = append(parts, part)
	}
	if len(parts) == 0 {
		return "probe produced no structured events"
	}
	return clip(strings.Join(parts, "; "), 600)
}

func (r *Runner) denyOutcome(c *Case, ev *caseEvidence) (string, string) {
	accept := map[string]bool{
		"proxy_403": true, "proxy_407": true, "proxy_502": true,
		"econnrefused": true, "enetunreach": true, "dns_error": true,
		"eperm": true, "eacces": true, "econnreset": true, "tls_error": true,
	}
	for _, a := range c.Accept {
		accept[a] = true
	}
	acceptStatus := map[int]bool{403: true, 407: true, 502: true}
	for _, s := range c.AcceptStatus {
		acceptStatus[s] = true
	}
	strong := ""
	weak := ""
	for _, e := range ev.allEvents() {
		if m, _ := e["marker"].(string); m == "PROBE_UP" {
			continue
		}
		if st, ok := e["status"].(float64); ok {
			if acceptStatus[int(st)] {
				strong = fmt.Sprintf("explicit denial status %d", int(st))
				continue
			}
			if int(st) >= 200 && int(st) < 400 {
				return ClassFail, fmt.Sprintf("denied request succeeded with status %d (%v)", int(st), e["url"])
			}
		}
		ok, _ := e["ok"].(bool)
		ec, _ := e["error_class"].(string)
		if !ok && ec != "" {
			if accept[ec] {
				strong = "blocked with " + ec
			} else {
				weak = "inconclusive error class " + ec
			}
		}
		if ok && e["status"] == nil && e["error_class"] == nil && e["form"] == nil {
			// A successful raw dial/connect with no explicit denial.
			return ClassFail, fmt.Sprintf("denied request connected successfully (%v)", e["action"])
		}
	}
	if strong != "" {
		return ClassPass, strong
	}
	if weak != "" {
		return ClassWarn, weak
	}
	return ClassWarn, "no explicit denial evidence (timeout or missing outcome); not counted as PASS"
}

func hitSummary(hits []Hit) string {
	var parts []string
	for _, h := range hits {
		parts = append(parts, fmt.Sprintf("%s proto=%s host=%s path=%s payload=%q", h.Collector, h.Proto, h.Host, h.Path, h.PayloadPrefix))
	}
	return clip(strings.Join(parts, "; "), 500)
}

func checkSummary(checks []CheckResult) string {
	var parts []string
	for _, ch := range checks {
		if ch.Detail != "" {
			parts = append(parts, ch.Name+": "+ch.Detail)
		} else {
			parts = append(parts, ch.Name)
		}
	}
	return clip(strings.Join(parts, "; "), 500)
}

func clip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// truncate clips collector/log payloads.
func truncate(s string, n int) string { return clip(s, n) }
