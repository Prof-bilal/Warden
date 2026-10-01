package mcpbridge

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"time"
)

type Process struct {
	cmd     *exec.Cmd
	input   io.WriteCloser
	replies chan Message
	done    chan struct{}
	mu      sync.Mutex
	once    sync.Once
	limit   int
	stopped chan struct{}
}

// StartProcess must be given a Warden run command, never an upstream directly.
// CLI construction binds policy bytes and resolves the backend before this call.
func StartProcess(ctx context.Context, warden string, args []string, env []string, stderr io.Writer, limit int, denied func(string) error) (*Process, error) {
	if len(args) < 2 || args[0] != "run" {
		return nil, fmt.Errorf("managed process must launch Warden run")
	}
	cmd := exec.CommandContext(ctx, warden, args...)
	cmd.Env = env
	cmd.Stderr = stderr
	configureProcess(cmd)
	in, e := cmd.StdinPipe()
	if e != nil {
		return nil, e
	}
	out, e := cmd.StdoutPipe()
	if e != nil {
		_ = in.Close()
		return nil, e
	}
	if e = cmd.Start(); e != nil {
		_ = in.Close()
		return nil, e
	}
	p := &Process{cmd: cmd, input: in, replies: make(chan Message, 8), done: make(chan struct{}), limit: limit, stopped: make(chan struct{})}
	go func() {
		defer close(p.done)
		defer close(p.replies)
		s := bufio.NewScanner(out)
		s.Buffer(make([]byte, 4096), limit)
		for s.Scan() {
			m, e := Parse(s.Bytes())
			if e != nil {
				break
			}
			if m.Method != "" {
				if denied == nil || denied(m.Method) != nil {
					break
				}
				// Explicitly deny server-originated requests. Notifications have
				// no authority and are deliberately not advertised or forwarded.
				if len(m.ID) > 0 {
					p.mu.Lock()
					_, e = fmt.Fprintln(p.input, string(Error(m.ID, -32601, "server requests unsupported")))
					p.mu.Unlock()
					if e != nil {
						break
					}
				}
				continue
			}
			select {
			case p.replies <- m:
			case <-p.stopped:
				p.kill()
				_ = cmd.Wait()
				return
			case <-ctx.Done():
				p.kill()
				_ = cmd.Wait()
				return
			}
		}
		p.kill()
		_ = cmd.Wait()
	}()
	return p, nil
}
func (p *Process) kill() {
	p.once.Do(func() { close(p.stopped); _ = p.input.Close(); terminateProcess(p.cmd) })
}
func (p *Process) Close() error {
	p.kill()
	select {
	case <-p.done:
	case <-time.After(5 * time.Second):
		return fmt.Errorf("managed process cleanup timed out")
	}
	return nil
}
func (p *Process) Notify(ctx context.Context, m Message) error {
	// A pipe write can block before RoundTrip reaches its response select.
	// Request cancellation must also unblock that write and reap the process.
	finished := make(chan struct{})
	defer close(finished)
	go func() {
		select {
		case <-ctx.Done():
			// The caller cancels its per-request context after Notify returns.
			// If the write already finished, that cleanup must not kill a
			// healthy connection when both channels become ready together.
			select {
			case <-finished:
				return
			default:
				p.kill()
			}
		case <-finished:
		}
	}()
	p.mu.Lock()
	defer p.mu.Unlock()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	_, e := fmt.Fprintln(p.input, string(Encode(m)))
	return e
}
func (p *Process) RoundTrip(ctx context.Context, m Message) (Message, error) {
	if e := p.Notify(ctx, m); e != nil {
		return Message{}, e
	}
	select {
	case out, ok := <-p.replies:
		if !ok {
			return Message{}, fmt.Errorf("upstream closed")
		}
		if string(out.ID) != string(m.ID) {
			p.kill()
			return Message{}, fmt.Errorf("unsolicited response")
		}
		return out, nil
	case <-ctx.Done():
		p.kill()
		return Message{}, ctx.Err()
	}
}
