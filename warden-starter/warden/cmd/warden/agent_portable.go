//go:build darwin || windows

package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/warden-sandbox/warden/internal/agentworkspace"
	"github.com/warden-sandbox/warden/internal/audit"
	"github.com/warden-sandbox/warden/internal/policy"
	"github.com/warden-sandbox/warden/internal/sandbox"
)

func cmdAgent(args []string) (int, error) {
	if len(args) > 0 && args[0] == "exec" {
		fs := flag.NewFlagSet("agent exec", flag.ContinueOnError)
		name := fs.String("profile", "", "approved HTTPS operation")
		if err := fs.Parse(args[1:]); err != nil {
			return 2, err
		}
		if fs.NArg() != 0 {
			return 2, fmt.Errorf("agent exec accepts a profile only; arbitrary commands never receive secrets")
		}
		socket := os.Getenv("WARDEN_AGENT_SOCKET")
		if socket == "" {
			return 1, fmt.Errorf("agent exec requires a Warden agent session")
		}
		return 0, agentworkspace.RequestProfile(socket, *name, os.Stdout)
	}

	fs := flag.NewFlagSet("agent", flag.ContinueOnError)
	path := fs.String("policy", "", "policy containing agent.workspace and redaction rules")
	backend := fs.String("backend", "auto", "native projection backend")
	if err := fs.Parse(args); err != nil {
		return 2, err
	}
	wantBackend := nativeAgentBackend()
	if *backend != "auto" && *backend != wantBackend {
		return 2, fmt.Errorf("agent projection on %s requires backend %q", runtime.GOOS, wantBackend)
	}
	p, err := policy.Load(*path)
	if err != nil {
		return 2, err
	}
	if p.Agent == nil {
		return 2, fmt.Errorf("policy must contain agent.workspace and redaction rules")
	}
	cmd, err := p.ResolveCommand(fs.Args())
	if err != nil {
		return 2, err
	}
	return runNativeAgent(p, cmd)
}

func runNativeAgent(p policy.Policy, cmd []string) (int, error) {
	var input, output *os.File
	var reader *bufio.Reader
	defer func() {
		if reader != nil {
			input.Close()
			output.Close()
		}
	}()
	approve := func(prompt, answer string) bool {
		if reader == nil {
			var err error
			input, err = openAgentInput()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warden: agent review requires a controlling terminal: %v\n", err)
				return false
			}
			output, err = openAgentOutput()
			if err != nil {
				input.Close()
				fmt.Fprintf(os.Stderr, "Warden: agent review requires a controlling terminal: %v\n", err)
				return false
			}
			reader = bufio.NewReader(input)
		}
		if _, err := fmt.Fprintln(output, prompt); err != nil {
			return false
		}
		line, err := reader.ReadString('\n')
		return err == nil && strings.TrimSpace(line) == answer
	}

	return runAgentWithReview(p, cmd, approve)
}

