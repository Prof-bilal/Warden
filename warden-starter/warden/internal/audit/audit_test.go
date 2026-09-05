package audit

import (
	"strings"
	"testing"
)

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
