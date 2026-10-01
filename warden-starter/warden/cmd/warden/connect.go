package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/warden-sandbox/warden/internal/envfilter"
	"github.com/warden-sandbox/warden/internal/mcpbridge"
	"github.com/warden-sandbox/warden/internal/policy"
	"github.com/warden-sandbox/warden/internal/privatefile"
	"github.com/warden-sandbox/warden/internal/sandbox"
)

func printConnectHelp(cmd string) {
	fmt.Fprint(os.Stderr, `Usage:
  warden connect --rules rules.json --policy policy.yaml [--backend auto] -- <command> [args...]
  warden serve --rules rules.json --policy policy.yaml --token-env WARDEN_GATEWAY_TOKEN [--listen 127.0.0.1:8788] -- <command> [args...]
  warden serve --rules rules.json --upstream https://host/mcp --upstream-token-env UPSTREAM_TOKEN --token-env WARDEN_GATEWAY_TOKEN
  warden stop --control-file <private-control.json>

connect speaks stdio; serve speaks authenticated Streamable HTTP. Local
upstreams always use digest-pinned Warden run. Remote upstreams are filtered,
not process-sandboxed. Rules deny tools/resources/prompts unless listed.
Protocol: 2025-11-25, bounded request/response subset. No MRTR, tasks,
subscriptions, pagination, roots, sampling or elicitation. Unsupported paths
return correlated errors. HTTP disconnect kills a pending local connection.

serve options: --public-url https://your-host/mcp --origins https://your-host
  --issuer https://trusted-issuer --jwks-file trusted-local-jwks.json (RS256,
  audience = public URL, mcp scope; mutually exclusive with token-env)
  --control-file new-private-file --journal private-events.jsonl
  --allow-loopback-upstream (explicit numeric loopback test endpoint only)
Gateway binds numeric loopback; public hosting requires your TLS reverse proxy.
Secrets come from explicit environment names, never command-line values.
`)
}

type connectionOptions struct {
	policyPath, rulesPath, backend, remote, remoteToken string
	loopback                                            bool
	command                                             []string
	journal                                             string
	protectedEnv                                        []string
	controlPath                                         string
	expectedPolicyHash, expectedRulesHash               string
}

