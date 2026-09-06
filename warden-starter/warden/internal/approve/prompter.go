package approve

import (
	"bufio"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/warden-sandbox/warden/internal/audit"
	"github.com/warden-sandbox/warden/internal/proxy"
)

// FileOutcome is the user's answer to a filesystem approval prompt.
// Unlike network grants, every non-deny outcome persists to the policy file:
// a restart reloads policy from disk, so a memory-only filesystem approval
// would silently do nothing.
type FileOutcome int

const (
	// FileDeny keeps running without the grant (default).
	FileDeny FileOutcome = iota
	// FileRecord saves the grant and keeps running; it takes effect on the
	// next run.
	FileRecord
	// FileRestart saves the grant and restarts the server immediately.
	FileRestart
)

// Prompter serializes all approval prompts for one run and remembers
// decisions so the user is asked once per resource ("the first time").
// The zero value is unusable; construct with NewPrompter.
type Prompter struct {
	mu         sync.Mutex
	timeout    time.Duration
	policyPath string
	logger     *audit.Logger

	netGranted map[string]bool
	netDenied  map[string]bool
	fileAsked  map[string]FileOutcome
}

// NewPrompter builds a Prompter for one sandboxed run. logger may be nil
// (decisions are then not audit-logged, e.g. in unit tests).
func NewPrompter(policyPath string, timeout time.Duration, logger *audit.Logger) *Prompter {
	return &Prompter{
		timeout:    timeout,
		policyPath: policyPath,
		logger:     logger,
		netGranted: make(map[string]bool),
		netDenied:  make(map[string]bool),
		fileAsked:  make(map[string]FileOutcome),
	}
}

// NetworkApprover returns the proxy.Approver closure for this run. It
// prompts once per host; later requests are resolved from memory (granted
// hosts also sit in the proxy allowlist, so they never reach this closure
// again — the cache here mainly suppresses re-prompts for denials and wins
// races between concurrent first requests).
func (p *Prompter) NetworkApprover() proxy.Approver {
	return func(host, port string) proxy.Decision { return p.DecideNetwork(host, port) }
}

// DecideNetwork resolves one blocked host:port to a proxy.Decision,
// prompting on the terminal for the first request per host.
func (p *Prompter) DecideNetwork(host, port string) proxy.Decision {
	key := strings.ToLower(host)
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.netDenied[key] {
		p.log("approval", "deny", hostPort(host, port), false, "previously denied by user this run; not re-prompting")
		return proxy.Deny
	}
	if p.netGranted[key] {
		return proxy.AllowSession
	}

	decision := p.promptNetworkLocked(host, port)
	switch decision {
	case proxy.AllowAndSave:
		// Persist before granting: if the save fails the user was promised
		// a durable grant, so fail closed instead of granting silently.
		if err := SaveHostGrant(p.policyPath, key); err != nil {
			p.log("approval", "deny", hostPort(host, port), false, fmt.Sprintf("user approved with save, but persisting failed: %v", err))
			p.netDenied[key] = true
			return proxy.Deny
		}
		p.netGranted[key] = true
	case proxy.AllowSession:
		p.netGranted[key] = true
	case proxy.AllowOnce:
		// Deliberately uncached: "once" means prompt again next time.
	default:
		decision = proxy.Deny
		p.netDenied[key] = true
	}
	return decision
}

// promptNetworkLocked asks about one host. Callers must hold p.mu so
// concurrent requests cannot interleave prompts.
func (p *Prompter) promptNetworkLocked(host, port string) proxy.Decision {
	question := fmt.Sprintf("warden approval: server requested network access to %s (not in network.allow)\n", hostPort(host, port))
	const options = "  [o] allow once  [s] allow for this session  [p] allow and save to policy  [d] deny (default): "
	answer, err := p.askLocked(question, options)
	if err != nil {
		p.log("approval", "deny", hostPort(host, port), false, fmt.Sprintf("prompt unavailable (%v); failing closed", err))
		return proxy.Deny
	}
	decision := parseNetAnswer(answer)
	var action, reason string
	switch decision {
	case proxy.AllowOnce:
		action, reason = "allow", "approved by user for this request only"
	case proxy.AllowSession:
		action, reason = "allow", "approved by user for this session"
	case proxy.AllowAndSave:
		action, reason = "allow", "approved by user and saved to policy"
	default:
		action, reason = "deny", fmt.Sprintf("denied by user (answer %q)", answer)
	}
	p.log("approval", action, hostPort(host, port), decision != proxy.Deny, reason)
	return decision
}

