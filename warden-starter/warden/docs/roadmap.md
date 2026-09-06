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
      [docs/security.md](security.md)

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
      (18 servers in [`testdata/compat/matrix.yaml`](compatibility.md#manifest),
      published in [`docs/compatibility.md`](compatibility.md):
      14 pass, 2 conditional, 2 fail)
- [x] For every failure, classify it — is it a Warden bug, a policy-schema
      gap (some access pattern the schema can't express yet), or a server
      doing something inherently incompatible with sandboxing (e.g.
      expecting arbitrary filesystem access by design)?
      (see [docs/compatibility.md](compatibility.md#failures-classified))
- [x] Recruit a small external beta group of MCP server maintainers/users
      (5-10 people) to run their own servers under Warden and report
      friction — this is the first real signal on whether the policy
      schema is usable by people who didn't design it
      (program + report template: [`docs/beta.md`](beta.md),
      [compatibility report template](beta.md#report-template))
- [x] Turn the matrix into a public compatibility page/README table, so
      prospective users can check "will this work with my server" before
      installing
      ([`docs/compatibility.md`](compatibility.md), README table,
      mkdocs nav)
- [x] Add every server from the matrix as a permanent regression fixture
      under `testdata/`, so a future change can't silently break
      compatibility with something that used to work
      ([`testdata/compat/`](compatibility.md#pinned-policies), enforced by
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