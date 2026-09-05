// Package windows implements the Windows AppContainer sandbox backend.
//
// Enforcement stack, applied in order, with every step failing closed:
//
//  1. AppContainer (LowBox) token — the process runs with a package SID,
//     no unneeded groups, and capability SIDs derived from the policy.
//  2. Filesystem capabilities — DACL grants on the policy-granted paths
//     only (plus a mandatory low-integrity label where the DACL is
//     rewritten). Everything else is denied by the token boundary.
//  3. WFP filters — a dedicated sublayer that hard-permits only loopback
//     traffic to the egress proxy and hard-blocks all other outbound IPv4
//     and IPv6 traffic from the container, including direct DNS.
//  4. ETW audit — file and network events for the container's PIDs,
//     including blocked attempts, exported into the JSONL audit log.
//  5. Job Object — process-tree memory limit, wall-clock timeout, and
//     kill-on-close termination of the whole tree.
//
// Until all of those are installed successfully, warden refuses to run the
// command. A plain CreateProcess fallback is never acceptable.
//
// plan.go holds the policy→primitive translation that is pure and therefore
// unit-testable on every OS. The syscall surface lives in the other files of
// this package (//go:build windows).
package windows

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/warden-sandbox/warden/internal/policy"
)

// runtimeWriteDeny are top-level directories that are never writable,
// mirroring the read-only runtime base of the Linux and macOS backends.
var runtimeWriteDeny = []string{
	`C:\Windows`,
	`C:\Program Files`,
	`C:\Program Files (x86)`,
	`C:\ProgramData`,
}

// RuntimeReadPaths are always readable so a typical Windows userspace binary
// can start (system DLLs and their resources). They are never writable.
var RuntimeReadPaths = []string{
	`C:\Windows\System32`,
	`C:\Windows\SysWOW64`,
	`C:\Windows\WinSxS`,
	`C:\Windows\Fonts`,
	`C:\Windows\Globalization`,
}

// IsRuntimeWriteDenied reports whether path is inside (or equal to) one of
// the read-only runtime base directories.
func IsRuntimeWriteDenied(path string) bool {
	for _, base := range runtimeWriteDeny {
		if strings.EqualFold(path, base) || isUnder(path, base) {
			return true
		}
	}
	return false
}

// IsRuntimeReadPath reports whether path is inside one of the runtime read
// paths, or the Windows system root.
func IsRuntimeReadPath(path string) bool {
	if strings.EqualFold(path, `C:\Windows`) {
		return true
	}
	for _, base := range RuntimeReadPaths {
		if strings.EqualFold(path, base) || isUnder(path, base) {
			return true
		}
	}
	return false
}

// Access masks for file grants (GENERIC_* values, which the object manager
// maps to FILE_GENERIC_* on files and directories).
const (
	accessRead  uint32 = 0x80000000 // GENERIC_READ
	accessWrite uint32 = 0x40000000 // GENERIC_WRITE
	accessExec  uint32 = 0x20000000 // GENERIC_EXECUTE
)

// GrantAccessMode value (accctrl.h GRANT_ACCESS). Kept here so plan
// semantics and the syscall layer agree on the ACE mode. GRANT_ACCESS
// appends ACEs to an existing DACL (SET_ACCESS=2 would replace it).
const grantAccess uint32 = 1

// bridgeHost is the loopback host of the egress proxy bridge. The port is
// assigned by proxy.StartTCP at run time and threaded through BuildPlan.
const bridgeHost = "127.0.0.1"

// proxyEnvPairs returns the proxy environment variables injected into the
// sandboxed process, pointing at the loopback bridge.
func proxyEnvPairs(addr string) []string {
	u := "http://" + addr
	return []string{
		"HTTP_PROXY=" + u,
		"HTTPS_PROXY=" + u,
		"ALL_PROXY=" + u,
		"NO_PROXY=",
	}
}

// pathGrant is one filesystem capability to install as a DACL grant.
type pathGrant struct {
	Path       string
	AccessMask uint32
	AccessMode uint32
}

