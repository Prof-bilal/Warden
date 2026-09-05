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

- Linux tests should use `bwrap` if available.
- macOS tests should use `sandbox-exec`/Seatbelt when available.
- Docker-based fallback tests should run only when the Docker backend is expected to be in use.

These tests should be skipped gracefully when the primitive is unavailable; they should not fail the suite on a developer machine that does not have the right sandboxing toolchain installed.

## Escape tests

Escape tests are the tests that validate the actual security promise of the product. They are not optional extras and should be treated as core correctness tests. Every escape test should do one of the following and assert the sandbox blocks it:

- attempt to read a file outside the allowed filesystem grant
- attempt to write outside the allowed write path
- connect to a non-allowlisted host
- resolve or query a blocked DNS name
- exceed a configured resource limit

These should be explicit, reproducible tests with small fixtures, not implicit assumptions hidden inside a broader integration test.

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

For the first milestones, the practical approach is likely a self-hosted Linux runner with `bwrap` installed. The repo does not yet specify a final CI matrix, so the expectation is that integration and escape tests run where the platform primitive exists, and gracefully skip where it does not.

## Coverage expectations

The project should aim for high coverage in the pure policy logic and meaningful coverage in sandbox enforcement. A blanket requirement of 100% line coverage for `internal/sandbox` is less important than covering the real escape scenarios: blocked filesystem access, denied network access, and handled backend failures. The risk profile of the project means the highest-value tests are the ones that prove sandbox boundaries hold.

## Open questions

- The repo does not yet define the exact directory layout for test fixtures, so `testdata/` is the correct convention to adopt as a standard but the final repo structure is still open.
- There is no committed CI configuration yet, so the exact self-hosted runner or privileged container strategy is still an implementation decision.
- The escape tests are clearly required by the product promise, but the repo does not yet specify which test names or file locations they should use in the first implementation.
