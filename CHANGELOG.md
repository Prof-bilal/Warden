# Changelog

All notable Warden changes are recorded here. Platform behavior and verification
scope are documented separately.

## 0.2.0-beta.12

- Add the Linux Autopilot preview for approved offline local programs. Warden
  detects project markers and generates the policy without a YAML file.
  Project changes require explicit approval in edit mode.
- Document the current agent startup limits in the [Autopilot guide](warden-starter/warden/docs/autopilot.md).

## Warden 2.0 (unreleased)

### Added

- `warden run` resolves local policies and reviewed launchers. The Filesystem
  launcher uses the bundled exact release.
- `warden learn` reviews starter grants on Linux. `warden doctor --prove`
  exercises the native sandbox on Linux, macOS, and Windows using disposable
  fixtures.
- `warden guard` checks package metadata before an isolated install, and CI
  policy reviews use the native validator and policy diff.
- Bridge-mode MCP tool pins hide changed definitions until explicitly accepted.
  The Claude Code hook provides advisory screening of tool requests.
- `warden migrate` previews and applies schema-version updates and matching
  client pins, with backups and reports for policies requiring manual review.

### Upgrade notes

- Explicit schema v1 policies remain accepted with a migration warning.
- `warden add` is a deprecated compatibility command scheduled for removal in
  2.2. Use `warden run` or `warden wrap` for new setups.

- Use `warden migrate` to preview a v2 policy. Inspect permission changes with
  `warden policy-diff` before deploying a new policy.
- Proxies require a numeric loopback listener. Empty tool allowlists deny every
  tool call, and upstream sessions are kept per downstream client. Direct stdio
  proxy upstreams require the explicit `--unsafe-unconfined` flag.

## 0.2.0-beta.5

This preview release follows `0.2.0-beta.4` and is published to npm with the
`beta` dist-tag. `latest` is unchanged.

### Breaking

- `warden proxy` with a `stdio:` upstream now requires `--unsafe-unconfined`,
  because stdio upstreams are not OS-sandboxed. Run the proxy under
  `warden run` instead, or pass the flag to accept the risk.
- The proxy listener must be a numeric loopback address. Remote binding is
  refused.

### Security

- An empty or omitted MCP tool allowlist now denies every tool call.
- Upstream MCP session IDs are isolated per downstream client, so one client
  cannot reuse another client's upstream session.
- A missing process state no longer crashes the MCP server. Exit codes are
  reported as -1 when unknown.

### Fixed

- `warden init` now grants files the traced program opened with relative paths.
  Before, the generated policy could not run the program it was built from.
- `warden add` no longer sets a 300-second wall-clock limit on the server it
  creates. Long-lived MCP servers were being killed after 5 minutes. The
  memory cap is kept.
- Documentation: rewrote the quickstart as a single tutorial that needs no
  repository fixtures, restructured the CLI reference (all commands listed),
  rewrote the troubleshooting page, and fixed the sidebar and broken wording.

## 0.2.0-beta.4 — 2026-10-29

This preview release follows `0.2.0-beta.3` and is published to npm with the
`beta` dist-tag.

### Added

- Sanitized agent workspaces for local AI agents and coding tools. Selected
  dotenv, JSON, YAML, and TOML fields are replaced with placeholders while the
  surrounding project remains editable.
- Human-reviewed apply flow for agent changes. Protected values stay in the
  supervisor-owned project, are never copied into the agent projection, and
  cannot be changed through the normal workspace diff.
- Agent workspace notices, manifests, redaction policies, environment
  filtering, size limits, and brokered approved secret operations.
- Portable workspace enforcement for Windows, including no-follow
  checks, reparse/symlink rejection, hard-link checks, native replacement
  behavior, and platform-specific test coverage.
- Linux agent workspace integration and escape/security fixtures.
- Windows AppContainer, WFP, Job Object, elevation, ETW, ACL, and unprivileged
  execution coverage reflected in the current implementation and CI.
- Real Linux and Windows terminal proof images in `warden-shots/` for the
  supported evidence workflow.
- Simplified public positioning across the README, landing page, navigation,
  setup flow, compatibility discovery, and documentation sidebar.
- Dedicated documentation for agent workspaces, platform testing, proof
  results, policy reference, security limitations, and verification workflows.
- CI checks for cross-platform agent workspace behavior, strict docs builds,
  website builds, npm wrapper tests, repository hygiene, and native backend
  verification.

### Changed

- Warden is now described first as a deny-by-default sandbox for AI agents,
  MCP servers, and local AI tools rather than as an MCP-only product.
- README onboarding now leads with the security boundary, installation, a
  minimal policy, four protected resource types, platform status, and links to
  deeper documentation.
- Homepage content is shorter and more direct: problem, boundary demo, agent
  workspace, policy, proof, and installation CTA.
- Documentation navigation now separates getting started, security, reference,
  platforms, integrations, verification, and contribution material.
- Root-level proof, planning, status, and platform-testing documents now live
  under categorized documentation directories instead of cluttering the public
  repository root.
- Repository description now clearly states: “Sandbox AI agents and MCP
  servers. Block access to files, secrets, and network by default.”

### Removed or moved

- Removed generated website architecture images that were not part of the
  current product UI.
- Removed the old root-level copies of temporary status reports, release
  previews, planning notes, and test scripts from the public root. Their useful
  content was preserved under `warden-starter/warden/docs/` and
  `warden-starter/warden/scripts/`.
- Removed outdated homepage sections that duplicated full CLI references,
  backend internals, release notes, and large implementation tables.
- No Warden runtime command or supported policy capability was intentionally
  removed in this release.

### Verification

- GitHub Actions run
  [40818875880](https://github.com/Prof-bilal/Warden/actions/runs/40818875880)
  passed all 12 jobs on the tagged source base before this release preparation:
  Linux, Windows, website, docs, repository hygiene, attack simulation,
  npm wrappers, and cross-builds.
- The verified CI run covered the portable workspace and proxy lifecycle fixes
  included in this release line.
- The npm release workflow packs the exact wrapper tarball, checks launcher and
  runtime files, and publishes prereleases under the `beta` dist-tag.

### Known limitations

- This is a pre-1.0 preview. Native enforcement maturity differs by operating
  system; verify the backend on the host where Warden will run. The release
  gate for this preview covers the Linux and Windows runtime jobs.
- Remote provider processes are not OS-sandboxed by Warden.
- The MCP gateway remains a bounded 2025-11-25 request/response subset; tasks,
  pagination, subscriptions, sampling, elicitation, roots, and newer protocol
  revisions are not supported.
- Checksums authenticate downloaded bytes against the GitHub release manifest;
  the current release path does not provide independent publisher signatures.
- CI, cross-build, and fixture results do not replace native security testing
  on every supported machine or a full GUI client workflow.

See the [security model](https://warden.blog/docs/security),
[verification notes](https://warden.blog/docs/proof), and
[agent workspace guide](https://warden.blog/docs/agent-workspace) before using
this preview with sensitive data.
