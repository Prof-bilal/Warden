//go:build linux

package agentworkspace

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/warden-sandbox/warden/internal/policy"
	"golang.org/x/sys/unix"
)

const maxFile = 8 << 20
const maxWorkspace = 256 << 20

type protectedField struct{ field, token, value string }
type fileState struct {
	original, sanitized []byte
	mode                os.FileMode
	format              string
	fields              []protectedField
}
type Session struct {
	Dir          string
	Policy       policy.Policy
	original     string
	files        map[string]*fileState
	writable     bool
	rootFD       int
	instructions map[string][]byte
}
type Change struct {
	Path          string
	Before, After []byte
	data          []byte
	mode          os.FileMode
	candidateHash [32]byte
}

// Prepare copies an explicitly granted root. No original file is mounted in the
// child. Snapshots containing real values stay only in supervisor memory.
func Prepare(p policy.Policy) (s *Session, err error) {
	if p.Agent == nil {
		return nil, fmt.Errorf("agent policy required")
	}
	root := p.Agent.Workspace
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil || resolved != root {
		return nil, fmt.Errorf("workspace must exist and contain no symlink ancestors")
	}
	fd, err := unix.Open(root, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	dir, err := os.MkdirTemp("", "warden-agent-")
	if err != nil {
		unix.Close(fd)
		return nil, err
	}
	s = &Session{Dir: dir, original: root, files: map[string]*fileState{}, rootFD: fd, writable: len(p.Filesystem.Write) > 0}
	defer func() {
		if err != nil {
			s.Close()
		}
	}()
	denied, err := p.DeniedPaths()
	if err != nil {
		return nil, err
	}
	specs := map[string]policy.RedactedFile{}
	for _, f := range p.Agent.Redaction.Files {
		specs[f.Path] = f
	}
	total := 0
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("cannot enumerate workspace")
		}
		if path == root {
			return nil
		}
		rel, e := filepath.Rel(root, path)
		if e != nil {
			return e
		}
		first := strings.Split(rel, string(filepath.Separator))[0]
		if hasGitComponent(rel) || first == ".warden" {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		for _, d := range denied {
			if path == d || policy.Within(path, d) {
				if entry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("workspace symlinks are unsupported: %s", safeName(rel))
		}
		if entry.IsDir() {
			return os.MkdirAll(filepath.Join(dir, rel), 0700)
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("workspace special files are unsupported")
		}
		data, mode, e := readAt(fd, rel)
		if e != nil {
			return fmt.Errorf("unsafe or unreadable workspace file: %s", safeName(rel))
		}
		total += len(data)
		if total > maxWorkspace {
			return fmt.Errorf("workspace exceeds 256 MiB projection limit")
		}
		st := &fileState{original: data, sanitized: append([]byte{}, data...), mode: mode}
		if spec, ok := specs[rel]; ok {
			st.format = spec.Format
			d, e := parse(data, spec.Format)
			if e != nil {
				return fmt.Errorf("protected file %s: %w", safeName(rel), e)
			}
			for _, field := range spec.Fields {
				value, e := d.replace(field, nil)
				if e != nil {
					return fmt.Errorf("protected selector in %s: %w", safeName(rel), e)
				}
				if value == "" {
					return fmt.Errorf("protected values must be nonempty")
				}
				nonce := make([]byte, 16)
				if _, e = rand.Read(nonce); e != nil {
					return e
				}
				token := "__WARDEN_REDACTED_" + hex.EncodeToString(nonce) + "__"
				if _, e = d.replace(field, &token); e != nil {
					return e
				}
				st.fields = append(st.fields, protectedField{field, token, value})
			}
			st.sanitized, e = d.bytes()
			if e != nil {
				return fmt.Errorf("cannot serialize protected file")
			}
			delete(specs, rel)
		}
		s.files[rel] = st
		return os.WriteFile(filepath.Join(dir, rel), st.sanitized, mode.Perm())
	})
	if err != nil {
		return nil, err
	}
	if len(specs) > 0 {
		return nil, fmt.Errorf("a redaction file is missing, excluded, or hard-denied; resolve the policy before running")
	}
	noticeDir := filepath.Join(dir, ".warden")
	if err = os.Mkdir(noticeDir, 0700); err != nil {
		return nil, err
	}
	notice := "# Warden sanitized workspace\n\nDeclared secret fields contain Warden placeholders. The original values exist outside this workspace. Keep every placeholder at its original field, exactly once. Protected values cannot be edited. Host changes require human review after exit. Git history is excluded.\n\nSee manifest.json for protected file selectors and environment names. For an approved API operation use: /.warden/proxy-bridge agent exec --profile NAME.\n"
	if err = os.WriteFile(filepath.Join(noticeDir, "NOTICE.md"), []byte(notice), 0400); err != nil {
		return nil, err
	}
	manifest := struct {
		Files []policy.RedactedFile `json:"files"`
		Env   []string              `json:"environment"`
	}{p.Agent.Redaction.Files, p.Agent.Redaction.Env}
	b, _ := json.MarshalIndent(manifest, "", "  ")
	if err = os.WriteFile(filepath.Join(noticeDir, "manifest.json"), b, 0400); err != nil {
		return nil, err
	}
	// Conventional instruction files help clients discover the notice. Existing
	// sanitized instructions are retained; these overlays never merge to host.
	s.instructions = map[string][]byte{}
	for _, name := range []string{"AGENTS.md", "CLAUDE.md"} {
		original := []byte{}
		if f := s.files[name]; f != nil {
			original = f.sanitized
		}
		content := append(append([]byte{}, original...), []byte("\n\n## Warden protected workspace\n\nDeclared credential values exist in the original project and are hidden from you here. Read `.warden/NOTICE.md` and `.warden/manifest.json`. Preserve every placeholder exactly. Edit surrounding code normally. Host changes require review. Approved API operations use `/.warden/proxy-bridge agent exec --profile NAME`.\n")...)
		if _, exists := s.files[name]; exists {
			if err = os.Chmod(filepath.Join(dir, name), 0600); err != nil {
				return nil, fmt.Errorf("cannot prepare agent instructions")
			}
		}
		if err = os.WriteFile(filepath.Join(dir, name), content, 0400); err != nil {
			return nil, fmt.Errorf("cannot create agent instruction overlay")
		}
		s.instructions[name] = content
	}
	projected := p
	projected.Agent = nil
	projected.Filesystem = policy.Filesystem{}
	if s.writable {
		projected.Filesystem.Write = []string{dir}
	} else {
		projected.Filesystem.Read = []string{dir}
	}
	projected.ReadOnlyPaths = []string{noticeDir}
	for name := range s.instructions {
		projected.ReadOnlyPaths = append(projected.ReadOnlyPaths, filepath.Join(dir, name))
	}
	sort.Strings(projected.ReadOnlyPaths)
	// Redacted variables are removed even if explicitly allowed by the base policy.
	remove := map[string]bool{}
	for _, name := range p.Agent.Redaction.Env {
		remove[name] = true
	}
	for _, profile := range p.Agent.SecretExec.Profiles {
		remove[profile.Env] = true
	}
	projected.Env.Allow = nil
	for _, name := range p.Env.Allow {
		if !remove[name] {
			projected.Env.Allow = append(projected.Env.Allow, name)
		}
	}
	projected.Env.Allow = append(projected.Env.Allow, "WARDEN_AGENT_NOTICE", "WARDEN_AGENT_SOCKET")
	s.Policy = projected
	s.Policy.AgentCwd = dir
	return s, nil
}

