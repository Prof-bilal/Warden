// Package envfilter implements the policy's environment passthrough rule:
// only names listed in env.allow are forwarded from the parent process into
// the sandbox. Deny by default — an empty allowlist means an empty
// environment.
package envfilter

import (
	"strings"
)

// Filter returns the subset of environment entries (name=value, from
// os.Environ-style slice) whose names are in allow, with duplicates keeping
// their last value — matching how a POSIX shell presents the environment to
// a child.
func Filter(parent []string, allow []string) []string {
	allowed := make(map[string]bool, len(allow))
	for _, name := range allow {
		allowed[name] = true
	}
	// First pass: find the last occurrence of each allowed name.
	lastIdx := make(map[string]int)
	for i, kv := range parent {
		name, _, ok := strings.Cut(kv, "=")
		if ok && allowed[name] {
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
