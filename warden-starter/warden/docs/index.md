# Warden

**A lightweight sandbox runtime for MCP servers.**

MCP servers routinely run as plain Node or Python processes on your machine
with full filesystem and network access — even ones you just cloned from
GitHub five minutes ago. Warden runs them in a restricted sandbox so a server
only ever gets the files, network hosts, and environment variables you
explicitly grant it.

> **Status:** All backends implemented. Homebrew, npm, and GitHub Releases
> distribution is in place. See [ROADMAP.md](https://github.com/warden-sandbox/warden/blob/main/warden-starter/warden/ROADMAP.md) for details.

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

See [Example Policies](https://github.com/warden-sandbox/warden/tree/main/warden-starter/warden/examples) and the [Schema Reference](schema.md)
for details.

## Get started

- **[Schema Reference](schema.md)** — complete field-by-field guide to `policy.yaml`
- **[Example Policies](https://github.com/warden-sandbox/warden/tree/main/warden-starter/warden/examples)** — real-world policy files
- **[Security Review](security.md)** — threat model and known limitations
- **[Architecture](architecture.md)** — how Warden works under the hood
- **[ROADMAP.md](https://github.com/warden-sandbox/warden/blob/main/warden-starter/warden/ROADMAP.md)** — milestones and current status

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

For the full design, see the [Architecture doc](architecture.md) or the
[ARCHITECTURE.md](https://github.com/warden-sandbox/warden/blob/main/warden-starter/warden/ARCHITECTURE.md) in the repo.
