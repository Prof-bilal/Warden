package main

import (
	"fmt"
	"net"
	"strings"
)

// evalCheck implements the named checks referenced from matrix.yaml. An
// unknown check name is reported as UNSUPPORTED; it is never silently
// treated as passing.
func (r *Runner) evalCheck(name string, ev *caseEvidence) CheckResult {
	res := CheckResult{Name: name, Required: true}
	hostOf := func(v string) string {
		if h, ok := ev.vars[v]; ok {
			return strings.ToLower(strings.TrimSuffix(h, "."))
		}
		return strings.ToLower(strings.TrimSuffix(v, "."))
	}
	switch {
	case name == "probe_status_2xx":
		res.OK = anyEvent(ev, func(e map[string]any) bool {
			st, ok := e["status"].(float64)
			return ok && int(st) >= 200 && int(st) < 300
		})
		res.Detail = "expected a 2xx probe response"
	case name == "probe_status_403":
		res.OK = anyEvent(ev, func(e map[string]any) bool {
			st, ok := e["status"].(float64)
			return ok && int(st) == 403
		})
		res.Detail = "expected a 403 from the proxy"
	case name == "probe_error":
		res.OK = anyEvent(ev, func(e map[string]any) bool {
			ok, has := e["ok"].(bool)
			return has && !ok
		})
		res.Detail = "expected the probe to fail"
	case strings.HasPrefix(name, "collector_hit:"):
		want := strings.TrimPrefix(name, "collector_hit:")
		count := countCollector(ev.hits, want)
		res.OK = count > 0
		res.Detail = fmt.Sprintf("collector %s hits=%d", want, count)
	case strings.HasPrefix(name, "collector_absent:"):
		want := strings.TrimPrefix(name, "collector_absent:")
		count := countCollector(ev.hits, want)
		res.OK = count == 0
		res.Detail = fmt.Sprintf("collector %s hits=%d (want 0)", want, count)
	case name == "no_forbidden":
		forbidden := ev.forbiddenHits(r.forbiddenNames(ev.c))
		res.OK = len(forbidden) == 0
		res.Detail = hitSummary(forbidden)
	case strings.HasPrefix(name, "dns_hit:"):
		h := hostOf(strings.TrimPrefix(name, "dns_hit:"))
		res.OK, res.Detail = dnsHas(ev, h)
	case strings.HasPrefix(name, "dns_absent:"):
		h := hostOf(strings.TrimPrefix(name, "dns_absent:"))
		ok, detail := dnsHas(ev, h)
		res.OK = !ok
		res.Detail = "queries for " + h + ": " + detail
	case name == "audit_allow":
		res.OK, res.Detail = auditHas(ev, true)
	case name == "audit_deny":
		res.OK, res.Detail = auditHas(ev, false)
	case name == "audit_present":
		res.OK = len(ev.audit) > 0
		res.Detail = fmt.Sprintf("%d audit events", len(ev.audit))
	case name == "env_proxy_set":
		res.OK, res.Detail = envProxySet(ev)
	case name == "env_no_decoy":
		res.OK, res.Detail = envNoDecoy(ev)
	case name == "control_write":
		res.OK, res.Detail = tamperAttempt(ev, "control_write")
	case name == "writes_denied":
		res.OK, res.Detail = tamperWritesDenied(ev)
	case name == "policy_read_closed":
		ok, detail := tamperAttempt(ev, "policy_read")
		res.OK = !ok
		res.Detail = "policy_read denied=" + fmt.Sprint(!ok) + " (" + detail + ")"
	case name == "unshare_denied":
		res.OK, res.Detail = unshareDenied(ev)
	case name == "post_unshare_denied":
		res.OK, res.Detail = postUnshareDenied(ev)
	case name == "kill_fail_closed":
		res.OK, res.WarnOnly, res.Detail = killFailClosed(ev)
	case name == "numeric_all_denied":
		res.OK, res.Detail = numericAllDenied(ev)
	case name == "variants_allowed":
		res.WarnOnly = true
		res.OK, res.Detail = variantsAllowed(ev)
	case name == "lookalike_denied":
		res.OK = strongDenialOr403(ev)
		res.Detail = "lookalike destination must be denied"
	case name == "wildcard_allowed":
		res.OK = countCollector(ev.hits, "allowed-http") > 0 || countCollector(ev.hits, "allowed-https") > 0
		res.Detail = "wildcard subdomain must reach the allowed collector"
	case name == "redirect_positive":
		res.OK = countCollector(ev.hits, "allowed-http") > 0
		res.Detail = "first redirect hop must reach the allowed collector"
	case name == "no_proxy_auth_upstream":
		res.WarnOnly = true
		res.OK, res.Detail = noProxyAuthUpstream(ev)
	case name == "host_mismatch_observed":
		res.WarnOnly = true
		res.OK, res.Detail = hostMismatchObserved(ev)
	case name == "ws_ok":
		res.OK = anyEvent(ev, func(e map[string]any) bool {
			st, ok := e["status"].(float64)
			return ok && int(st) == 101
		})
		res.Detail = "WebSocket upgrade must complete with 101"
	case name == "socks_rejected":
		res.OK = !anyEvent(ev, func(e map[string]any) bool {
			ok, has := e["socks_supported"].(bool)
			return has && ok
		})
		res.Detail = "proxy must not speak SOCKS"
	case name == "flood_denied":
		res.OK, res.Detail = floodDenied(ev)
	case name == "flood_allowed":
		res.OK, res.Detail = floodAllowed(ev)
	case name == "large_body":
		res.OK, res.Detail = largeBody(ev)
	case name == "stream_ok":
		res.OK = anyEvent(ev, func(e map[string]any) bool {
			n, ok := e["stream_lines"].(float64)
			return ok && int(n) > 0
		})
		res.Detail = "expected at least one streamed event"
	case name == "h2_negotiated":
		if anyEvent(ev, func(e map[string]any) bool {
			p, _ := e["proto"].(string)
			return p == "HTTP/2.0"
		}) {
			res.OK = true
			res.Detail = "HTTP/2 negotiated through the proxy"
		} else {
			res.Unsupported = true
			res.Detail = "HTTP/2 was not negotiated (collector is HTTP/1.1); HTTP/2 behavior not covered"
		}
	case name == "unix_granted_path_reachable":
		connected := anyEvent(ev, func(e map[string]any) bool {
			ok, has := e["connected"].(bool)
			return has && ok
		})
		res.OK = connected
		if connected {
			res.Detail = "AF_UNIX socket in a granted path is reachable, as intended: the filesystem grant is the local-IPC boundary, not the network policy"
		} else {
			res.Detail = "AF_UNIX socket in a granted path was not reachable; the filesystem grant did not provide the expected local IPC"
		}
	case name == "unix_proxy_denied":
		res.OK, res.Detail = unixProxyDenied(ev)
	case name == "second_proxy_denied":
		res.OK, res.Detail = secondProxyDenied(ev)
	case name == "interpreter_denied":
		res.OK, res.Detail = interpreterDenied(ev)
	case name == "resolver_fail_closed":
		res.OK, res.Detail = namedStepDenied(ev, "no-dns")
	case name == "collector_fail_closed":
		res.OK, res.Detail = namedStepDenied(ev, "no-collector")
	case name == "timeout_fail_closed":
		res.OK, res.Detail = timeoutFailClosed(ev)
	case name == "mcp_initialize":
		res.OK = anyEvent(ev, func(e map[string]any) bool {
			a, _ := e["action"].(string)
			b, _ := e["mcp_ok"].(bool)
			return a == "mcp-initialize" && b
		})
		res.Detail = "MCP initialize must succeed inside the sandbox"
	case name == "mcp_allowed_call":
		res.OK, res.Detail = mcpCall(ev, "mcp-allowed")
	case name == "mcp_denied_call":
		res.OK, res.Detail = mcpCall(ev, "mcp-denied")
	case name == "dns_rebind_denied":
		res.OK = strongDenialOr403(ev)
		res.Detail = "rebinding to loopback must not produce a usable connection"
	case name == "no_fake_proxy_reach":
		count := 0
		for _, h := range ev.hits {
			if h.FakeProxy {
				count++
			}
		}
		res.OK = count == 0
		res.Detail = fmt.Sprintf("fake proxy hits=%d (want 0)", count)
	case name == "env_tamper_denied":
		res.OK, res.WarnOnly, res.Detail = envTamperDenied(ev)
	case strings.HasPrefix(name, "extra_write_denied:"):
		p := strings.TrimPrefix(name, "extra_write_denied:")
		ok, detail := tamperAttempt(ev, "write:"+p)
		res.OK = !ok
		res.Detail = "write " + p + " denied=" + fmt.Sprint(!ok) + " (" + detail + ")"
	case name == "policy_replace_denied":
		res.OK, res.Detail = policyReplaceDenied(ev)
	case name == "exec_self_denied":
		res.OK, res.Detail = execSelfDenied(ev)
	case name == "interpreter_allowed":
		res.OK, res.Detail = interpreterAllowed(ev)
	case strings.HasPrefix(name, "dns_at_least:"):
		var want int
		fmt.Sscanf(strings.TrimPrefix(name, "dns_at_least:"), "%d", &want)
		res.OK = len(ev.dns) >= want
		res.Detail = fmt.Sprintf("dns queries=%d want>=%d", len(ev.dns), want)
	case name == "flood_ran":
		res.OK, res.Detail = floodRan(ev)
	case name == "inflight_lost":
		res.OK, res.Detail = inflightLost(ev)
	case name == "tls_floor_denied":
		res.OK, res.Detail = tlsFloorDenied(ev)
	case name == "tls_verify_failed":
		res.OK = anyEvent(ev, func(e map[string]any) bool {
			ec, _ := e["error_class"].(string)
			return ec == "tls_error"
		})
		res.Detail = "system-root validation must reject the test certificate"
	case name == "error_class_strong":
		res.OK = strongDenialOr403(ev)
		res.Detail = "expected an explicit denial or blocked transport error"
	case strings.HasPrefix(name, "step_allow:"):
		want := strings.TrimPrefix(name, "step_allow:")
		res.OK, res.Detail = stepOutcome(ev, want, "allow")
	case strings.HasPrefix(name, "step_denied:"):
		want := strings.TrimPrefix(name, "step_denied:")
		res.OK, res.Detail = stepOutcome(ev, want, "denied")
	case strings.HasPrefix(name, "step_timeout:"):
		want := strings.TrimPrefix(name, "step_timeout:")
		res.OK, res.Detail = stepOutcome(ev, want, "timeout")
	default:
		res.Required = false
		res.Unsupported = true
		res.Detail = "check not implemented by this runner; case cannot be judged"
	}
	return res
}

