package ui

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestProgressNonTTYFallback(t *testing.T) {
	origCI := os.Getenv("CI")
	t.Cleanup(func() { os.Setenv("CI", origCI) })
	os.Setenv("CI", "true") // force non-interactive
	var buf bytes.Buffer
	p := NewProgress(&buf, 4)
	p.Success("Checking platform")
	p.Success("Installing runtime")
	p.Failure("Verifying installation")

	out := buf.String()
	if !strings.Contains(out, "[1/4]") {
		t.Errorf("Progress non-TTY should contain [1/4], got %q", out)
	}
	if !strings.Contains(out, "OK") {
		t.Error("Progress non-TTY should contain OK")
	}
	if !strings.Contains(out, "FAIL") {
		t.Error("Progress non-TTY should contain FAIL")
	}
	// Ensure deterministic bracket format, not spinner frames
	if strings.Contains(out, "⠋") || strings.Contains(out, "⠙") {
		t.Error("Progress non-TTY should not contain spinner braille")
	}
}

func TestProgressColorDisabled(t *testing.T) {
	origNoColor := os.Getenv("NO_COLOR")
	origCI := os.Getenv("CI")
	t.Cleanup(func() {
		os.Setenv("NO_COLOR", origNoColor)
		os.Setenv("CI", origCI)
	})
	os.Setenv("NO_COLOR", "1")
	os.Setenv("CI", "true")
	var buf bytes.Buffer
	p := NewProgress(&buf, 2)
	p.Success("Installing CLI")
	out := buf.String()
	if strings.Contains(out, "\x1b[") {
		t.Error("Progress with NO_COLOR should not contain ANSI")
	}
}

func TestPrintStaticSteps(t *testing.T) {
	var buf bytes.Buffer
	origCI := os.Getenv("CI")
	t.Cleanup(func() { os.Setenv("CI", origCI) })
	os.Setenv("CI", "true")
	PrintStaticSteps(&buf, []string{"a", "b"})
	out := buf.String()
	if !strings.Contains(out, "[1/2]") || !strings.Contains(out, "[2/2]") {
		t.Errorf("PrintStaticSteps should print bracketed steps, got %q", out)
	}
}

func TestProgressIsAnimatedRespectsCI(t *testing.T) {
	origCI := os.Getenv("CI")
	t.Cleanup(func() { os.Setenv("CI", origCI) })
	os.Setenv("CI", "true")
	var buf bytes.Buffer
	p := NewProgress(&buf, 1)
	if p.isAnimated() {
		t.Error("Progress should not be animated in CI")
	}
}
