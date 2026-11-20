package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/warden-sandbox/warden/internal/approve"
	"github.com/warden-sandbox/warden/internal/audit"
	"github.com/warden-sandbox/warden/internal/packs"
	"github.com/warden-sandbox/warden/internal/policy"
)

func printGrantHelp() {
	fmt.Fprintln(os.Stderr, "Usage: warden grant host <hostname> [--policy <file>] [--for 30m]")
	fmt.Fprintln(os.Stderr, "       warden grant path <path> [--policy <file>] [--for 30m]")
	fmt.Fprintln(os.Stderr, "Review one exact hostname or read grant and approve it on the terminal.")
}

func classifyGrant(kind, value string, reviewedHosts []string) string {
	switch kind {
	case "write", "unconfined", "disable_protect_secrets", "widen_upgrade":
		return "requires explicit policy review"
	case "read":
		if policy.IsProtectedSecretPath(value) {
			return "matches a protected secret path"
		}
	case "env":
		for _, fragment := range []string{"KEY", "TOKEN", "SECRET", "PASSWORD", "CRED"} {
			if strings.Contains(strings.ToUpper(value), fragment) {
				return "resembles a credential variable"
			}
		}
	case "host":
		if reviewedHosts != nil && !slices.Contains(reviewedHosts, value) {
			return "host is outside the reviewed profile"
		}
		if strings.Contains(value, "*") || net.ParseIP(value) != nil {
			return "IP literals and wildcard hosts require explicit policy review"
		}
		for _, label := range strings.FieldsFunc(strings.ToLower(value), func(r rune) bool { return r == '.' || r == '-' }) {
			switch label {
			case "auth", "login", "signin", "oauth", "oauth2", "sso", "accounts", "identity", "idp", "sts", "vault", "credentials":
				return "credential host requires explicit policy review"
			}
		}
		for _, pack := range packs.List() {
			if slices.Contains(pack.Hosts, value) {
				for _, env := range pack.Env {
					if classifyGrant("env", env, nil) != "" {
						return "credential host requires explicit policy review"
					}
				}
			}
		}
	}
	return ""
}

func cmdGrant(args []string) error {
	if len(args) < 2 || (args[0] != "host" && args[0] != "path") {
		return fmt.Errorf("expected grant host <hostname> or grant path <path>")
	}
	flags := flag.NewFlagSet("warden grant", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	explicit := flags.String("policy", "", "policy file to update (default: ./warden.yaml)")
	duration := flags.Duration("for", 0, "grant lifetime")
	if err := flags.Parse(args[2:]); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected grant arguments")
	}
	timed := false
	flags.Visit(func(f *flag.Flag) {
		if f.Name == "for" {
			timed = true
		}
	})
	if timed && *duration <= 0 {
		return fmt.Errorf("--for requires a positive duration")
	}
	policyPath, err := runPolicyPath(*explicit, true)
	if err != nil {
		return err
	}
	p, digest, _, err := resolvePolicy(policySource{path: policyPath})
	if err != nil {
		return err
	}
	value := args[1]
	if value == "" {
		return fmt.Errorf("grant resource must not be empty")
	}
	kind := "host"
	if args[0] == "path" {
		kind = "read"
		value, err = filepath.Abs(value)
		if err != nil {
			return err
		}
		value, err = filepath.EvalSymlinks(value)
		if err != nil {
			return err
		}
		info, err := os.Stat(value)
		if err != nil {
			return err
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		if policy.Within(home, value) || (!info.Mode().IsRegular() && !(info.IsDir() && p.Filesystem.ProtectSecrets)) {
			return fmt.Errorf("this path requires explicit policy review")
		}
	} else {
		value = strings.ToLower(strings.TrimSpace(value))
		if value != policy.HostOnly(value) {
			return fmt.Errorf("use a bare hostname without a port")
		}
	}
	if reason := classifyGrant(kind, value, nil); reason != "" {
		return fmt.Errorf("%s. Review and edit the policy directly", reason)
	}
	if timed && (p.Agent != nil || p.MCP != nil) {
		return fmt.Errorf("temporary grants require a process policy")
	}
	var changed bool
	if kind == "host" {
		changed, err = p.AddHost(value)
	} else {
		changed, err = p.AddFileGrant(value, false)
	}
	if err != nil {
		return err
	}
	if !changed {
		fmt.Fprintln(os.Stdout, "The policy already grants this resource. Check deny rules and the audit log.")
		return nil
	}
	prompt := fmt.Sprintf("Add %s grant %q to %q?\nPolicy SHA-256: %s", kind, value, policyPath, digest)
	if timed {
		prompt = fmt.Sprintf("Add %s grant %q to %q for %s?\nPolicy SHA-256: %s", kind, value, policyPath, *duration, digest)
	}
	ok, err := approve.NewPrompter("", time.Minute, nil).ConfirmGrant(prompt)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("grant declined")
	}
	if timed {
		grant := policy.TemporaryGrant{Expires: time.Now().UTC().Add(*duration)}
		if kind == "host" {
			grant.Host = value
		} else {
			grant.Path = value
		}
		if err := approve.SaveTemporaryGrant(policyPath, digest, grant); err != nil {
			return err
		}
		fmt.Fprintf(os.Stdout, "Grant expires at %s. Restart with the updated policy.\n", grant.Expires.Format(time.RFC3339))
		return nil
	}
	if err := approve.SaveReviewedGrant(policyPath, digest, value, kind == "host"); err != nil {
		return err
	}
	fmt.Fprintln(os.Stdout, "Grant saved. Restart with the updated policy.")
	return nil
}

func explainDeniedEvents(w io.Writer, events []audit.Event) error {
	var out strings.Builder
	for _, event := range events {
		if event.Allowed {
			continue
		}
		fmt.Fprintf(&out, "%q %q was denied or failed", event.Type, event.Resource)
		if event.Reason != "" {
			fmt.Fprintf(&out, ": %q", event.Reason)
		}
		switch event.Type {
		case "network":
			host := strings.ToLower(policy.HostOnly(event.Resource))
			if classifyGrant("host", host, nil) == "" {
				out.WriteString(". Review network.allow with warden grant host")
			} else {
				out.WriteString(". Review network.allow directly")
			}
		case "file":
			grant, write, ok := policy.ProposeFileGrant(event.Action, event.Resource)
			if ok && !write && classifyGrant("read", grant, nil) == "" {
				out.WriteString(". Check whether this path exists before reviewing a read grant with warden grant path")
			} else {
				out.WriteString(". Review filesystem grants and deny rules directly")
			}
		}
		out.WriteByte('\n')
	}
	if out.Len() == 0 {
		out.WriteString("No denied events in the selected audit log entries.\n")
	}
	_, err := io.WriteString(w, out.String())
	return err
}
