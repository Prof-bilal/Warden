package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/warden-sandbox/warden/internal/approve"
	"github.com/warden-sandbox/warden/internal/audit"
	"github.com/warden-sandbox/warden/internal/learn"
	"github.com/warden-sandbox/warden/internal/policy"
)

func printLearnHelp() {
	fmt.Fprintln(os.Stderr, "Usage: warden learn [--output warden.learned.yaml] [--] <command> [args...]")
	fmt.Fprintln(os.Stderr, "Run an observed Linux sandbox and review each proposed read grant at the terminal.")
}

func cmdLearn(args []string) error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("Linux only; use warden trace / init")
	}
	flags := flag.NewFlagSet("warden learn", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	output := flags.String("output", "warden.learned.yaml", "new policy file")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() == 0 || *output == "" {
		return fmt.Errorf("supply a command and a policy output path")
	}
	if err := approve.CheckTTY(); err != nil {
		return fmt.Errorf("learning requires an interactive terminal for policy review")
	}
	file, err := filepath.Abs(*output)
	if err != nil {
		return err
	}
	reportFile := file + ".report.json"
	for _, path := range []string{file, reportFile} {
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			if err != nil {
				return err
			}
			return fmt.Errorf("output %q already exists", path)
		}
	}
	workspace, err := os.Getwd()
	if err != nil {
		return err
	}
	workspace, err = filepath.EvalSymlinks(workspace)
	if err != nil {
		return err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	home, err = canonicalOutput(home)
	if err != nil {
		return err
	}
	if policy.Within(home, workspace) || policy.IsProtectedSecretPath(workspace) {
		return fmt.Errorf("run learn from a project directory separate from protected or broad home paths")
	}
	state, err := audit.StateDir()
	if err != nil {
		return err
	}
	state, err = canonicalOutput(state)
	if err != nil {
		return err
	}
	if policy.Within(state, workspace) || policy.Within(workspace, state) {
		return fmt.Errorf("learning workspace must be separate from Warden audit storage")
	}
	cmd, err := policy.ResolveExecutable(flags.Args())
	if err != nil {
		return err
	}
	cmd[0], err = filepath.Abs(cmd[0])
	if err != nil {
		return err
	}
	if err := checkSandboxCommand(cmd); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Observing %q in read-only workspace %q with egress denied.\n", cmd[0], workspace)
	events, exit, limit, err := learn.Observe(cmd, workspace)
	if err != nil {
		return err
	}
	p, report := learn.Propose(events, cmd, workspace)
	report.ExitCode, report.Limit = exit, limit
	p.Filesystem.Read = nil
	if limit != "" {
		fmt.Fprintf(os.Stderr, "Observation reached a limit: %s. Review the partial results.\n", limit)
	} else if exit != 0 {
		fmt.Fprintf(os.Stderr, "Observation exited %d. Review the partial results.\n", exit)
	}
	for _, host := range report.RequestedHosts {
		fmt.Fprintf(os.Stderr, "Requested network host %q requires a separate policy review.\n", host)
	}
	prompt := approve.NewPrompter("", time.Minute, nil)
	for _, path := range report.ProposedReads {
		ok, err := prompt.ConfirmGrant(fmt.Sprintf("Include read access to %q?", path))
		if err != nil {
			return err
		}
		if ok {
			p.Filesystem.Read = append(p.Filesystem.Read, path)
			report.ApprovedReads = append(report.ApprovedReads, path)
		}
	}
	if err := p.Validate(); err != nil {
		return err
	}
	body, err := p.Marshal()
	if err != nil {
		return err
	}
	_, report.PolicySHA256, report.PolicySource, err = resolvePolicy(policySource{
		path: file, provenance: "learned", generated: body,
	})
	if err != nil {
		return err
	}
	approved, err := prompt.Confirm(fmt.Sprintf("Save reviewed policy to %q?\nPolicy SHA-256: %s", file, report.PolicySHA256))
	if err != nil {
		return err
	}
	if !approved {
		return fmt.Errorf("learning policy declined")
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	if err := writeExclusive(reportFile, append(data, '\n')); err != nil {
		return err
	}
	if err := writeExclusive(file, body); err != nil {
		_ = os.Remove(reportFile)
		return err
	}
	fmt.Fprintf(os.Stdout, "Policy: %s\nPolicy SHA-256: %s\nReport: %s\n", file, report.PolicySHA256, reportFile)
	return nil
}
