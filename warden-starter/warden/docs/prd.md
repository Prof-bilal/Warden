# Product Requirements Document

## Problem

MCP servers are commonly run as plain local processes with the developer's full machine permissions. The project README describes this as a security gap: an MCP server can be malicious or simply buggy, and in the default installation path it may read SSH keys, exfiltrate data, or write to arbitrary filesystem locations without any protocol-level protection. Warden exists to run those servers in a restricted sandbox so the server only gets the filesystem paths, network hosts, and environment variables the user explicitly grants.

## Target user

The target user is a developer who installs and runs third-party MCP servers locally on a personal machine. This is not a platform team or enterprise deployment product: the architecture explicitly scopes Warden to a local developer workflow, not a multi-tenant gateway or hosted service. The user is responsible for deciding what each installed MCP server is allowed to access, usually by writing a small YAML policy and invoking the CLI directly from their shell.

## Goals

- Keep the MCP client experience transparent: stdio should behave as if the server were running normally, even when the process is sandboxed.
- Enforce deny-by-default access controls for filesystem, network, and environment variables.
- Work in a normal developer environment without root privileges when the OS supports unprivileged sandbox primitives.
- Prefer OS-native sandboxing backends so the trust boundary stays small and the tool remains lightweight.
- Produce a clear audit trail of blocked and allowed access attempts for debugging and review.
- Support a basic, easy-to-understand policy model that maps to the actual sandbox backend primitives on Linux and macOS.

## Non-goals

This section is copied forward from ARCHITECTURE.md and is not re-derived here:

- Agent identity / audit-trail-of-who-asked.
- Multi-tenant / server-side deployment.
- Windows support in v1.
- Full hosted governance or platform-layer controls for many users.

## Functional requirements

- Filesystem restriction: the sandbox must expose only declared read and write paths, with everything else hidden inside the sandbox; derived from the M1 milestone and the README's filesystem-only quickstart.
- Network allowlisting: the sandbox must allow only configured hosts and block direct egress, DNS leakage, and non-allowlisted connections; required by ROADMAP.md M2.
- Environment passthrough: only named environment variables may pass through to the server; values are inherited from the parent environment and not stored in the policy file; required by M1+M2 policy parsing and enforcement.
- Resource limits: the sandbox should enforce memory and timeout constraints and terminate the process when limits are exceeded; tracked under M3.
- Audit logging: every file access and network attempt should be recorded, including blocks, and exposed through a structured log stream; required by M2 and M3.
- Trace/init UX: `trace` mode should observe access without enforcing, and `init` should generate a starter policy from a trace log; both are M3 features.

## Non-functional requirements

- No-root requirement: Warden should work in a normal developer setup without sudo, using unprivileged namespaces where the OS supports them.
- Fail loud, not silent: blocked access attempts must be logged and surfaced in a way that makes sandbox policy violations obvious instead of quietly ignored.
- Single-binary distribution goal: the project aims to ship as a lightweight single binary with minimal overhead compared to Docker or VM-based isolation.
- Performance expectations: the design prefers low overhead and a small trust boundary, with sandboxing should be no harder than running a local MCP server directly; a concrete performance target is not specified in the repo, so no numeric SLA is assumed.

## Constraints & assumptions

- Platform support order is Linux first, then macOS, then Windows.
- The native backends in scope are bubblewrap on Linux, Seatbelt/sandbox-exec
  on macOS, and AppContainer plus Windows Filtering Platform on Windows.
- Windows support is an M5 milestone. Until it is complete, Windows builds
  fail closed rather than running an unrestricted process.
- The CLI is expected to be a thin orchestration layer over OS-native primitives, not a custom sandbox implementation.
- The project is early-stage and pre-alpha; the README explicitly says it is not ready for production use yet.

## Success metrics

- A real MCP server runs successfully under a correct policy with transparent stdio.
- A deliberate sandbox-escape attempt (for example, reading a forbidden path or connecting to a disallowed host) is blocked and logged.
- A malformed policy fails early with clear validation errors and no silent fallback to an unsandboxed process.
- The local quickstart in README is demonstrably usable against the MVP build on Linux.

## Open questions

- The architecture doc suggests Docker is a fallback backend for macOS/Windows, but the README's quickstart says Warden prefers native primitives and only mentions Docker as a reason not to use it. The repo does not yet specify whether Docker is a true fallback for all non-Linux environments or an explicit opt-in mode.
- Trace sessions are JSON Lines files in Warden's state directory; `init` uses
  the newest session by default and never overwrites an existing policy. An
  interactive approval flow for individual suggested grants remains open.
- The Linux egress proxy is launched by the sandbox backend for each run and
  is reached through an in-namespace loopback bridge.
- Limits are optional policy fields. When configured on Linux,
  `memory_mb` and `timeout_s` are enforced for the sandboxed process tree.