func countCollector(hits []Hit, name string) int {
	n := 0
	for _, h := range hits {
		if h.Collector == name {
			n++
		}
	}
	return n
}

func anyEvent(ev *caseEvidence, pred func(map[string]any) bool) bool {
	for _, e := range ev.allEvents() {
		if pred(e) {
			return true
		}
	}
	return false
}

func dnsHas(ev *caseEvidence, name string) (bool, string) {
	var names []string
	for _, q := range ev.dns {
		names = append(names, q.Name+"/"+q.QType)
		if strings.EqualFold(strings.TrimSuffix(q.Name, "."), strings.TrimSuffix(name, ".")) {
			return true, q.Name + "/" + q.QType + " rcode=" + fmt.Sprint(q.RCode)
		}
	}
	return false, "no matching query (observed: " + strings.Join(names, ",") + ")"
}

func auditHas(ev *caseEvidence, allowed bool) (bool, string) {
	for _, a := range ev.audit {
		b, has := a["allowed"].(bool)
		if !has || b != allowed {
			continue
		}
		if t, _ := a["type"].(string); t != "network" {
			continue
		}
		action, _ := a["action"].(string)
		// The proxy logs allowed decisions as action=connect and denials with
		// the request method (GET, CONNECT, ...). Strace's raw connect events
		// are filtered out by requiring the proxy's allow action.
		if allowed && !strings.EqualFold(action, "connect") {
			continue
		}
		resource, _ := a["resource"].(string)
		if resource == "" || strings.Contains(strings.ToLower(resource), ev.caseID) ||
			strings.Contains(strings.ToLower(resource), strings.ToLower(ev.vars["allowedHost"])) {
			return true, fmt.Sprintf("audit %v resource=%s action=%v", b, resource, a["action"])
		}
	}
	return false, "no matching audit event"
}

