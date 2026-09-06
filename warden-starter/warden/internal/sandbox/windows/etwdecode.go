// This file holds the pure helpers of the Windows ETW auditor: decoding a
// file path out of a Kernel-File event payload and choosing the audit action
// for an operation. The helpers only touch []byte, so they are unit-tested on
// every OS while the session machinery lives in etw.go (//go:build windows).

package windows

import (
	"encoding/binary"
	"strings"
	"unicode/utf16"
)

// maxDecodedName is the longest file name Warden will accept from an event
// payload. Kernel events report names shorter than MAX_PATH; anything longer
// is not a name we can trust to be a path.
const maxDecodedName = 260

// classic kernel FileIo payloads lay the file name out as
//
//	FileObject(pointer) FileKey(pointer) [optional fields] FileNameLength(USHORT) FileName(UTF-16)
//
// on x64 the two pointers occupy offsets 0 and 8, so the name length lives at
// offset 16 for events without extra fields. Some operations (create, rename)
// prefix additional fields, so a small set of candidate offsets is tried and
// the first that yields a plausible path wins. These offsets are x64 mirrors
// of the FileIo event layouts (evntrace.h / TraceEvent kernel parser) and are
// pinned by the Windows escape test TestEtwFileActivityAudited.
var kernelFileNameOffsets = [...]int{16, 20, 24, 28}

// decodeKernelFileEvent extracts the file path from a Microsoft-Windows-
// Kernel-File event payload. It reports false when the payload does not carry
// a name Warden can verify, so undecodable events are dropped rather than
// written to the audit log with a guessed resource.
func decodeKernelFileEvent(payload []byte) (string, bool) {
	for _, off := range kernelFileNameOffsets {
		if name, ok := kernelFileNameAt(payload, off); ok {
			return name, true
		}
	}
	return "", false
}

func kernelFileNameAt(payload []byte, lengthOff int) (string, bool) {
	if lengthOff < 0 || lengthOff+2 > len(payload) {
		return "", false
	}
	n := int(binary.LittleEndian.Uint16(payload[lengthOff : lengthOff+2]))
	if n < 1 || n > maxDecodedName {
		return "", false
	}
	start := lengthOff + 2
	if start+2*n > len(payload) {
		return "", false
	}
	name, ok := decodeUTF16LE(payload[start : start+2*n])
	if !ok || !plausibleFileName(name) {
		return "", false
	}
	return name, true
}

// decodeUTF16LE converts a little-endian UTF-16 byte slice to a string,
// reporting false on malformed input (unpaired surrogates or an odd byte
// count).
func decodeUTF16LE(b []byte) (string, bool) {
	if len(b)%2 != 0 {
		return "", false
	}
	u := make([]uint16, len(b)/2)
	for i := range u {
		u[i] = binary.LittleEndian.Uint16(b[i*2:])
	}
	runes := utf16.Decode(u)
	var sb strings.Builder
	sb.Grow(len(runes))
	for _, r := range runes {
		if r == 0 {
			return "", false
		}
		sb.WriteRune(r)
	}
	return sb.String(), true
}

// plausibleFileName rejects strings that are structurally unlikely to be a
// file path reported by the kernel: control characters, NULs, path-shaped
// separators are required (kernel name events report "C:\..." or
// "\Device\..."-style paths), and nothing longer than a Windows path.
func plausibleFileName(s string) bool {
	if len(s) < 3 || len(s) > maxDecodedName {
		return false
	}
	hasSep := strings.ContainsAny(s, `\/`)
	if !hasSep {
		return false
	}
	for _, r := range s {
		if r < 0x20 {
			return false
		}
	}
	return true
}

// classic file-operation opcodes carried by Kernel-File events (evntrace.h
// EVENT_TRACE_TYPE_IO_*). They are best-effort: when an opcode is not one of
// the classic set (e.g. provider-routed differently on a given Windows
// build), the record is logged with a generic action rather than a guessed
// verb. Pinned by the Windows escape tests.
const (
	fileOpCreate = 32 // EVENT_TRACE_TYPE_IO_CREATE
	fileOpRead   = 33
	fileOpWrite  = 34
	fileOpClose  = 35
	fileOpDelete = 36
	fileOpRename = 37
	fileOpName   = 43 // EVENT_TRACE_TYPE_IO_NAME
)

// kernelFileAction maps a Kernel-File opcode to the audit event action.
func kernelFileAction(opcode uint8) string {
	switch opcode {
	case fileOpCreate:
		return "create"
	case fileOpRead:
		return "read"
	case fileOpWrite:
		return "write"
	case fileOpClose:
		return "close"
	case fileOpDelete:
		return "delete"
	case fileOpRename:
		return "rename"
	case fileOpName:
		return "name"
	default:
		return "file-op"
	}
}
