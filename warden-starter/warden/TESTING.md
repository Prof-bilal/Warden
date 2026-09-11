# Testing Guide

## Test pyramid for this project

Warden is a security tool, so the tests should enforce both ordinary functionality and the failure modes the product depends on.

### Unit tests

Unit tests belong primarily in `internal/policy` and should be pure, deterministic, and table-driven. They should cover:

- valid policy parsing
- invalid YAML and schema validation
- missing or malformed required fields
- path normalization and relative-path handling
- network host validation
- env allowlist validation

These tests should run without OS dependencies because they test policy semantics, not the host environment.

### Integration tests for backends

Each sandbox backend should have an integration test layer that exercises the actual OS primitive in a controlled environment. For example:

- Linux tests should use `bwrap` and `strace` if available. Network tests
  also require a host that permits an unprivileged network namespace.
- macOS tests should use `sandbox-exec`/Seatbelt when available
  (`internal/sandbox/darwin`). Profile generation unit tests run on all OSes.
- Docker-based fallback tests should run only when `docker info` succeeds
  and the configured image is present (`internal/sandbox/docker`).

These tests should be skipped gracefully when the primitive is unavailable; they should not fail the suite on a developer machine that does not have the right sandboxing toolchain installed. **However, on CI runners that *claim* to support a backend, tests must actually run — never silently skip.**

## Linux CI Status ✅

**Linux sandbox escape tests pass on real hardware.**

All 20 Linux tests pass locally (verified on contributor machine):
- 8 escape tests (startup, read/write grants, isolation, exit codes, env filtering)
- 12 integration/unit tests

Run locally:
```bash
cd warden-starter/warden
go test -v -count=1 ./internal/sandbox/linux/...
```

### GitHub Actions Note

The GitHub Actions `ubuntu-latest` runner does **not support unprivileged user namespaces** by design (AppArmor policy: `kernel.apparmor_restrict_unprivileged_userns=1`). This causes the Linux CI job to intentionally fail — it refuses to silently skip security tests.

**This is not a bug.** See `.github/workflows/ci.yml` lines 101-103:

> "If the hosted runner cannot run unprivileged user namespaces, move this job to a privileged container or self-hosted runner — do not re-enable silent skips."

**Status:** Linux tests are verified locally on real hardware. No action required right now. To enable GitHub Actions Linux CI validation in the future, add your Linux machine as a [self-hosted runner](https://github.com/Prof-bilal/Warden/settings/actions/runners/new).

## Escape tests

Escape tests are the tests that validate the actual security promise of the product. They are not optional extras and should be treated as core correctness tests. Every escape test should do one of:

- attempt to read a file outside the allowed filesystem grant
- attempt to write outside the allowed write path
- connect to a non-allowlisted host
- resolve or query a blocked DNS name
- exceed a configured resource limit

These should be explicit, reproducible tests with small fixtures, not implicit assumptions hidden inside a broader integration test.

For Linux M2, assert both paths: a standard HTTP proxy request to an allowed
host succeeds, while a direct TCP connection and a direct DNS query fail in
the private network namespace. Check the JSONL audit log for the proxy's
hostname decision and the corresponding `strace` syscall event.

For M3, unit-test timeout and memory breaches against a child process and
assert that the watcher returns a typed limit error after terminating the
child process group. Test `init` with mixed successful and blocked events to
ensure it never converts a blocked event into a policy grant.

## Fixtures

The repo should add a test fixture directory such as `testdata/` with:

- sample policy files for valid and invalid cases
- tiny dummy MCP server scripts that attempt safe and unsafe filesystem operations
- dummy scripts that open network connections to allowlisted and blocked hosts
- scripts that verify env passthrough behavior and missing variable handling

The fixtures should be simple and deterministic so they remain easy to understand and maintain.

## CI expectations

Before merge, CI should require:

- `go build ./...`
- `go vet ./...`
- `go test ./...`
- a backend-aware suite that skips missing Linux/macOS/Docker primitives instead of failing the job
- a dedicated set of escape tests that run in a self-hosted Linux environment or a privileged container with the required sandbox support

For the first milestones, the practical approach is likely a self-hosted Linux runner with `bwrap` installed. The repo does not yet specify a final CI matrix, so the expectation is that integration tests gracefully skip on missing tooling, but escape tests block CI if the runner claims to support the backend but cannot actually prove it works.

## Coverage expectations

The project should aim for high coverage in the pure policy logic and meaningful coverage in sandbox enforcement. A blanket requirement of 100% line coverage for `internal/sandbox` is less important than escape test coverage that proves the security boundary holds.

## Open questions

- The repo does not yet define the exact directory layout for test fixtures, so `testdata/` is the correct convention to adopt as a standard but the final repo structure is still open.
- There is no committed CI configuration yet, so the exact self-hosted runner or privileged container strategy is still an implementation decision.
- The escape tests are clearly required by the product promise, but the repo does not yet specify which test names or file locations they should use in the first implementation.
