package docker

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// idMapEntry is one line of /proc/self/{uid,gid}_map:
//
//	inside outside length
//
// An ID in the current namespace is mapped to outside+ (id-inside) on the
// host when inside <= id < inside+length.
type idMapEntry struct {
	inside  uint64
	outside uint64
	length  uint64
}

// parseIDMap parses the kernel's uid_map/gid_map format. Empty lines are
// ignored. Malformed input is an error so callers can fall back explicitly
// rather than silently using a wrong identity.
func parseIDMap(r io.Reader) ([]idMapEntry, error) {
	var entries []idMapEntry
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 3 {
			return nil, fmt.Errorf("invalid id map line %q", line)
		}
		var vals [3]uint64
		for i, field := range fields {
			v, err := strconv.ParseUint(field, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid id map line %q: %w", line, err)
			}
			vals[i] = v
		}
		entries = append(entries, idMapEntry{inside: vals[0], outside: vals[1], length: vals[2]})
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}

// mapID maps a namespace ID to the host ID using the parsed map.
func mapID(id uint64, entries []idMapEntry) (uint64, bool) {
	for _, e := range entries {
		if e.length == 0 || id < e.inside {
			continue
		}
		offset := id - e.inside
		if offset < e.length {
			return e.outside + offset, true
		}
	}
	return 0, false
}
