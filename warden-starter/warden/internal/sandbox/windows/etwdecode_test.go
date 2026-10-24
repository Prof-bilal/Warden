package windows

import (
	"encoding/binary"
	"testing"
)

// kernelFilePayload builds a synthetic Kernel-File payload: off zero bytes of
// fixed fields followed by name as NUL-terminated UTF-16LE.
func kernelFilePayload(off int, name string) []byte {
	b := make([]byte, off)
	for _, r := range name {
		var u [2]byte
		binary.LittleEndian.PutUint16(u[:], uint16(r))
		b = append(b, u[:]...)
	}
	return append(b, 0, 0)
}

func TestDecodeKernelFileEvent(t *testing.T) {
	cases := []struct {
		name    string
		id      uint16
		payload []byte
		want    string
		ok      bool
	}{
		{"Create (12): name at 32", kfCreate, kernelFilePayload(32, `\Device\HarddiskVolume3\data\in.txt`), `\Device\HarddiskVolume3\data\in.txt`, true},
		{"CreateNewFile (30): name at 32", kfCreateNewFile, kernelFilePayload(32, `C:\out\new.log`), `C:\out\new.log`, true},
		{"NameCreate (10): name at 8", kfNameCreate, kernelFilePayload(8, `C:\data\file.txt`), `C:\data\file.txt`, true},
		{"RenamePath (27): path at 40", kfRenamePath, kernelFilePayload(40, `C:\tmp\renamed.bin`), `C:\tmp\renamed.bin`, true},
		{"unmapped event id", 16, kernelFilePayload(32, `C:\x\y`), "", false},
		{"payload shorter than offset", kfCreate, []byte{1, 2, 3}, "", false},
		{"no NUL terminator", kfNameCreate, kernelFilePayload(8, `C:\a\b`)[:16], "", false},
		{"empty name", kfNameCreate, kernelFilePayload(8, ""), "", false},
		{"no separator", kfNameCreate, kernelFilePayload(8, "plainname"), "", false},
	}
	for _, c := range cases {
		got, ok := decodeKernelFileEvent(c.id, c.payload)
		if ok != c.ok || got != c.want {
			t.Errorf("%s: got (%q, %v), want (%q, %v)", c.name, got, ok, c.want, c.ok)
		}
	}
}

func TestDecodeKernelFileEventRejectsControlChars(t *testing.T) {
	if _, ok := decodeKernelFileEvent(kfNameCreate, kernelFilePayload(8, "C:\a\x01b")); ok {
		t.Fatal("name with a control character accepted")
	}
}

func TestKernelFileAction(t *testing.T) {
	for id, want := range map[uint16]string{
		kfCreate: "open", kfCreateNewFile: "create", kfDeletePath: "delete",
		kfNameDelete: "delete", kfRenamePath: "rename", kfNameCreate: "name", 99: "file-op",
	} {
		if got := kernelFileAction(id); got != want {
			t.Errorf("kernelFileAction(%d) = %q, want %q", id, got, want)
		}
	}
}