func connectionFactory(ctx context.Context, o connectionOptions) (mcpbridge.Factory, mcpbridge.Rules, error) {
	r, rulesHash, e := mcpbridge.LoadRules(o.rulesPath)
	if e != nil {
		return nil, r, e
	}
	var p policy.Policy
	if o.expectedRulesHash != "" && o.expectedRulesHash != rulesHash {
		return nil, r, fmt.Errorf("rules changed before launch")
	}
	var digest string
	var warden string
	env := []string{}
	if o.remote == "" {
		abs, e := filepath.Abs(o.policyPath)
		if e != nil || o.policyPath == "" {
			return nil, r, fmt.Errorf("local upstream requires --policy")
		}
		o.policyPath = abs
		b, e := os.ReadFile(abs)
		if e != nil {
			return nil, r, e
		}
		digest = mcpbridge.Hash(b)
		if o.expectedPolicyHash != "" && o.expectedPolicyHash != digest {
			return nil, r, fmt.Errorf("policy changed before launch")
		}
		p, e = policy.LoadVerified(abs, digest)
		if e != nil {
			return nil, r, e
		}
		if len(o.command) == 0 {
			o.command = p.Command
		}
		for _, name := range p.Env.Allow {
			for _, protected := range o.protectedEnv {
				if protected != "" && name == protected {
					return nil, r, fmt.Errorf("gateway credentials cannot be granted to a local upstream")
				}
			}
		}
		if len(o.command) == 0 {
			return nil, r, fmt.Errorf("local command required")
		}
		if o.backend, e = sandbox.Resolve(o.backend); e != nil {
			return nil, r, e
		}
		if o.controlPath != "" {
			if e = outsideUpstream(o.controlPath, p, o.command, o.backend); e != nil {
				return nil, r, e
			}
		}
		warden, e = os.Executable()
		if e != nil {
			return nil, r, e
		}
		// Only reviewed upstream env names plus Warden's runtime/state setup
		// reach the subprocess; gateway authentication values are excluded.
		allow := append(append([]string{}, p.Env.Allow...), "PATH", "HOME", "XDG_STATE_HOME")
		env = envfilter.Filter(os.Environ(), allow)
	} else {
		if o.policyPath != "" || len(o.command) > 0 {
			return nil, r, fmt.Errorf("remote mode cannot mix with local command/policy")
		}
		if _, e = mcpbridge.NewRemote(o.remote, o.remoteToken, o.loopback, r.MaxBytes); e != nil {
			return nil, r, e
		}
	}
	if o.journal == "" {
		dir, e := os.UserConfigDir()
		if e != nil {
			return nil, r, e
		}
		o.journal = filepath.Join(dir, "warden", "gateway", "events.jsonl")
	}
	if info, e := os.Lstat(o.journal); e == nil && !info.Mode().IsRegular() {
		return nil, r, fmt.Errorf("journal symlink refused")
	}
	abs, e := canonicalOutput(o.journal)
	if e != nil {
		return nil, r, e
	}
	if e = outsideUpstream(abs, p, o.command, o.backend); e != nil {
		return nil, r, e
	}
	if e = os.MkdirAll(filepath.Dir(abs), 0700); e != nil {
		return nil, r, e
	}
	if info, e := os.Lstat(abs); e == nil && !info.Mode().IsRegular() {
		return nil, r, fmt.Errorf("journal symlink refused")
	}
	f, e := os.OpenFile(abs, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if e != nil {
		return nil, r, e
	}
	if e = privatefile.Protect(abs, false); e != nil {
		_ = f.Close()
		return nil, r, e
	}
	j := &mcpbridge.Journal{Writer: f}
	go func() { <-ctx.Done(); _ = f.Close() }()
	return func(parent context.Context, id string) (*mcpbridge.Session, error) {
		var up mcpbridge.Upstream
		var e error
		if o.remote != "" {
			up, e = mcpbridge.NewRemote(o.remote, o.remoteToken, o.loopback, r.MaxBytes)
		} else {
			args := []string{"run", "--policy", o.policyPath, "--policy-sha256", digest, "--backend", o.backend, "--"}
			args = append(args, o.command...)
			up, e = mcpbridge.StartProcess(parent, warden, args, env, os.Stderr, r.MaxBytes, func(method string) error {
				return j.Record(id, method, "server-originated capability not forwarded", false)
			})
		}
		if e != nil {
			return nil, e
		}
		return mcpbridge.NewSession(up, r, j, id), nil
	}, r, nil
}

func cmdConnect(cmd string, args []string) error {
	if cmd == "stop" {
		return cmdStop(args)
	}
	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	pp := fs.String("policy", "", "local process policy")
	rp := fs.String("rules", "", "MCP rules JSON")
	backend := fs.String("backend", "auto", "sandbox")
	remote := fs.String("upstream", "", "remote HTTP URL")
	remoteEnv := fs.String("upstream-token-env", "", "upstream credential env name")
	loop := fs.Bool("allow-loopback-upstream", false, "test upstream")
	journal := fs.String("journal", "", "private decision journal")
	listen := fs.String("listen", "127.0.0.1:8788", "numeric loopback")
	public := fs.String("public-url", "", "TLS proxy endpoint")
	tokenEnv := fs.String("token-env", "", "downstream bearer env name")
	issuer := fs.String("issuer", "", "trusted OAuth issuer")
	jwks := fs.String("jwks-file", "", "trusted JWKS")
	origins := fs.String("origins", "", "comma-separated exact browser origins")
	control := fs.String("control-file", "", "private stop credentials")
	if e := fs.Parse(args); e != nil {
		return e
	}
	if *rp == "" {
		return fmt.Errorf("--rules required")
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if *remoteEnv != "" && *remoteEnv == *tokenEnv {
		return fmt.Errorf("upstream and downstream credential names must differ")
	}
	if *pp != "" {
		p, e := policy.Load(*pp)
		if e != nil {
			return e
		}
		for _, name := range p.Env.Allow {
			if (name == *tokenEnv && *tokenEnv != "") || (name == *remoteEnv && *remoteEnv != "") {
				return fmt.Errorf("gateway credentials cannot be granted to a local upstream")
			}
		}
	}
	if *remoteEnv != "" && os.Getenv(*remoteEnv) == "" {
		return fmt.Errorf("upstream credential environment variable is empty")
	}
	var auth *mcpbridge.Auth
	var e error
	if cmd == "serve" {
		auth, e = mcpbridge.LoadAuth(os.Getenv(*tokenEnv), *issuer, *public, *jwks)
		if e != nil {
			return e
		}
	}
	if *control != "" {
		*control, e = canonicalOutput(*control)
		if e != nil {
			return e
		}
	}
	factory, r, e := connectionFactory(ctx, connectionOptions{policyPath: *pp, rulesPath: *rp, backend: *backend, remote: *remote, remoteToken: os.Getenv(*remoteEnv), loopback: *loop, command: fs.Args(), journal: *journal, controlPath: *control, protectedEnv: []string{*tokenEnv, *remoteEnv}})
	if e != nil {
		return e
	}
	if cmd == "connect" {
		s, e := factory(ctx, mcpbridge.RandomID())
		if e != nil {
			return e
		}
		defer s.Close()
		messages := make(chan []byte)
		readError := make(chan error, 1)
		go func() {
			scanner := bufio.NewScanner(os.Stdin)
			scanner.Buffer(make([]byte, 4096), r.MaxBytes)
			for scanner.Scan() {
				b := append([]byte{}, scanner.Bytes()...)
				m, _ := mcpbridge.Parse(b)
				if m.Method == "notifications/cancelled" {
					cancel()
					break
				}
				select {
				case messages <- b:
				case <-ctx.Done():
					return
				}
			}
			readError <- scanner.Err()
			cancel()
		}()
		for {
			select {
			case b := <-messages:
				out := s.Handle(ctx, b)
				if len(out) > 0 {
					if _, e = fmt.Fprintln(os.Stdout, string(out)); e != nil {
						return e
					}
				}
			case e := <-readError:
				return e
			case <-ctx.Done():
				return nil
			}
		}
	}
	host, _, e := net.SplitHostPort(*listen)
	if e != nil || net.ParseIP(host) == nil || !net.ParseIP(host).IsLoopback() {
		return fmt.Errorf("gateway listener must use numeric loopback")
	}
	l, e := net.Listen("tcp", *listen)
	if e != nil {
		return e
	}
	defer l.Close()
	if *public == "" {
		*public = "http://" + l.Addr().String() + "/mcp"
	}
	controlToken := ""
	if *control != "" {
		controlToken = mcpbridge.RandomID()
		u, _ := url.Parse(*public)
		record, _ := json.Marshal(map[string]string{"url": "http://" + l.Addr().String() + "/__warden/stop", "authority": u.Host, "token": controlToken})
		if e = writeExclusive(*control, record); e != nil {
			return e
		}
		defer os.Remove(*control)
	}
	var allowed []string
	if *origins != "" {
		allowed = strings.Split(*origins, ",")
	}
	h, e := mcpbridge.NewHTTP(ctx, factory, auth, *public, allowed, r.MaxBytes, controlToken, cancel)
	if e != nil {
		return e
	}
	defer h.Close()
	server := &http.Server{Handler: h, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 32768}
	go func() {
		<-ctx.Done()
		h.Close()
		shutdown, c := context.WithTimeout(context.Background(), 5*time.Second)
		defer c()
		_ = server.Shutdown(shutdown)
	}()
	fmt.Fprintln(os.Stderr, "MCP gateway listening:", *public, "protocol", mcpbridge.Protocol)
	e = server.Serve(l)
	if e == http.ErrServerClosed {
		return nil
	}
	return e
}
func cmdStop(args []string) error {
	fs := flag.NewFlagSet("stop", flag.ContinueOnError)
	path := fs.String("control-file", "", "private gateway control")
	if e := fs.Parse(args); e != nil {
		return e
	}
	if *path == "" || fs.NArg() != 0 {
		return fmt.Errorf("--control-file required")
	}
	info, e := os.Lstat(*path)
	if e != nil || !info.Mode().IsRegular() || info.Size() > 16384 {
		return fmt.Errorf("invalid control file")
	}
	b, e := os.ReadFile(*path)
	if e != nil {
		return e
	}
	if mcpbridge.StrictJSON(b) != nil {
		return fmt.Errorf("invalid control JSON")
	}
	var r struct{ URL, Authority, Token string }
	if json.Unmarshal(b, &r) != nil || len(r.Token) < 32 {
		return fmt.Errorf("invalid control record")
	}
	u, e := url.Parse(r.URL)
	if e != nil || u.Scheme != "http" || u.User != nil || net.ParseIP(u.Hostname()) == nil || !net.ParseIP(u.Hostname()).IsLoopback() || u.Path != "/__warden/stop" || u.RawQuery != "" || u.Fragment != "" {
		return fmt.Errorf("control endpoint must be numeric loopback")
	}
	req, _ := http.NewRequest("POST", r.URL, nil)
	req.Host = r.Authority
	req.Header.Set("X-Warden-Control", r.Token)
	c := &http.Client{Timeout: 5 * time.Second, Transport: &http.Transport{Proxy: nil}, CheckRedirect: func(*http.Request, []*http.Request) error { return fmt.Errorf("control redirects refused") }}
	resp, e := c.Do(req)
	if e != nil {
		return fmt.Errorf("gateway control unavailable")
	}
	defer resp.Body.Close()
	if resp.StatusCode != 202 {
		return fmt.Errorf("stop refused")
	}
	fmt.Println("Managed connections stopping. Independent provider effects cannot be undone.")
	return nil
}
