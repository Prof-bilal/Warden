# Roadmap

Rough milestones, roughly in order. Each one should leave you with something
you could actually demo, not just an internal refactor.

## M0Repo groundwork
- [x] Folder structure, README, architecture doc (this scaffold)
- [x] Pick and reserve the project name (check GitHub/npm/crates.io/Homebrew
      for collisions before publishing anything)
- [x] `LICENSE` (MIT recommended for a project you want easy adoption of)
- [x] `CONTRIBUTING.md` + issue templates
- [x] CI skeleton (lint + build, even before there's real logic to test)

## M1Linux prototype (prove the core idea works)
- [x] Policy struct + YAML parsing, with schema validation and clear error
      messages on malformed policies
- [x] Shell out to `bwrap` with bind mounts derived from `filesystem.read`
      / `filesystem.write`
- [x] Transparent stdio passthrough from sandboxed process to parent
- [x] Manual test: wrap a real, simple MCP server (e.g. the reference
      filesystem MCP server) and confirm it works normally when the policy
      grants the right paths, and fails cleanly when it doesn't
- [x] **Demo-able milestone:** `warden run --policy policy.yaml -- <server>`
      works end-to-end on Linux for filesystem restriction only (no network
      enforcement yet)

**Known limitations (M1):** No network enforcement, no resource limits.

## M2Network enforcement
- [x] Local egress proxy with hostname allowlist
- [x] Network namespace setup that forces the sandboxed process through the
      proxy (no direct egress path)
- [x] Block DNS resolution for non-allowlisted hosts (not just the TCP
      connectprevents leakage via DNS queries themselves)
- [x] Audit logger: structured log of every file/network attempt, allowed
      or blocked

## M3Usability layer
- [x] `warden trace`run unsandboxed but instrumented, log all access
      attempts
- [x] `warden init`generate a starter policy.yaml from a trace log
- [x] `warden logs`tail/inspect the audit log
- [x] Resource limits: memory and wall-clock timeout enforcement, process
      killed cleanly on breach

## M4macOS support
- [x] `sandbox-exec`/Seatbelt backend
- [x] Docker fallback backend for when native sandboxing isn't available
- [x] Backend auto-detection (`warden run` picks the best available backend
      unless `--backend` is passed explicitly)

## M5Windows support
- [x] AppContainer backend with restricted token and explicit filesystem
      capabilities derived from the policy
- [x] Windows Filtering Platform (WFP) egress filtering and blocked DNS resolution
- [x] ETW-based file/network audit events (allowed/blocked)
- [x] Job Object memory and wall-clock limits with process-tree termination
- [x] Backend-aware Windows integration and escape tests (warden run --backend windows)

**Security gate:** Windows support must fail closed when AppContainer,
network filtering, auditing, or Job Object setup cannot be applied. A plain
`CreateProcess` fallback is never acceptable.

### M5 completion (ETW + CI + truth-sync)landed

**Policy decision:** real ETW auditing fails closed. If the trace session cannot
start, `warden run` refuses to runauditing is part of the security gate.
Because the kernel (WPP-style) providers accept a single enabling session, a
machine where another controller holds them (PerfView, an EDR) fails closed
with a clear error rather than running unaudited.

**What landed:**
- `etw.go` starts a real private real-time session (`StartTraceW` →
  `EnableTraceEx2` for Kernel-File + Kernel-Network → `OpenTraceW` →
  `ProcessTrace`) scoped to the sandboxed process tree, mapped into the
  `audit.Event` JSONL stream (`etwdecode.go` for the pure decode logic).
  Kernel-File records that carry a verifiable path become `file` events;
  Kernel-Network records are captured but their payloads are not mapped yet —
  blocked/allowed network audit is reported with exact host:port decisions by
  the egress proxy and the WFP deny filters (denied packets never reach a
  provider that could log them).
- Escape/lifecycle tests on the new Windows CI job pin the session lifecycle
  and assert a blocked network request lands in the audit log as
  `allowed=false`; see `etw_test.go` and `integration_test.go`.
- `.github/workflows/ci.yml` gains a `windows-latest` job (elevated runners,
  so the AppContainer/WFP/ETW tests execute instead of skipping). The
  syscalls.go "re-verify when CI gains a runner" caveat stays until that job
  is green; remove it after the first passing run.
- Docs and landing page now describe Windows as Ready with the elevation
  caveat (WFP/ETW require admin); see `docs/security.md`.

## M6Polish & distribution
- [x] Package as a single static binary (Homebrew tap formula, npm wrapper
      for Node users, plus manual download links in GitHub Releases)
- [x] Docs site with a policy schema reference and worked examples
      (MkDocs Material, deployed to GitHub Pages)
- [x] Example gallery: policies for 5 popular MCP servers (filesystem,
      GitHub, Slack, PostgreSQL, Brave Search) so people can copy-paste
      rather than write a policy from scratch
- [x] Security review passdocumented threat model (7 categories),
      known-limitations table, and best-practices section in
      [docs/security.md](warden-starter/warden/docs/security.md)

## M7Stretch goals (post-1.0)
- [x] Interactive approval modeprompt the user the first time a server
      requests an access not in the policy, rather than hard-failing
      (`warden run --approve`: network prompts apply live via the egress
      proxy; filesystem prompts save to the policy and restart. Seatbelt /
      Docker / Windows support network approval onlyno live filesystem
      signal there. See docs/approve.md.)
- [x] Integration with MCP gateways (e.g. wrap servers registered in a
      gateway config automatically)

## M8Real-world MCP server compatibility testing
- [x] Build a compatibility matrix: run Warden against the 15-20 most-used
      public MCP servers (filesystem, GitHub, Slack, Postgres/SQLite,
      Google Drive, a couple of the popular community ones) and record
      pass/fail plus the exact policy each one needed
      (      18 servers in [`testdata/compat/matrix.yaml`](warden-starter/warden/testdata/compat/matrix.yaml),
      published      in [`docs/compatibility.md`](warden-starter/warden/docs/compatibility.md):
      14 pass, 2 conditional, 2 fail)
- [x] For every failure, classify itis it a Warden bug, a policy-schema
      gap (some access pattern the schema can't express yet), or a server
      doing something inherently incompatible with sandboxing (e.g.
      expecting arbitrary filesystem access by design)?
      (see [docs/compatibility.md](warden-starter/warden/docs/compatibility.md#failures-classified))
- [x] Recruit a small external beta group of MCP server maintainers/users
      (5-10 people) to run their own servers under Warden and report
      frictionthis is the first real signal on whether the policy
      schema is usable by people who didn't design it
      (program + report template: [`docs/beta.md`](warden-starter/warden/docs/beta.md),
      [`.github/ISSUE_TEMPLATE/compat_report.md`](.github/ISSUE_TEMPLATE/compat_report.md))
- [x] Turn the matrix into a public compatibility page/README table, so
      prospective users can check "will this work with my server" before
      installing
      ([`docs/compatibility.md`](warden-starter/warden/docs/compatibility.md), README table,
      mkdocs nav)
- [x] Add every server from the matrix as a permanent regression fixture
      under `testdata/`, so a future change can't silently break
      compatibility with something that used to work
      ([`testdata/compat/`](warden-starter/warden/testdata/compat/), enforced by
      `go test ./internal/compat/`)
- [x] Triage and fix the highest-impact gaps found (most-used servers
      first) before moving on to M6/M7-style polish
      (fixed: read/write overlap coalescing via `policy.Normalize`;
      bare-launcher PATH resolution via `policy.ResolveExecutable`;
      remaining gaps documented as schema-gap vs inherent)

**Why this comes before polish:** M6's distribution and docs work assumes
the tool actually works against real servers, not just the fixtures written
alongside the code that implements each backend. This milestone is the
reality check between "the backends pass their own tests" and "this is
something people can actually adopt."

---

**First thing to actually build:** M1's policy parser + bwrap wrapper. It's
the smallest slice that proves the core mechanism (transparent stdio +
filesystem restriction) actually works before investing in the network
proxy, which is the more complex piece.

---

## The Long Bet — AI agent & browser isolation (Speculative)

> **Status:** research-grounded strategy sketch, explicitly **not** a
> near-term roadmap item. This is a sequenced bet with evidence, kept here
> so the direction is visible without implying a commitment. Sections cite
> Warden mechanisms that already exist and would be reused.

### The problem

AI agents that control a desktop (see [Anthropic's computer use](https://www.anthropic.com/news/3-5-models-and-computer-use))
can see screens, move mice, click buttons, and type — full desktop control.
Running these agents on your main computer is dangerous: a malicious
instruction hidden in a website could steal passwords, send emails, or
exfiltrate files. Today users must either buy a VPS ($5–50/mo) or configure
a local VM (complex, resource-heavy). Neither is simple.

**Agents in scope:** Claude Computer Use, OpenAI Operator (browser agent),
AutoGPT, and general-purpose agent platforms — all require full computer
access to operate.

**Security risks introduced:** prompt injection (attacker-hidden
instructions in web content — OpenAI states this is "unfixable"),
same-origin bypass in agentic browsers, credential theft via file access,
data exfiltration, and supply-chain weaknesses in browser automation
libraries (CVE-2025-47241 affected 1,500+ projects).

**Current solutions and why they fall short:**

| Option | Pros | Cons |
|---|---|---|
| Cloud VM (EC2, DigitalOcean) | Complete isolation | Recurring cost, technical barrier |
| Local VM (VirtualBox/QEMU) | Free, complete isolation | Complex setup, resource-heavy |
| Docker | Lightweight | No GUI support — poor fit for desktop control |

None are simple, purpose-built for AI agents, or auditable.

### The opportunity

A new `warden agent` command would run any AI agent inside an isolated
environment defined by the **same `policy.yaml` model** already used for
MCP servers:

```bash
warden agent run --policy policy.yaml -- agent-command
```

Reuse map — everything below exists today for MCP servers:

- **Tiers:** Tier 1 (namespace/bwrap) → Tier 2 (container/Docker) →
  Tier 3 (microVM/Firecracker)
- **Display:** virtual display (Xvfb) for screen access
- **Policy:** same `policy.yaml` restricts filesystem, network, env, limits
- **Audit:** log every agent action (JSONL stream)
- **Approval:** human-in-the-loop for sensitive actions (`--approve`)

**Competition:** cloud VMs and local VMs (cost/complexity), Daytona
(microVM, paid), and Warden itself today (namespaces only). Warden
differentiates with one policy model across all tiers and a local-first
approach.

### Sequencing (if signal justifies it)

1. **Phase 1:** Tier 1 agent mode — namespace isolation + virtual display
2. **Phase 2:** Tier 2 — container isolation
3. **Phase 3:** Tier 3 (microVM) + full approval flow
4. **Phase 4:** marketplace / policy templates for popular agents

### Sources

Anthropic [computer use](https://www.anthropic.com/news/3-5-models-and-computer-use) ·
OpenAI [Operator](https://openai.com/index/introducing-operator) ·
[AutoGPT](https://en.wikipedia.org/wiki/AutoGPT) ·
Techtimes [AI browser security review](https://techtimes.com/articles/318528/20260616/ai-browser-comparison-2026) ·
Axis Intelligence [browser agent security guide](https://axis-intelligence.com/browser-agent-security-risk-guide/) ·
PiunikaWeb [browser credential leak](https://piunikaweb.com/2026/06/25/chatgpt-atlas-perplexity-comet-ai-browsers-leaking-credentials/) ·
[OWASP LLM Top 10](https://owasp.org/www-project-top-10-for-large-language-model-applications) ·
[NIST AI RMF](https://www.nist.gov/itl/ai-risk-management-framework) ·
[Daytona](https://daytona.io/) ·
[Firecracker](https://firecracker-microvm.github.io/) ·
[Xvfb](https://www.x.org/releases/X11R7.7/doc/man/Xvfb)