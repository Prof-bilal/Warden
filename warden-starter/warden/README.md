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
# Manual download — static binaries for Linux, macOS, and Windows:
# https://github.com/Prof-bilal/Warden/releases
```

Or build from source with Go 1.22+ (`go build -o warden ./cmd/warden`).
Homebrew and npm distribution is coming soon. Full per-OS guide, including
required sandbox primitives (`bwrap`, `sandbox-exec`, Docker fallback):
[Install](./docs/install.md).

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
warden run --policy <file> [--backend auto|linux|seatbelt|windows|docker] [--approve] [--approve-timeout <dur>] -- <command...>
    Run a server under a policy (--approve prompts on first out-of-policy access)

warden trace -- <command...>
    Run unsandboxed and record access attempts

warden init [--log <file>] [--output <file>] [-- <command...>]
    Generate a starter policy from an audit log

warden logs [--tail <n>] [--follow] [--log <file>]
    Inspect or follow the audit log

warden gateway init --config <file> --policies <dir>
    Generate per-server starter policies from a gateway config

warden gateway run --config <file> --policies <dir> --server <name>
    Run one registered server sandboxed

warden gateway wrap --config <file> --policies <dir> [--output <file>]
    Emit a gateway config whose commands run through Warden
```

See [Gateway Integration](./docs/gateway.md) and the
[example configs](./examples/gateway-mcp.json) (`gateway-registry.yaml`).

See [Interactive Approval Mode](./docs/approve.md) for `--approve` semantics
(live network prompts; filesystem prompts save + restart; fail-closed
without a terminal).

## Compatibility

Tested against 18 real-world MCP servers — **14 pass, 2 conditional, 2 fail**.
Each row links to the exact policy and has a permanent regression fixture
under [`testdata/compat/`](./testdata/compat/). Full details, failure
classification, and triage notes: [Compatibility Matrix](./docs/compatibility.md).

| Server | Verdict | Policy |
|---|---|---|
| Filesystem, GitHub, Slack, PostgreSQL, SQLite, Brave Search, Google Drive, Git, Memory, Time, Sequential Thinking, Notion, Linear, Tavily | ✅ pass | [`testdata/compat/`](./testdata/compat/) |
| Fetch, Kubernetes | ⚠️ conditional (deployment-specific hosts) | [`testdata/compat/fetch/`](./testdata/compat/fetch/) · [`testdata/compat/kubernetes/`](./testdata/compat/kubernetes/) |
| Docker (needs daemon socket), Playwright (needs wildcard hosts) | ❌ fail — documented gaps | [Failure analysis](./docs/compatibility.md#failures-classified) |

Running your own server? Trace it, generate a policy, and
[file a compatibility report](./docs/beta.md#filing-a-compatibility-report) —
external beta reports are what proves the schema is usable by people who
didn't design it.

## Documentation

- [Schema Reference](./docs/schema.md) — complete field-by-field guide to `policy.yaml`
- [Example Policies](./examples/) — copy-paste policies for popular MCP servers
- [Security Review](./docs/security.md) — threat model, known limitations, and best practices
- [Architecture](./docs/architecture.md) — how Warden works under the hood
 - [Interactive Approval Mode](./docs/approve.md) — `--approve` prompts instead of hard-fails
 - [Compatibility Matrix](./docs/compatibility.md) — 18 tested servers, exact policies, failure analysis
 - [Beta Program](./docs/beta.md) — run your server under Warden and report friction
 - [ROADMAP.md](./ROADMAP.md) — milestones and current status

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md) and [ARCHITECTURE.md](./ARCHITECTURE.md).
Good first contributions: adding example policies for new MCP servers, testing
the backends on your platform, or improving the docs.

## License

MIT