func envProxySet(ev *caseEvidence) (bool, string) {
	for _, e := range ev.allEvents() {
		if a, _ := e["action"].(string); a != "env" {
			continue
		}
		m, _ := e["env"].(map[string]any)
		get := func(k string) string {
			if v, ok := m[k].(string); ok {
				return v
			}
			return ""
		}
		http := get("HTTP_PROXY")
		https := get("HTTPS_PROXY")
		all := get("ALL_PROXY")
		no := get("NO_PROXY")
		ok := http == "http://127.0.0.1:18080" && https == "http://127.0.0.1:18080" &&
			all == "http://127.0.0.1:18080" && no == ""
		return ok, fmt.Sprintf("HTTP_PROXY=%q HTTPS_PROXY=%q ALL_PROXY=%q NO_PROXY=%q", http, https, all, no)
	}
	return false, "probe did not report its environment"
}

func envNoDecoy(ev *caseEvidence) (bool, string) {
	for _, e := range ev.allEvents() {
		if isMarker(e) {
			continue
		}
		if a, _ := e["action"].(string); a != "env" {
			continue
		}
		m, _ := e["env"].(map[string]any)
		if v, ok := m["EGRESS_DECOY"]; ok && v != "" {
			return false, "decoy environment variable leaked into the sandbox"
		}
		if v, ok := m["HTTP_PROXY"].(string); ok && strings.Contains(v, "decoy.invalid") {
			return false, "parent proxy environment leaked into the sandbox"
		}
		return true, "no decoy or parent proxy variables present"
	}
	return false, "probe did not report its environment"
}

