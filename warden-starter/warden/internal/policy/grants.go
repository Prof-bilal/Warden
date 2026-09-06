package policy

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// This file holds the grant-mapping helpers shared by `warden init` (which
// derives starter policies from trace logs) and interactive approval mode
// (which proposes one grant per blocked access at runtime). Keeping the
// mapping in one place guarantees an approved grant is exactly what `init`
// would have generated for the same access.

// Within reports whether path is dir itself or strictly under it. Both are
// cleaned lexically so "./a/.." cases are handled.
func Within(path, dir string) bool {
	rel, err := filepath.Rel(dir, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

// RuntimePath reports whether path is provided by the sandbox runtime itself
// (system libraries, pseudo-filesystems, scratch tmpfs, the proxy bridge)
// rather than by a policy grant. Such paths must never become policy grants.
func RuntimePath(path string) bool {
	for _, dir := range []string{"/usr", "/lib", "/lib64", "/proc", "/dev", "/tmp", "/.warden"} {
		if path == dir || Within(path, dir) {
			return true
		}
	}
	return false
}

// WriteAction reports whether a file syscall action mutates the filesystem
// (and therefore needs a write grant) as opposed to merely observing it.
func WriteAction(action string) bool {
	switch action {
	case "creat", "unlink", "unlinkat", "rename", "renameat", "renameat2", "mkdir", "mkdirat", "rmdir", "chmod", "fchmodat", "chown", "fchownat":
		return true
	default:
		return false
	}
}

// HostOnly strips an optional :port suffix from a network resource,
// returning the bare hostname for allowlist comparison.
func HostOnly(resource string) string {
	if host, _, err := net.SplitHostPort(resource); err == nil {
		return host
	}
	return resource
}

// ProposeFileGrant maps one file access (strace action + resource) to the
// policy grant `init` would generate for it: a read grant on the path
// itself, or — for mutating actions — a write grant on the parent directory
// (a writable mount must name an existing directory when the target creates
// a new file). ok is false for non-absolute or runtime paths, which must
// never become grants.
func ProposeFileGrant(action, resource string) (grant string, write bool, ok bool) {
	path := filepath.Clean(resource)
	if !filepath.IsAbs(path) || RuntimePath(path) {
		return "", false, false
	}
	if WriteAction(action) {
		return filepath.Dir(path), true, true
	}
	return path, false, true
}

// Normalize coalesces redundant filesystem grants so a policy that lists
// the same path (or a path under a write grant) in both read and write
// degrades to a single write grant instead of failing at backend time.
//
// A write grant subsumes reads underneath it (see CoversFile), so dropping
// the shadowed read grant changes no enforcement decision — it only stops
// the backends from rejecting the overlap as ambiguous. This matters for
// real-world MCP servers (e.g. a cache dir that is both read and written);
// without it, a hand-written policy with the same dir in both lists passes
// Validate but fails in every backend with "granted as both read and
// write". Normalize is deny-by-default preserving: it only ever removes
// read grants that a write grant already covers, never adds access.
//
// Normalize is called by Load after ResolvePaths, so grants are absolute
// here. It also dedups and sorts both lists for stable output.
func (p *Policy) Normalize() {
	p.Filesystem.Read = dedupSorted(p.Filesystem.Read)
	p.Filesystem.Write = dedupSorted(p.Filesystem.Write)
	kept := p.Filesystem.Read[:0]
	for _, r := range p.Filesystem.Read {
		covered := false
		for _, w := range p.Filesystem.Write {
			if r == w || Within(r, w) {
				covered = true
				break
			}
		}
		if !covered {
			kept = append(kept, r)
		}
	}
	// Avoid aliasing surprises when Read was fully subsumed.
	if len(kept) == 0 {
		p.Filesystem.Read = nil
	} else {
		p.Filesystem.Read = kept
	}
}

func dedupSorted(in []string) []string {
	if len(in) == 0 {
		return in
	}
	cp := append([]string(nil), in...)
	sort.Strings(cp)
	out := cp[:0]
	for i, s := range cp {
		if i == 0 || s != cp[i-1] {
			out = append(out, s)
		}
	}
	return out
}

// CoversFile reports whether the policy already grants path: an exact or
// enclosing read grant covers reads, and an exact or enclosing write grant
// covers both reads and writes (a write grant subsumes reads underneath it).
func CoversFile(p Policy, path string) bool {
	path = filepath.Clean(path)
	for _, dir := range p.Filesystem.Write {
		if path == dir || Within(path, dir) {
			return true
		}
	}
	for _, dir := range p.Filesystem.Read {
		if path == dir || Within(path, dir) {
			return true
		}
	}
	return false
}

// AddHost adds host to network.allow (lowercased, like `init` generates).
// It reports whether the list changed. The policy is validated before
// returning success; an invalid host leaves the policy untouched.
func (p *Policy) AddHost(host string) (bool, error) {
	host = strings.ToLower(strings.TrimSpace(HostOnly(host)))
	if err := validateHost(host); err != nil {
		return false, err
	}
	for _, h := range p.Network.Allow {
		if h == host {
			return false, nil
		}
	}
	p.Network.Allow = append(p.Network.Allow, host)
	sort.Strings(p.Network.Allow)
	return true, nil
}

// AddFileGrant adds a filesystem grant, mirroring `init` semantics: a new
// write grant drops read grants it subsumes, and a read grant under an
// existing write grant is a no-op. It reports whether the policy changed.
func (p *Policy) AddFileGrant(grant string, write bool) (bool, error) {
	grant = filepath.Clean(grant)
	if !filepath.IsAbs(grant) {
		return false, fmt.Errorf("grant %q must be absolute", grant)
	}
	if write {
		for _, dir := range p.Filesystem.Write {
			if grant == dir || Within(grant, dir) {
				return false, nil
			}
		}
		kept := p.Filesystem.Read[:0]
		for _, r := range p.Filesystem.Read {
			if r == grant || Within(r, grant) {
				continue
			}
			kept = append(kept, r)
		}
		p.Filesystem.Read = kept
		p.Filesystem.Write = append(p.Filesystem.Write, grant)
		sort.Strings(p.Filesystem.Write)
	} else {
		if CoversFile(*p, grant) {
			return false, nil
		}
		p.Filesystem.Read = append(p.Filesystem.Read, grant)
		sort.Strings(p.Filesystem.Read)
	}
	if err := p.Validate(); err != nil {
		return false, err
	}
	return true, nil
}

// Save writes the policy back to path with owner-only permissions,
// validating first. It marshals the struct, so comments and formatting from
// a hand-written file are not preserved — the data is.
func (p Policy) Save(path string) error {
	if err := p.Validate(); err != nil {
		return err
	}
	data, err := p.Marshal()
	if err != nil {
		return fmt.Errorf("encode policy: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write policy %q: %w", path, err)
	}
	return nil
}
