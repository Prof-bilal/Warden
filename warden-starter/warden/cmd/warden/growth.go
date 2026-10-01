package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	evidence "github.com/warden-sandbox/warden/internal/attestation"
	"github.com/warden-sandbox/warden/internal/maintenance"
	"github.com/warden-sandbox/warden/internal/mcpbridge"
	"github.com/warden-sandbox/warden/internal/policy"
	"github.com/warden-sandbox/warden/internal/sandbox"
	"github.com/warden-sandbox/warden/internal/version"
)

func printGrowthHelp(cmd string) {
	fmt.Fprint(os.Stderr, `Usage:
  warden creator init --output <new-directory>
  warden creator keygen --output <new-private-key> (public key: .pub)
  warden creator definition-hash --manifest <single-tool-definition.json>
  warden creator check --policy <yaml> --rules <json> --artifact <directory> --manifest <checks.json> --key <private-key> --output <evidence.json> [--backend auto] -- <prepared-server> [args...]
  warden creator verify --policy <yaml> --rules <json> --artifact <directory> --manifest <checks.json> --public-key <trusted-key> --revocations <trusted-fresh-json> --evidence <evidence.json> [--badge <new.svg>] -- <same-command> [args...]
  warden policy-diff --before <yaml> --after <yaml>
  warden report record --file <local.jsonl> --client <id> --event <setup-completed|setup-failed|first-protected-task|repeat-use|rollback|badge-activation> [--seconds N]
  warden report summary --file <local.jsonl>

Creator checks run a sandboxed stdio workflow using fake/test data. Evidence is
signed only after allowed and policy-denied operations pass with a liveness
control. Verifiers require an independently trusted public key and fresh local
revocation snapshot; no self-trust from an evidence upload. Platform, Warden,
artifact tree, policy, rules, manifest and command are bound. Evidence expires
after 7 days. A passing badge describes these checks, not a safety certification.
Reports are explicit local events; no network telemetry or tool bodies.
`)
}
func cmdGrowth(cmd string, args []string) error {
	if cmd == "policy-diff" {
		fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
		before := fs.String("before", "", "old policy")
		after := fs.String("after", "", "new policy")
		if e := fs.Parse(args); e != nil {
			return e
		}
		if fs.NArg() != 0 || *before == "" || *after == "" {
			return fmt.Errorf("--before and --after required")
		}
		a, e := policy.Load(*before)
		if e != nil {
			return e
		}
		b, e := policy.Load(*after)
		if e != nil {
			return e
		}
		return json.NewEncoder(os.Stdout).Encode(maintenance.Compare(a, b))
	}
	if cmd == "report" {
		return cmdReport(args)
	}
	if len(args) == 0 {
		return fmt.Errorf("creator subcommand required")
	}
	sub := args[0]
	fs := flag.NewFlagSet("creator "+sub, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	output := fs.String("output", "", "new output")
	pp := fs.String("policy", "", "policy")
	rp := fs.String("rules", "", "rules")
	artifact := fs.String("artifact", "", "artifact tree")
	manifest := fs.String("manifest", "", "test manifest")
	keyfile := fs.String("key", "", "private key")
	pub := fs.String("public-key", "", "trusted public key")
	rev := fs.String("revocations", "", "trusted fresh revocations")
	envelope := fs.String("evidence", "", "signed report")
	badge := fs.String("badge", "", "new badge output")
	backend := fs.String("backend", "auto", "sandbox")
	if e := fs.Parse(args[1:]); e != nil {
		return e
	}
	switch sub {
	case "definition-hash":
		if *manifest == "" || fs.NArg() != 0 {
			return fmt.Errorf("--manifest single tool definition required")
		}
		raw, e := os.ReadFile(*manifest)
		if e != nil {
			return e
		}
		if len(raw) > 1024*1024 {
			return fmt.Errorf("definition too large")
		}
		digest, e := mcpbridge.DefinitionHash(raw)
		if e != nil {
			return e
		}
		fmt.Println(digest)
		return nil
	case "init":
		if *output == "" || fs.NArg() != 0 {
			return fmt.Errorf("new --output directory required")
		}
		if e := os.Mkdir(*output, 0700); e != nil {
			return e
		}
		for name, body := range creatorTemplates {
			if e := writeExclusive(filepath.Join(*output, name), []byte(body)); e != nil {
				return e
			}
		}
		fmt.Println("Candidate creator kit created. Replace fixture paths, review grants, and use fake data before checks.")
		return nil
	case "keygen":
		if *output == "" || fs.NArg() != 0 {
			return fmt.Errorf("--output required")
		}
		pub, key, e := ed25519.GenerateKey(rand.Reader)
		if e != nil {
			return e
		}
		if e = writeExclusive(*output, []byte(hex.EncodeToString(key)+"\n")); e != nil {
			return e
		}
		if e = writeExclusive(*output+".pub", []byte(hex.EncodeToString(pub)+"\n")); e != nil {
			_ = os.Remove(*output)
			return e
		}
		fmt.Println("Issuer key created. Keep the private key private; distribute the public fingerprint through an independent trusted channel.")
		return nil
	case "check", "verify":
	default:
		return fmt.Errorf("unknown creator subcommand")
	}
	if *pp == "" || *rp == "" || *artifact == "" || *manifest == "" {
		return fmt.Errorf("policy, rules, artifact and manifest required")
	}
	command := fs.Args()
	if len(command) == 0 {
		p, e := policy.Load(*pp)
		if e != nil {
			return e
		}
		command = p.Command
	}
	if len(command) == 0 {
		return fmt.Errorf("same prepared command required")
	}
	warden, e := os.Executable()
	if e != nil {
		return e
	}
	if sub == "verify" {
		r, status, e := evidence.Verify(*envelope, evidence.Inputs{Artifact: *artifact, Policy: *pp, Rules: *rp, Manifest: *manifest, PublicKey: *pub, Revocations: *rev, Warden: warden, Command: command}, time.Now().UTC())
		_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"status": status, "issuer": r.Issuer, "scope": "signed sandboxed stdio checks; not a safety certification"})
		if e != nil {
			return e
		}
		if *badge != "" {
			return writeExclusive(*badge, []byte(evidence.Badge(status)))
		}
		return nil
	}
	if *output == "" || *keyfile == "" {
		return fmt.Errorf("--output and private --key required")
	}
	key, e := evidence.Key(*keyfile, true)
	if e != nil {
		return e
	}
	ph, e := evidence.FileHash(*pp)
	if e != nil {
		return e
	}
	reviewedPolicy, e := policy.LoadVerified(*pp, ph)
	if e != nil {
		return e
	}
	resolved, e := sandbox.Resolve(*backend)
	if e != nil {
		return e
	}
	if e = outsideUpstream(*keyfile, reviewedPolicy, command, resolved); e != nil {
		return e
	}
	privatePath, e := filepath.Abs(*keyfile)
	if e != nil {
		return e
	}
	privatePath, e = filepath.EvalSymlinks(privatePath)
	if e != nil {
		return e
	}
	artifactPath, e := filepath.Abs(*artifact)
	if e != nil {
		return e
	}
	artifactPath, e = filepath.EvalSymlinks(artifactPath)
	if e != nil {
		return e
	}
	if policy.CoversFile(reviewedPolicy, privatePath) || policy.Within(privatePath, artifactPath) {
		return fmt.Errorf("issuer private key must be outside the artifact and upstream grants")
	}
	for _, grant := range append(append([]string{}, reviewedPolicy.Filesystem.Read...), reviewedPolicy.Filesystem.Write...) {
		if real, e := filepath.EvalSymlinks(grant); e == nil && policy.Within(privatePath, real) {
			return fmt.Errorf("issuer private key overlaps a resolved upstream grant")
		}
	}
	var spec evidence.Manifest
	raw, e := evidence.DecodeFile(*manifest, &spec)
	if e != nil {
		return e
	}
	if spec.Version != 1 || len(spec.Checks) < 2 || len(spec.Checks) > 50 {
		return fmt.Errorf("manifest needs 2–50 checks and version 1")
	}
	allow, deny := false, false
	names := map[string]bool{}
	for _, test := range spec.Checks {
		if test.Name == "" || names[test.Name] || len(test.Name) > 128 {
			return fmt.Errorf("unique check names required")
		}
		names[test.Name] = true
		if test.Method != "tools/call" && test.Method != "resources/read" && test.Method != "prompts/get" {
			return fmt.Errorf("check method must exercise tools/resources/prompts")
		}
		switch test.Decision {
		case "allow":
			allow = true
		case "deny":
			deny = true
		default:
			return fmt.Errorf("check decision must be allow or deny")
		}
	}
	if !allow || !deny {
		return fmt.Errorf("both allowed and denied checks required")
	}
	before, e := evidence.TreeHash(*artifact)
	if e != nil {
		return e
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	rh, e := evidence.FileHash(*rp)
	if e != nil {
		return e
	}
	factory, _, e := connectionFactory(ctx, connectionOptions{policyPath: *pp, rulesPath: *rp, backend: resolved, command: command, expectedPolicyHash: ph, expectedRulesHash: rh})
	if e != nil {
		return e
	}
	s, e := factory(ctx, mcpbridge.RandomID())
	if e != nil {
		return e
	}
	defer s.Close()
	init := s.Handle(ctx, []byte(`{"jsonrpc":"2.0","id":0,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"warden-creator-check","version":"1"}}}`))
	im, e := mcpbridge.Parse(init)
	if e != nil || len(im.Error) > 0 {
		return fmt.Errorf("upstream initialization failed")
	}
	if out := s.Handle(ctx, []byte(`{"jsonrpc":"2.0","method":"notifications/initialized"}`)); len(out) > 0 {
		return fmt.Errorf("upstream initialization notification failed")
	}
	checks := []evidence.Check{}
	for i, test := range spec.Checks {
		id, _ := json.Marshal(i + 1)
		response := s.Handle(ctx, mcpbridge.Encode(mcpbridge.Message{JSONRPC: "2.0", ID: id, Method: test.Method, Params: test.Params}))
		m, e := mcpbridge.Parse(response)
		passed := e == nil
		if test.Decision == "allow" {
			var result struct {
				IsError bool `json:"isError"`
			}
			_ = json.Unmarshal(m.Result, &result)
			passed = passed && len(m.Error) == 0 && !result.IsError && (test.Contains == "" || strings.Contains(string(m.Result), test.Contains))
		} else {
			var failure struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			}
			_ = json.Unmarshal(m.Error, &failure)
			passed = passed && failure.Code == -32001 && (failure.Message == "tool or arguments denied by policy" || failure.Message == "resource denied by policy" || failure.Message == "prompt denied by policy" || failure.Message == "tool absent or definition changed")
		}
		if !passed {
			return fmt.Errorf("creator check %d failed; no evidence signed", i+1)
		}
		// A denied operation must not pass merely because the target died.
		ping := s.Handle(ctx, []byte(`{"jsonrpc":"2.0","id":"liveness","method":"ping"}`))
		pm, e := mcpbridge.Parse(ping)
		if e != nil || len(pm.Error) > 0 {
			return fmt.Errorf("liveness control failed; no evidence signed")
		}
		checks = append(checks, evidence.Check{Name: test.Name, Decision: test.Decision, Passed: true})
	}
	after, e := evidence.TreeHash(*artifact)
	if e != nil || before != after {
		return fmt.Errorf("artifact changed during checks; no evidence signed")
	}
	finalPolicy, e := evidence.FileHash(*pp)
	if e != nil || finalPolicy != ph {
		return fmt.Errorf("policy changed during checks; no evidence signed")
	}
	finalRules, e := evidence.FileHash(*rp)
	if e != nil || finalRules != rh {
		return fmt.Errorf("rules changed during checks; no evidence signed")
	}
	wh, e := evidence.FileHash(warden)
	if e != nil {
		return e
	}
	argv, _ := json.Marshal(command)
	now := time.Now().UTC()
	r := evidence.Report{Version: 1, ArtifactSHA256: before, PolicySHA256: ph, RulesSHA256: rh, ManifestSHA256: mcpbridge.Hash(raw), CommandSHA256: mcpbridge.Hash(argv), WardenSHA256: wh, WardenVersion: version.Version, Platform: runtime.GOOS + "/" + runtime.GOARCH, Backend: resolved, Transport: "stdio", Protocol: mcpbridge.Protocol, IssuedAt: now, ExpiresAt: now.Add(7 * 24 * time.Hour), Checks: checks}
	b, e := evidence.Sign(r, ed25519.PrivateKey(key))
	if e != nil {
		return e
	}
	if e = writeExclusive(*output, b); e != nil {
		return e
	}
	fmt.Println("Signed allowed/denied stdio evidence created. Verification requires independent issuer trust and a fresh revocation snapshot.")
	return nil
}
func cmdReport(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("record or summary required")
	}
	fs := flag.NewFlagSet("report", flag.ContinueOnError)
	file := fs.String("file", "", "local report")
	client := fs.String("client", "", "client ID")
	event := fs.String("event", "", "enum event")
	seconds := fs.Int("seconds", 0, "duration")
	if e := fs.Parse(args[1:]); e != nil {
		return e
	}
	if *file == "" || fs.NArg() != 0 {
		return fmt.Errorf("explicit --file required")
	}
	switch args[0] {
	case "record":
		return maintenance.Append(*file, maintenance.Record{Time: time.Now().UTC().Truncate(time.Hour), Client: *client, Event: *event, Seconds: *seconds})
	case "summary":
		rows, e := maintenance.Read(*file)
		if e != nil {
			return e
		}
		return json.NewEncoder(os.Stdout).Encode(maintenance.Summarize(rows))
	default:
		return fmt.Errorf("unknown report subcommand")
	}
}

var creatorTemplates = map[string]string{
	"rules.json":          `{"version":1,"tools":{"read_text_file":{}},"resources":[],"prompts":[],"max_bytes":1048576,"timeout_seconds":30}`,
	"checks.json":         `{"version":1,"checks":[{"name":"read fixture","method":"tools/call","params":{"name":"read_text_file","arguments":{"path":"/absolute/test-data/hello.txt"}},"decision":"allow","contains":"hello"},{"name":"block write tool","method":"tools/call","params":{"name":"write_file","arguments":{"path":"/absolute/test-data/hello.txt","content":"blocked"}},"decision":"deny"}]}`,
	"README.md":           "# Warden creator kit\n\nThis kit is a candidate. Use a dedicated prepared artifact and disposable data, review policy/rules, then run `warden creator check`. It never invokes unsandboxed trace. A signature establishes issuer provenance; it does not certify safety. Keep normal agent trust/approval controls. Include the signed evidence, public key fingerprint, pinned policy/rules and verification command in your repository. Distribute public-key trust and fresh revocations independently.\n\n[![Warden: policy included](policy-included.svg)](README.md)\n",
	"policy-included.svg": evidence.Badge("policy-included"),
}
