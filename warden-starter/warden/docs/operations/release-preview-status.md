# 0.2.0-beta.4 release status

Verified 2026-10-29. This page records the release checks for the current
preview. It is an operational record, not a claim of universal agent or MCP
conformance.

## Release checks

| Check | Result |
| --- | --- |
| Linux runtime and escape fixtures | Passed in the release workflow |
| Windows runtime and workspace checks | Passed in the release workflow |
| Cross-builds | Passed for Linux, Windows, and the published release assets |
| Website and documentation builds | Passed |
| npm wrapper tests and tarball inspection | Passed |
| GitHub release assets and checksums | Built and uploaded |

The release workflow deliberately keeps the runtime gate focused on the
platforms with an available hosted runner for this preview. Platform-specific
implementation details and limitations belong in the platform and security
documentation, not in this release checklist.

## Package publication

The wrapper tarball is prepared as:

```text
warden-sandbox-cli@0.2.0-beta.4
```

The package tests pass and the six-file tarball contains the launcher, runtime,
package metadata, license, README, and verification script. Publication still
requires an npm automation token with permission to publish
`warden-sandbox-cli`; npm returns a masked `E404` when the configured token is
not authorized.

## Known limitations

- Warden is pre-1.0 security software. Verify the selected backend on the host
  where it will run.
- Remote provider processes are not OS-sandboxed by Warden.
- The MCP gateway remains a bounded 2025-11-25 request/response subset; tasks,
  pagination, subscriptions, sampling, elicitation, roots, and newer protocol
  revisions are not supported.
- Checksums authenticate downloaded bytes against the GitHub release manifest;
  the current release path does not provide independent publisher signatures.
- CI, cross-build, and fixture results do not replace native security testing
  on every supported machine or a full GUI client workflow.

See the [security model](../security.md), [verification notes](../proof.md),
and [agent workspace guide](../agent-workspace.md) before using this preview
with sensitive data.
