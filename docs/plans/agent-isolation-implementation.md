# Agent isolation implementation plan

## Phase 0 — Linux proof

Build fake-secret fixtures and launch a real command under Warden. Prove allowed
source reads/writes and normal Node, Python, Go, Rust, package-manager, build, test,
and Git workflows. Prove that denied secrets remain inaccessible through shell,
interpreters, symlinks, hard links, traversal, absolute paths, subprocesses,
archives, `/proc`, descriptors, and environment enumeration.

No production secret scanner or UI is required in this phase.

## Phase 1 — native Linux enforcement

Add workspace projection for fine-grained denies, Landlock setup after namespace
construction, explicit descriptor hygiene, and capability detection. Keep the
existing egress proxy and process-group limits. Refuse unsupported guarantees.

## Phase 2 — policy

Add deny globs, workspace mode, protected-secret defaults, deterministic canonical
matching, Git visibility controls, and validation tests. Update architecture and
example policies together.

## Phase 3 — approvals and audit

Add scoped filesystem decisions, protected-secret approval restrictions, and
structured decision events without content logging.

## Phase 4/5 — macOS and Windows

Extend Seatbelt and AppContainer/DACL paths with platform-specific capability checks
and adversarial tests. Preserve documented differences rather than claiming parity.

## Phase 6 — integrations

Validate Claude Code, Codex, OpenCode, Gemini CLI, and Aider inside the same Warden
boundary. Do not add agent-specific security logic.

## Acceptance criteria

The phase is complete only when the adversarial harness passes on the target host,
the audit contains decisions without secret contents, unsupported capabilities fail
closed, and the documented guarantee matches the tested backend.