func (s *Session) Close() {
	if s == nil {
		return
	}
	unix.Close(s.rootFD)
	s.rootFD = -1
	// Dir is an exact supervisor-created temporary directory, never a user path.
	os.RemoveAll(s.Dir)
	for _, f := range s.files {
		clear(f.original)
		f.fields = nil
	}
}

// Changes validates the entire candidate before presenting anything for review.
// Protected slots are immutable and tokens cannot be moved or copied.
func (s *Session) Changes() ([]Change, error) {
	fd, err := unix.Open(s.Dir, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	defer unix.Close(fd)
	candidate := map[string][]byte{}
	modes := map[string]os.FileMode{}
	total := 0
	err = filepath.WalkDir(s.Dir, func(path string, e os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("cannot enumerate projection")
		}
		if path == s.Dir {
			return nil
		}
		rel, _ := filepath.Rel(s.Dir, path)
		if strings.Split(rel, string(filepath.Separator))[0] == ".warden" {
			if e.IsDir() {
				return filepath.SkipDir
			}
			return fmt.Errorf("invalid notice directory")
		}
		if hasGitComponent(rel) {
			return fmt.Errorf("git metadata cannot be applied")
		}
		if e.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("projection contains a symlink")
		}
		if e.IsDir() {
			return nil
		}
		b, m, err := readAt(fd, rel)
		if err != nil {
			return fmt.Errorf("unsafe projection entry")
		}
		if instruction, ok := s.instructions[rel]; ok {
			if !bytes.Equal(b, instruction) {
				return fmt.Errorf("agent instruction overlay was modified")
			}
			if f := s.files[rel]; f != nil {
				candidate[rel] = f.sanitized
				modes[rel] = f.mode
			}
			return nil
		}
		total += len(b)
		if total > maxWorkspace {
			return fmt.Errorf("projection exceeds size limit")
		}
		candidate[rel] = b
		modes[rel] = m
		return nil
	})
	if err != nil {
		return nil, err
	}
	for name := range s.instructions {
		if _, err := os.Lstat(filepath.Join(s.Dir, name)); err != nil {
			return nil, fmt.Errorf("agent instruction overlay was removed")
		}
	}
	for path, f := range s.files {
		if len(f.fields) == 0 {
			continue
		}
		b, exists := candidate[path]
		if !exists {
			return nil, fmt.Errorf("protected file was deleted")
		}
		d, e := parse(b, f.format)
		if e != nil {
			return nil, e
		}
		for _, slot := range f.fields {
			v, e := d.replace(slot.field, nil)
			if e != nil || v != slot.token {
				return nil, fmt.Errorf("protected field was modified or removed")
			}
			count := 0
			for _, content := range candidate {
				count += bytes.Count(content, []byte(slot.token))
			}
			if count != 1 {
				return nil, fmt.Errorf("protected placeholder was copied or removed")
			}
		}
	}
	// Also reject foreign/forged markers and transformations retaining the marker.
	allTokens := 0
	expected := 0
	for _, b := range candidate {
		allTokens += bytes.Count(b, []byte("__WARDEN_REDACTED_"))
	}
	for _, f := range s.files {
		expected += len(f.fields)
	}
	if allTokens != expected {
		return nil, fmt.Errorf("unknown or damaged placeholder")
	}
	var changes []Change
	paths := map[string]bool{}
	for p := range s.files {
		paths[p] = true
	}
	for p := range candidate {
		paths[p] = true
	}
	for path := range paths {
		f := s.files[path]
		b, exists := candidate[path]
		if f != nil && exists && bytes.Equal(b, f.sanitized) {
			continue
		}
		if !exists {
			return nil, fmt.Errorf("file deletion is unsupported; remove files manually after review")
		}
		c := Change{Path: path, After: b, data: b, mode: modes[path], candidateHash: sha256.Sum256(b)}
		if f != nil {
			c.Before = f.sanitized
			c.mode = f.mode
			if len(f.fields) > 0 {
				d, e := parse(b, f.format)
				if e != nil {
					return nil, e
				}
				// Review the exact serialization that will be written, with placeholders.
				c.After, e = d.bytes()
				if e != nil {
					return nil, fmt.Errorf("cannot serialize review")
				}
				for _, slot := range f.fields {
					if _, e = d.replace(slot.field, &slot.value); e != nil {
						return nil, e
					}
				}
				c.data, e = d.bytes()
				if e != nil {
					return nil, fmt.Errorf("cannot restore protected fields")
				}
			}
		}
		changes = append(changes, c)
	}
	sort.Slice(changes, func(i, j int) bool { return changes[i].Path < changes[j].Path })
	if !s.writable && len(changes) > 0 {
		return nil, fmt.Errorf("workspace is read-only")
	}
	return changes, nil
}

