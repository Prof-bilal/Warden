# Warden

**A lightweight sandbox runtime for MCP servers.**

MCP servers routinely run as plain Node or Python processes on your machine
with full filesystem and network access — even ones you just cloned from
GitHub five minutes ago. Warden runs them in a restricted sandbox so a server
only ever gets the files, network hosts, and environment variables you
explicitly grant it.

> **Status:** All backends implemented (Linux, macOS, Windows). Distribution
> tooling (Homebrew, npm) and docs are in place. See [ROADMAP.md](./ROADMAP.md)
> for what's built and what's next.

## Quickstart

Write a policy that describes what the server is allowed to touch:

```yaml
# policy.yaml
command: ["/usr/bin/node", "server.js"]
filesystem:
  read: ["./data"]
  write: ["./output"]
network:
  allow: ["api.github.com"]
env:
  allow: ["GITHUB_TOKEN"]
limits:
  memory_mb: 512
  timeout_s: 300
```

Then run the server inside the sandbox:

```bash
warden run --policy policy.yaml
```

Or generate a starter policy automatically by tracing an unsandboxed run:

```bash
warden trace -- /usr/bin/node server.js
warden init
```

See [Example Policies](./examples/) and the [Schema Reference](./docs/schema.md)
for details.

## Install

```bash
# macOS
brew install warden-sandbox/warden/warden

# npm
npm install -g @warden-sandbox/mcp-warden

# Manual download — static binaries for Linux, macOS, and Windows:
# https://github.com/warden-sandbox/warden/releases
```

## What Warden does

```
warden run --policy ./policy.yaml -- node ./my-mcp-server/index.js
```

Warden spawns the server inside a sandbox that:

- **Filesystem** — only sees paths you list, read-only or read-write as you
  specify. Everything else is invisible, not just "permission denied."
- **Network** — can only reach hostnames you allowlist. Connections to other
  hosts are blocked at the sandbox boundary before DNS even resolves.
- **Environment** — only receives the env vars you pass through. No automatic
  inheritance of your shell environment.
- **Stdio** — passed through transparently, so the MCP client (Claude, an IDE,
  etc.) talks to the sandboxed process exactly like an unsandboxed one.
- **Audit log** — records every file access attempt and network connection
  attempt (including blocked ones) at `${XDG_STATE_HOME:-~/.local/state}/warden/audit.jsonl`. View with `warden logs`.

## Backends

| OS | Default backend | Fallback |
|---|---|---|
| Linux | bubblewrap (`bwrap`) | Docker |
| macOS | `sandbox-exec` (Seatbelt) | Docker |
| Windows | AppContainer / WFP | Fail closed |
| Other | Docker | Fail closed |

Force a specific backend with `--backend linux|seatbelt|docker|windows`.
Docker is **never** preferred over a working native backend.

## Commands

```
warden run --policy <file> [--backend auto|linux|seatbelt|windows|docker] -- <command...>
    Run a server under a policy

warden trace -- <command...>
    Run unsandboxed and record access attempts

warden init [--log <file>] [--output <file>] [-- <command...>]
    Generate a starter policy from an audit log

warden logs [--tail <n>] [--follow] [--log <file>]
    Inspect or follow the audit log
```

## Documentation

- [Schema Reference](./docs/schema.md) — complete field-by-field guide to `policy.yaml`
- [Example Policies](./examples/) — copy-paste policies for popular MCP servers
- [Security Review](./docs/security.md) — threat model, known limitations, and best practices
- [Architecture](./docs/architecture.md) — how Warden works under the hood
- [ROADMAP.md](./ROADMAP.md) — milestones and current status

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md) and [ARCHITECTURE.md](./ARCHITECTURE.md).
Good first contributions: adding example policies for new MCP servers, testing
the backends on your platform, or improving the docs.

## License

MIT
