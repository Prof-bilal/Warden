package audit

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestEventOptionalFields(t *testing.T) {
	event := Event{Cmd: "node", Tool: "read_file", PID: 42, Hash: "abc", Kind: "command"}
	want := map[string]any{"cmd": "node", "tool": "read_file", "pid": float64(42), "hash": "abc", "kind": "command"}
	for _, input := range []Event{{}, event} {
		data, err := json.Marshal(input)
		if err != nil {
			t.Fatal(err)
		}
		var fields map[string]any
		if err := json.Unmarshal(data, &fields); err != nil {
			t.Fatal(err)
		}
		for key, value := range want {
			got, present := fields[key]
			if present != (input == event) || (present && got != value) {
				t.Errorf("field %q encoded incorrectly in %s", key, data)
			}
		}
	}
}

func TestReadEvents(t *testing.T) {
	events, err := ReadEvents(strings.NewReader("{\"timestamp\":\"2026-01-01T00:00:00Z\",\"type\":\"file\",\"action\":\"openat\",\"resource\":\"/x\",\"allowed\":true}\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Resource != "/x" {
		t.Fatalf("events = %+v", events)
	}
	if _, err := ReadEvents(strings.NewReader("not json\n")); err == nil {
		t.Fatal("expected malformed log error")
	}
}