// Plan is the fully translated enforcement plan for one run. Building it is
// deterministic and side-effect free; the Run path applies it.
type Plan struct {
	// AppContainerName is the per-run package moniker for the LowBox token.
	AppContainerName string
	// PackageSID is the SDDL form of the AppContainer (package) SID,
	// derived from the moniker (S-1-15-3-...).
	PackageSID string
	// CapabilitySIDs are the SDDL capability SIDs granted to the token.
	// Access is granted by the DACLs on granted paths, not by well-known
	// capability names; egress is enforced by the WFP filters.
	CapabilitySIDs []string
	// Grants are the filesystem capabilities to install, in deterministic
	// (case-insensitive path) order.
	Grants []pathGrant
	// Command is the resolved command with an absolute executable path.
	Command []string
	// CommandLine is the fully quoted command line for CreateProcessW.
	CommandLine string
	// ProxyAddress is the loopback address of the host-side egress proxy.
	ProxyAddress string
	// AllowIPs are the loopback destinations WFP hard-permits (the proxy
	// listener) for TCP and UDP. All other outbound traffic is blocked.
	AllowIPs []string
	// AllowPorts are the remote ports of AllowIPs to permit.
	AllowPorts []uint16
	// EnvBlock is the NUL-separated environment block for the sandboxed
	// process, reduced to env.allow plus the proxy variables.
	EnvBlock string // Limits carries the resource limits for the Job Object.
	Limits   policy.Limits
}

// BuildPlan translates a validated policy into the enforcement plan for a
// run. sessionID must be unique per run (see NewSessionID). proxyAddr is the
// loopback "host:port" of the host-side egress proxy; its exact address is
// hard-permitted by the WFP filters and injected into the environment.
func BuildPlan(cmd []string, p policy.Policy, sessionID, proxyAddr string) (*Plan, error) {
	if len(cmd) == 0 {
		return nil, fmt.Errorf("windows plan: no command")
	}
	exe := cmd[0]
	if !isAbsWin(exe) {
		return nil, fmt.Errorf("windows plan: command %q must be an absolute path", exe)
	}
	for _, arg := range cmd {
		if strings.ContainsRune(arg, '\x00') {
			return nil, fmt.Errorf("windows plan: command arguments must not contain NUL bytes")
		}
	}

	grants, err := deriveFilesystemGrants(exe, p)
	if err != nil {
		return nil, err
	}

	port, err := proxyPort(proxyAddr)
	if err != nil {
		return nil, err
	}
	name := fmt.Sprintf("warden.%s", sessionID)
	if len(name) > 64 {
		name = name[:64]
	}
	return &Plan{
		AppContainerName: name,
		PackageSID:       appContainerSID(name),
		CapabilitySIDs:   []string{capabilitySID(sessionID)},
		Grants:           grants,
		Command:          append([]string{}, cmd...),
		CommandLine:      buildCommandLine(cmd),
		ProxyAddress:     proxyAddr,
		AllowIPs:         []string{bridgeHost},
		AllowPorts:       []uint16{port},
		EnvBlock:         buildEnvBlock(os.Environ(), p.EnvAllowlist(), proxyEnvPairs(proxyAddr)),
		Limits:           p.Limits,
	}, nil
}

// proxyPort validates that addr is a loopback host:port and returns the port.
func proxyPort(addr string) (uint16, error) {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return 0, fmt.Errorf("windows plan: proxy address %q must be host:port", addr)
	}
	if ip := net.ParseIP(host); ip == nil || !ip.IsLoopback() {
		return 0, fmt.Errorf("windows plan: proxy address %q must be loopback", addr)
	}
	port, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil || port == 0 {
		return 0, fmt.Errorf("windows plan: proxy address %q must have a valid port", addr)
	}
	return uint16(port), nil
}

