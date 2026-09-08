package ui

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestBannerConstant(t *testing.T) {
	if Banner == "" {
		t.Fatal("Banner constant is empty")
	}
	lines := strings.Split(Banner, "\n")
	if len(lines) != 5 {
		t.Fatalf("Banner should be 5 lines, got %d", len(lines))
	}
	if BannerWidth() == 0 {
		t.Fatal("BannerWidth is 0")
	}
	if BannerWidth() > 80 {
		t.Fatalf("BannerWidth %d exceeds 80 col target", BannerWidth())
	}
	if !strings.Contains(Banner, "██") {
		t.Error("Banner should contain block characters")
	}
}

func TestPrintBanner(t *testing.T) {
	origNoColor := os.Getenv("NO_COLOR")
	origColumns := os.Getenv("COLUMNS")
	origWidth := os.Getenv("WARDEN_TERM_WIDTH")
	t.Cleanup(func() {
		os.Setenv("NO_COLOR", origNoColor)
		os.Setenv("COLUMNS", origColumns)
		os.Setenv("WARDEN_TERM_WIDTH", origWidth)
	})
	os.Setenv("WARDEN_TERM_WIDTH", "80")
	os.Setenv("NO_COLOR", "1")
	var buf bytes.Buffer
	n, err := PrintBanner(&buf)
	if err != nil || n == 0 {
		t.Fatalf("PrintBanner failed: n=%d err=%v", n, err)
	}
	out := buf.String()
	if !strings.Contains(out, "██") {
		t.Error("PrintBanner output missing block art in NO_COLOR mode")
	}
	if !strings.Contains(out, "MCP SERVER SANDBOX") {
		t.Error("PrintBanner should contain subtitle")
	}
	// Ensure no ANSI codes when NO_COLOR=1
	if strings.Contains(out, "\x1b[") {
		t.Error("PrintBanner emitted ANSI when NO_COLOR=1")
	}
}

func TestPrintBannerNarrowFallback(t *testing.T) {
	origCols := os.Getenv("COLUMNS")
	origW := os.Getenv("WARDEN_TERM_WIDTH")
	t.Cleanup(func() {
		os.Setenv("COLUMNS", origCols)
		os.Setenv("WARDEN_TERM_WIDTH", origW)
	})
	os.Setenv("COLUMNS", "40")
	os.Setenv("WARDEN_TERM_WIDTH", "40")
	var buf bytes.Buffer
	_, err := PrintBanner(&buf)
	if err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	// Narrow should fallback to header, not large art
	if strings.Contains(out, "██     ██") {
		t.Error("Narrow PrintBanner should not contain large art")
	}
	if !strings.Contains(out, "WARDEN") {
		t.Error("Narrow PrintBanner should contain WARDEN header")
	}
}

func TestPrintHeader(t *testing.T) {
	var buf bytes.Buffer
	_, err := PrintHeader(&buf, HeaderOptions{ShowVersion: true})
	if err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "WARDEN") {
		t.Error("PrintHeader missing WARDEN")
	}
	if !strings.Contains(out, "MCP Server Sandbox Runtime") {
		t.Error("PrintHeader missing subtitle")
	}
	if !strings.Contains(out, "Version") {
		t.Error("PrintHeader with ShowVersion should contain Version line")
	}
}

func TestColorEnabledRespectsNoColor(t *testing.T) {
	orig := os.Getenv("NO_COLOR")
	t.Cleanup(func() { os.Setenv("NO_COLOR", orig) })
	os.Setenv("NO_COLOR", "1")
	if ColorEnabled() {
		t.Error("ColorEnabled should be false when NO_COLOR=1")
	}
	os.Unsetenv("NO_COLOR")
	// TERM=dumb
	origTerm := os.Getenv("TERM")
	t.Cleanup(func() { os.Setenv("TERM", origTerm) })
	os.Setenv("TERM", "dumb")
	os.Setenv("NO_COLOR", "")
	if ColorEnabled() {
		t.Error("ColorEnabled should be false when TERM=dumb")
	}
}

