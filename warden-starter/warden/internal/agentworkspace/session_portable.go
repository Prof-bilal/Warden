//go:build darwin || windows

package agentworkspace

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/warden-sandbox/warden/internal/policy"
)

// The portable projection deliberately has the same conservative limits as
// Linux. The platform sandbox, rather than this copier, is the boundary that
// prevents the child from reaching the original workspace.
const (
	maxFile      = 8 << 20
	maxWorkspace = 256 << 20
)

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
	rootInfo     os.FileInfo
	files        map[string]*fileState
	writable     bool
	instructions map[string][]byte
}

type Change struct {
	Path          string
	Before, After []byte
	data          []byte
	mode          os.FileMode
	candidateHash [32]byte
}

// Prepare creates a temporary projection containing only regular files from
// the explicitly granted workspace. Original bytes containing protected values
// remain in supervisor memory and are never copied into the child workspace.
func Prepare(p policy.Policy) (s *Session, err error) {
	if p.Agent == nil {
		return nil, fmt.Errorf("agent policy required")
	}
	root := filepath.Clean(p.Agent.Workspace)
	rootInfo, err := os.Stat(root)
	if err != nil || !rootInfo.IsDir() {
		return nil, fmt.Errorf("agent workspace must be an existing directory")
	}
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, fmt.Errorf("workspace symlink check failed")
	}
	resolvedInfo, err := os.Stat(resolved)
	if err != nil || !os.SameFile(rootInfo, resolvedInfo) || filepath.Clean(resolved) != root {
		return nil, fmt.Errorf("workspace must contain no symlink ancestors")
	}

	dir, err := os.MkdirTemp("", "warden-agent-")
	if err != nil {
		return nil, err
	}
	s = &Session{Dir: dir, original: root, rootInfo: rootInfo, files: map[string]*fileState{}, writable: len(p.Filesystem.Write) > 0}
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
		specs[filepath.Clean(f.Path)] = f
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
		if hasGitComponentPortable(rel) || strings.EqualFold(first, ".warden") {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		for _, d := range denied {
			if sameOrWithinPortable(path, d) {
				if entry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("workspace symlinks are unsupported: %s", safeNamePortable(rel))
		}
		if entry.IsDir() {
			return os.MkdirAll(filepath.Join(dir, rel), 0700)
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("workspace special files are unsupported")
		}
		data, mode, e := readPortable(filepath.Join(root, rel))
		if e != nil {
			return fmt.Errorf("unsafe or unreadable workspace file: %s", safeNamePortable(rel))
		}
		total += len(data)
		if total > maxWorkspace {
			return fmt.Errorf("workspace exceeds 256 MiB projection limit")
		}
		st := &fileState{original: data, sanitized: append([]byte{}, data...), mode: mode}
		if spec, ok := specs[filepath.Clean(rel)]; ok {
			st.format = spec.Format
			d, e := parse(data, spec.Format)
			if e != nil {
				return fmt.Errorf("protected file %s: %w", safeNamePortable(rel), e)
			}
			for _, field := range spec.Fields {
				value, e := d.replace(field, nil)
				if e != nil {
					return fmt.Errorf("protected selector in %s: %w", safeNamePortable(rel), e)
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
			delete(specs, filepath.Clean(rel))
		}
		s.files[filepath.Clean(rel)] = st
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
	os.RemoveAll(s.Dir)
	for _, f := range s.files {
		clear(f.original)
		f.fields = nil
	}
}

func (s *Session) Changes() ([]Change, error) {
	candidate := map[string][]byte{}
	modes := map[string]os.FileMode{}
	total := 0
	err := filepath.WalkDir(s.Dir, func(path string, e os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("cannot enumerate projection")
		}
		if path == s.Dir {
			return nil
		}
		rel, _ := filepath.Rel(s.Dir, path)
		first := strings.Split(rel, string(filepath.Separator))[0]
		if strings.EqualFold(first, ".warden") {
			if e.IsDir() {
				return filepath.SkipDir
			}
			return fmt.Errorf("invalid notice directory")
		}
		if hasGitComponentPortable(rel) {
			return fmt.Errorf("git metadata cannot be applied")
		}
		if e.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("projection contains a symlink")
		}
		if e.IsDir() {
			return nil
		}
		b, m, err := readPortable(path)
		if err != nil {
			return fmt.Errorf("unsafe projection entry")
		}
		if instruction, ok := s.instructions[filepath.Clean(rel)]; ok {
			if !bytes.Equal(b, instruction) {
				return fmt.Errorf("agent instruction overlay was modified")
			}
			if f := s.files[filepath.Clean(rel)]; f != nil {
				candidate[filepath.Clean(rel)] = f.sanitized
				modes[filepath.Clean(rel)] = f.mode
			}
			return nil
		}
		total += len(b)
		if total > maxWorkspace {
			return fmt.Errorf("projection exceeds size limit")
		}
		rel = filepath.Clean(rel)
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
	if err := validateProtectedPortable(s, candidate); err != nil {
		return nil, err
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

func validateProtectedPortable(s *Session, candidate map[string][]byte) error {
	allTokens, expected := 0, 0
	for path, f := range s.files {
		if len(f.fields) == 0 {
			continue
		}
		expected += len(f.fields)
		b, exists := candidate[path]
		if !exists {
			return fmt.Errorf("protected file was deleted")
		}
		d, err := parse(b, f.format)
		if err != nil {
			return err
		}
		for _, slot := range f.fields {
			v, e := d.replace(slot.field, nil)
			if e != nil || v != slot.token {
				return fmt.Errorf("protected field was modified or removed")
			}
			count := 0
			for _, content := range candidate {
				count += bytes.Count(content, []byte(slot.token))
			}
			if count != 1 {
				return fmt.Errorf("protected placeholder was copied or removed")
			}
		}
	}
	for _, b := range candidate {
		allTokens += bytes.Count(b, []byte("__WARDEN_REDACTED_"))
	}
	if allTokens != expected {
		return fmt.Errorf("unknown or damaged placeholder")
	}
	return nil
}

func Review(w io.Writer, changes []Change) error {
	for _, c := range changes {
		if _, err := fmt.Fprintf(w, "\nFile %q\n", c.Path); err != nil {
			return err
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

func (s *Session) Apply(reviewed []Change) error {
	currentInfo, err := os.Stat(s.original)
	if err != nil || !os.SameFile(s.rootInfo, currentInfo) {
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
		b, _, e := readPortable(filepath.Join(s.original, path))
		if e != nil || sha256.Sum256(b) != sha256.Sum256(f.original) {
			return fmt.Errorf("host workspace changed; restart and review again")
		}
	}
	for _, c := range current {
		parent := filepath.Dir(filepath.Join(s.original, c.Path))
		if info, e := os.Stat(parent); e != nil || !info.IsDir() {
			return fmt.Errorf("all target parent directories must already exist")
		}
		if s.files[c.Path] == nil {
			if _, e := os.Lstat(filepath.Join(s.original, c.Path)); e == nil {
				return fmt.Errorf("new file conflicts with host")
			} else if !os.IsNotExist(e) {
				return fmt.Errorf("cannot inspect new host file")
			}
		}
	}
	var written []Change
	for _, c := range current {
		if err := writePortable(filepath.Join(s.original, c.Path), c.data, c.mode, s.files[c.Path] != nil); err != nil {
			rollbackFailed := false
			for i := len(written) - 1; i >= 0; i-- {
				previous := written[i]
				b, _, readErr := readPortable(filepath.Join(s.original, previous.Path))
				if readErr != nil || !bytes.Equal(b, previous.data) {
					rollbackFailed = true
					continue
				}
				if original := s.files[previous.Path]; original != nil {
					if e := writePortable(filepath.Join(s.original, previous.Path), original.original, original.mode, true); e != nil {
						rollbackFailed = true
					}
				} else if e := os.Remove(filepath.Join(s.original, previous.Path)); e != nil {
					rollbackFailed = true
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

func readPortable(path string) ([]byte, os.FileMode, error) {
	if err := checkPortableParents(path); err != nil {
		return nil, 0, err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return nil, 0, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, 0, fmt.Errorf("not a regular file")
	}
	if info.Size() > maxFile {
		return nil, 0, fmt.Errorf("file too large")
	}
	f, err := openPortable(path)
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()
	if err := validatePortableFile(f, info); err != nil {
		return nil, 0, err
	}
	data, err := io.ReadAll(io.LimitReader(f, maxFile+1))
	if err != nil || len(data) > maxFile {
		return nil, 0, fmt.Errorf("file read failed")
	}
	after, err := f.Stat()
	if err != nil || after.Mode()&os.ModeSymlink != 0 || !after.Mode().IsRegular() || after.Size() != info.Size() {
		return nil, 0, fmt.Errorf("file changed while reading")
	}
	if err := validatePortableFile(f, after); err != nil {
		return nil, 0, err
	}
	return data, info.Mode(), nil
}

func checkPortableParents(path string) error {
	for parent := filepath.Dir(filepath.Clean(path)); ; parent = filepath.Dir(parent) {
		info, err := os.Lstat(parent)
		if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("unsafe path parent")
		}
		next := filepath.Dir(parent)
		if next == parent {
			break
		}
	}
	return nil
}

func hasGitComponentPortable(path string) bool {
	for _, part := range strings.Split(path, string(filepath.Separator)) {
		if strings.EqualFold(part, ".git") {
			return true
		}
	}
	return false
}

func sameOrWithinPortable(path, base string) bool {
	path, base = filepath.Clean(path), filepath.Clean(base)
	if path == base {
		return true
	}
	rel, err := filepath.Rel(base, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return false
	}
	return true
}

func safeNamePortable(s string) string { return fmt.Sprintf("%q", s) }
