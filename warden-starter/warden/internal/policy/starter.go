package policy

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/warden-sandbox/warden/internal/audit"
	"gopkg.in/yaml.v3"
)

// StarterFromAudit derives the narrowest practical starter policy from
// successful audit events. It deliberately omits failed attempts and unknown
// resources: a trace is a suggestion for an allowlist, not permission to
// widen it. The caller may add environment grants after reviewing the file.
// Relative file paths are resolved against baseDir, the directory the traced
// program ran in; pass "" when that is unknown and relative paths are skipped.
func StarterFromAudit(events []audit.Event, command []string, baseDir string) Policy {
	return starterFromAudit(events, command, baseDir, "")
}

// StarterFromObservedReads accepts files intentionally mounted under a scratch
// runtime path, such as a project in /tmp, while learning.
func StarterFromObservedReads(events []audit.Event, command []string, mount string) Policy {
	return starterFromAudit(events, command, "", mount)
}

func starterFromAudit(events []audit.Event, command []string, baseDir, readMount string) Policy {
	read := make(map[string]struct{})
	write := make(map[string]struct{})
	hosts := make(map[string]struct{})
	for _, event := range events {
		if !event.Allowed {
			continue
		}
		switch event.Type {
		case "file":
			resource := event.Resource
			if !filepath.IsAbs(resource) && baseDir != "" {
				resource = filepath.Join(baseDir, resource)
			}
			grant, isWrite, ok := ProposeFileGrant(event.Action, resource)
			if !ok && readMount != "" && filepath.IsAbs(resource) && Within(resource, readMount) && !WriteAction(event.Action) {
				grant, isWrite, ok = filepath.Clean(resource), false, true
			}
			if !ok {
				continue
			}
			if isWrite {
				write[grant] = struct{}{}
			} else {
				read[grant] = struct{}{}
			}
		case "network":
			if !strings.EqualFold(event.Action, "connect") {
				continue
			}
			host := HostOnly(event.Resource)
			if validateHost(host) == nil {
				hosts[strings.ToLower(host)] = struct{}{}
			}
		}
	}
	// A write grant subsumes a read grant for the same path tree.
	for path := range read {
		for dir := range write {
			if path == dir || Within(path, dir) {
				delete(read, path)
				break
			}
		}
	}
	return Policy{
		Command: command,
		Filesystem: Filesystem{
			Read:  sortedSet(read),
			Write: sortedSet(write),
		},
		Network: Network{Allow: sortedSet(hosts)},
	}
}

// Marshal returns a readable policy document suitable for warden init.
func (p Policy) Marshal() ([]byte, error) {
	if !p.Limits.GrantDeadline.IsZero() {
		return nil, fmt.Errorf("cannot save a policy with temporary grants applied")
	}
	return yaml.Marshal(p)
}

func sortedSet(set map[string]struct{}) []string {
	values := make([]string, 0, len(set))
	for value := range set {
		values = append(values, value)
	}
	sort.Strings(values)
	return values
}
