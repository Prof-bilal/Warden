//go:build linux

package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
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
	backend := fs.String("backend", "linux", "Linux projection backend")
	if err := fs.Parse(args); err != nil {
		return 2, err
	}
	if *backend != "linux" && *backend != "auto" {
		return 2, fmt.Errorf("agent projection currently requires Linux bubblewrap")
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
	cmd, err = policy.ResolveExecutable(cmd)
	if err != nil {
		return 2, err
	}
	if err = checkSandboxCommand(cmd); err != nil {
		return 2, err
	}
	// Outside executables are trusted runtime programs. Executables in the
	// workspace run from the projection so hardcoded source never bypasses it.
	session, err := agentworkspace.Prepare(p)
	if err != nil {
		return 1, err
	}
	defer session.Close()
	if cmd[0] == p.Agent.Workspace || policy.Within(cmd[0], p.Agent.Workspace) {
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
	// All approvals are read from a controlling terminal before the agent starts.
	// An agent cannot approve its own operations by writing to stdin.
	var tty *os.File
	defer func() {
		if tty != nil {
			tty.Close()
		}
	}()
	approve := func(prompt, answer string) bool {
		if tty == nil {
			tty, _ = os.OpenFile("/dev/tty", os.O_RDWR, 0)
		}
		if tty == nil {
			return false
		}
		fmt.Fprintln(tty, prompt)
		line, err := bufio.NewReader(tty).ReadString('\n')
		return err == nil && strings.TrimSpace(line) == answer
	}
	profiles := map[string]policy.SecretProfile{}
	names := []string{}
	for name := range p.Agent.SecretExec.Profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		profile := p.Agent.SecretExec.Profiles[name]
		approved := approve(fmt.Sprintf("Allow fixed GET %q using credential %q for this session? Type %q", profile.URL, profile.Env, "allow "+name), "allow "+name)
		if err = logger.Log(audit.Event{Type: "agent", Action: "profile_approval", Resource: name, Allowed: approved}); err != nil {
			return 1, err
		}
		if approved {
			profiles[name] = profile
		}
	}
	closeBroker, brokerEndpoint, err := agentworkspace.StartBroker(session.Dir, profiles, logger)
	if err != nil {
		return 1, err
	}
	defer closeBroker()
	session.Policy.AgentSocket = brokerEndpoint
	// The supervisor's broker captured real credentials before replacing child
	// environment values. Restore the parent environment when the session ends.
	restore := map[string]*string{}
	set := func(name, value string) {
		if _, ok := restore[name]; !ok {
			if old, ok := os.LookupEnv(name); ok {
				v := old
				restore[name] = &v
			} else {
				restore[name] = nil
			}
		}
		os.Setenv(name, value)
	}
	defer func() {
		for k, v := range restore {
			if v == nil {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, *v)
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
	set("WARDEN_AGENT_SOCKET", brokerEndpoint)
	cwd, err := os.Getwd()
	if err != nil {
		return 1, err
	}
	if err = os.Chdir(session.Dir); err != nil {
		return 1, err
	}
	defer os.Chdir(cwd)
	fmt.Fprintln(os.Stderr, "Warden: declared secrets are hidden. Read .warden/NOTICE.md. Host changes require review after exit.")
	code, err := sandbox.Run(cmd, session.Policy, "linux")
	closeBroker() // stop all broker activity before reviewing files
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
	approved := approve("Apply the reviewed changes, preserving protected values? Type apply (anything else discards).", "apply")
	if err = logger.Log(audit.Event{Type: "agent", Action: "apply_review", Resource: "workspace", Allowed: approved}); err != nil {
		return 1, err
	}
	if !approved {
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
