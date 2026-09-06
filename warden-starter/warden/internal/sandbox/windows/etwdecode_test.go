package windows

import (
	"encoding/binary"
	"strings"
	"testing"
)

// utf16LEBytes encodes s as little-endian UTF-16 bytes for building synthetic
// Kernel-File payloads.
func utf16LEBytesForTest(s string) []byte {
	out := make([]byte, 0, len(s)*2)
	for _, r := range s {
		var u [2]byte
		binary.LittleEndian.PutUint16(u[:], uint16(r))
		out = append(out, u[:]...)
	}
	return out
}

func TestDecodeKernelFileEvent(t *testing.T) {
	cases := []struct {
		name    string
		payload func() []byte
		want    string
		ok      bool
	}{
		{
			name: "name event layout (FileKey then name at 16)",
			payload: func() []byte {
				b := make([]byte, 18)
				n := utf16LEBytesForTest(`C:\data\in\file.txt`)
				binary.LittleEndian.PutUint16(b[16:18], uint16(len(n)/2))
				return append(b, n...)
			},
			want: `C:\data\in\file.txt`,
			ok:   true,
		},
		{
			name: "rename event layout (extra prefix, name at 24)",
			payload: func() []byte {
				b := make([]byte, 26)
				n := utf16LEBytesForTest(`\Device\HarddiskVolume3\tmp\renamed.bin`)
				binary.LittleEndian.PutUint16(b[24:26], uint16(len(n)/2))
				return append(b, n...)
			},
			want: `\Device\HarddiskVolume3\tmp\renamed.bin`,
			ok:   true,
		},
		{
			name: "create event layout (name at 28)",
			payload: func() []byte {
				b := make([]byte, 30)
				n := utf16LEBytesForTest(`C:\Users\me\out\new.log`)
				binary.LittleEndian.PutUint16(b[28:30], uint16(len(n)/2))
				return append(b, n...)
			},
			want: `C:\Users\me\out\new.log`,
			ok:   true,
		},
		{
			name: "payload too short",
			payload: func() []byte {
				return []byte{0, 1, 2, 3}
			},
			ok: false,
		},
		{
			name: "length beyond payload",
			payload: func() []byte {
				b := make([]byte, 18)
				binary.LittleEndian.PutUint16(b[16:18], 500)
				return b
			},
			ok: false,
		},
		{
			name: "length of zero",
			payload: func() []byte {
				b := make([]byte, 20)
				binary.LittleEndian.PutUint16(b[16:18], 0)
				return b
			},
			ok: false,
		},
		{
			name: "non-path trailing bytes do not decode",
			payload: func() []byte {
				// Random-looking tail that is not a plausible path.
				b := make([]byte, 18)
				noise := []byte{0x01, 0x00, 0x02, 0x00, 0xff, 0xff, 0x00, 0x00, 0x01, 0x00}
				binary.LittleEndian.PutUint16(b[16:18], uint16(len(noise)/2))
				return append(b, noise...)
			},
			ok: false,
		},
		{
			name: "trailing bytes after the name are ignored",
			payload: func() []byte {
				b := make([]byte, 18)
				n := utf16LEBytesForTest(`C:\x`)
				binary.LittleEndian.PutUint16(b[16:18], uint16(len(n)/2))
				body := append(b, n...)
				return append(body, 0x00, 0x00) // padding past the declared length
			},
			want: `C:\x`,
			ok:   true, // the length prefix bounds the name; padding is ignored
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := decodeKernelFileEvent(c.payload())
			if ok != c.ok {
				t.Fatalf("decodeKernelFileEvent ok = %v, want %v", ok, c.ok)
			}
			if ok && !strings.EqualFold(got, c.want) {
				t.Fatalf("decoded name = %q, want %q", got, c.want)
			}
		})
	}
}

func TestDecodeKernelFileEventRejectsControlChars(t *testing.T) {
	payload := make([]byte, 18)
	name := []byte{0x43, 0x00, 0x3a, 0x00, 0x5c, 0x00, 0x09, 0x00} // "C:\" + tab
	binary.LittleEndian.PutUint16(payload[16:18], uint16(len(name)/2))
	if _, ok := decodeKernelFileEvent(append(payload, name...)); ok {
		t.Fatal("decoded a path containing a control character")
	}
}

func TestKernelFileAction(t *testing.T) {
	if kernelFileAction(fileOpCreate) != "create" {
		t.Fatal("create opcode mapped wrong")
	}
	if kernelFileAction(fileOpDelete) != "delete" {
		t.Fatal("delete opcode mapped wrong")
	}
	if kernelFileAction(fileOpRename) != "rename" {
		t.Fatal("rename opcode mapped wrong")
	}
	if kernelFileAction(200) != "file-op" {
		t.Fatal("unknown opcode should fall back to file-op")
	}
}
