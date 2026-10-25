// Package envfilter implements the policy's environment passthrough rule:
// only names listed in env.allow are forwarded from the parent process into
// the sandbox. Deny by defaultan empty allowlist means an empty
// environment.
package envfilter

import (
	"path/filepath"
	"strings"
)

// Filter returns the subset of environment entries (name=value, from
// os.Environ-style slice) whose names are in allow, with duplicates keeping
// their last valuematching how a POSIX shell presents the environment to
// a child.
func Filter(parent []string, allow []string) []string {
	return FilterDenied(parent, allow, nil)
}

// FilterDenied applies an allowlist followed by deny-pattern overrides.
// Patterns match variable names only; values are never inspected.
func FilterDenied(parent []string, allow, deny []string) []string {
	allowed := make(map[string]bool, len(allow))
	for _, name := range allow {
		allowed[name] = true
	}
	// First pass: find the last occurrence of each allowed name.
	lastIdx := make(map[string]int)
	for i, kv := range parent {
		name, _, ok := strings.Cut(kv, "=")
		if ok && allowed[name] && !isDenied(name, deny) {
			lastIdx[name] = i
		}
	}
	// Second pass: emit only those last occurrences, in parent order.
	result := make([]string, 0, len(lastIdx))
	for i, kv := range parent {
		name, _, ok := strings.Cut(kv, "=")
		if !ok {
			continue
		}
		if last, found := lastIdx[name]; found && last == i {
			result = append(result, kv)
		}
	}
	return result
}

func isDenied(name string, patterns []string) bool {
	for _, pattern := range patterns {
		matched, err := filepath.Match(pattern, name)
		if err == nil && matched {
			return true
		}
	}
	return false
}
