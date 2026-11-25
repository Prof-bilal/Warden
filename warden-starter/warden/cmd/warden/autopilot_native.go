//go:build linux || darwin || windows

package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/warden-sandbox/warden/internal/approve"
	"github.com/warden-sandbox/warden/internal/audit"
	"github.com/warden-sandbox/warden/internal/autopilot"
	"github.com/warden-sandbox/warden/internal/policy"
	"github.com/warden-sandbox/warden/internal/sandbox"
)

func cmdAutopilot(args []string) (int, error) {
	if len(args) == 0 {
		printAutopilotHelp(os.Stderr)
		return 2, fmt.Errorf("a subcommand is required")
	}
	op := args[0]
	tail := args[1:]
	switch op {
	case "setup", "plan", "status", "revoke":
		if len(tail) != 0 {
			return 2, fmt.Errorf("unexpected arguments for %s", op)
		}
	case "run":
		if len(tail) > 0 {
			if tail[0] != "--" {
				return 2, fmt.Errorf("place agent arguments after --")
			}
			tail = tail[1:]
		}
	case "approve":
	default:
		return 2, fmt.Errorf("unknown subcommand %q", op)
	}
	if op == "setup" {
		fmt.Fprintf(os.Stderr, "Autopilot host %s/%s · native sandbox probe\n", runtime.GOOS, runtime.GOARCH)
		if err := autopilotDependencies(); err != nil {
			return 1, err
		}
		if err := doctorProve(); err != nil {
			return 1, err
		}
		return 0, nil
	}

	project, err := autopilot.Inspect(".")
	if err != nil {
		return 1, err
	}
	if op == "plan" {
		dependency := "tools available. Run setup to check containment"
		if err := autopilotDependencies(); err != nil {
			dependency = err.Error()
		}
		return 0, printAutopilotJSON(struct {
			autopilot.Project
			DefaultMode string `json:"default_mode"`
			Readiness   string `json:"readiness"`
		}{project, autopilot.ReadOnly, dependency})
	}

	dir, err := audit.StateDir()
	if err != nil {
		return 1, err
	}
	if !filepath.IsAbs(dir) {
		return 1, fmt.Errorf("Warden state requires an absolute home or XDG state path")
	}
	dir, err = canonicalOutput(dir)
	if err != nil {
		return 1, err
	}
	auditPath, err := canonicalOutput(filepath.Join(dir, "audit.jsonl"))
	if err != nil {
		return 1, err
	}
	if policy.Within(auditPath, project.Root) {
		return 1, fmt.Errorf("Warden audit storage must be outside the project")
	}
	stateDir := filepath.Join(dir, "autopilot")
	state, err := autopilot.OpenState(stateDir, project.Root, false)
	if err != nil && !(errors.Is(err, os.ErrNotExist) && op == "approve") {
		return 1, autopilotStateError(err)
	}
	if state != nil {
		defer state.Close()
	}

	switch op {
	case "approve":
		return 0, approveAutopilot(tail, project, state, stateDir, auditPath)
	case "revoke":
		return 0, autopilotStateError(autopilot.Revoke(state, project.Root))
	}
	e, err := autopilot.Load(state, project.Root)
	if err != nil {
		return 1, autopilotStateError(err)
	}
	p, err := e.Policy(project.Root)
	if err != nil {
		return 1, err
	}
	dependencyErr := errors.Join(autopilotDependencies(), e.VerifyProgram(), checkAutopilotBoundary(e, stateDir, auditPath))
	if op == "status" {
		readiness := "tools available. Run setup to check containment"
		if dependencyErr != nil {
			readiness = dependencyErr.Error()
		}
		body, err := p.Marshal()
		if err != nil {
			return 1, err
		}
		return 0, printAutopilotJSON(struct {
			autopilot.Enrollment
			Policy    string `json:"policy_yaml"`
			Audit     string `json:"audit_log"`
			Readiness string `json:"readiness"`
		}{e, string(body), auditPath, readiness})
	}
	if dependencyErr != nil {
		return 1, dependencyErr
	}
	if err := recordAutopilot("launch", e); err != nil {
		return 1, err
	}
	return runNativeAgent(p, append(p.Command, tail...))
}