// deriveFilesystemGrants turns the policy grants into DACL capabilities:
// read grants get GENERIC_READ|GENERIC_EXECUTE (execute so a granted
// directory of binaries can run), write grants additionally get
// GENERIC_WRITE. The executable's parent directory is granted read-only so
// the image and its loader dependencies are reachable. Grants under the
// Windows system root are never writable and the filesystem root is never
// granted at all.
func deriveFilesystemGrants(exe string, p policy.Policy) ([]pathGrant, error) {
	mode := make(map[string]string)
	var grants []pathGrant

	add := func(path string, mask uint32) error {
		if path == "" {
			return fmt.Errorf("filesystem: path must not be empty")
		}
		if !isAbsWin(path) {
			return fmt.Errorf("filesystem: %q must be absolute", path)
		}
		clean := cleanWinPath(path)
		if isDriveRootWin(clean) {
			return fmt.Errorf("filesystem: refusing to grant %q (filesystem root)", path)
		}
		if mask&accessWrite != 0 && IsRuntimeWriteDenied(clean) {
			return fmt.Errorf("filesystem: %q is inside the read-only runtime base and cannot be granted write", path)
		}
		key := strings.ToLower(clean)
		if prev, ok := mode[key]; ok {
			if prev == "read" && mask&accessWrite != 0 {
				return fmt.Errorf("filesystem: %q is granted as both read and write", path)
			}
			return nil // already covered by an existing grant
		}
		if mask&accessWrite != 0 {
			mode[key] = "write"
		} else {
			mode[key] = "read"
		}
		grants = append(grants, pathGrant{Path: clean, AccessMask: mask, AccessMode: grantAccess})
		return nil
	}

	// Runtime read paths first: they must never be widened to write.
	for _, base := range RuntimeReadPaths {
		if err := add(base, accessRead|accessExec); err != nil {
			return nil, err
		}
	}
	// Policy grants, reads then writes.
	for _, path := range p.Filesystem.Read {
		if err := add(path, accessRead|accessExec); err != nil {
			return nil, err
		}
	}
	for _, path := range p.Filesystem.Write {
		if err := add(path, accessRead|accessWrite|accessExec); err != nil {
			return nil, err
		}
	}
	// The executable's parent directory, read-only, so the image and its
	// loader dependencies are reachable.
	if err := add(parentDirWin(exe), accessRead|accessExec); err != nil {
		return nil, err
	}

	sort.Slice(grants, func(i, j int) bool {
		return strings.ToLower(grants[i].Path) < strings.ToLower(grants[j].Path)
	})
	return grants, nil
}

// appContainerSID derives the AppContainer package SID for a moniker in
// SDDL form: S-1-15-3-<w0>-<w1>-<w2>-<w3>, where w0..w3 are the first four
// little-endian 32-bit words of the SHA-256 of the UTF-16LE moniker, each
// masked to 30 bits to fit a subauthority. This matches
// DeriveAppContainerSidFromAppContainerName and keeps plan construction
// pure and testable on every OS.
func appContainerSID(moniker string) string {
	sum := sha256.Sum256(utf16LE(moniker))
	parts := make([]string, 4)
	for i := range parts {
		parts[i] = fmt.Sprintf("%d", binary.LittleEndian.Uint32(sum[i*4:])&0x3FFFFFFF)
	}
	return "S-1-15-3-" + strings.Join(parts, "-")
}

// capabilitySID derives a unique capability SID (S-1-15-7-<w0>..<w3>) for a
// run, bound to the session id so ACLs minted for one run never match
// another run's token.
func capabilitySID(sessionID string) string {
	sum := sha256.Sum256(utf16LE("warden.cap." + sessionID))
	parts := make([]string, 4)
	for i := range parts {
		parts[i] = fmt.Sprintf("%d", binary.LittleEndian.Uint32(sum[i*4:])&0x3FFFFFFF)
	}
	return "S-1-15-7-" + strings.Join(parts, "-")
}

// utf16LE returns the little-endian UTF-16 encoding of s without a
// terminator, the input format used for Windows name hashing.
func utf16LE(s string) []byte {
	u := utf16.Encode([]rune(s))
	b := make([]byte, len(u)*2)
	for i, v := range u {
		binary.LittleEndian.PutUint16(b[i*2:], v)
	}
	return b
}

// buildCommandLine quotes cmd for CreateProcessW (lpCommandLine): the
// executable is quoted as argv[0], then each argument with the Windows
// argument-quoting rules (backslashes before a quote double, embedded
// quotes are escaped, a trailing backslash before the closing quote
// doubles).
func buildCommandLine(cmd []string) string {
	quoted := make([]string, len(cmd))
	for i, arg := range cmd {
		quoted[i] = quoteArg(arg)
	}
	return strings.Join(quoted, " ")
}

// quoteArg quotes one argument following the Microsoft C runtime rules.
func quoteArg(arg string) string {
	if arg != "" && !strings.ContainsAny(arg, " \t\"") {
		return arg
	}
	var b strings.Builder
	b.WriteByte('"')
	slashes := 0
	for _, r := range arg {
		switch r {
		case '\\':
			slashes++
			b.WriteByte('\\')
		case '"':
			for i := 0; i < slashes; i++ {
				b.WriteByte('\\')
			}
			slashes = 0
			b.WriteString(`\"`)
		default:
			slashes = 0
			b.WriteRune(r)
		}
	}
	for i := 0; i < slashes; i++ {
		b.WriteByte('\\')
	}
	b.WriteByte('"')
	return b.String()
}

