# Warden

Development ecosystem gateway: `warden connect` (stdio) and `warden serve`
(authenticated HTTP) combine reviewed MCP rules with local process sandboxing.
`warden creator` creates/verifies signed allowed/denied evidence;
`policy-diff` and `report` support upgrades and explicit local pilot measurement.
See [gateway guide](docs/mcp-gateway.md), [creator program](docs/creator-program.md)
and [maintenance guide](docs/ecosystem-maintenance.md). Full protocol, native
platform and public release gates remain documented work.

Development checkout: new `packs`, `clients`, `wrap`, `unwrap`, and `inventory`
commands provide candidate policy profiles and reversible local setup. These
are not yet a published-release or live-client compatibility claim. Read
[the boundary baseline](docs/ecosystem-baseline.md) and
[the setup guide](../../warden-landing/content/docs/ecosystem-setup.md).

**A lightweight sandbox runtime for MCP servers.**

MCP servers routinely run as plain Node or Python processes on your machine
with full filesystem and network accesseven ones you just cloned from
GitHub five minutes ago. Warden runs them in a restricted sandbox so a server
only ever gets the files, network hosts, and environment variables you
explicitly grant it.

> **Status:** All backends implemented (Linux, macOS, Windows); npm and GitHub
> Releases distribution is live. Verification state: Linux verified on real
> hardware, Windows verified via CI escape tests, macOS CI-green and pending a
> real-hardware harness run. See [TESTING.md](./TESTING.md) and
> [REMAINING_WORK.md](./REMAINING_WORK.md) for the exact state.

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
# npm (auto-downloads the release binary for your platform):
npm install -g warden-sandbox-cli

# Or manual downloadstatic binaries for Linux, macOS, and Windows:
# https://github.com/Prof-bilal/Warden/releases
```

Or build from source with Go 1.22+ (`go build -o warden ./cmd/warden`).
A Homebrew tap is pending. Full per-OS guide, including required sandbox
primitives (`bwrap`, `sandbox-exec`, Docker fallback) and the strace
requirement for `warden run` on Linux: [Install](./docs/install.md).

## What Warden does

```
warden run --policy ./policy.yaml -- node ./my-mcp-server/index.js
```

Warden spawns the server inside a sandbox that:

- **Filesystem**only sees paths you list, read-only or read-write as you
  specify. Everything else is invisible, not just "permission denied."
- **Network**can only reach hostnames you allowlist. Connections to other
  hosts are blocked at the sandbox boundary before DNS even resolves.
- **Environment**only receives the env vars you pass through. No automatic
  inheritance of your shell environment.
- **Stdio**passed through transparently, so the MCP client (Claude, an IDE,
  etc.) talks to the sandboxed process exactly like an unsandboxed one.
- **Audit log**records every file access attempt and network connection
  attempt (including blocked ones) at `${XDG_STATE_HOME:-~/.local/state}/warden/audit.jsonl`. View with `warden logs`.

## Backends

| OS | Default backend | Fallback |
|---|---|---|
| Linux | bubblewrap (`bwrap`) | Docker |
| macOS | `sandbox-exec` (Seatbelt) | Docker |
| Windows | AppContainer / WFP / ETW | Fail closed |
| Other | Docker | Fail closed |

Force a specific backend with `--backend linux|seatbelt|docker|windows`.
Docker is **never** preferred over a working native backend.

## Commands

```
warden add <pack> to <client> [--yes] [--allow-host <host>] ...
    Connect an MCP server to a client in one command (see below)

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

warden doctor
    Check sandbox readiness (backend, namespaces, proxy, policy engine)

warden update [--check] [--version <ver>] [--yes]
    Update warden via npm registry + verified GitHub Release binary

warden version
    Print the build version (also `--version`)
```

`warden` with no args shows the branded header and usage; `warden init` and
`warden doctor` show the large ASCII banner when run interactively (never
in CI or for `warden run`/`--help`/`--version`). Colors respect
`NO_COLOR`, `TERM=dumb`, and `WARDEN_NO_UNICODE`; see
[CLI ReferenceCLI Experience](./docs/cli.md#cli-experience). First-run
shows a one-time welcome (`Welcome to Warden`); it never blocks
`warden run` and is suppressed in CI.

### One-command client connection (`warden add`)

```bash
warden add slack to claude-desktop
warden add notion to claude-desktop
warden add @scope/mcp-server@1.2.3 to claude-code \
    --allow-host api.example.com --allow-env SCOPE_KEY
```

`add` prepares the pinned npm release inside the sandbox (install scripts
disabled), generates a deny-by-default policy, and registers a wrapped
launcher in the client's config after a preview + confirmation prompt.
Catalog packs (`warden packs list`) carry reviewed grants; any other npm
MCP package works with explicit `--allow-host/--allow-env/--allow-read/
--allow-write` flags or your own `--policy`. Credentials are reported by
name only and never written into config files. `warden add --list-clients`
shows every detected client config and its current entries (read-only).
Undo with `warden unwrap --client <client> --server <entry>`. See
`warden help add`.

See [Gateway Integration](./docs/gateway.md) and the
[example configs](./examples/gateway-mcp.json) (`gateway-registry.yaml`).

See [Interactive Approval Mode](./docs/approve.md) for `--approve` semantics
(live network prompts; filesystem prompts save + restart; fail-closed
without a terminal).

## Compatibility

Tested against 18 real-world MCP servers**14 pass, 2 conditional, 2 fail**.
Each row links to the exact policy and has a permanent regression fixture
under [`testdata/compat/`](./testdata/compat/). Full details, failure
classification, and triage notes: [Compatibility Matrix](./docs/compatibility.md).

| Server | Verdict | Policy |
|---|---|---|
| Filesystem, GitHub, Slack, PostgreSQL, SQLite, Brave Search, Google Drive, Git, Memory, Time, Sequential Thinking, Notion, Linear, Tavily | ✅ pass | [`testdata/compat/`](./testdata/compat/) |
| Fetch, Kubernetes | ⚠️ conditional (deployment-specific hosts) | [`testdata/compat/fetch/`](./testdata/compat/fetch/) · [`testdata/compat/kubernetes/`](./testdata/compat/kubernetes/) |
| Docker (needs daemon socket), Playwright (needs wildcard hosts) | ❌ faildocumented gaps | [Failure analysis](./docs/compatibility.md#failures-classified) |

Running your own server? Trace it, generate a policy, and
[file a compatibility report](./docs/beta.md#filing-a-compatibility-report) —
external beta reports are what proves the schema is usable by people who
didn't design it.

## Documentation

- [Schema Reference](./docs/schema.md)complete field-by-field guide to `policy.yaml`
- [Example Policies](./examples/)copy-paste policies for popular MCP servers
- [Security Review](./docs/security.md)threat model, known limitations, and best practices
- [Architecture](./docs/architecture.md)how Warden works under the hood
 - [Interactive Approval Mode](./docs/approve.md)`--approve` prompts instead of hard-fails
 - [Compatibility Matrix](./docs/compatibility.md)18 tested servers, exact policies, failure analysis
 - [Beta Program](./docs/beta.md)run your server under Warden and report friction
 - [ROADMAP.md](./ROADMAP.md)milestones and current status

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md) and [ARCHITECTURE.md](./ARCHITECTURE.md).
Good first contributions: adding example policies for new MCP servers, testing
the backends on your platform, or improving the docs.

## License

MIT
