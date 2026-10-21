package main

import "testing"

func TestTruncateForTerminalClampsNarrowWidths(t *testing.T) {
	// COLUMNS is untrusted; none of these widths may panic or produce an
	// empty result.
	for _, w := range []int{-5, 0, 1, 14, 15, 16, 17, 18, 80} {
		if got := truncateForTerminal("a-very-long-command", w); got == "" {
			t.Fatalf("width %d produced an empty string", w)
		}
	}
	if got := truncateForTerminal("short", 80); got != "short" {
		t.Fatalf("short string was truncated: %q", got)
	}
	if got := truncateForTerminal("a-very-long-command", 16); len([]rune(got)) > 16 {
		t.Fatalf("width 16 produced %q (%d runes)", got, len([]rune(got)))
	}
}