func tamperEvent(ev *caseEvidence) map[string]any {
	for _, e := range ev.allEvents() {
		if isMarker(e) {
			continue
		}
		if a, _ := e["action"].(string); a == "tamper" {
			return e
		}
	}
	return nil
}

func tamperAttempt(ev *caseEvidence, name string) (bool, string) {
	ok, _, detail := attemptStatus(ev, name)
	return ok, detail
}

// attemptStatus reports whether a named tamper attempt exists and, if so,
// whether it succeeded. Distinguishing "missing" from "denied" is important:
// a missing attempt is never evidence of enforcement.
func attemptStatus(ev *caseEvidence, name string) (ok, exists bool, detail string) {
	e := tamperEvent(ev)
	if e == nil {
		return false, false, "probe did not run the tamper action"
	}
	attempts, _ := e["attempts"].([]any)
	for _, a := range attempts {
		m, _ := a.(map[string]any)
		if n, _ := m["attempt"].(string); n == name {
			ok, _ := m["ok"].(bool)
			errStr, _ := m["error"].(string)
			return ok, true, fmt.Sprintf("attempt %s ok=%v error=%s", name, ok, clip(errStr, 120))
		}
	}
	return false, false, "attempt " + name + " missing"
}

func tamperWritesDenied(ev *caseEvidence) (bool, string) {
	var details []string
	denied := true
	for _, name := range []string{"policy_write", "audit_write"} {
		ok, exists, detail := attemptStatus(ev, name)
		details = append(details, detail)
		if !exists || ok {
			denied = false
		}
	}
	// Rename can only be attempted if the temp file was creatable. If temp
	// creation itself was denied, replacement is impossible by construction.
	if ok, exists, detail := attemptStatus(ev, "policy_replace"); exists {
		details = append(details, detail)
		if ok {
			denied = false
		}
	} else if tok, texists, tdetail := attemptStatus(ev, "policy_replace_temp"); texists {
		details = append(details, tdetail)
		if tok {
			denied = false
			details = append(details, "temp policy write succeeded but replace was not attempted")
		}
	} else {
		details = append(details, detail+" and "+tdetail)
		denied = false
	}
	return denied, strings.Join(details, "; ")
}

func policyReplaceDenied(ev *caseEvidence) (bool, string) {
	ok, exists, detail := attemptStatus(ev, "policy_replace")
	if exists {
		return !ok, detail
	}
	tok, texists, tdetail := attemptStatus(ev, "policy_replace_temp")
	if texists && !tok {
		return true, "replacement impossible: temp write denied (" + tdetail + ")"
	}
	return false, "replace attempt missing and temp write not shown denied (" + tdetail + ")"
}

func envTamperDenied(ev *caseEvidence) (bool, bool, string) {
	e := findEvent(ev, "env-tamper")
	if e == nil {
		return false, false, "probe did not run env-tamper"
	}
	strong := func(ec string) bool {
		switch ec {
		case "proxy_403", "proxy_407", "proxy_502", "econnrefused", "enetunreach", "dns_error", "eperm", "eacces", "econnreset", "tls_error":
			return true
		}
		return false
	}
	var details []string
	sawStrong, sawTimeout := false, false
	if ok, _ := e["direct_ok"].(bool); ok {
		return false, false, "direct dial to the denied address succeeded after the environment override"
	}
	if st, ok := e["proxy_status"].(float64); ok {
		if int(st) < 400 {
			return false, false, fmt.Sprintf("request via the attacker-controlled proxy returned status %d", int(st))
		}
		details = append(details, fmt.Sprintf("proxy_status=%d", int(st)))
		sawStrong = true
	}
	for _, key := range []string{"direct_error_class", "proxy_error_class"} {
		ec, _ := e[key].(string)
		if ec == "" {
			continue
		}
		details = append(details, key+"="+ec)
		switch {
		case strong(ec):
			sawStrong = true
		case ec == "timeout":
			sawTimeout = true
		default:
			return false, false, "inconclusive error class " + ec + " for " + key
		}
	}
	if len(details) == 0 {
		return false, false, "no attempt recorded"
	}
	if !sawStrong {
		if sawTimeout {
			return false, true, strings.Join(details, ", ")
		}
		return false, false, strings.Join(details, ", ")
	}
	return true, false, strings.Join(details, ", ")
}

func execSelfDenied(ev *caseEvidence) (bool, string) {
	e := findEvent(ev, "exec-self")
	if e == nil {
		return false, "probe did not run exec-self"
	}
	child, _ := e["child_events"].([]any)
	if len(child) == 0 {
		return false, "child process emitted no probe events"
	}
	for _, c := range child {
		m, _ := c.(map[string]any)
		if strongEvent(m) {
			return true, fmt.Sprintf("child action %v was denied (%v)", m["action"], m["error_class"])
		}
		if st, ok := m["status"].(float64); ok && int(st) >= 200 && int(st) < 300 {
			return false, fmt.Sprintf("child reached a destination with status %d", int(st))
		}
	}
	return false, "child events show no explicit denial"
}

