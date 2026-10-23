//go:build linux

package docker

import "os"

// mapCurrentID maps the process's current namespace ID to the host ID using
// /proc/self/{uid,gid}_map. It returns the original ID and false when the
// map cannot be read or does not contain the ID, so callers can fall back to
// the pre-userns behavior rather than dropping the --user flag.
func mapCurrentID(id int, path string) (int, bool) {
	if id < 0 {
		return id, false
	}
	f, err := os.Open(path)
	if err != nil {
		return id, false
	}
	defer f.Close()
	entries, err := parseIDMap(f)
	if err != nil {
		return id, false
	}
	mapped, ok := mapID(uint64(id), entries)
	if !ok {
		return id, false
	}
	return int(mapped), true
}

// hostUIDGID returns the host uid/gid that the invoking process maps to.
//
// Inside an unprivileged user namespace (for example
// `unshare --user --map-root-user`), os.Getuid/os.Getgid return the
// namespace IDs, often 0. Passing those directly to `docker run --user`
// makes the container run as host root, while bind mounts and the proxy
// socket belong to the mapped host user. Resolve the mapping so the
// container keeps the invoking user's host identity.
func hostUIDGID() (int, int, bool) {
	uid, gid := os.Getuid(), os.Getgid()
	if uid < 0 || gid < 0 {
		return uid, gid, false
	}
	hostUID, okUID := mapCurrentID(uid, "/proc/self/uid_map")
	if !okUID {
		hostUID = uid
	}
	hostGID, okGID := mapCurrentID(gid, "/proc/self/gid_map")
	if !okGID {
		hostGID = gid
	}
	return hostUID, hostGID, true
}
