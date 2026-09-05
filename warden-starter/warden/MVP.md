# Minimum Viable Product

## Scope choice

This document scopes the MVP to ROADMAP.md M1 only, not M1+M2. The roadmap explicitly calls M1 the smallest slice that proves the core idea works: policy parsing + `bwrap` filesystem sandbox + transparent stdio + a demo-able Linux end-to-end run. Network enforcement is important, but the repo's own roadmap says it is the next milestone after the core filesystem mechanism, and notes that the first thing to actually build is M1's policy parser + `bwrap` wrapper. We therefore treat filesystem restriction as the MVP, with network allowlisting deferred to M2.

## In scope

The MVP is considered done only when all of the following are true:

- `warden run --policy policy.yaml -- <cmd>` loads a policy file and validates the YAML schema before launching the target command.
- On Linux, `warden run` invokes the current OS sandbox backend via `bwrap` with bind mounts derived from `filesystem.read` and `filesystem.write`.
- The sandboxed process sees only the declared filesystem paths; everything else is invisible inside the sandbox instead of merely permission-denied.
- Stdio passes through transparently to the MCP client, so the client sees no protocol-level difference between the sandboxed and unsandboxed process.
- A policy that leaves `filesystem.read` and `filesystem.write` empty results in the sandboxed process seeing an empty or inaccessible filesystem, not a non-sandboxed fallback or a confusing error path.
- The CLI exits with a clear error if a required backend or policy file is unavailable, rather than silently running unsandboxed.
- The README quickstart is accurate for the MVP build: users can write a policy, run `warden run --policy ...`, and observe a server operating under the declared restrictions.

## Explicitly out of scope for MVP

- macOS support via `sandbox-exec`/Seatbelt.
- Windows support.
- Docker fallback backend.
- `trace` mode.
- `init` mode.
- `logs` inspection of the audit log.
- Resource limits such as memory and timeout enforcement.
- Network allowlisting and egress proxy enforcement.

## Acceptance criteria

The MVP is only acceptable when all of the following can be demonstrated in a real Linux environment with `bwrap` installed:

1. Policy validation: a malformed policy yields a clear validation error before the command starts.
2. Filesystem grant: a process started under a policy with `read: ["./data"]` can read `./data` but cannot read a sibling directory not listed in the policy.
3. Filesystem write: a process with `write: ["./output"]` can create files there, but cannot write to a different path outside the granted write tree.
4. Transparent stdio: an MCP client can communicate with the sandboxed server over stdin/stdout without protocol changes.
5. Deny-by-default behavior: a missing grant for a path does not produce a permission-denied fallback in user space; it is denied at the sandbox boundary.
6. No silent unsandboxed fallback: if `bwrap` is unavailable, the command exits with an explicit error and does not continue in an unprotected mode.

## Definition of done

- `go build ./...` succeeds on the target Linux environment.
- `go vet ./...` succeeds.
- `go test ./...` succeeds, including table-driven validation tests for the policy parser and any sandbox smoke tests that are safe to run in CI.
- The README quickstart works as written against the MVP binary.
- The repo includes at least one documented example policy and a test case that exercises a denied filesystem path.
- The codebase follows the repository's security invariants: deny-by-default, fail loud on backend failure, and log denied attempts.

## Status

M1 is complete and all acceptance criteria have been met.

## Open questions resolved

- Network allowlisting: deliberately deferred to M2. The policy `network.allow` is parsed but not enforced in M1 (stderr warning emitted).
- Package boundaries: `internal/policy`, `internal/sandbox/linux`, `internal/envfilter` are now implemented and checked in.
- Resource limits (`limits`): parsed but ignored during M1 validation, with a stderr warning. Enforcement deferred to M3.