func interpreterAllowed(ev *caseEvidence) (bool, string) {
	e := findEvent(ev, "exec")
	if e == nil {
		return false, "probe did not run exec"
	}
	out, _ := e["output"].(string)
	if strings.Contains(out, "ALLOWED-OK") {
		return true, "interpreter reached the allowlisted destination"
	}
	return false, "interpreter did not reach the allowlisted destination: " + clip(out, 200)
}

func floodRan(ev *caseEvidence) (bool, string) {
	e := findEvent(ev, "flood")
	if e == nil {
		return false, "probe did not run flood"
	}
	n, _ := e["requests"].(float64)
	counts, _ := e["counts"].(map[string]any)
	if int(n) < 1 || len(counts) == 0 {
		return false, fmt.Sprintf("requests=%v counts=%v", n, counts)
	}
	return true, fmt.Sprintf("%d requests exercised: %v", int(n), counts)
}

func inflightLost(ev *caseEvidence) (bool, string) {
	for _, s := range ev.steps {
		if s.Name != "inflight" {
			continue
		}
		if s.TimedOut {
			return false, "in-flight request outlived the supervisor kill (timeout)"
		}
		if s.ExitCode != 0 {
			return true, fmt.Sprintf("workload exited %d after the supervisor was killed", s.ExitCode)
		}
		if anyEventIn(s.Events, func(e map[string]any) bool {
			if ok, has := e["ok"].(bool); has && !ok {
				return true
			}
			_, has := e["stream_error"]
			return has
		}) {
			return true, "in-flight request ended in an error after the supervisor was killed"
		}
		return false, "in-flight request completed successfully despite the supervisor being killed"
	}
	return false, "step inflight missing"
}

func unshareDenied(ev *caseEvidence) (bool, string) {
	e := findEvent(ev, "unshare")
	if e == nil {
		return false, "probe did not run the unshare action"
	}
	if ok, _ := e["unshare_ok"].(bool); !ok {
		return true, "unshare(CLONE_NEWNET) was refused"
	}
	if post, has := e["post_unshare_ok"].(bool); has && !post {
		return true, "unshare succeeded but the new namespace has no route to the denied destination"
	}
	return false, "unshare succeeded and post-unshare egress was not shown to be blocked"
}

func postUnshareDenied(ev *caseEvidence) (bool, string) {
	e := findEvent(ev, "unshare")
	if e == nil {
		return false, "probe did not run the unshare action"
	}
	post, has := e["post_unshare_ok"].(bool)
	if !has {
		return false, "post-unshare dial was not attempted"
	}
	return !post, fmt.Sprintf("post_unshare_ok=%v", post)
}

func findEvent(ev *caseEvidence, action string) map[string]any {
	for _, e := range ev.allEvents() {
		if isMarker(e) {
			continue
		}
		if a, _ := e["action"].(string); a == action {
			return e
		}
	}
	return nil
}

