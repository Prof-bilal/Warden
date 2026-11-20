// Package learn builds reviewed starter policies from contained observations.
package learn

import (
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/warden-sandbox/warden/internal/audit"
	"github.com/warden-sandbox/warden/internal/policy"
)

type Report struct {
	ExitCode       int      `json:"exit_code"`
	Limit          string   `json:"limit,omitempty"`
	ProposedReads  []string `json:"proposed_reads"`
	ApprovedReads  []string `json:"approved_reads"`
	RequestedHosts []string `json:"requested_hosts"`
	PolicySHA256   string   `json:"policy_sha256"`
	PolicySource   string   `json:"policy_source"`
}

func Propose(events []audit.Event, cmd []string, workspace string) (policy.Policy, Report) {
	var observations []audit.Event
	report := Report{ProposedReads: []string{}, ApprovedReads: []string{}, RequestedHosts: []string{}}
	for _, event := range events {
		if event.Kind == "learn-egress" && event.Resource != "" {
			report.RequestedHosts = append(report.RequestedHosts, event.Resource)
		}
		if !event.Allowed || event.Type != "file" ||
			(event.Action != "open" && event.Action != "openat" && event.Action != "openat2") {
			continue
		}
		resource := event.Resource
		if !filepath.IsAbs(resource) || (len(cmd) > 0 && resource == cmd[0]) || !policy.Within(resource, workspace) ||
			policy.IsProtectedSecretPath(resource) {
			continue
		}
		rel, err := filepath.Rel(workspace, resource)
		if err != nil || slices.Contains(strings.Split(filepath.ToSlash(rel), "/"), ".git") {
			continue
		}
		real, err := filepath.EvalSymlinks(resource)
		if err != nil || real != resource {
			continue
		}
		info, err := os.Stat(resource)
		if err == nil && info.Mode().IsRegular() {
			observations = append(observations, event)
		}
	}
	sort.Strings(report.RequestedHosts)
	report.RequestedHosts = slices.Compact(report.RequestedHosts)
	p := policy.StarterFromObservedReads(observations, cmd, workspace)
	report.ProposedReads = append(report.ProposedReads, p.Filesystem.Read...)
	p.Filesystem.ProtectSecrets = true
	return p, report
}
