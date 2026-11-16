# What Warden Is

**Warden is a sandbox for AI tools.** It runs an MCP server, a coding agent, or any local script so it can only see the files, network hosts, and environment variables you allow.

Most MCP servers you install run as a normal process with **your** permissions: your SSH keys, your tokens, your files, and unrestricted network access. Nothing in the MCP protocol stops a buggy or malicious server from reading `~/.ssh` or sending data out. Warden fixes that without changing how your client talks to the server.

```bash
warden run --policy ./policy.yaml -- node ./my-mcp-server/index.js
```

## How it works

You describe what the program may touch in a small policy file:

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

Then run it inside the sandbox:

```bash
warden run --policy policy.yaml
```

Not sure what to put in the policy? Let Warden watch the program once, without the sandbox, and draft one for you:

```bash
warden trace -- /usr/bin/node server.js && warden init
```

Review the draft before you rely on it.

If a local coding agent needs to edit your project, use an [AI agent workspace](agent-workspace.md). The agent works on a sanitized copy, declared secrets stay hidden, and you review the changes before they touch your project.

## Why Warden

- **Files:** Only the paths you grant exist inside the sandbox, read-only or read-write as you choose. Everything else is not there at all, so the program cannot even see it.
- **Network:** Only the hosts you allow can be reached. Other hosts are blocked before DNS lookup.
- **Environment:** Only the variables you name are passed in. Your shell environment does not leak in by default.
- **Audit log:** Every access, allowed and blocked, is recorded. Run `warden logs` to see what a program tried to do.
- **One binary:** No daemon and no container per run. Sandboxing a program is about as easy as running it.

## Proven against real servers

Warden is tested against 18 real MCP servers: 14 pass, 2 pass with conditions, and 2 fail (documented). Each tested server has its policy pinned as a regression test, so updates cannot silently break what works. Check your server before you install:

**[Compatibility matrix](compatibility.md)** · **[Beta program](beta.md)**

## Get started

1. **[Install](install.md):** Linux, macOS, Windows, Docker fallback, or build from source.
2. **[Quickstart](quickstart.md):** Your first sandboxed run in about five minutes.
3. **[Protect an AI agent](agent-workspace.md):** Let an agent edit code without seeing your secrets.
4. **[Example policies](examples.md):** Copy-paste policies for filesystem, GitHub, Slack, Postgres, and Brave Search.
5. **[Schema reference](schema.md):** Every `policy.yaml` field.
6. **[CLI reference](cli.md):** Every command and flag.
7. **[FAQ and troubleshooting](faq.md):** Common errors and how to fix them.

> **Status: beta.** Sandboxing on Linux, macOS, and Windows, plus Docker fallback, tracing, approval mode, and gateway integration are implemented and tested. Install from [npm](https://www.npmjs.com/package/warden-sandbox-cli) (`warden-sandbox-cli`) or [GitHub Releases](https://github.com/Prof-bilal/Warden/releases). A Homebrew tap is not available yet. You can also [build from source](install.md#3-build-from-source). See the [roadmap](roadmap.md) and [about](about.md) pages for details.
