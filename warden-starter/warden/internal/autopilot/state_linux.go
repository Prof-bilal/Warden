//go:build linux

package autopilot

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"

	"github.com/warden-sandbox/warden/internal/policy"
	"golang.org/x/sys/unix"
)

// OpenState opens the canonical, owner-only state directory outside the
// projection. The caller supplies the canonical Warden state directory.
func OpenState(dir, project string, create bool) (*os.Root, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	home, err = filepath.EvalSymlinks(home)
	if err != nil {
		return nil, err
	}
	if !filepath.IsAbs(dir) || overlap(project, dir) || policy.Within(home, project) || policy.IsProtectedSecretPath(project) {
		return nil, fmt.Errorf("project must be separate from Warden state, home root and protected paths")
	}
	if create {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return nil, err
		}
	}
	before, err := os.Lstat(dir)
	if err != nil {
		return nil, err
	}
	if !before.IsDir() || !ownedPrivate(before, 0700) {
		return nil, fmt.Errorf("Autopilot state directory must be owned by the current user with mode 0700")
	}
	state, err := os.OpenRoot(dir)
	if err != nil {
		return nil, err
	}
	info, err := state.Stat(".")
	if err != nil || !os.SameFile(before, info) || !ownedPrivate(info, 0700) {
		state.Close()
		return nil, fmt.Errorf("Autopilot state directory must be owned by the current user with mode 0700")
	}
	return state, nil
}

func ownedPrivate(info os.FileInfo, mode os.FileMode) bool {
	if info == nil || info.Mode().Perm() != mode {
		return false
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && stat.Uid == uint32(os.Geteuid())
}

func enrollmentName(project string) string {
	return fmt.Sprintf("%x.json", sha256.Sum256([]byte(project)))
}

// Save publishes a complete enrollment without overwriting any prior approval.
func Save(state *os.Root, e Enrollment) error {
	if _, err := e.Policy(e.Project); err != nil {
		return err
	}
	body, err := json.Marshal(e)
	if err != nil {
		return err
	}
	tmp := ".enroll-" + rand.Text()
	f, err := state.OpenFile(tmp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer state.Remove(tmp)
	_, writeErr := f.Write(body)
	if writeErr == nil {
		writeErr = f.Sync()
	}
	closeErr := f.Close()
	if writeErr != nil {
		return writeErr
	}
	if closeErr != nil {
		return closeErr
	}
	return state.Link(tmp, enrollmentName(e.Project))
}

func Load(state *os.Root, project string) (Enrollment, error) {
	var e Enrollment
	f, err := state.OpenFile(enrollmentName(project), os.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0)
	if err != nil {
		return e, err
	}
	defer f.Close()
	info, err := f.Stat()
	const maxEnrollment = 32 << 10
	if err != nil || !info.Mode().IsRegular() || !ownedPrivate(info, 0600) || info.Size() > maxEnrollment {
		return e, fmt.Errorf("invalid Autopilot enrollment file")
	}
	raw, err := io.ReadAll(io.LimitReader(f, maxEnrollment+1))
	if err != nil || len(raw) > maxEnrollment {
		return e, fmt.Errorf("could not read Autopilot enrollment")
	}
	if err := json.Unmarshal(raw, &e); err != nil {
		return e, fmt.Errorf("invalid Autopilot enrollment")
	}
	canonical, err := json.Marshal(e)
	if err != nil || !bytes.Equal(raw, canonical) {
		return e, fmt.Errorf("Autopilot enrollment has an unsupported format")
	}
	_, err = e.Policy(project)
	return e, err
}

func Revoke(state *os.Root, project string) error {
	return state.Remove(enrollmentName(project))
}
