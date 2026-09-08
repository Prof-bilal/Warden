package ui

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// Progress renders polished step-by-step installation feedback with
// automatic TTY vs non-TTY degradation.
//
// In a TTY it shows an animated spinner (⠋⠙⠹…) that is short-lived and
// never leaves stray characters. In CI / piped output it falls back to
// deterministic "[n/total] message... OK" lines so logs stay clean.
type Progress struct {
	w       io.Writer
	total   int
	current int
	mu      sync.Mutex

	spinMu sync.Mutex
	spinning bool
	stopCh   chan struct{}
	doneCh   chan struct{}
}

// NewProgress creates a progress renderer writing to w.
func NewProgress(w io.Writer, total int) *Progress {
	if w == nil {
		w = os.Stderr
	}
	return &Progress{w: w, total: total, stopCh: make(chan struct{}), doneCh: make(chan struct{})}
}

// spinnerFrames are the braille spinner used by many modern CLIs. They are
// short and degrade gracefully: when Unicode is unavailable we use ASCII.
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
var asciiFrames = []string{"-", "\\", "|", "/"}

func frames() []string {
	if SupportsUnicode() {
		return spinnerFrames
	}
	return asciiFrames
}

// isAnimated reports whether we should animate.
func (p *Progress) isAnimated() bool {
	if IsCI() {
		return false
	}
	if IsNonInteractive(p.w) {
		return false
	}
	if !IsTerminalWriter(p.w) {
		return false
	}
	// Respect explicit opt-out.
	if os.Getenv("WARDEN_NO_SPINNER") != "" {
		return false
	}
	return true
}

// Start begins a spinner for the given message. It returns a function to
// call when the step completes. The spinner runs only in interactive TTYs;
// otherwise this is a no-op and the caller should use Step directly.
func (p *Progress) Start(msg string) func(success bool, finalMsg string) {
	if !p.isAnimated() {
		return func(bool, string) {}
	}
	p.spinMu.Lock()
	if p.spinning {
		p.spinMu.Unlock()
		return func(bool, string) {}
	}
	p.spinning = true
	p.stopCh = make(chan struct{})
	p.doneCh = make(chan struct{})
	p.spinMu.Unlock()

	fr := frames()
	go func() {
		defer close(p.doneCh)
		ticker := time.NewTicker(80 * time.Millisecond)
		defer ticker.Stop()
		idx := 0
		for {
			select {
			case <-p.stopCh:
				return
			case <-ticker.C:
				// Clear line and redraw spinner. Use \r to overwrite.
				frame := fr[idx%len(fr)]
				if ColorEnabled() {
					frame = Cyan(frame)
				}
				fmt.Fprintf(p.w, "\r%s %s   ", frame, msg)
				idx++
			}
		}
	}()
	return func(success bool, finalMsg string) {
		p.spinMu.Lock()
		if !p.spinning {
			p.spinMu.Unlock()
			return
		}
		p.spinning = false
		p.spinMu.Unlock()
		close(p.stopCh)
		<-p.doneCh
		// Clear spinner line.
		fmt.Fprintf(p.w, "\r%s\r", strings.Repeat(" ", 60))
		if finalMsg == "" {
			finalMsg = msg
		}
		if success {
			mark := CheckMark()
			if ColorEnabled() {
				mark = Green(mark)
			}
			fmt.Fprintf(p.w, "%s %s\n", mark, finalMsg)
		} else {
			mark := CrossMark()
			if ColorEnabled() {
				mark = Red(mark)
			}
			fmt.Fprintf(p.w, "%s %s\n", mark, finalMsg)
		}
	}
}

// Step records a completed step in non-animated mode. In animated mode it
// just prints a success line (caller should have used Start otherwise).
func (p *Progress) Step(msg string, success bool) {
	p.mu.Lock()
	p.current++
	cur := p.current
	total := p.total
	p.mu.Unlock()

	if p.isAnimated() {
		var mark string
		if success {
			mark = CheckMark()
			if ColorEnabled() {
				mark = Green(mark)
			}
		} else {
			mark = CrossMark()
			if ColorEnabled() {
				mark = Red(mark)
			}
		}
		fmt.Fprintf(p.w, "%s %s\n", mark, msg)
		return
	}
	// Non-TTY / CI fallback: deterministic bracketed progress.
	status := "OK"
	if !success {
		status = "FAIL"
	}
	if ColorEnabled() {
		if success {
			status = Green(status)
		} else {
			status = Red(status)
		}
	}
	if total > 0 {
		fmt.Fprintf(p.w, "[%d/%d] %s... %s\n", cur, total, msg, status)
	} else {
		fmt.Fprintf(p.w, "%s %s\n", CheckMark(), msg)
		if !success {
			fmt.Fprintf(p.w, "%s %s\n", CrossMark(), msg)
		}
	}
}

// Success is a convenience for a successful step.
func (p *Progress) Success(msg string) { p.Step(msg, true) }

// Failure is a convenience for a failed step.
func (p *Progress) Failure(msg string) { p.Step(msg, false) }

// Static helpers for one-shot install output without managing a Progress.

// PrintStaticSteps prints steps in the CI-friendly format without animation.
func PrintStaticSteps(w io.Writer, steps []string) {
	prog := NewProgress(w, len(steps))
	for _, s := range steps {
		prog.Success(s)
	}
}