func runAgentWithReview(p policy.Policy, cmd []string, approve func(prompt, answer string) bool) (int, error) {
	cmd, err := policy.ResolveExecutable(cmd)
	if err != nil {
		return 2, err
	}
	if err = checkSandboxCommand(cmd); err != nil {
		return 2, err
	}
	session, err := agentworkspace.Prepare(p)
	if err != nil {
		return 1, err
	}
	defer session.Close()
	if sameAgentPath(cmd[0], p.Agent.Workspace) || policy.Within(cmd[0], p.Agent.Workspace) {
		rel, _ := filepath.Rel(p.Agent.Workspace, cmd[0])
		cmd[0] = filepath.Join(session.Dir, rel)
	}

	logFile, _, err := audit.OpenDefault()
	if err != nil {
		return 1, err
	}
	defer logFile.Close()
	logger := audit.New(logFile)
	if err = logger.Log(audit.Event{Type: "agent", Action: "projection", Resource: "workspace", Allowed: true, Reason: "explicit selectors; originals excluded"}); err != nil {
		return 1, err
	}

	profiles := map[string]policy.SecretProfile{}
	names := make([]string, 0, len(p.Agent.SecretExec.Profiles))
	for name := range p.Agent.SecretExec.Profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		profile := p.Agent.SecretExec.Profiles[name]
		ok := approve(fmt.Sprintf("Allow fixed GET %q using credential %q for this session? Type %q", profile.URL, profile.Env, "allow "+name), "allow "+name)
		if err = logger.Log(audit.Event{Type: "agent", Action: "profile_approval", Resource: name, Allowed: ok}); err != nil {
			return 1, err
		}
		if ok {
			profiles[name] = profile
		}
	}
	closeBroker := func() {}
	if len(profiles) != 0 {
		stop, endpoint, err := agentworkspace.StartBroker(session.Dir, profiles, logger)
		if err != nil {
			return 1, err
		}
		closeBroker = stop
		defer closeBroker()
		session.Policy.AgentSocket = endpoint
	}

	restore := map[string]*string{}
	set := func(name, value string) {
		if _, exists := restore[name]; !exists {
			if old, ok := os.LookupEnv(name); ok {
				v := old
				restore[name] = &v
			} else {
				restore[name] = nil
			}
		}
		_ = os.Setenv(name, value)
	}
	defer func() {
		for k, v := range restore {
			if v == nil {
				_ = os.Unsetenv(k)
			} else {
				_ = os.Setenv(k, *v)
			}
		}
	}()
	for _, name := range p.Agent.Redaction.Env {
		set(name, "__WARDEN_REDACTED_ENV__")
	}
	for _, profile := range p.Agent.SecretExec.Profiles {
		set(profile.Env, "__WARDEN_REDACTED_ENV__")
	}
	set("WARDEN_AGENT_NOTICE", filepath.Join(session.Dir, ".warden", "NOTICE.md"))
	set("WARDEN_AGENT_SOCKET", session.Policy.AgentSocket)

	oldCwd, err := os.Getwd()
	if err != nil {
		return 1, err
	}
	if err = os.Chdir(session.Dir); err != nil {
		return 1, err
	}
	defer os.Chdir(oldCwd)
	fmt.Fprintln(os.Stderr, "Warden: declared secrets are hidden. Read .warden/NOTICE.md. Host changes require review after exit.")
	code, err := sandbox.Run(cmd, session.Policy, nativeAgentBackend())
	closeBroker()
	if err != nil {
		return code, err
	}
	if code != 0 {
		return code, fmt.Errorf("agent exited unsuccessfully; host files were not changed")
	}
	changes, err := session.Changes()
	if err != nil {
		_ = logger.Log(audit.Event{Type: "agent", Action: "candidate_validation", Resource: "workspace", Allowed: false, Reason: "unsafe projection changes rejected"})
		return 1, err
	}
	if len(changes) == 0 {
		fmt.Fprintln(os.Stderr, "Warden: no workspace changes.")
		return code, nil
	}
	if err = agentworkspace.Review(os.Stderr, changes); err != nil {
		return 1, err
	}
	ok := approve("Apply the reviewed changes, preserving protected values? Type apply (anything else discards).", "apply")
	if err = logger.Log(audit.Event{Type: "agent", Action: "apply_review", Resource: "workspace", Allowed: ok}); err != nil {
		return 1, err
	}
	if !ok {
		fmt.Fprintln(os.Stderr, "Warden: changes discarded.")
		return code, nil
	}
	if err = session.Apply(changes); err != nil {
		_ = logger.Log(audit.Event{Type: "agent", Action: "apply_complete", Resource: "workspace", Allowed: false, Reason: "apply or conflict check failed"})
		return 1, err
	}
	if err = logger.Log(audit.Event{Type: "agent", Action: "apply_complete", Resource: "workspace", Allowed: true}); err != nil {
		return 1, fmt.Errorf("files applied but completion audit failed")
	}
	fmt.Fprintln(os.Stderr, "Warden: reviewed changes applied; protected values preserved.")
	return code, nil
}

func nativeAgentBackend() string {
	if runtime.GOOS == "windows" {
		return sandbox.BackendWindows
	}
	return sandbox.BackendSeatbelt
}

func sameAgentPath(a, b string) bool {
	if runtime.GOOS == "windows" {
		return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
	}
	return filepath.Clean(a) == filepath.Clean(b)
}

func openAgentInput() (*os.File, error) {
	if runtime.GOOS == "windows" {
		return os.Open("CONIN$")
	}
	return os.Open("/dev/tty")
}

func openAgentOutput() (*os.File, error) {
	if runtime.GOOS == "windows" {
		return os.OpenFile("CONOUT$", os.O_WRONLY, 0)
	}
	return os.OpenFile("/dev/tty", os.O_WRONLY, 0)
}
