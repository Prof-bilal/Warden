# Example Policies

Copy any of these policies as a starting point for your own MCP server, then
adjust the `command` path, grants, and env allowlist to match your setup.
Relative paths in these files are resolved against the policy file's own
directory — not your shell's current working directory.

> **Tip:** The easiest way to get a policy right is to run
> `warden trace -- <your-server-command>` first, then `warden init`. The
> auto-generated policy is conservative by design; review and tighten it
> from there.

The source files live under `examples/` in the repo. The full content of each one is reproduced below so you don't have to leave this site.

## Policies

| Server | What it covers |
|---|---|
| [Filesystem](#filesystem) | Local file read/write — no network |
| [GitHub](#github) | REST + GraphQL against `api.github.com`, needs `GITHUB_TOKEN` |
| [Slack](#slack) | API + web UI, needs `SLACK_BOT_TOKEN` and `SLACK_TEAM_ID` |
| [PostgreSQL](#postgresql) | Database connections, `.pgpass` / `.postgresql` credentials |
| [Brave Search](#brave-search) | Search API only, needs `BRAVE_API_KEY` |
| [Comprehensive reference](#comprehensive-reference) | Full schema with annotations for every field — use as a template |
| [Gateway registry](#gateway-registry) | Wrapping servers registered in an MCP gateway config |

## Filesystem

Local file access only — no network egress, no env passthrough.

```yaml
# Example Warden policy — @modelcontextprotocol/server-filesystem
# Copy this and adjust the command path, read grants, and env as needed.
#
# Relative paths below are resolved against THIS FILE's directory, not your
# shell's CWD. The executable must be an absolute path: the sandbox binds its
# parent directory, and a bare command name would resolve nowhere inside.

command: ["/usr/bin/node", "./server/dist/index.js"]

filesystem:
  # Read-only: the filesystem server reads project files. Grant only the
  # directories the server needs; deny-by-default prevents arbitrary reads.
  read:
    - "./data"
  # Read-write: the official filesystem server doesn't need to write.
  # If your use-case requires writes (e.g. creating files), grant a specific
  # directory instead of broadening this.
  write: []

network:
  # The filesystem server operates locally and has no network egress.
  allow: []

env:
  # No environment variables are required by this server. If your Node.js
  # runtime needs specific vars, add them here — values are never stored,
  # only the names are forwarded from the parent shell.
  allow: []

limits:
  # Minimal footprint — the server does pure local filesystem I/O.
  memory_mb: 128
  timeout_s: 60
```

## GitHub

REST + GraphQL against `api.github.com`, needs `GITHUB_TOKEN`.

```yaml
# Example Warden policy — @modelcontextprotocol/server-github
# Copy this and adjust the command path, cache/output dirs, and token name.
#
# Relative paths below are resolved against THIS FILE's directory, not your
# shell's CWD. The executable must be an absolute path.

command: ["/usr/bin/node", "./server/dist/index.js"]

filesystem:
  # Read-only cache for GitHub API responses or local artifacts.
  read:
    - "./data/cache"
  # Read-write output dir for any generated files (e.g. diffs, reports).
  # Keep this separate from your project source to limit blast radius.
  write:
    - "./data/output"

network:
  # Only the GitHub API is reachable; DNS lookups for other hosts are blocked.
  # If you need GraphQL (api.github.com) and REST simultaneously, they share
  # the same hostname so a single entry covers both.
  allow:
    - "api.github.com"

env:
  # GITHUB_TOKEN must be set in your parent shell (or passed via --env).
  # Warden forwards only the listed names; everything else is dropped.
  allow:
    - "GITHUB_TOKEN"
    - "HOME"

limits:
  # API calls and git operations can be memory-heavy; 300s covers long PR reviews.
  memory_mb: 256
  timeout_s: 300
```

## Slack

Slack API + web UI, needs `SLACK_BOT_TOKEN` and `SLACK_TEAM_ID`.

```yaml
# Example Warden policy — @modelcontextprotocol/server-slack
# Copy this and adjust the command path and token name as needed.
#
# Relative paths below are resolved against THIS FILE's directory, not your
# shell's CWD. The executable must be an absolute path.

command: ["/usr/bin/node", "./server/dist/index.js"]

filesystem:
  # Read-write cache for Slack OAuth tokens or rate-limit state.
  # A write grant subsumes reads underneath it, so a separate read grant
  # for the same dir is redundant (Load drops it during Normalize).
  read: []
  write:
    - "./data/cache"

network:
  # Slack's web UI and API endpoints. Both hostnames are typically needed.
  allow:
    - "slack.com"
    - "api.slack.com"

env:
  # SLACK_BOT_TOKEN and SLACK_TEAM_ID are required by the Slack MCP server.
  # Set them in your shell before running; only the listed names are forwarded.
  allow:
    - "SLACK_BOT_TOKEN"
    - "SLACK_TEAM_ID"
    - "HOME"

limits:
  # Network-bound servers allocate more memory for buffers; 300s covers
  # extended conversations or batch history exports.
  memory_mb: 256
  timeout_s: 300
```

## PostgreSQL

Database connections with file-based credentials.

```yaml
# Example Warden policy — PostgreSQL database MCP server
# Copy this and adjust hostnames, paths, and credentials as needed.
#
# Relative paths below are resolved against THIS FILE's directory, not your
# shell's CWD. The executable must be an absolute path.

command: ["/usr/bin/node", "./server/dist/index.js"]

filesystem:
  # PostgreSQL client libraries read .pgpass and .postgresql for credentials
  # and SSL config. Grant only the exact paths you need — never the whole home.
  read:
    - "./config/.pgpass"
    - "./config/.postgresql"
  # Database servers typically don't write files; keep this empty.
  write: []

network:
  # Allow connections to the database host(s) only. Replace with your
  # actual hostnames; wildcard hosts are not supported — list each one.
  allow:
    - "localhost"
    - "db.internal.example.com"

env:
  # Standard PostgreSQL connection variables. Values are set in your shell,
  # not in this file. Warden forwards only the names in this list.
  allow:
    - "PGHOST"
    - "PGPORT"
    - "PGUSER"
    - "PGPASSWORD"
    - "PGSSLMODE"
    - "HOME"

limits:
  # Database servers hold connection pools and query buffers in memory.
  # 600s covers long-running migrations or bulk exports.
  memory_mb: 512
  timeout_s: 600
```

## Brave Search

Search API only, needs `BRAVE_API_KEY`.

```yaml
# Example Warden policy — @modelcontextprotocol/server-brave-search
# Copy this and adjust the command path and API key name as needed.
#
# Relative paths below are resolved against THIS FILE's directory, not your
# shell's CWD. The executable must be an absolute path.

command: ["/usr/bin/node", "./server/dist/index.js"]

filesystem:
  # The search server doesn't read local files, but may cache responses.
  read: []
  write:
    - "./data/cache"

network:
  # Brave Search API endpoint only.
  allow:
    - "api.search.brave.com"

env:
  # BRAVE_API_KEY must be set in your parent shell. Warden forwards only
  # the listed names; everything else is dropped.
  allow:
    - "BRAVE_API_KEY"
    - "HOME"

limits:
  # Light memory footprint for a search proxy; short timeout is sufficient.
  memory_mb: 128
  timeout_s: 60
```

## Comprehensive reference

Full schema with annotations for every field — use as a template.

```yaml
# Warden Policy — Comprehensive Reference
# =========================================
#
# This file documents the full policy schema with examples for every field.
# Copy this file (or any example in this directory) and adjust it for your
# MCP server. Relative paths are resolved against THIS FILE's directory, not
# your shell's CWD. The executable must be an absolute path because the
# sandbox binds its parent directory — a bare command name would resolve
# nowhere inside.
#
# Run with: warden run --policy policy.example.yaml -- <command...>
# (The -- <command...> is optional if `command` is set in the file.)

# ── Command ──────────────────────────────────────────────────────────────────
#
# How to start the sandboxed server. If omitted, you must provide the command
# after `--` on the CLI. Prefer an absolute path to the executable; a bare
# name on PATH (npx, uvx, node) is resolved via LookPath at launch and fails
# closed when unresolvable. The remaining items are arguments.
#
# Tip: use `warden trace -- <cmd>` first to observe what your server touches,
# then feed the JSONL log into `warden init` to auto-generate a starter policy.
command: ["/usr/bin/node", "./server/dist/index.js"]

# ── Filesystem ───────────────────────────────────────────────────────────────
#
# The sandbox is deny-by-default. Only paths listed here are visible inside
# the sandbox, mounted at the same absolute path as on the host.
filesystem:
  # Read-only grants: the server can open and read files here but cannot
  # modify, create, or delete anything. Grant the smallest set of directories
  # the server actually needs — never the whole home or project root unless
  # required.
  read:
    - "./data/cache"      # Example: local response cache
    - "./config/secrets"  # Example: credential files the server must read

  # Read-write grants: the server can create, modify, and delete files here.
  # Keep this separate from source code to limit blast radius if the server
  # is compromised. An empty list means no write access at all. A write
  # grant subsumes reads underneath it: listing the same path in both read
  # and write is coalesced to a single write grant at load time.
  write:
    - "./data/output"     # Example: generated artifacts, caches that need writes

# ── Network ──────────────────────────────────────────────────────────────────
#
# The sandbox gets a fresh network namespace with no external route. A loopback
# bridge forwards HTTP/HTTPS traffic to the host-side egress proxy, which checks
# `network.allow` before performing any DNS lookup or connection. DNS lookups
# for disallowed hosts are blocked at the sandbox boundary.
#
# Only bare hostnames or IP literals are accepted — ports, paths, and URLs are
# rejected (the egress layer handles port selection). Wildcards are not
# supported; list each host your server needs to reach.
network:
  allow:
    - "api.github.com"   # Example: REST and GraphQL both live under this hostname
    - "slack.com"        # Example: web UI
    - "api.slack.com"    # Example: API
    - "localhost"        # Example: local database
    - "db.internal.example.com"  # Example: internal service

# ── Environment ──────────────────────────────────────────────────────────────
#
# Only the names listed here are forwarded from the parent shell. Values are
# NEVER stored in the policy file — this list only controls what gets passed
# through. An empty allowlist means the sandboxed process starts with no
# environment variables (except those explicitly set by the runtime).
#
# Variable names must be valid identifiers (no spaces, tabs, newlines, or `=`).
env:
  allow:
    - "GITHUB_TOKEN"      # Example: authentication credential
    - "SLACK_BOT_TOKEN"   # Example: bot auth
    - "HOME"              # Example: often needed by Node.js or library init
    - "PATH"              # Example: if the server spawns child processes

# ── Limits ───────────────────────────────────────────────────────────────────
#
# Resource constraints enforced on Linux (and best-effort on other platforms).
# Warden terminates the process group cleanly when a limit is breached.
# Set to 0 or omit to disable that particular limit.
limits:
  # Resident memory cap for the entire process tree (in MB). Use this to bound
  # memory leaks or runaway allocations. Example: 128 for lightweight servers,
  # 512+ for database or media-heavy workloads.
  memory_mb: 256

  # Wall-clock time limit (in seconds). Useful for preventing hangs or
  # indefinite loops. Long-running tasks (migrations, exports) may need higher
  # values or a longer timeout.
  timeout_s: 300
```

## Gateway registry

Example gateway registry mixing both supported YAML families. Remote entries
(`endpoint` / `http_url`) have no local process and are skipped by
`warden gateway init` with a clear reason. See [Gateway Integration](gateway.md).

```yaml
# Example gateway registry mixing both supported YAML families.
#
# - `upstreams` follows the jonfairbanks/mcp-gateway shape
#   (id, transport, command as string-or-list, args, env, endpoint).
# - `backends` follows the MikkoParkkola/mcp-gateway shape
#   (command as a single string, env, http_url for remote servers).
#
# Remote entries (endpoint/http_url) have no local process and are skipped by
# `warden gateway init` with a clear reason. Stdio entries get a
# deny-by-default policy in --policies <name>.yaml.
#
# Use ${VAR} for required secrets (fails closed when unset) and
# ${VAR:-default} for optional values. Values stay in the gateway file and the
# parent environment — generated policies store NAMES only, never secrets.

upstreams:
  - id: "context7"
    transport: "stdio"
    command: "npx"
    args:
      - "-y"
      - "@upstash/context7-mcp"

  - id: "github-remote"
    transport: "streamable_http"
    endpoint: "https://api.githubcopilot.com/mcp/"

backends:
  tavily:
    command: "npx -y @anthropic/mcp-server-tavily"
    description: "Web search via Tavily"
    env:
      TAVILY_API_KEY: "${TAVILY_API_KEY}"

  sentry:
    http_url: "https://mcp.sentry.dev/mcp"
    description: "Sentry issues (remote — not sandboxable)"
```

Example MCP gateway config (`gateway-mcp.json` shape):

```json
{
  "mcpServers": {
    "filesystem": {
      "command": "/usr/bin/node",
      "args": ["./server/dist/index.js"],
      "env": {}
    },
    "github": {
      "command": "/usr/bin/node",
      "args": ["./github/dist/index.js"],
      "env": {
        "GITHUB_TOKEN": "${GITHUB_TOKEN}"
      }
    },
    "remote-docs": {
      "type": "streamable-http",
      "url": "https://docs.example.com/mcp"
    }
  }
}
```

## Common patterns

### Where to put the server binary

All examples assume you've installed the MCP server (usually via `npm install`)
and the compiled entry point lives under `./server/dist/index.js` relative to
the policy file. Adjust the `command` array to point at your actual binary:

```yaml
command: ["/usr/bin/node", "./path/to/your/server.js"]
```

The executable **must be an absolute path** inside the sandbox because Warden
bind-mounts only the paths you list — a bare name like `node` would resolve
nowhere.

### Home directory grants

Several examples pass `HOME` through `env.allow` and grant read access to
`~/.config`-style paths. This is intentional — many Node.js and Python
libraries expect `HOME` to be set. **Do not combine `HOME` with a broad
`filesystem.read` grant** (especially `~/.ssh`); see
[Security Review](security.md#5-credential-exposure) for details.

### Empty allowlists are explicit

Notice that `network.allow: []` and `env.allow: []` appear in the filesystem
example. An empty list is *not* the same as omitting the key — the former
explicitly denies everything, the latter falls back to the default (which
varies by field). Being explicit in your policy makes intent clear and helps
`warden init` produce a correct starter.

## Next steps

- Check the [Compatibility Matrix](compatibility.md) — 18 tested
  servers with exact policies pinned as regression fixtures.
- Read the [Schema Reference](schema.md) to understand every field.
- Run `warden trace -- <your-server>` to see what a server actually touches.
- Run `warden init` to generate a starter policy from the trace output.
- Review [Security Review](security.md) before relying on any policy
  in production.
