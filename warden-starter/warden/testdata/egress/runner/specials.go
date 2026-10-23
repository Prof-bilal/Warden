package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// runMCP drives an MCP stdio conversation through the sandbox: initialize,
// tools/list, then two tool calls (one allowed, one denied). It proves that
// MCP transport traffic is subject to the same network policy.
func (r *Runner) runMCP(c Case, ev *caseEvidence, policyPath string) {
	dir := ev.dir
	step := stepEvidence{Name: "mcp"}
	if policyPath == "" {
		step.Stderr = "harness error: case has no policy fixture"
		step.ExitCode = -1
		ev.steps = append(ev.steps, step)
		return
	}
	wardenArgs := []string{"run", "--backend", r.backend, "--policy", policyPath, "--",
		r.probePath, "mcp-stdio", "--ca", r.caFile, "--proxy-from-all"}
	step.CommandLine = r.wardenPath + " " + strings.Join(wardenArgs, " ")

	ctx, cancel := context.WithTimeout(context.Background(), r.cfg.Timeout)
	defer cancel()
	cmd := execCommandContext(ctx, r.wardenPath, wardenArgs...)
	cmd.Env = r.wardenEnv(dir)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		step.Stderr = err.Error()
		step.ExitCode = -1
		ev.steps = append(ev.steps, step)
		return
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		step.Stderr = err.Error()
		step.ExitCode = -1
		ev.steps = append(ev.steps, step)
		return
	}
	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf
	configureProcess(cmd)
	if err := cmd.Start(); err != nil {
		step.Stderr = err.Error()
		step.ExitCode = -1
		ev.steps = append(ev.steps, step)
		return
	}

	lines := make(chan string, 64)
	go func() {
		sc := bufio.NewScanner(stdout)
		sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
		for sc.Scan() {
			lines <- sc.Text()
		}
		close(lines)
	}()

	send := func(v any) {
		b, _ := json.Marshal(v)
		_, _ = fmt.Fprintln(stdin, string(b))
	}
	await := func(id int) map[string]any {
		deadline := time.After(15 * time.Second)
		for {
			select {
			case line, ok := <-lines:
				if !ok {
					return nil
				}
				var m map[string]any
				if json.Unmarshal([]byte(line), &m) != nil {
					continue
				}
				if got, ok := m["id"].(float64); ok && int(got) == id {
					return m
				}
			case <-deadline:
				return nil
			}
		}
	}

	send(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "initialize",
		"params": map[string]any{"protocolVersion": "2025-11-25", "capabilities": map[string]any{},
			"clientInfo": map[string]any{"name": "warden-egress-runner", "version": "1"}}})
	initResp := await(1)
	initOK := initResp != nil && initResp["result"] != nil
	step.Events = append(step.Events, map[string]any{
		"probe": "egress", "action": "mcp-initialize", "mcp_ok": initOK, "raw": initResp,
	})

	send(map[string]any{"jsonrpc": "2.0", "id": 2, "method": "tools/list", "params": map[string]any{}})
	listResp := await(2)
	_ = listResp

	send(map[string]any{"jsonrpc": "2.0", "id": 3, "method": "tools/call",
		"params": map[string]any{"name": "egress", "arguments": map[string]any{"url": ev.vars["allowedURL"]}}})
	allowed := await(3)
	step.Events = append(step.Events, mcpCallEvent("mcp-allowed", ev.vars["allowedURL"], allowed))

	send(map[string]any{"jsonrpc": "2.0", "id": 4, "method": "tools/call",
		"params": map[string]any{"name": "egress", "arguments": map[string]any{"url": ev.vars["deniedURL"]}}})
	denied := await(4)
	step.Events = append(step.Events, mcpCallEvent("mcp-denied", ev.vars["deniedURL"], denied))

	_ = stdin.Close()
	done := make(chan struct{})
	go func() { _ = cmd.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		_ = cmd.Process.Kill()
	}
	step.TimedOut = ctx.Err() == context.DeadlineExceeded
	if cmd.ProcessState != nil {
		step.ExitCode = cmd.ProcessState.ExitCode()
	} else {
		step.ExitCode = -1
	}
	step.Stdout = ""
	step.Stderr = stderrBuf.String()
	// The probe writes its JSON-RPC responses to stdout (already consumed)
	// and probe metadata to stderr.
	for _, e := range parseProbeEvents(stderrBuf.String()) {
		if a, _ := e["action"].(string); a == "mcp-call" {
			// Map probe-reported call results onto the synthetic events so
			// checks see structured status/error evidence.
			url, _ := e["url"].(string)
			for i := range step.Events {
				if u, _ := step.Events[i]["url"].(string); u == url {
					step.Events[i]["mcp_ok"] = e["ok"]
					step.Events[i]["status"] = e["status"]
					step.Events[i]["error_class"] = e["error_class"]
				}
			}
		}
	}
	step.Positive = hasPositive(stderrBuf.String())
	writeStepFiles(dir, step)
	ev.steps = append(ev.steps, step)
}

