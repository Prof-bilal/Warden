# Roadmap

Rough milestones, roughly in order. Each one should leave you with something
you could actually demo, not just an internal refactor.

## M0 — Repo groundwork
- [x] Folder structure, README, architecture doc (this scaffold)
- [x] Pick and reserve the project name (check GitHub/npm/crates.io/Homebrew
      for collisions before publishing anything)
- [x] `LICENSE` (MIT recommended for a project you want easy adoption of)
- [x] `CONTRIBUTING.md` + issue templates
- [x] CI skeleton (lint + build, even before there's real logic to test)

## M1 — Linux prototype (prove the core idea works)
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

## M2 — Network enforcement
- [x] Local egress proxy with hostname allowlist
- [x] Network namespace setup that forces the sandboxed process through the
      proxy (no direct egress path)
- [x] Block DNS resolution for non-allowlisted hosts (not just the TCP
      connect — prevents leakage via DNS queries themselves)
- [x] Audit logger: structured log of every file/network attempt, allowed
      or blocked

## M3 — Usability layer
- [x] `warden trace` — run unsandboxed but instrumented, log all access
      attempts
- [x] `warden init` — generate a starter policy.yaml from a trace log
- [x] `warden logs` — tail/inspect the audit log
- [x] Resource limits: memory and wall-clock timeout enforcement, process
      killed cleanly on breach

## M4 — macOS support
- [x] `sandbox-exec`/Seatbelt backend
- [x] Docker fallback backend for when native sandboxing isn't available
- [x] Backend auto-detection (`warden run` picks the best available backend
      unless `--backend` is passed explicitly)

## M5 — Windows support
- [x] AppContainer backend with restricted token and explicit filesystem
      capabilities derived from the policy
- [x] Windows Filtering Platform (WFP) egress filtering and blocked DNS resolution
- [x] ETW-based file/network audit events (allowed/blocked)
- [x] Job Object memory and wall-clock limits with process-tree termination
- [x] Backend-aware Windows integration and escape tests (warden run --backend windows)

**Security gate:** Windows support must fail closed when AppContainer,
network filtering, auditing, or Job Object setup cannot be applied. A plain
`CreateProcess` fallback is never acceptable.

### M5 completion (ETW + CI + truth-sync) — landed

**Policy decision:** real ETW auditing fails closed. If the trace session cannot
start, `warden run` refuses to run — auditing is part of the security gate.
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

## M6 — Polish & distribution
- [x] Package as a single static binary (Homebrew tap formula, npm wrapper
      for Node users, plus manual download links in GitHub Releases)
- [x] Docs site with a policy schema reference and worked examples
      (MkDocs Material, deployed to GitHub Pages)
- [x] Example gallery: policies for 5 popular MCP servers (filesystem,
      GitHub, Slack, PostgreSQL, Brave Search) so people can copy-paste
      rather than write a policy from scratch
- [x] Security review pass — documented threat model (7 categories),
      known-limitations table, and best-practices section in
      [docs/security.md](./docs/security.md)

## M7 — Stretch goals (post-1.0)
- [x] Interactive approval mode — prompt the user the first time a server
      requests an access not in the policy, rather than hard-failing
      (`warden run --approve`: network prompts apply live via the egress
      proxy; filesystem prompts save to the policy and restart. Seatbelt /
      Docker / Windows support network approval only — no live filesystem
      signal there. See docs/approve.md.)
- [x] Integration with MCP gateways (e.g. wrap servers registered in a
      gateway config automatically)

## M8 — Real-world MCP server compatibility testing
- [x] Build a compatibility matrix: run Warden against the 15-20 most-used
      public MCP servers (filesystem, GitHub, Slack, Postgres/SQLite,
      Google Drive, a couple of the popular community ones) and record
      pass/fail plus the exact policy each one needed
      (18 servers in [`testdata/compat/matrix.yaml`](./testdata/compat/matrix.yaml),
      published in [`docs/compatibility.md`](./docs/compatibility.md):
      14 pass, 2 conditional, 2 fail)
- [x] For every failure, classify it — is it a Warden bug, a policy-schema
      gap (some access pattern the schema can't express yet), or a server
      doing something inherently incompatible with sandboxing (e.g.
      expecting arbitrary filesystem access by design)?
      (see [docs/compatibility.md](./docs/compatibility.md#failures-classified))
- [x] Recruit a small external beta group of MCP server maintainers/users
      (5-10 people) to run their own servers under Warden and report
      friction — this is the first real signal on whether the policy
      schema is usable by people who didn't design it
      (program + report template: [`docs/beta.md`](./docs/beta.md),
      [`.github/ISSUE_TEMPLATE/compat_report.md`](./.github/ISSUE_TEMPLATE/compat_report.md))
- [x] Turn the matrix into a public compatibility page/README table, so
      prospective users can check "will this work with my server" before
      installing
      ([`docs/compatibility.md`](./docs/compatibility.md), README table,
      mkdocs nav)
- [x] Add every server from the matrix as a permanent regression fixture
      under `testdata/`, so a future change can't silently break
      compatibility with something that used to work
      ([`testdata/compat/`](./testdata/compat/), enforced by
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