// Review is deliberately quoted: filenames and agent text cannot emit terminal
// controls. It contains sanitized before/after data, never restored host bytes.
func Review(w io.Writer, changes []Change) error {
	for _, c := range changes {
		if _, e := fmt.Fprintf(w, "\nFile %q\n", c.Path); e != nil {
			return e
		}
		for _, side := range []struct {
			prefix string
			data   []byte
		}{{"-", c.Before}, {"+", c.After}} {
			for _, line := range strings.Split(string(side.data), "\n") {
				if _, err := fmt.Fprintf(w, "%s %q\n", side.prefix, line); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// Apply revalidates the immutable reviewed snapshot and original host files.
// Each file replacement is atomic. Multi-file application is not transactional.
func (s *Session) Apply(reviewed []Change) error {
	resolved, err := filepath.EvalSymlinks(s.original)
	if err != nil || resolved != s.original {
		return fmt.Errorf("host workspace root changed")
	}
	fd, err := unix.Open(s.original, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return fmt.Errorf("host workspace root unavailable")
	}
	var previous, now unix.Stat_t
	e1 := unix.Fstat(s.rootFD, &previous)
	e2 := unix.Fstat(fd, &now)
	unix.Close(fd)
	if e1 != nil || e2 != nil || previous.Dev != now.Dev || previous.Ino != now.Ino {
		return fmt.Errorf("host workspace root changed")
	}
	current, err := s.Changes()
	if err != nil {
		return err
	}
	if len(current) != len(reviewed) {
		return fmt.Errorf("projection changed after review")
	}
	for i, c := range current {
		if c.Path != reviewed[i].Path || !bytes.Equal(c.After, reviewed[i].After) || c.candidateHash != reviewed[i].candidateHash {
			return fmt.Errorf("projection changed after review")
		}
	}
	for path, f := range s.files {
		b, _, e := readAt(s.rootFD, path)
		if e != nil || sha256.Sum256(b) != sha256.Sum256(f.original) {
			return fmt.Errorf("host workspace changed; restart and review again")
		}
	}
	for _, c := range current {
		fd, _, e := parentAt(s.rootFD, c.Path)
		if e != nil {
			return fmt.Errorf("all target parent directories must already exist")
		}
		unix.Close(fd)
		if s.files[c.Path] == nil {
			if _, _, e := readAt(s.rootFD, c.Path); !os.IsNotExist(e) {
				return fmt.Errorf("new file conflicts with host")
			}
		}
	}
	var written []Change
	for _, c := range current {
		if err := writeAt(s.rootFD, c.Path, c.data, c.mode, s.files[c.Path] != nil); err != nil {
			var committed *committedWriteError
			if errors.As(err, &committed) {
				written = append(written, c)
			}
			rollbackFailed := false
			for i := len(written) - 1; i >= 0; i-- {
				previous := written[i]
				b, _, readErr := readAt(s.rootFD, previous.Path)
				if readErr != nil || !bytes.Equal(b, previous.data) {
					rollbackFailed = true
					continue
				}
				if original := s.files[previous.Path]; original != nil {
					if e := writeAt(s.rootFD, previous.Path, original.original, original.mode, true); e != nil {
						rollbackFailed = true
					}
				} else {
					fd, name, e := parentAt(s.rootFD, previous.Path)
					if e != nil {
						rollbackFailed = true
						continue
					}
					if e = unix.Unlinkat(fd, name, 0); e != nil {
						rollbackFailed = true
					}
					unix.Fsync(fd)
					unix.Close(fd)
				}
			}
			if rollbackFailed {
				return fmt.Errorf("apply failed and rollback was incomplete; inspect host files before continuing")
			}
			return fmt.Errorf("apply failed; completed replacements rolled back")
		}
		written = append(written, c)
	}
	return nil
}

func safeName(s string) string { return fmt.Sprintf("%q", s) }

func hasGitComponent(path string) bool {
	for _, part := range strings.Split(path, string(filepath.Separator)) {
		if part == ".git" {
			return true
		}
	}
	return false
}

// parentAt anchors every path component to directory descriptors. Symlinks and
// hardlinks are rejected rather than trusting lexical path checks.
func parentAt(root int, path string) (int, string, error) {
	if path == "." || filepath.IsAbs(path) || filepath.Clean(path) != path || path == ".." || strings.HasPrefix(path, "../") {
		return -1, "", fmt.Errorf("unsafe path")
	}
	fd, err := unix.Dup(root)
	if err != nil {
		return -1, "", err
	}
	unix.CloseOnExec(fd)
	parts := strings.Split(path, "/")
	for _, p := range parts[:len(parts)-1] {
		next, e := unix.Openat(fd, p, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
		unix.Close(fd)
		if e != nil {
			return -1, "", e
		}
		fd = next
	}
	return fd, parts[len(parts)-1], nil
}
func readAt(root int, path string) ([]byte, os.FileMode, error) {
	fd, name, err := parentAt(root, path)
	if err != nil {
		return nil, 0, err
	}
	defer unix.Close(fd)
	f, e := unix.Openat(fd, name, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if e != nil {
		return nil, 0, e
	}
	file := os.NewFile(uintptr(f), name)
	defer file.Close()
	var st unix.Stat_t
	if e = unix.Fstat(f, &st); e != nil {
		return nil, 0, e
	}
	if st.Mode&unix.S_IFMT != unix.S_IFREG || st.Nlink != 1 || st.Size > maxFile {
		return nil, 0, fmt.Errorf("not a regular single-link file within size limit")
	}
	b, e := io.ReadAll(io.LimitReader(file, maxFile+1))
	if len(b) > maxFile {
		return nil, 0, fmt.Errorf("file too large")
	}
	return b, os.FileMode(st.Mode & 0777), e
}
func writeAt(root int, path string, b []byte, mode os.FileMode, replace bool) error {
	fd, name, err := parentAt(root, path)
	if err != nil {
		return fmt.Errorf("parent must already exist on host")
	}
	defer unix.Close(fd)
	var original unix.Stat_t
	if replace {
		if err = unix.Fstatat(fd, name, &original, unix.AT_SYMLINK_NOFOLLOW); err != nil {
			return err
		}
		if original.Mode&unix.S_IFMT != unix.S_IFREG || original.Nlink != 1 {
			return fmt.Errorf("unsafe replacement target")
		}
	}
	nonce := make([]byte, 16)
	if _, err = rand.Read(nonce); err != nil {
		return err
	}
	temp := ".warden-apply-" + hex.EncodeToString(nonce)
	f, err := unix.Openat(fd, temp, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_CLOEXEC, 0600)
	if err != nil {
		return err
	}
	defer unix.Unlinkat(fd, temp, 0)
	file := os.NewFile(uintptr(f), temp)
	if _, err = file.Write(b); err != nil {
		file.Close()
		return err
	}
	if replace {
		if err = file.Chown(int(original.Uid), int(original.Gid)); err != nil {
			file.Close()
			return err
		}
	}
	if err = file.Chmod(mode.Perm()); err != nil {
		file.Close()
		return err
	}
	if err = file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	if replace {
		err = unix.Renameat(fd, temp, fd, name)
	} else {
		err = unix.Renameat2(fd, temp, fd, name, unix.RENAME_NOREPLACE)
	}
	if err != nil {
		return err
	}
	if err = unix.Fsync(fd); err != nil {
		return &committedWriteError{err}
	}
	return nil
}

type committedWriteError struct{ error }
