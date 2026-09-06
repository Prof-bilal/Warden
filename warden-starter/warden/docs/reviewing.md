# Code Review Guide

This guide is distinct from `CONTRIBUTING.md`: it is the review checklist reviewers should apply to each PR, with security-sensitive concerns prioritized.

## Security-first review checklist

- Does this change alter any default behavior? Any new default-allow is a red flag and must be justified explicitly in the PR description.
- Does the change add a new grant surface — a new filesystem path type, network capability, or environment variable surface? If so, it must be documented in `ARCHITECTURE.md`'s schema section and reflected in `examples/policy.example.yaml`.
- Does the change affect the CLI contract or user-facing command behavior? If the README quickstart or examples would no longer be valid, the PR should update the docs in the same change.
- Is there an escape test that covers the new behavior, not just a happy-path test? Security-sensitive changes need failure-mode validation.
- Does the PR maintain deny-by-default semantics, fail-loud behavior, and audit logging in the affected path?
- Does the change keep the code in a minimal trust boundary and avoid introducing an unsandboxed escape valve?

## Standard checks

- `go test ./...` passes.
- `go build ./...` passes.
- `gofmt` and linting are clean.
- New exported types and functions have Go doc comments.
- No committed binaries, secrets, or temporary artifacts are included.
- The change does not widen the default policy or silently change YAML semantics in a way that contradicts the current architecture.

## Review etiquette for security-relevant changes

- Require a second reviewer on security-relevant changes even when CI is green.
- Prefer asking for explicit rationale in the PR description when a change touches policy semantics, sandbox behavior, or the audit logger.
- Treat the failure cases as the product behavior, not as optional follow-up work.
- If a change weakens the sandbox boundary, blocks, or logging semantics, ask for a design note or a specific regression test before approval.

## Open questions

- The repo does not yet define a formal PR template or required reviewer policy beyond this guide, so exact enforcement is still open to repo maintainers.
- The current codebase has no backend implementation yet, so "security-relevant" review scope is still being established in practice rather than codified in a larger ruleset.
- The project does not yet define the exact review gate for privileged integration tests required in a self-hosted Linux environment.
