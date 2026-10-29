# Warden

**A deny-by-default sandbox for AI agents, MCP servers, and local AI tools.**

Stop AI tools from accessing files, secrets, or network hosts you did not
approve. Warden turns broad user permissions into a small, reviewable policy.

[Website](https://warden.blog) · [Documentation](https://warden.blog/docs) ·
[npm](https://www.npmjs.com/package/warden-sandbox-cli) ·
[License](LICENSE)

[![npm version](https://img.shields.io/npm/v/warden-sandbox-cli?style=flat-square)](https://www.npmjs.com/package/warden-sandbox-cli)
[![CI](https://img.shields.io/github/actions/workflow/status/Prof-bilal/Warden/ci.yml?branch=main&style=flat-square)](https://github.com/Prof-bilal/Warden/actions/workflows/ci.yml)
[![license](https://img.shields.io/badge/license-MIT-blue?style=flat-square)](LICENSE)

> Warden is pre-1.0 security software. Read the [security model and
> limitations](warden-starter/warden/docs/security.md) before protecting
> sensitive workloads. Verify native enforcement on the operating system where
> you will run it.

## See the boundary

```text
AI tool → ./project              ✅ allowed
AI tool → .env                   ❌ denied
AI tool → ~/.ssh/id_rsa          ❌ denied
AI tool → api.github.com         ✅ allowed when declared
AI tool → evil.example.com       ❌ denied
```

The exact result depends on the policy and backend. Warden refuses to run when
it cannot establish the requested sandbox instead of silently running without
protection.

## Install

```bash
npm install -g warden-sandbox-cli
warden doctor
```

Use `warden doctor` before the first protected run. It checks whether the
selected platform backend and its required primitives are ready. For release
downloads, source builds, and platform prerequisites, see the
[installation guide](https://warden.blog/docs/install).

## Quickstart

Observe a trusted local command, turn the observed accesses into a starter
policy, review it, then run the command inside Warden:

```bash
warden trace -- npx your-server
warden init
warden run --policy policy.yaml -- npx your-server
```

Tracing is intentionally unsandboxed. Use disposable data and fake credentials,
then narrow the generated policy before running it.

## A small policy

```yaml
filesystem:
  read:
    - ./project

network:
  allow:
    - api.github.com

env:
  allow:
    - GITHUB_TOKEN
```

With this policy, the process receives only the declared filesystem, network,
and environment access. Anything else is not an implicit permission. The full
[policy reference](https://warden.blog/docs/schema) explains write grants,
limits, MCP rules, redaction, and platform differences.

## Protect an AI coding agent

When an agent needs to edit code near credentials, use a sanitized workspace:

```bash
warden agent --policy agent-policy.yaml -- claude
```

Warden creates a temporary project projection. Selected environment values and
fields in dotenv, JSON, YAML, or TOML files become placeholders. The agent can
edit the surrounding code, but Warden shows a sanitized diff and requires your
approval before applying changes to the real project.

Read the [AI agent workspace guide](https://warden.blog/docs/agent-workspace)
for the policy format, supported clients, fixed credential requests, limits,
and failure behavior.

## What Warden controls

- **Files:** grant selected paths read or write access.
- **Secrets:** pass only named environment variables, or hide selected values
  from an agent workspace.
- **Network:** allow only declared hosts through Warden's policy-aware path.
- **Processes:** run the whole child process tree inside the selected sandbox,
  with supported timeout and memory supervision.

## How it works

```text
AI agent / MCP server / local tool
                 ↓
              Warden
                 ↓
        policy + OS sandbox backend
                 ↓
          approved resources only
```

MCP is a first-class workflow, not the product boundary. Warden can wrap local
commands directly, and it also provides MCP proxy, gateway, and client setup
workflows.

## Evidence and platforms

The repository maintains an 18-entry MCP compatibility matrix: 14 pass, 2 are
conditional, and 2 are documented failures. These are fixture classifications,
not a promise that every upstream service works without configuration.

Native platform status and reproducible proof are documented separately:

- **Linux:** bubblewrap namespaces and the native proof harness.
- **macOS:** Seatbelt with platform-specific limitations and fallback paths.
- **Windows:** AppContainer/WFP/Job Objects with native verification requirements.
- **Docker:** explicit fallback backend with a larger trust surface.

See [test evidence](https://warden.blog/docs/proof), the
[compatibility matrix](https://warden.blog/docs/compatibility), and the
[security review](https://warden.blog/docs/security). A CI or compile result is
not automatically proof of native enforcement on your machine.

## Documentation

- [Quickstart](https://warden.blog/docs/quickstart)
- [Installation](https://warden.blog/docs/install)
- [AI agent workspace](https://warden.blog/docs/agent-workspace)
- [Policy reference](https://warden.blog/docs/schema)
- [CLI reference](https://warden.blog/docs/cli)
- [Compatibility](https://warden.blog/docs/compatibility)
- [Security model and limitations](https://warden.blog/docs/security)
- [Architecture](https://warden.blog/docs/architecture)
- [Verification and testing](https://warden.blog/docs/proof)
- [Contributing](CONTRIBUTING.md)

## Contributing

Contributions are welcome, especially platform evidence, escape tests, policy
fixtures, documentation, and reproducible bug reports. Start with
[CONTRIBUTING.md](CONTRIBUTING.md). Security issues should follow
[SECURITY.md](SECURITY.md).

Warden is released under the [MIT License](LICENSE).