// buildEnvBlock builds a NUL-separated, NUL-terminated environment block
// (lpEnvironment) with deny-by-default semantics: only names in allow are
// forwarded from parent (last value wins, matching CreateProcess), then the
// proxy variables are added so egress can reach the bridge. Entries are
// sorted case-insensitively as CreateProcess expects.
func buildEnvBlock(parent []string, allow []string, injected []string) string {
	allowed := make(map[string]struct{}, len(allow))
	for _, name := range allow {
		allowed[strings.ToUpper(name)] = struct{}{}
	}
	keep := make(map[string]string)
	for _, kv := range parent {
		name, value, ok := strings.Cut(kv, "=")
		if !ok || name == "" {
			continue
		}
		upper := strings.ToUpper(name)
		if _, ok := allowed[upper]; ok {
			keep[upper] = value // later entries overwrite earlier ones
		}
	}
	for _, kv := range injected {
		name, value, _ := strings.Cut(kv, "=")
		keep[name] = value
	}
	names := make([]string, 0, len(keep))
	for name := range keep {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool {
		return strings.ToLower(names[i]) < strings.ToLower(names[j])
	})
	block := make([]string, 0, len(names))
	for _, name := range names {
		block = append(block, name+"="+keep[name])
	}
	return strings.Join(block, "\x00") + "\x00\x00"
}

// isAbsWin reports whether path is an absolute Windows path (drive-letter
// rooted or UNC). Implemented explicitly because the generic filepath
// package only applies Windows semantics when compiled for Windows, and the
// plan logic must behave identically on every OS (unit tests, CI).
func isAbsWin(path string) bool {
	if strings.HasPrefix(path, `\\`) {
		return true // UNC
	}
	if len(path) >= 3 && path[1] == ':' && (path[2] == '\\' || path[2] == '/') {
		c := path[0]
		return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
	}
	return false
}

// isDriveRootWin reports whether path is a drive root like C:\.
func isDriveRootWin(path string) bool {
	if len(path) == 3 && path[1] == ':' && (path[2] == '\\' || path[2] == '/') {
		c := path[0]
		return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
	}
	return false
}

// cleanWinPath normalizes slashes, collapses duplicate separators, resolves
// \. and \.. lexically, and drops trailing separators (except the root).
func cleanWinPath(path string) string {
	path = strings.ReplaceAll(path, "/", `\`)
	if strings.HasPrefix(path, `\\`) {
		// UNC: preserve the leading pair.
		rest := path[2:]
		return `\\` + cleanSegments(rest)
	}
	return cleanSegments(path)
}

func cleanSegments(path string) string {
	prefix := ""
	if len(path) >= 2 && path[1] == ':' {
		prefix, path = path[:2], path[2:]
	}
	parts := strings.Split(path, `\`)
	var out []string
	for _, part := range parts {
		switch part {
		case "", ".":
			continue
		case "..":
			if len(out) > 0 {
				out = out[:len(out)-1]
			}
			// .. at the root is dropped, matching Windows normalization.
		default:
			out = append(out, part)
		}
	}
	joined := strings.Join(out, `\`)
	if prefix != "" {
		if joined == "" {
			return prefix + `\`
		}
		return prefix + `\` + joined
	}
	if strings.HasPrefix(path, `\`) { // rooted without drive
		if joined == "" {
			return `\`
		}
		return `\` + joined
	}
	return joined
}

// parentDirWin returns the parent directory of an absolute Windows path.
func parentDirWin(path string) string {
	path = cleanWinPath(path)
	if isDriveRootWin(path) {
		return path
	}
	i := strings.LastIndexAny(path, `\/`)
	if i < 0 {
		return path
	}
	if i <= 2 { // parent is the drive root (X:\)
		return path[:3]
	}
	return path[:i]
}

// isUnder reports whether path is strictly inside dir, case-insensitively.
func isUnder(path, dir string) bool {
	p, d := strings.ToLower(cleanWinPath(path)), strings.ToLower(cleanWinPath(dir))
	if p == d || d == "" {
		return false
	}
	return strings.HasPrefix(p, d+`\`)
}

// NewSessionID returns a short unique id for one sandboxed run, used as the
// AppContainer moniker suffix and capability binding. It mixes the current
// time with crypto/rand so concurrent runs never collide.
func NewSessionID() string {
	var rnd [8]byte
	if _, err := rand.Read(rnd[:]); err != nil {
		// Crypto/rand never fails on Windows/Linux in practice; if it did,
		// the time component still disambiguates sequential runs.
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("%d.%s", time.Now().UnixNano(), fmt.Sprintf("%016x", binary.BigEndian.Uint64(rnd[:])))
}
