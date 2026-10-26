// This file holds the pure helpers of the Windows ETW auditor: decoding a
// file path out of a Kernel-File event payload and choosing the audit action
// for an operation. The helpers only touch []byte, so they are unit-tested on
// every OS while the session machinery lives in etw.go (//go:build windows).

package windows

import (
	"encoding/binary"
	"fmt"
	"strings"
	"unicode/utf16"
)

// maxDecodedName is the longest file name Warden will accept from an event
// payload. Kernel events report names shorter than MAX_PATH; anything longer
// is not a name we can trust to be a path.
const maxDecodedName = 260

// Microsoft-Windows-Kernel-File is a manifest provider: events are told apart
// by event ID (the opcode is 0) and names are NUL-terminated UTF-16 strings at
// a fixed offset per event. x64 layouts (pointers are 8 bytes):
//
//	10 NameCreate, 11 NameDelete:     FileKey, FileName                       -> 8
//	12 Create, 30 CreateNewFile:      Irp, FileObject, IssuingThreadId,
//	                                  CreateOptions, CreateAttributes,
//	                                  ShareAccess, FileName                   -> 32
//	26 DeletePath, 27 RenamePath:     Irp, FileObject, FileKey,
//	                                  ExtraInformation, IssuingThreadId,
//	                                  InfoClass, FilePath                     -> 40
//
// Pinned on a real host by TestEtwFileActivityAudited.
const (
	kfNameCreate    = 10
	kfNameDelete    = 11
	kfCreate        = 12
	kfDeletePath    = 26
	kfRenamePath    = 27
	kfCreateNewFile = 30
)

var kernelFileNameOffset = map[uint16]int{
	kfNameCreate:    8,
	kfNameDelete:    8,
	kfCreate:        32,
	kfCreateNewFile: 32,
	kfDeletePath:    40,
	kfRenamePath:    40,
}

// decodeKernelFileEvent extracts the file path from a Kernel-File event
// payload. It reports false for events without a name or a name Warden cannot
// verify, so undecodable events are dropped rather than logged with a guess.
func decodeKernelFileEvent(id uint16, payload []byte) (string, bool) {
	off, ok := kernelFileNameOffset[id]
	if !ok || off >= len(payload) {
		return "", false
	}
	b := payload[off:]
	end := -1
	for k := 0; k+1 < len(b) && k/2 <= maxDecodedName; k += 2 {
		if b[k] == 0 && b[k+1] == 0 {
			end = k
			break
		}
	}
	if end <= 0 {
		return "", false
	}
	name, ok := decodeUTF16LE(b[:end])
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

// kernelFileAction maps a Kernel-File event ID to the audit event action.
func kernelFileAction(id uint16) string {
	switch id {
	case kfCreate:
		return "open"
	case kfCreateNewFile:
		return "create"
	case kfDeletePath, kfNameDelete:
		return "delete"
	case kfRenamePath:
		return "rename"
	case kfNameCreate:
		return "name"
	default:
		return "file-op"
	}
}

// kfOperationEnd is the Kernel-File completion event (Irp, ExtraInformation,
// Status; x64 offsets 0, 8, 16). The request events above report only the
// attempt, so the outcome of an open, delete or rename comes from the
// OperationEnd whose Irp matches the request's.
const kfOperationEnd = 24

// kernelFileIrp returns the IRP pointer of a request event that is completed
// by an OperationEnd. Name events (10, 11) carry a FileKey instead and report
// false.
func kernelFileIrp(id uint16, payload []byte) (uint64, bool) {
	switch id {
	case kfCreate, kfCreateNewFile, kfDeletePath, kfRenamePath:
	default:
		return 0, false
	}
	if len(payload) < 8 {
		return 0, false
	}
	return binary.LittleEndian.Uint64(payload), true
}

// decodeOperationEnd extracts the IRP pointer and NTSTATUS of an
// OperationEnd payload.
func decodeOperationEnd(payload []byte) (irp uint64, status uint32, ok bool) {
	if len(payload) < 20 {
		return 0, 0, false
	}
	return binary.LittleEndian.Uint64(payload), binary.LittleEndian.Uint32(payload[16:]), true
}

// ntStatusNames names the failures a sandboxed open commonly ends with.
var ntStatusNames = map[uint32]string{
	0xC0000022: "STATUS_ACCESS_DENIED",
	0xC0000034: "STATUS_OBJECT_NAME_NOT_FOUND",
	0xC0000035: "STATUS_OBJECT_NAME_COLLISION",
	0xC000003A: "STATUS_OBJECT_PATH_NOT_FOUND",
	0xC0000043: "STATUS_SHARING_VIOLATION",
	0xC00000BA: "STATUS_FILE_IS_A_DIRECTORY",
	0xC0000103: "STATUS_NOT_A_DIRECTORY",
}

// ntStatusReason describes a failed NTSTATUS for an audit record, like the
// errno the Linux backend records for a failed syscall.
func ntStatusReason(status uint32) string {
	if name, ok := ntStatusNames[status]; ok {
		return fmt.Sprintf("NTSTATUS 0x%08X %s", status, name)
	}
	return fmt.Sprintf("NTSTATUS 0x%08X", status)
}

// ntSuccess reports whether an NTSTATUS is a success or informational code
// (NT_SUCCESS: the severity bit is clear).
func ntSuccess(status uint32) bool { return int32(status) >= 0 }
