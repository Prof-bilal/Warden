package audit

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestImportStraceRecordsAllowedAndBlockedAccesses(t *testing.T) {
	trace := strings.NewReader(`[pid 12] openat(AT_FDCWD, "/allowed/file", O_RDONLY) = 3
[pid 12] openat(AT_FDCWD, "/hidden/file", O_RDONLY) = -1 ENOENT (No such file or directory)
[pid 12] connect(3, {sa_family=AF_INET, sin_port=htons(53), sin_addr=inet_addr("8.8.8.8")}, 16) = -1 ENETUNREACH (Network is unreachable)
`)
	var out bytes.Buffer
	if err := ImportStrace(trace, New(&out)); err != nil {
		t.Fatal(err)
	}
	var events []Event
	for _, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		var event Event
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Fatal(err)
		}
		events = append(events, event)
	}
	if len(events) != 3 {
		t.Fatalf("got %d events, want 3: %s", len(events), out.String())
	}
	if !events[0].Allowed || events[0].Resource != "/allowed/file" || events[0].Type != "file" {
		t.Errorf("allowed open event = %+v", events[0])
	}
	if events[1].Allowed || events[1].Resource != "/hidden/file" {
		t.Errorf("blocked open event = %+v", events[1])
	}
	if events[2].Allowed || events[2].Type != "network" || events[2].Resource != "8.8.8.8" {
		t.Errorf("blocked connect event = %+v", events[2])
	}
}