func approveAutopilot(args []string, project autopilot.Project, state *os.Root, stateDir, auditPath string) error {
	fs := flag.NewFlagSet("autopilot approve", flag.ContinueOnError)
	edit := fs.Bool("edit", false, "propose a writable copy with reviewed host apply")
	useAdvisor := fs.Bool("advisor", false, "ask the configured provider for a mode suggestion")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 || (*edit && *useAdvisor) {
		return fmt.Errorf("supply one installed agent with at most one of --edit or --advisor")
	}
	if state != nil {
		if _, err := autopilot.Load(state, project.Root); !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("an enrollment already exists or requires repair. Use revoke before approving again")
		}
	}
	if err := approve.CheckTTY(); err != nil {
		return fmt.Errorf("Autopilot approval requires a controlling terminal")
	}
	if err := autopilotDependencies(); err != nil {
		return err
	}
	mode := autopilot.ReadOnly
	if *edit {
		mode = autopilot.ReviewedEdits
	}
	if *useAdvisor {
		var advisor autopilot.Advisor = autopilot.CompatibleAdvisor{
			Endpoint: os.Getenv("WARDEN_AUTOPILOT_ENDPOINT"),
			Model:    os.Getenv("WARDEN_AUTOPILOT_MODEL"),
			APIKey:   os.Getenv("WARDEN_AUTOPILOT_API_KEY"),
		}
		suggestion, err := advisor.Suggest(context.Background(), project.Hints)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Autopilot advisor unavailable (%v). Using read-only proposal.\n", err)
		} else {
			mode = suggestion
		}
	}
	e, err := autopilot.Enroll(project.Root, fs.Arg(0), mode)
	if err != nil {
		return err
	}
	p, err := e.Policy(project.Root)
	if err != nil {
		return err
	}
	if err := checkAutopilotBoundary(e, stateDir, auditPath); err != nil {
		return err
	}
	body, err := p.Marshal()
	if err != nil {
		return err
	}
	prompt := approve.NewPrompter("", time.Minute, nil)
	ok, err := prompt.Confirm(fmt.Sprintf("Enroll %q to run %q with workspace mode %q?\n%s", e.Project, e.Program, e.Mode, body))
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("Autopilot enrollment declined")
	}
	if state == nil {
		state, err = autopilot.OpenState(stateDir, project.Root, true)
		if err != nil {
			return err
		}
		defer state.Close()
	}
	if err := recordAutopilot("approve", e); err != nil {
		return err
	}
	if err := autopilot.Save(state, e); err != nil {
		return fmt.Errorf("save approval: %w", err)
	}
	fmt.Fprintln(os.Stderr, "Autopilot enrollment saved. Run warden autopilot run from this project.")
	return nil
}

func autopilotDependencies() error {
	if _, err := sandbox.Resolve(nativeAgentBackend()); err != nil || runtime.GOOS != "linux" {
		return err
	}
	for _, tool := range []string{"setpriv", "strace"} {
		if _, err := exec.LookPath(tool); err != nil {
			return fmt.Errorf("Autopilot requires %s on PATH", tool)
		}
	}
	return nil
}

func autopilotStateError(err error) error {
	if errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("no enrollment for this project. Run warden autopilot approve")
	}
	return err
}

func printAutopilotJSON(value any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func recordAutopilot(action string, e autopilot.Enrollment) error {
	file, _, err := audit.OpenDefault()
	if err != nil {
		return err
	}
	defer file.Close()
	return audit.New(file).Log(audit.Event{Type: "autopilot", Action: action, Resource: e.Project, Allowed: true, Reason: e.Mode})
}

func checkAutopilotBoundary(e autopilot.Enrollment, stateDir, auditPath string) error {
	if runtime.GOOS == "linux" {
		return nil
	}
	if runtime.GOOS == "windows" && !strings.EqualFold(filepath.Ext(e.Program), ".exe") {
		return fmt.Errorf("Autopilot on Windows requires an installed .exe")
	}
	command := []string{e.Program}
	backend := nativeAgentBackend()
	if err := outsideUpstream(e.Project, policy.Policy{}, command, backend); err != nil {
		return fmt.Errorf("project overlaps native sandbox runtime permissions: %w", err)
	}
	temp, err := canonicalOutput(os.TempDir())
	if err != nil {
		return err
	}
	if policy.Within(temp, e.Project) ||
		(runtime.GOOS == "darwin" && policy.Within(strings.ToLower(temp), strings.ToLower(e.Project))) {
		return fmt.Errorf("project must not contain the host temporary directory")
	}
	if err := outsideUpstream(auditPath, policy.Policy{}, command, backend); err != nil {
		return fmt.Errorf("audit storage overlaps native sandbox runtime permissions: %w", err)
	}
	return outsideUpstream(stateDir, policy.Policy{}, command, backend)
}
