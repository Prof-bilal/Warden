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
- [ ] Local egress proxy with hostname allowlist
- [ ] Network namespace setup that forces the sandboxed process through the
      proxy (no direct egress path)
- [ ] Block DNS resolution for non-allowlisted hosts (not just the TCP
      connect — prevents leakage via DNS queries themselves)
- [ ] Audit logger: structured log of every file/network attempt, allowed
      or blocked

## M3 — Usability layer
- [ ] `warden trace` — run unsandboxed but instrumented, log all access
      attempts
- [ ] `warden init` — generate a starter policy.yaml from a trace log
- [ ] `warden logs` — tail/inspect the audit log
- [ ] Resource limits: memory and wall-clock timeout enforcement, process
      killed cleanly on breach

## M4 — macOS support
- [ ] `sandbox-exec`/Seatbelt backend
- [ ] Docker fallback backend for when native sandboxing isn't available
- [ ] Backend auto-detection (`warden run` picks the best available backend
      unless `--backend` is passed explicitly)

## M5 — Polish & distribution
- [ ] Package as a single static binary (Homebrew tap, npm wrapper for
      Node users, or both)
- [ ] Docs site with a policy schema reference and worked examples
- [ ] Example gallery: policies for 4-5 popular MCP servers (filesystem,
      GitHub, Slack, a database connector) so people can copy-paste rather
      than write a policy from scratch
- [ ] Security review pass — this is a security tool, so a documented
      threat model and known-limitations section matters before anyone
      relies on it

## M6 — Stretch goals (post-1.0)
- [ ] Windows backend (AppContainer or WSL2 delegation)
- [ ] Interactive approval mode — prompt the user the first time a server
      requests an access not in the policy, rather than hard-failing
- [ ] Integration with MCP gateways (e.g. wrap servers registered in a
      gateway config automatically)

---

**First thing to actually build:** M1's policy parser + bwrap wrapper. It's
the smallest slice that proves the core mechanism (transparent stdio +
filesystem restriction) actually works before investing in the network
proxy, which is the more complex piece.
