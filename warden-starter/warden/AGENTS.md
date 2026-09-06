# AGENTS.md

## Orientation

Warden is a lightweight sandbox runtime for MCP servers. The repo is intentionally small and early-stage: the core idea is to run third-party MCP servers under OS-native sandboxing so that a local AI tooling stack only gets the filesystem paths, network hosts, and environment variables an explicit policy allows. M1–M5 are implemented: Linux bubblewrap, macOS Seatbelt, Docker fallback, Windows AppContainer + WFP + Job Objects, egress proxy, audit/trace/init/logs, and resource limits.

## Where things live

- `cmd/warden` is the CLI entry point only. It parses subcommands and should not contain the policy-engine logic or sandbox enforcement itself.
- `internal/policy` is the intended home for YAML parsing, validation, and policy normalization.
- `internal/sandbox` selects the backend (`auto` prefers native, then Docker).
- `internal/sandbox/<os>` holds platform-specific backends (`linux` bubblewrap,
  `darwin` Seatbelt, `windows` AppContainer + WFP + ETW + Job Objects).
  `internal/sandbox/docker` is the cross-platform fallback.
- `internal/audit` is the intended home for structured access logging, both for enforcement and for trace mode.
- `examples/` contains sample policies and usage patterns that should be kept aligned with the architecture doc.
- `README.md`, `ARCHITECTURE.md`, `ROADMAP.md`, and `CONTRIBUTING.md` are the canonical design and workflow documents; changes to the implementation should be checked against them before merging.

## Security invariants: hard rules

- Deny by default for filesystem, network, and environment grants. If a path, host, or variable is not explicitly allowed by the policy, it must be blocked.
- No silent fallback to an unsandboxed run. If a backend is unavailable or a required primitive is missing, the CLI must fail loudly with a clear error instead of continuing without sandboxing.
- Every blocked access attempt must be logged, not swallowed. The audit log is part of the product contract.
- Policy parsing and backend selection must fail closed. Invalid or incomplete policy input is not a reason to run the target command without restrictions.
- Never add a new default-allow surface in the policy engine or any backend without a matching change to the architecture and example policy.

If a change would violate one of the above invariants, stop and ask for clarification instead of proceeding.

## Build and test commands

Before considering a task done, the agent should run the project checks in this order:

```bash
cd /home/abdullah/Downloads/warden/warden-starter/warden
go build ./...
go vet ./...
go test ./...
```

If a task touches backend enforcement, also run the relevant backend-specific integration or escape tests described in `TESTING.md` before finalizing the change.

## When uncertain

For anything touching sandbox enforcement, policy schema semantics, or audit logging, prefer asking a clarifying question over guessing. These are security-sensitive decisions, not ordinary feature work. A wrong assumption here can weaken the runtime guarantee rather than just failing a unit test.

## Prohibited actions

- Do not commit built binaries or generated artifacts to the repo.
- Do not add a new default-allow anywhere in the policy engine.
- Do not alter the policy YAML schema without updating both `ARCHITECTURE.md` and `examples/policy.example.yaml` in the same change.
- Do not silently change CLI output or flags in a way that breaks the README quickstart without updating the relevant docs in the same PR.
- Do not move scope beyond the roadmap without documenting the decision in the architecture and release docs.

## Open questions

- The repo currently has no checked-in `internal` packages, so the exact naming of the future policy and sandbox packages remains a convention to be established by the implementation rather than a fact already present in the code.
- The architecture doc mentions a local egress proxy and DNS blocking, but the repo does not yet specify whether the proxy is mandatory for M2 or can be implemented as a backend-specific wrapper around the process network namespace.
- The current CLI stub defines `trace`, `init`, and `logs` subcommands, but those behaviors are not yet implemented; their exact UX and output format remain open design decisions.
