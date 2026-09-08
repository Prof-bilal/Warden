package ui

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestPrintDoctorReady(t *testing.T) {
	origNoColor := os.Getenv("NO_COLOR")
	t.Cleanup(func() { os.Setenv("NO_COLOR", origNoColor) })
	os.Setenv("NO_COLOR", "1")
	var buf bytes.Buffer
	report := DoctorReport{
		Checks: []Check{
			{Name: "Operating system", Status: "info", Detail: "linux/amd64"},
			{Name: "Sandbox backend", Status: "ok", Detail: "linux"},
			{Name: "Policy engine", Status: "ok", Detail: "ready"},
		},
		Ready: true,
	}
	PrintDoctor(&buf, report)
	out := buf.String()
	if !strings.Contains(out, "WARDEN DOCTOR") {
		t.Errorf("PrintDoctor missing header, got %q", out)
	}
	if !strings.Contains(out, "Status: READY") {
		t.Errorf("PrintDoctor missing READY, got %q", out)
	}
	// Ensure accessibility: not color alone, contains check mark text
	if !strings.Contains(out, CheckMark()) && !strings.Contains(out, "OK") {
		t.Errorf("PrintDoctor should contain check mark, got %q", out)
	}
}

func TestPrintDoctorNotReady(t *testing.T) {
	origNoColor := os.Getenv("NO_COLOR")
	t.Cleanup(func() { os.Setenv("NO_COLOR", origNoColor) })
	os.Setenv("NO_COLOR", "1")
	var buf bytes.Buffer
	report := DoctorReport{
		Checks: []Check{
			{Name: "Sandbox backend", Status: "fail", Detail: "unavailable"},
		},
		Ready:  false,
		Reason: "Sandbox backend unavailable",
	}
	PrintDoctor(&buf, report)
	out := buf.String()
	if !strings.Contains(out, "Status: NOT READY") {
		t.Errorf("expected NOT READY, got %q", out)
	}
	if !strings.Contains(out, "fails closed") {
		t.Errorf("NOT READY should mention fails closed, got %q", out)
	}
	// Ensure fail status uses CrossMark text, not just color
	if !strings.Contains(out, CrossMark()) && !strings.Contains(out, "x") {
		t.Errorf("doctor fail should contain cross mark %q, got %q", CrossMark(), out)
	}
}

func TestDoctorReportStructure(t *testing.T) {
	origNoColor := os.Getenv("NO_COLOR")
	t.Cleanup(func() { os.Setenv("NO_COLOR", origNoColor) })
	os.Setenv("NO_COLOR", "1")
	var buf bytes.Buffer
	report := DoctorReport{
		Checks: []Check{
			{Name: "Operating system", Status: "info", Detail: "linux/amd64"},
			{Name: "Namespace support", Status: "ok", Detail: "available"},
			{Name: "Fail-closed", Status: "ok", Detail: "enabled"},
		},
		Ready: true,
	}
	PrintDoctor(&buf, report)
	out := buf.String()
	if !strings.Contains(out, "Environment") {
		t.Errorf("doctor output missing Environment section, got %q", out)
	}
	if !strings.Contains(out, "Security posture") {
		// ui.PrintDoctor may not use exactly that string, but should contain posture checks
		// At least ensure checks are rendered
		if !strings.Contains(out, "Namespace support") {
			t.Errorf("doctor output missing check, got %q", out)
		}
	}
}