func mcpCallEvent(action, url string, resp map[string]any) map[string]any {
	e := map[string]any{"probe": "egress", "action": action, "url": url, "mcp_ok": false}
	if resp == nil {
		e["error_class"] = "no_response"
		return e
	}
	e["raw"] = resp
	if resp["error"] != nil {
		e["error_class"] = "rpc_error"
		return e
	}
	result, _ := resp["result"].(map[string]any)
	content, _ := result["content"].([]any)
	if len(content) > 0 {
		if m, ok := content[0].(map[string]any); ok {
			text, _ := m["text"].(string)
			e["mcp_ok"] = strings.Contains(text, "ok:true")
			if strings.Contains(text, "403") {
				e["status"] = float64(403)
			}
		}
	}
	return e
}

// runKillWarden starts an in-flight request, kills the Warden supervisor,
// and records whether the workload lost access without opening a new path.
func (r *Runner) runKillWarden(c Case, ev *caseEvidence, policyPath string) {
	dir := ev.dir
	step := stepEvidence{Name: "inflight"}
	if policyPath == "" {
		step.Stderr = "harness error: case has no policy fixture"
		step.ExitCode = -1
		ev.steps = append(ev.steps, step)
		return
	}
	wardenArgs := append([]string{"run", "--backend", r.backend, "--policy", policyPath, "--", r.probePath},
		substituteAll(c.Probe, ev.vars)...)
	step.CommandLine = r.wardenPath + " " + strings.Join(wardenArgs, " ")
	ctx, cancel := context.WithTimeout(context.Background(), r.cfg.Timeout)
	defer cancel()
	cmd := execCommandContext(ctx, r.wardenPath, wardenArgs...)
	cmd.Env = r.wardenEnv(dir)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	configureProcess(cmd)
	if err := cmd.Start(); err != nil {
		step.Stderr = err.Error()
		step.ExitCode = -1
		ev.steps = append(ev.steps, step)
		return
	}
	time.Sleep(2 * time.Second)
	killProcessGroup(cmd)
	_ = cmd.Wait()
	step.TimedOut = ctx.Err() == context.DeadlineExceeded
	if cmd.ProcessState != nil {
		step.ExitCode = cmd.ProcessState.ExitCode()
	} else {
		step.ExitCode = -1
	}
	step.Stdout = stdout.String()
	step.Stderr = stderr.String()
	step.Events = parseProbeEvents(stdout.String())
	step.Events = append(step.Events,
		map[string]any{"probe": "egress", "action": "kill-parent", "phase": "signal", "ok": true, "signal": "SIGKILL", "killed_by": "runner"},
	)
	step.Positive = hasPositive(stdout.String()) || hasPositive(stderr.String())
	writeStepFiles(dir, step)
	ev.steps = append(ev.steps, step)
}

// runStopDNS proves that resolver failure fails closed, then restores the
// resolver and proves recovery.
func (r *Runner) runStopDNS(c Case, ev *caseEvidence, policyPath string) {
	dir := ev.dir
	r.collectors.StopDNS()
	noDNS := r.execProbe(c, dir, policyPath, substituteAll(c.Probe, ev.vars), "no-dns", 45*time.Second)
	ev.steps = append(ev.steps, noDNS)
	r.collectors.StartDNS()
	time.Sleep(300 * time.Millisecond)
	if len(c.Steps) > 0 {
		rec := r.execProbe(c, dir, policyPath, substituteAll(c.Steps[0].Probe, ev.vars), "recovery", 45*time.Second)
		ev.steps = append(ev.steps, rec)
	}
}

// runStopCollector proves that upstream collector failure fails closed (no
// silent fallback), then restarts the collector and proves recovery.
func (r *Runner) runStopCollector(c Case, ev *caseEvidence, policyPath string) {
	dir := ev.dir
	_ = r.collectors.StopHTTP("allowed-http")
	noCollector := r.execProbe(c, dir, policyPath, substituteAll(c.Probe, ev.vars), "no-collector", 45*time.Second)
	ev.steps = append(ev.steps, noCollector)
	if err := r.collectors.StartHTTP("allowed-http"); err != nil {
		ev.notes = append(ev.notes, "could not restart allowed-http: "+err.Error())
		return
	}
	time.Sleep(300 * time.Millisecond)
	newVars := r.buildVars(ev.c.ID, dir, ev.token)
	ev.vars = newVars
	if len(c.Steps) > 0 {
		rec := r.execProbe(c, dir, policyPath, substituteAll(c.Steps[0].Probe, newVars), "recovery", 45*time.Second)
		ev.steps = append(ev.steps, rec)
	}
}

// runMulti executes a case's named steps in order.
func (r *Runner) runMulti(c Case, ev *caseEvidence, policyPath string) {
	dir := ev.dir
	for i, s := range c.Steps {
		name := s.Named(i)
		step := r.execProbe(c, dir, policyPath, substituteAll(s.Probe, ev.vars), name, 0)
		ev.steps = append(ev.steps, step)
	}
}