func killFailClosed(ev *caseEvidence) (bool, bool, string) {
	signalPhase := func(e map[string]any) string {
		a, _ := e["action"].(string)
		if a != "kill-parent" && a != "signal-parent" {
			return ""
		}
		p, _ := e["phase"].(string)
		return p
	}
	signalled := anyEvent(ev, func(e map[string]any) bool { return signalPhase(e) == "signal" })
	attempted := signalled || anyEvent(ev, func(e map[string]any) bool { return signalPhase(e) == "before" })
	if !attempted {
		return false, false, "the probe never signalled the enforcement process"
	}
	parent := findPhase(ev, "after-parent")
	if parent == nil {
		// Two fail-closed teardown signatures are accepted, both requiring a
		// non-zero workload exit so a surviving workload is never a PASS:
		//
		//  1. The probe recorded the kill(2) result and then died. On Linux the
		//     in-sandbox bridge is PID 1 of the sandbox PID namespace; when it
		//     dies the kernel terminates every process in the namespace before
		//     the probe can emit after-parent.
		//  2. The probe emitted the pre-signal phase but was torn down before
		//     it could record the kill(2) result. On Docker the in-container
		//     bridge is the probe's parent; when the bridge dies the container
		//     stops and every process in it, including the probe, is killed
		//     first. The pre-signal event plus a non-zero workload exit is the
		//     same evidence, observed slightly earlier in the sequence.
		if step := signalStep(ev); step != nil && step.ExitCode != 0 {
			return true, false, fmt.Sprintf("the workload was terminated (exit %d) when the enforcement process received the signal; no traffic survived (fail closed)", step.ExitCode)
		}
		if step := preSignalStep(ev); step != nil && step.ExitCode != 0 {
			return true, false, fmt.Sprintf("the workload was terminated (exit %d) when the enforcement process was signalled, before the kill result could be recorded; no traffic survived (fail closed)", step.ExitCode)
		}
		return false, true, "the signal was sent but the workload's fate could not be determined; fail-closed behavior is not proven"
	}
	if unknown, _ := parent["alive_unknown"].(bool); unknown {
		return false, true, "the signal was sent but parent liveness could not be determined; fail-closed behavior is not proven"
	}
	alive, _ := parent["parent_alive"].(bool)
	if alive {
		return false, true, "the signalled proxy process is still alive (the sandbox PID-namespace init ignores signals it cannot handle); traffic stayed proxied and policy-conforming, but enforcement death was not exercised here (see F15/G9)"
	}
	var details []string
	for _, e := range ev.allEvents() {
		a, _ := e["action"].(string)
		p, _ := e["phase"].(string)
		if (a != "kill-parent" && a != "signal-parent") || !strings.HasPrefix(p, "after") || p == "after-parent" {
			continue
		}
		ok, _ := e["ok"].(bool)
		st, hasSt := e["status"].(float64)
		ec, _ := e["error_class"].(string)
		details = append(details, fmt.Sprintf("%s ok=%v status=%v err=%s", p, ok, e["status"], ec))
		if ok && (!hasSt || (int(st) >= 200 && int(st) < 400)) {
			return false, false, "traffic still succeeded after the enforcement process died: " + strings.Join(details, "; ")
		}
	}
	if len(details) == 0 {
		return false, false, "no post-death probe was recorded"
	}
	return true, false, "the proxy process died and no post-death success was observed (" + strings.Join(details, "; ") + ")"
}

// signalStep returns the step that emitted the signal phase.
func signalStep(ev *caseEvidence) *stepEvidence {
	for i := range ev.steps {
		for _, e := range ev.steps[i].Events {
			a, _ := e["action"].(string)
			p, _ := e["phase"].(string)
			if (a == "kill-parent" || a == "signal-parent") && p == "signal" {
				return &ev.steps[i]
			}
		}
	}
	return nil
}

// preSignalStep returns the step that emitted the probe's pre-signal phase.
// It is used when the probe was torn down before it could record the kill(2)
// result, which happens when the signalled process is the probe's container
// parent and its death stops the whole container.
func preSignalStep(ev *caseEvidence) *stepEvidence {
	for i := range ev.steps {
		for _, e := range ev.steps[i].Events {
			a, _ := e["action"].(string)
			p, _ := e["phase"].(string)
			if (a == "kill-parent" || a == "signal-parent") && p == "before" {
				return &ev.steps[i]
			}
		}
	}
	return nil
}

func findPhase(ev *caseEvidence, phase string) map[string]any {
	for _, e := range ev.allEvents() {
		if p, _ := e["phase"].(string); p == phase {
			return e
		}
	}
	return nil
}

func findStep(ev *caseEvidence, name string) *stepEvidence {
	for i := range ev.steps {
		if ev.steps[i].Name == name {
			return &ev.steps[i]
		}
	}
	return nil
}

func unixProxyDenied(ev *caseEvidence) (bool, string) {
	step := findStep(ev, "proxy-control-socket")
	if step == nil {
		return false, "step proxy-control-socket missing"
	}
	for _, e := range step.Events {
		if st, ok := e["status"].(float64); ok && int(st) == 403 {
			return true, "the proxy Unix control socket denied the non-allowlisted host with 403"
		}
	}
	return false, "no 403 observed from the proxy Unix control socket: " + clip(step.Stderr, 160)
}

func numericAllDenied(ev *caseEvidence) (bool, string) {
	events := ev.events("numeric")
	if len(events) == 0 {
		return false, "probe did not run the numeric action"
	}
	var details []string
	for _, e := range events {
		form, _ := e["form"].(string)
		ok, _ := e["ok"].(bool)
		st, hasSt := e["status"].(float64)
		if ok && hasSt && int(st) >= 200 && int(st) < 400 {
			return false, fmt.Sprintf("numeric form %s reached a destination with status %d", form, int(st))
		}
		details = append(details, fmt.Sprintf("%s=%v/%v", form, ok, e["status"]))
	}
	return true, strings.Join(details, ", ")
}

