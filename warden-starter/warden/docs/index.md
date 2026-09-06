# Warden

**Your MCP servers run with your keys to the kingdom. Warden takes them back.**

Every MCP server you install — that filesystem helper, that Slack bot, that
script you cloned five minutes ago — runs as a plain process with **your**
full permissions: your SSH keys, your tokens, your files, unrestricted
network. Nothing in the MCP protocol stops a buggy or malicious server from
reading `~/.ssh` or exfiltrating data. Most installs are one `npx` command
away from total access.

Warden runs each server in a restricted sandbox instead. A server only ever
sees the files, network hosts, and environment variables you explicitly
grant it — everything else is invisible. Sandboxing is invisible to the
protocol: your MCP client talks to the sandboxed server exactly as before.

```bash
warden run --policy ./policy.yaml -- node ./my-mcp-server/index.js
```

## How it works in 30 seconds

Write a policy describing what the server may touch:

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

Run it sandboxed — or generate the policy automatically by watching the
server once, unsandboxed:

```bash
warden run --policy policy.yaml
# or:
warden trace -- /usr/bin/node server.js && warden init
```

## Why Warden

- **Filesystem** — only granted paths exist inside the sandbox, read-only or
  read-write as you specify. The rest isn't "permission denied" — it's gone.
- **Network** — only allowlisted hostnames resolve and connect. Everything
  else is blocked before DNS even resolves.
- **Environment** — only the variables you name are passed through. Your
  shell environment never leaks in by default.
- **Audit log** — every access attempt, allowed *and blocked*, is recorded.
  See what a server *tried* to do with `warden logs`.
- **Single static binary**, near-zero overhead — sandboxing a server is no
  harder than running it. No daemon, no containers per run.

## Proven against real servers

Warden is tested against **18 real-world MCP servers — 14 pass, 2
conditional, 2 fail (documented)** — each with its exact policy pinned as a
regression fixture, so updates can't silently break what used to work. Check
whether your server works before installing:

**[→ Compatibility Matrix](compatibility.md)** · **[→ Beta Program](beta.md)**

## Get started

1. **[Install](install.md)** — Linux, macOS, Windows, Docker fallback, or
   build from source
2. **[Quickstart](quickstart.md)** — your first sandboxed run in five minutes
3. **[Schema Reference](schema.md)** — every `policy.yaml` field
4. **[CLI Reference](cli.md)** — every command and flag
5. **[FAQ](faq.md)** — common failures and fixes

> **Status:** beta. Core sandboxing (Linux, macOS, Windows, Docker fallback),
> tracing, approval mode, and gateway integration are implemented and tested.
> Distribution via Homebrew and npm is coming soon — today, install from a
> [GitHub Release](https://github.com/Prof-bilal/Warden/releases) or build
> from source. See the [roadmap](https://github.com/Prof-bilal/Warden/blob/main/warden-starter/warden/ROADMAP.md)
> and [About](about.md) pages for details.
