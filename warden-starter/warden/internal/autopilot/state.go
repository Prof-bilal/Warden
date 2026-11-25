//go:build linux || darwin || windows

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

	"github.com/warden-sandbox/warden/internal/policy"
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
		if err := createStateDirectory(dir); err != nil {
			return nil, err
		}
	}
	before, err := os.Lstat(dir)
	if err != nil {
		return nil, err
	}
	if !before.IsDir() || !validStateEntry(before) {
		return nil, fmt.Errorf("Autopilot state directory must be private and owned by the current user")
	}
	state, err := os.OpenRoot(dir)
	if err != nil {
		return nil, err
	}
	info, err := state.Stat(".")
	if err != nil || !os.SameFile(before, info) || !privateStateDirectory(state, info) {
		state.Close()
		return nil, fmt.Errorf("Autopilot state directory must be private and owned by the current user")
	}
	return state, nil
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
	if err := checkNewEnrollment(state, tmp); err != nil {
		f.Close()
		return err
	}
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
	f, err := openEnrollment(state, enrollmentName(project))
	if err != nil {
		return e, err
	}
	defer f.Close()
	info, err := f.Stat()
	const maxEnrollment = 32 << 10
	if err != nil || !info.Mode().IsRegular() || !privateEnrollment(f, info) || info.Size() > maxEnrollment {
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

func checkNewEnrollment(state *os.Root, name string) error {
	f, err := openEnrollment(state, name)
	if err != nil {
		return err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || !privateEnrollment(f, info) {
		return fmt.Errorf("new Autopilot enrollment must be private and owned by the current user")
	}
	return nil
}
