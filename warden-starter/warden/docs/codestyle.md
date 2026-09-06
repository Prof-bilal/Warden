# Code Style Guide

## Go conventions

Warden is a Go project, and the repo expects Go code to be formatted and reviewed with the same standards as any other Go service, even though this is a security tool with a very small API surface.

- `gofmt` is mandatory. `goimports` is also expected for import ordering and cleanup when available.
- CI should enforce `gofmt` and, if practical, `go vet` and a lint step such as `golangci-lint`.
- A minimal lint set is acceptable, but it should include basic vet-like checks and style enforcement. The repo does not yet define a final lint configuration, so this is a documented requirement rather than an implemented one.

## Package layout

- `cmd/warden` should host only the CLI entry point and subcommand dispatch. It should not contain the policy parser or sandbox enforcement itself.
- `internal/` should contain the actual logic. New subpackages are warranted when a feature introduces a new responsibility, such as policy parsing, backend implementation, or audit logging. If a responsibility is small and clearly belongs with an existing package, avoid creating a new subpackage just to avoid a few files.
- The public API surface should be kept small; the repo is intentionally simple and the sandbox logic should remain in `internal` packages.

## Error handling

- Wrap errors with `%w` and include context, such as `fmt.Errorf("load policy: %w", err)`.
- Do not call `panic()` outside `main()`. If a fatal condition must terminate the process, prefer returning a structured error to the caller and handling it in the CLI.
- Do not silently ignore errors using `_ = err` without a specific comment explaining why it is safe to do so.
- On CLI failure, emit a clear message to stderr and exit non-zero. The project prefers fail-fast behavior over silent degradation.

## Naming and command style

The current CLI stub uses command-style names like `cmdRun`, `cmdTrace`, `cmdInit`, and `cmdLogs` in `main.go`. This appears to be the intended style for the entry point layer, and the project should keep that naming pattern unless there is a strong reason to switch. The convention is: `cmdX` for subcommand handlers in `main.go`, with package-level logic moved into `internal/` packages as the code grows.

## Documentation requirements

- Every exported type and exported function should have a doc comment in standard Go doc-comment form.
- Package comments are recommended for new packages if they add meaningful API surface.
- Public behavior that changes the CLI shape or policy schema must be documented in the repo docs that correspond to the change.

## Open questions

- The repo has not yet adopted a concrete lint configuration or CI lint step, so the exact `golangci-lint` config is still an implementation decision rather than a policy already enforced in the repo.
- The current CLI stub is deliberately a design skeleton, so the naming convention may still evolve as real package boundaries appear; the doc intentionally treats `cmdRun`/`cmdTrace` as the current convention, not a final commitment.
- The project does not yet have enough code to settle whether a package like `internal/sandbox` should expose a common interface or simply organize per-platform implementations under the same directory.
