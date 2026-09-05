package policy

import (
	"net"
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
func StarterFromAudit(events []audit.Event, command []string) Policy {
	read := make(map[string]struct{})
	write := make(map[string]struct{})
	hosts := make(map[string]struct{})
	for _, event := range events {
		if !event.Allowed {
			continue
		}
		switch event.Type {
		case "file":
			path := filepath.Clean(event.Resource)
			if !filepath.IsAbs(path) || runtimePath(path) {
				continue
			}
			if writeAction(event.Action) {
				// A writable mount must name an existing directory when the
				// trace target creates a new file.
				write[filepath.Dir(path)] = struct{}{}
			} else {
				read[path] = struct{}{}
			}
		case "network":
			if !strings.EqualFold(event.Action, "connect") {
				continue
			}
			host := hostOnly(event.Resource)
			if validateHost(host) == nil {
				hosts[strings.ToLower(host)] = struct{}{}
			}
		}
	}
	// A write grant subsumes a read grant for the same path tree.
	for path := range read {
		for dir := range write {
			if path == dir || isWithin(path, dir) {
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
func (p Policy) Marshal() ([]byte, error) { return yaml.Marshal(p) }

func runtimePath(path string) bool {
	for _, dir := range []string{"/usr", "/lib", "/lib64", "/proc", "/dev", "/tmp", "/.warden"} {
		if path == dir || isWithin(path, dir) {
			return true
		}
	}
	return false
}

func writeAction(action string) bool {
	switch action {
	case "creat", "unlink", "unlinkat", "rename", "renameat", "renameat2", "mkdir", "mkdirat", "rmdir", "chmod", "fchmodat", "chown", "fchownat":
		return true
	default:
		return false
	}
}

func hostOnly(resource string) string {
	if host, _, err := net.SplitHostPort(resource); err == nil {
		return host
	}
	return resource
}

func isWithin(path, dir string) bool {
	rel, err := filepath.Rel(dir, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

func sortedSet(set map[string]struct{}) []string {
	values := make([]string, 0, len(set))
	for value := range set {
		values = append(values, value)
	}
	sort.Strings(values)
	return values
}