func variantsAllowed(ev *caseEvidence) (bool, string) {
	events := ev.events("host-variants")
	if len(events) == 0 {
		return false, "probe did not run host variants"
	}
	var bad []string
	for _, e := range events {
		form, _ := e["form"].(string)
		ok, _ := e["ok"].(bool)
		st, hasSt := e["status"].(float64)
		if !ok || !hasSt || int(st) < 200 || int(st) >= 300 {
			bad = append(bad, fmt.Sprintf("%s(status=%v,err=%v)", form, e["status"], e["error_class"]))
		}
	}
	if len(bad) > 0 {
		return false, "variants not accepted by the proxy (over-blocking, not a bypass): " + strings.Join(bad, ", ")
	}
	return true, "canonical, uppercase, mixed-case and trailing-dot forms all reached the collector"
}

func strongDenialOr403(ev *caseEvidence) bool {
	return anyEvent(ev, strongEvent)
}

func noProxyAuthUpstream(ev *caseEvidence) (bool, string) {
	for _, h := range ev.hits {
		if h.Proto != "http" && h.Proto != "https" {
			continue
		}
		if vals, ok := h.Headers["Proxy-Authorization"]; ok && len(vals) > 0 {
			return false, "Proxy-Authorization header was forwarded to the upstream collector"
		}
	}
	return true, "no Proxy-Authorization header observed upstream"
}

// hostNoPort normalizes a Host header or policy hostname for comparison:
// lowercase, strip a trailing dot, and drop an optional port. Collector hits
// record the Host header as sent on the wire, which includes the ephemeral
// destination port; the policy value is the hostname only.
func hostNoPort(h string) string {
	if host, _, err := net.SplitHostPort(h); err == nil {
		h = host
	}
	return strings.ToLower(strings.TrimSuffix(h, "."))
}

func hostMismatchObserved(ev *caseEvidence) (bool, string) {
	allowed := hostNoPort(ev.vars["allowedHost"])
	for _, h := range ev.hits {
		if h.Proto != "http" && h.Proto != "https" {
			continue
		}
		if h.Host != "" && hostNoPort(h.Host) != allowed {
			return false, fmt.Sprintf("proxy forwarded mismatched Host header %q to upstream %s (authorized host %q)", h.Host, h.Dst, allowed)
		}
	}
	return true, "no mismatched Host header observed"
}

func floodDenied(ev *caseEvidence) (bool, string) {
	e := findEvent(ev, "flood")
	if e == nil {
		return false, "probe did not run the flood action"
	}
	counts, _ := e["counts"].(map[string]any)
	for k, v := range counts {
		if strings.HasPrefix(k, "status_2") || strings.HasPrefix(k, "status_3") {
			return false, fmt.Sprintf("flood produced %s=%v", k, v)
		}
	}
	return true, fmt.Sprintf("flood outcomes: %v", counts)
}

func floodAllowed(ev *caseEvidence) (bool, string) {
	e := findEvent(ev, "flood")
	if e == nil {
		return false, "probe did not run the flood action"
	}
	counts, _ := e["counts"].(map[string]any)
	n, _ := e["requests"].(float64)
	if v, ok := counts["status_200"].(float64); ok && int(v) == int(n) {
		return true, fmt.Sprintf("all %d allowed requests succeeded", int(n))
	}
	return false, fmt.Sprintf("requests=%v counts=%v", n, counts)
}

func largeBody(ev *caseEvidence) (bool, string) {
	min := 1 << 20
	if v, ok := ev.c.Extra["min_body"]; ok {
		fmt.Sscanf(v, "%d", &min)
	}
	e := findEvent(ev, "http")
	if e == nil {
		e = findEvent(ev, "https")
	}
	if e == nil {
		return false, "no HTTP event"
	}
	n, _ := e["body_len"].(float64)
	return int(n) >= min, fmt.Sprintf("body_len=%d want>=%d", int(n), min)
}

func secondProxyDenied(ev *caseEvidence) (bool, string) {
	e := findEvent(ev, "child-proxy")
	if e == nil {
		return false, "probe did not run child-proxy"
	}
	if st, ok := e["denied_status"].(float64); ok && int(st) == 403 {
		return true, "child proxy could not escape (403 from upstream proxy)"
	}
	ec, _ := e["denied_error_class"].(string)
	switch ec {
	case "econnrefused", "enetunreach", "dns_error", "proxy_502", "eacces", "eperm", "timeout":
		return true, "child proxy had no usable upstream (" + ec + ")"
	}
	return false, fmt.Sprintf("child proxy result: status=%v error=%v err=%v", e["denied_status"], e["denied_error_class"], clip(fmt.Sprint(e["denied_error"]), 160))
}

func interpreterDenied(ev *caseEvidence) (bool, string) {
	e := findEvent(ev, "exec")
	if e == nil {
		return false, "probe did not run exec"
	}
	out, _ := e["output"].(string)
	if strings.Contains(out, "DENIED-OK") || strings.Contains(out, "RAW-OK") {
		return false, "interpreter reached a denied destination: " + clip(out, 300)
	}
	if strings.Contains(out, "DENIED-ERR") && strings.Contains(out, "RAW-ERR") {
		return true, "interpreter's HTTP and raw-socket attempts were both denied: " + clip(out, 200)
	}
	return false, "interpreter output did not show the expected denial markers: " + clip(out, 300)
}

