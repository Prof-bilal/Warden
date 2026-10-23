//go:build linux

package docker

import (
	"fmt"
	"os"
)

// mapCurrentID maps the process's current namespace ID to the host ID using
// /proc/self/{uid,gid}_map. It returns an error when the map cannot be read,
// parsed, or does not contain the ID, so the caller can fail closed instead
// of running the container as namespace root.
func mapCurrentID(id int, path string) (int, error) {
	if id < 0 {
		return 0, fmt.Errorf("negative id %d", id)
	}
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	entries, err := parseIDMap(f)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", path, err)
	}
	mapped, ok := mapID(uint64(id), entries)
	if !ok {
		return 0, fmt.Errorf("id %d is not mapped in %s", id, path)
	}
	return int(mapped), nil
}

// hostUIDGID returns the host uid/gid that the invoking process maps to.
//
// Inside an unprivileged user namespace (for example
// `unshare --user --map-root-user`), os.Getuid/os.Getgid return the
// namespace IDs, often 0. Passing those directly to `docker run --user`
// makes the container run as host root, while bind mounts and the proxy
// socket belong to the mapped host user. Resolve the mapping so the
// container keeps the invoking user's host identity.
//
// If the mapping cannot be resolved, this returns an error; BuildDockerArgs
// then fails closed instead of silently launching the container as root.
func hostUIDGID() (int, int, bool, error) {
	uid, gid := os.Getuid(), os.Getgid()
	if uid < 0 || gid < 0 {
		return uid, gid, false, nil
	}
	hostUID, err := mapCurrentID(uid, "/proc/self/uid_map")
	if err != nil {
		return 0, 0, false, fmt.Errorf("resolve host uid: %w", err)
	}
	hostGID, err := mapCurrentID(gid, "/proc/self/gid_map")
	if err != nil {
		return 0, 0, false, fmt.Errorf("resolve host gid: %w", err)
	}
	return hostUID, hostGID, true, nil
}
