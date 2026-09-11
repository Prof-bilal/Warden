# Warden

**A lightweight sandbox runtime for MCP servers.**

MCP servers routinely run as a plain Node or Python process on your machine
with full filesystem and network access — even ones you just cloned from
GitHub five minutes ago. Warden runs them in a restricted sandbox instead,
so a server only ever gets the files, network hosts, and environment
variables you explicitly grant it.

> **Status: alpha.** Warden works today — the npm package (`warden-sandbox-cli`),
> GitHub Releases binaries, and docs are live. Verification state, honestly:
> **Linux verified on real hardware** (escape tests, CI, proof harness);
> **Windows verified via CI** (AppContainer/WFP/ETW escape tests on GitHub
> Windows runners); **macOS code-complete and CI-green, pending a real-hardware
> proof-harness run**. Not yet hardened against a determined local attacker —
> see [REMAINING_WORK.md](./REMAINING_WORK.md) and [TESTING.md](./TESTING.md).
> Not a design skeleton: see [ARCHITECTURE.md](./ARCHITECTURE.md) for how it
> actually works.

## The problem

MCP's spec doesn't require any process isolation. The default install path
for most servers is "run this script with your user's full permissions."
That means a malicious or buggy MCP server can read your SSH keys, exfiltrate
data over the network, or write anywhere on disk — and nothing in the
protocol stops it.

## What Warden does

```
warden run --policy ./policy.yaml -- node ./my-mcp-server/index.js
```

Warden spawns the server inside a sandbox that:

- **Filesystem**: only sees the paths you list, read-only or read-write as you specify. Everything else is invisible, not just "permission denied."
- **Network**: can only reach the hostnames you allowlist. Everything else is blocked at the sandbox boundary, before DNS even resolves.
- **Environment**: only receives the environment variables you pass through — no automatic inheritance of your full shell environment.
- **Stdio**: passed through transparently, so the MCP client (Claude, an IDE, etc.) talks to the sandboxed process exactly like it would an unsandboxed one. Sandboxing is invisible to the protocol.
- **Audit log**: records every file access attempt and network connection attempt — including blocked ones — so you can see what a server *tried* to do.

## Quickstart

These commands exist and work as shown:

```bash
# 1. Write a policy describing what the server is allowed to touch
cat > policy.yaml <<EOF
command: ["node", "server.js"]
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
EOF

# 2. Run the server inside the sandbox
warden run --policy policy.yaml

# 3. Or generate a starter policy by watching what the server does once,
#    unsandboxed, so you're not writing the policy blind
warden trace -- node server.js
```

## How it works

See [ARCHITECTURE.md](./ARCHITECTURE.md) for the full design, but in short:
Warden is a thin CLI over OS-native sandboxing primitives — [bubblewrap](https://github.com/containers/bubblewrap)
on Linux, `sandbox-exec`/Seatbelt on macOS (with a Docker fallback), and a
policy engine that translates a simple YAML file into the low-level
namespace/seccomp/network rules each platform actually needs.

## Why not just use Docker?

You can, and Warden's macOS fallback does. But Docker is heavyweight for
"run one npm script with a restricted home directory" — slow cold starts,
a daemon dependency, and a much bigger trust boundary than a namespace
sandbox needs. Warden aims to be a single static binary with near-zero
overhead, so sandboxing an MCP server is no harder than running it.

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md) for how to get involved, and
[REMAINING_WORK.md](./REMAINING_WORK.md) for the current priorities.
Good first contributions: example policies for new MCP servers, testing the
backends on your platform, or improving the docs.

## License

MIT
