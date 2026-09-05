package audit

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// ImportStrace converts strace's %file and %network records to Warden audit
// events.  Warden runs strace outside the sandbox, so the untrusted process
// cannot alter or suppress this evidence.  strace follows forks, covering a
// server's worker processes as well as its initial executable.
func ImportStrace(r io.Reader, logger *Logger) error {
	s := bufio.NewScanner(r)
	// Long argv values should not make an audit record disappear.
	s.Buffer(make([]byte, 4*1024), 1024*1024)
	for s.Scan() {
		line := s.Text()
		if strings.Contains(line, "+++ exited") || strings.Contains(line, "--- SIG") {
			continue
		}
		open := strings.IndexByte(line, '(')
		if open < 1 {
			continue
		}
		action := line[:open]
		if end := strings.LastIndex(action, "] "); end >= 0 { // strace -f PID prefix
			action = action[end+2:]
		}
		if !isAuditedCall(action) {
			continue
		}
		resource := syscallResource(action, line[open+1:])
		allowed := !strings.Contains(line, ") = -1 ")
		reason := "syscall succeeded"
		if !allowed {
			reason = syscallResult(line)
		}
		if err := logger.Log(Event{Type: eventType(action), Action: action, Resource: resource, Allowed: allowed, Reason: reason}); err != nil {
			return fmt.Errorf("write strace audit event: %w", err)
		}
	}
	if err := s.Err(); err != nil {
		return fmt.Errorf("read strace output: %w", err)
	}
	return nil
}

func isAuditedCall(action string) bool {
	switch action {
	case "open", "openat", "openat2", "creat", "access", "faccessat", "faccessat2", "stat", "lstat", "newfstatat", "statx", "readlink", "readlinkat", "execve", "execveat", "unlink", "unlinkat", "rename", "renameat", "renameat2", "mkdir", "mkdirat", "rmdir", "chmod", "fchmodat", "chown", "fchownat", "connect", "bind", "sendto", "sendmsg", "recvfrom", "recvmsg":
		return true
	default:
		return false
	}
}

func eventType(action string) string {
	switch action {
	case "connect", "bind", "sendto", "sendmsg", "recvfrom", "recvmsg":
		return "network"
	default:
		return "file"
	}
}

func syscallResource(action, args string) string {
	// %file calls carry their pathname as the first quoted argument (or the
	// second for *at variants). Network calls include a printable address.
	if first := strings.IndexByte(args, '"'); first >= 0 {
		if end := strings.IndexByte(args[first+1:], '"'); end >= 0 {
			quoted := args[first+1 : first+1+end]
			if unquoted, err := strconv.Unquote("\"" + quoted + "\""); err == nil {
				return unquoted
			}
			return quoted
		}
	}
	// Do not discard non-path calls such as connect with an abstract address.
	if close := strings.LastIndex(args, ") ="); close >= 0 {
		return args[:close]
	}
	return args
}

func syscallResult(line string) string {
	if i := strings.LastIndex(line, ") = "); i >= 0 {
		return line[i+4:]
	}
	return "syscall failed"
}