func namedStepDenied(ev *caseEvidence, name string) (bool, string) {
	for _, s := range ev.steps {
		if s.Name != name {
			continue
		}
		if anyEventIn(s.Events, func(e map[string]any) bool {
			if st, ok := e["status"].(float64); ok && (int(st) == 403 || int(st) == 502) {
				return true
			}
			ec, _ := e["error_class"].(string)
			switch ec {
			case "proxy_403", "proxy_502", "dns_error", "econnrefused", "enetunreach":
				return true
			}
			return false
		}) {
			return true, "workload lost access while the dependency was down"
		}
		return false, "step " + name + " did not show a fail-closed outcome: " + clip(s.Stderr, 200)
	}
	return false, "step " + name + " missing"
}

func anyEventIn(events []map[string]any, pred func(map[string]any) bool) bool {
	for _, e := range events {
		if pred(e) {
			return true
		}
	}
	return false
}

func timeoutFailClosed(ev *caseEvidence) (bool, string) {
	for _, s := range ev.steps {
		if s.Name != "timeout" {
			continue
		}
		timedOut := anyEventIn(s.Events, func(e map[string]any) bool {
			ec, _ := e["error_class"].(string)
			return ec == "timeout"
		}) || s.TimedOut
		if !timedOut {
			return false, "expected the slow request to time out"
		}
	}
	return namedStepDenied(ev, "after-timeout")
}

func mcpCall(ev *caseEvidence, action string) (bool, string) {
	for _, e := range ev.allEvents() {
		a, _ := e["action"].(string)
		if a != action {
			continue
		}
		ok, _ := e["mcp_ok"].(bool)
		return ok, fmt.Sprintf("%s: status=%v error=%v", action, e["status"], e["error_class"])
	}
	return false, "missing " + action + " event"
}

func tlsFloorDenied(ev *caseEvidence) (bool, string) {
	for _, e := range ev.allEvents() {
		if v, has := e["tls_version"].(float64); has {
			return false, fmt.Sprintf("TLS handshake completed at version 0x%04x; the proxy did not enforce the 1.2 floor", int(v))
		}
		ec, _ := e["error_class"].(string)
		if ec == "tls_error" {
			return true, "a ClientHello offering only TLS below 1.2 was rejected with a TLS error"
		}
	}
	return false, "the probe did not report a TLS-floor rejection"
}

// stepOutcome inspects one named step's events.
func stepOutcome(ev *caseEvidence, name, want string) (bool, string) {
	var step *stepEvidence
	for i := range ev.steps {
		if ev.steps[i].Name == name {
			step = &ev.steps[i]
			break
		}
	}
	if step == nil {
		return false, "step " + name + " missing"
	}
	events := stepEventsDeep(step)
	switch want {
	case "allow":
		if anyEventIn(events, func(e map[string]any) bool {
			st, ok := e["status"].(float64)
			return ok && int(st) >= 200 && int(st) < 300
		}) {
			return true, "step " + name + " reached the destination"
		}
		return false, "step " + name + " did not succeed: " + clip(step.Stderr, 160)
	case "denied":
		if anyEventIn(events, func(e map[string]any) bool {
			return strongEvent(e)
		}) {
			return true, "step " + name + " was denied"
		}
		return false, "step " + name + " was not explicitly denied"
	case "timeout":
		if step.TimedOut || anyEventIn(step.Events, func(e map[string]any) bool {
			ec, _ := e["error_class"].(string)
			return ec == "timeout"
		}) {
			return true, "step " + name + " timed out as expected"
		}
		return false, "step " + name + " did not time out"
	}
	return false, "unknown step outcome " + want
}

// stepEventsDeep includes events emitted by a child process that the probe
// wrapped (exec-self records them under child_events).
func stepEventsDeep(step *stepEvidence) []map[string]any {
	out := append([]map[string]any{}, step.Events...)
	for _, e := range step.Events {
		ce, _ := e["child_events"].([]any)
		for _, c := range ce {
			if m, ok := c.(map[string]any); ok {
				out = append(out, m)
			}
		}
	}
	return out
}

func strongEvent(e map[string]any) bool {
	if st, ok := e["status"].(float64); ok {
		switch int(st) {
		case 403, 407, 502:
			return true
		}
	}
	ec, _ := e["error_class"].(string)
	switch ec {
	case "proxy_403", "proxy_407", "proxy_502", "econnrefused", "enetunreach", "dns_error", "eperm", "eacces", "tls_error":
		return true
	}
	return false
}