func TestColorHelpersRespectNoColor(t *testing.T) {
	orig := os.Getenv("NO_COLOR")
	t.Cleanup(func() { os.Setenv("NO_COLOR", orig) })
	os.Setenv("NO_COLOR", "1")
	if got := Green("hello"); got != "hello" {
		t.Errorf("Green with NO_COLOR should be plain, got %q", got)
	}
	if got := Red("hello"); got != "hello" {
		t.Errorf("Red with NO_COLOR should be plain, got %q", got)
	}
	if got := Cyan("hello"); got != "hello" {
		t.Errorf("Cyan with NO_COLOR should be plain, got %q", got)
	}
}

func TestSupportsUnicodeFallback(t *testing.T) {
	orig := os.Getenv("WARDEN_NO_UNICODE")
	t.Cleanup(func() { os.Setenv("WARDEN_NO_UNICODE", orig) })
	os.Setenv("WARDEN_NO_UNICODE", "1")
	if SupportsUnicode() {
		t.Error("SupportsUnicode should be false when WARDEN_NO_UNICODE=1")
	}
	if got := CheckMark(); got != "OK" {
		t.Errorf("CheckMark fallback should be OK, got %q", got)
	}
	if got := CrossMark(); got != "x" {
		t.Errorf("CrossMark fallback should be x, got %q", got)
	}
}

func TestIsCI(t *testing.T) {
	orig := os.Getenv("CI")
	t.Cleanup(func() { os.Setenv("CI", orig) })
	os.Setenv("CI", "true")
	if !IsCI() {
		t.Error("IsCI should be true when CI=true")
	}
	os.Setenv("CI", "false")
	// Save and clear other CI indicator env vars that IsCI checks, so they
	// don't interfere with the CI=false assertion.
	ciVars := []string{"GITHUB_ACTIONS", "GITLAB_CI", "JENKINS_URL", "TF_BUILD", "CIRCLECI", "BUILDKITE", "TEAMCITY_VERSION"}
	saved := make(map[string]string)
	for _, k := range ciVars {
		saved[k] = os.Getenv(k)
		os.Unsetenv(k)
	}
	t.Cleanup(func() {
		for _, k := range ciVars {
			if v := saved[k]; v != "" {
				os.Setenv(k, v)
			}
		}
	})
	if IsCI() {
		t.Error("IsCI should be false when CI=false")
	}
	os.Unsetenv("CI")
	// GitHub Actions
	origGH := os.Getenv("GITHUB_ACTIONS")
	t.Cleanup(func() { os.Setenv("GITHUB_ACTIONS", origGH) })
	os.Setenv("GITHUB_ACTIONS", "true")
	if !IsCI() {
		t.Error("IsCI should be true when GITHUB_ACTIONS=true")
	}
}

func TestTerminalWidthDefault(t *testing.T) {
	orig := os.Getenv("COLUMNS")
	origW := os.Getenv("WARDEN_TERM_WIDTH")
	t.Cleanup(func() {
		os.Setenv("COLUMNS", orig)
		os.Setenv("WARDEN_TERM_WIDTH", origW)
	})
	os.Unsetenv("COLUMNS")
	os.Unsetenv("WARDEN_TERM_WIDTH")
	if got := TerminalWidth(); got != 80 {
		t.Errorf("TerminalWidth default should be 80, got %d", got)
	}
	os.Setenv("COLUMNS", "120")
	if got := TerminalWidth(); got != 120 {
		t.Errorf("TerminalWidth should respect COLUMNS, got %d", got)
	}
}

func TestPrintWelcome(t *testing.T) {
	origNoColor := os.Getenv("NO_COLOR")
	t.Cleanup(func() { os.Setenv("NO_COLOR", origNoColor) })
	os.Setenv("NO_COLOR", "1")
	var buf bytes.Buffer
	n, err := PrintWelcome(&buf)
	if err != nil || n == 0 {
		t.Fatalf("PrintWelcome failed: n=%d err=%v", n, err)
	}
	out := buf.String()
	if !strings.Contains(out, "Welcome to Warden") {
		t.Error("PrintWelcome should contain Welcome to Warden")
	}
	if !strings.Contains(out, "warden init") {
		t.Error("PrintWelcome should mention warden init")
	}
	if !strings.Contains(out, "Fail-closed") && !strings.Contains(out, "fails closed") {
		t.Error("PrintWelcome should mention fail-closed")
	}
}
