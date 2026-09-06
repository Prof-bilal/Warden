package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/warden-sandbox/warden/internal/version"
)

// captureStdout redirects os.Stdout for the duration of fn and returns
// whatever was written, so tests can assert on cmdVersion output.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	orig := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	fn()

	w.Close()
	return <-done
}

func TestCmdVersionDefault(t *testing.T) {
	out := captureStdout(t, cmdVersion)
	want := "warden version dev"
	if got := strings.TrimSpace(out); got != want {
		t.Fatalf("cmdVersion output = %q, want %q", got, want)
	}
}

func TestCmdVersionEmptyFallsBackToDev(t *testing.T) {
	orig := version.Version
	version.Version = ""
	t.Cleanup(func() { version.Version = orig })

	out := captureStdout(t, cmdVersion)
	want := "warden version dev"
	if got := strings.TrimSpace(out); got != want {
		t.Fatalf("cmdVersion with empty Version = %q, want %q", got, want)
	}
}

func TestCmdVersionStamped(t *testing.T) {
	orig := version.Version
	version.Version = "v9.9.9-test"
	t.Cleanup(func() { version.Version = orig })

	out := captureStdout(t, cmdVersion)
	want := "warden version v9.9.9-test"
	if got := strings.TrimSpace(out); got != want {
		t.Fatalf("cmdVersion with stamped Version = %q, want %q", got, want)
	}
}