// PromptFile asks about one blocked filesystem access, represented by the
// proposed grant (see policy.ProposeFileGrant). It reports the outcome and
// whether this run prompted just now (false = replaying a cached answer, so
// the caller must not save twice).
func (p *Prompter) PromptFile(action, resource, grant string, write bool) (FileOutcome, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if outcome, asked := p.fileAsked[grant]; asked {
		return outcome, false
	}
	outcome := p.promptFileLocked(action, resource, grant, write)
	p.fileAsked[grant] = outcome
	return outcome, true
}

func (p *Prompter) promptFileLocked(action, resource, grant string, write bool) FileOutcome {
	kind, op := "READ", "read"
	if write {
		kind, op = "WRITE", "write"
	}
	question := fmt.Sprintf("warden approval: server tried to %s %s (%s blocked; not covered by filesystem grants)\nProposed grant: filesystem.%s += %s\nFilesystem grants need a server restart to take effect (bind mounts are fixed at spawn).\n", kind, resource, op, op, grant)
	const options = "  [r] save grant + restart server now  [k] save grant, keep running (next run)  [d] deny (default): "
	answer, err := p.askLocked(question, options)
	if err != nil {
		p.log("approval", "deny", grant, false, fmt.Sprintf("prompt unavailable (%v); failing closed", err))
		return FileDeny
	}
	outcome := parseFileAnswer(answer)
	var act, reason string
	switch outcome {
	case FileRestart:
		act, reason = "allow", fmt.Sprintf("approved by user with restart (server %s %s)", action, resource)
	case FileRecord:
		act, reason = "allow", fmt.Sprintf("approved by user for next run (server %s %s)", action, resource)
	default:
		outcome = FileDeny
		act, reason = "deny", fmt.Sprintf("denied by user (answer %q for server %s %s)", answer, action, resource)
	}
	p.log("approval", act, grant, outcome != FileDeny, reason)
	return outcome
}

// askLocked writes a question to the controlling terminal and reads one
// answer line. It never touches the process stdio, which belongs to the MCP
// client. Any failure (no terminal, timeout, read error) is returned so the
// caller can fail closed.
func (p *Prompter) askLocked(question, options string) (string, error) {
	tty, err := openConsole()
	if err != nil {
		return "", fmt.Errorf("no interactive terminal: %w", err)
	}
	defer tty.Close()
	if _, err := fmt.Fprint(tty, "\n"+question+options); err != nil {
		return "", fmt.Errorf("write prompt: %w", err)
	}

	type result struct {
		line string
		err  error
	}
	answer := make(chan result, 1)
	go func() {
		line, err := bufio.NewReader(tty).ReadString('\n')
		answer <- result{line, err}
	}()

	if p.timeout > 0 {
		select {
		case r := <-answer:
			if r.err != nil {
				return "", fmt.Errorf("read answer: %w", r.err)
			}
			return strings.TrimSpace(r.line), nil
		case <-time.After(p.timeout):
			// The reader goroutine leaks, blocked on the terminal; it dies
			// with the process and never touches shared state.
			fmt.Fprintln(tty, "\nwarden approval: timed out waiting for an answer")
			return "", fmt.Errorf("timed out after %s", p.timeout)
		}
	}
	r := <-answer
	if r.err != nil {
		return "", fmt.Errorf("read answer: %w", r.err)
	}
	return strings.TrimSpace(r.line), nil
}

func (p *Prompter) log(typ, action, resource string, allowed bool, reason string) {
	if p.logger == nil {
		return
	}
	_ = p.logger.Log(audit.Event{Type: typ, Action: action, Resource: resource, Allowed: allowed, Reason: reason})
}

// parseNetAnswer maps a prompt answer to a network decision. Empty or
// unrecognized input denies (fail closed).
func parseNetAnswer(answer string) proxy.Decision {
	switch strings.ToLower(strings.TrimSpace(answer)) {
	case "o", "once":
		return proxy.AllowOnce
	case "s", "session":
		return proxy.AllowSession
	case "p", "persist", "save", "y", "yes", "a", "allow":
		return proxy.AllowAndSave
	default:
		return proxy.Deny
	}
}

// parseFileAnswer maps a prompt answer to a filesystem outcome. Empty or
// unrecognized input denies (fail closed).
func parseFileAnswer(answer string) FileOutcome {
	switch strings.ToLower(strings.TrimSpace(answer)) {
	case "r", "restart":
		return FileRestart
	case "k", "keep", "record", "save", "next":
		return FileRecord
	default:
		return FileDeny
	}
}

func hostPort(host, port string) string {
	if port == "" {
		return host
	}
	return host + ":" + port
}
