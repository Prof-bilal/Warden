# Warden

**A lightweight sandbox runtime for MCP servers.**

MCP servers routinely run as a plain Node or Python process on your machine
with full filesystem and network access — even ones you just cloned from
GitHub five minutes ago. Warden runs them in a restricted sandbox instead,
so a server only ever gets the files, network hosts, and environment
variables you explicitly grant it.

> Status: M3 complete on Linux. See [ROADMAP.md](./ROADMAP.md) for what's
> built and what's next.

> Windows builds are cross-compiled and fail closed today; native Windows
> enforcement is planned for M5.

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

- **Filesystem**: only sees the paths you list, read-only or read-write as you specify. Everything else is invisible, not just "permission denied." (Implemented in M1)
- **Network**: can only reach the hostnames you allowlist. Everything else is blocked at the sandbox boundary, before DNS even resolves. (Implemented on Linux)
- **Environment**: only receives the environment variables you pass through — no automatic inheritance of your full shell environment. (Implemented in M1)
- **Stdio**: passed through transparently, so the MCP client (Claude, an IDE, etc.) talks to the sandboxed process exactly like it would an unsandboxed one. Sandboxing is invisible to the protocol. (Implemented in M1)
- **Audit log**: records every file access attempt and network connection attempt — including blocked ones — so you can see what a server *tried* to do. (Implemented on Linux)

> **M3 Status**: Linux filesystem sandboxing, hostname-restricted egress,
> JSONL audit logging, trace/init tooling, and resource limits are implemented.

## Quickstart (M3 — Linux usability layer implemented)

```bash
# 1. Write a policy describing what the server is allowed to touch
cat > policy.yaml <<EOF
command: ["/usr/bin/node", "server.js"]
filesystem:
  read: ["./data"]
  write: ["./output"]
network:
  allow: ["api.github.com"]
env:
  allow: ["GITHUB_TOKEN"]
EOF

# 2. Run the server inside the sandbox
warden run --policy policy.yaml

# `network.allow` and limits are enforced on Linux.
```

```bash
# 1. Write a policy describing what the server is allowed to touch
cat > policy.yaml <<EOF
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
EOF

# 2. Run the server inside the sandbox
warden run --policy policy.yaml

# 3. Or generate a starter policy by watching the server once, unsandboxed.
warden trace -- /usr/bin/node server.js
warden init -- /usr/bin/node server.js
```

## How it works

See [ARCHITECTURE.md](./ARCHITECTURE.md) for the full design, but in short:
Warden is a thin CLI over OS-native sandboxing primitives — [bubblewrap](https://github.com/containers/bubblewrap)
on Linux, `sandbox-exec`/Seatbelt on macOS (with a Docker fallback), and a
policy engine that translates a simple YAML file into the low-level
namespace/seccomp/network rules each platform actually needs.

## Linux network and audit requirements

Linux network enforcement uses `bwrap` for a private network namespace and
an HTTP CONNECT proxy for the only egress path. Warden injects `HTTP_PROXY`,
`HTTPS_PROXY`, and `ALL_PROXY`; applications that need permitted network
access must honor standard proxy variables. An application that bypasses them
has no direct route, so its connection and DNS query fail instead.

Complete syscall-level auditing uses `strace`, which Warden runs outside the
sandbox so the server cannot modify the record. Both `bwrap` and `strace` are
required on Linux; Warden fails closed if either is unavailable. Events are
stored as JSON Lines at `${XDG_STATE_HOME:-~/.local/state}/warden/audit.jsonl`
and can be printed with `warden logs`.

## Trace, starter policies, and limits

`warden trace -- <command...>` runs a command unsandboxed but records its
file and network syscalls in an isolated trace session. Review that log, then
run `warden init -- <command...>` to create a non-overwriting `policy.yaml`
from the newest trace, with only successful observed grants. Use `--log` or
`--output` to select files.
`warden logs --tail 50 --follow` prints recent JSONL events and follows new
ones.

`limits.memory_mb` is a process-tree resident-memory cap, sampled every
25 ms. `limits.timeout_s` is a wall-clock cap. On either breach Warden sends
SIGTERM to the server process group, waits briefly, then sends SIGKILL if the
server did not exit. A limit breach is also recorded in the audit log.

## Why not just use Docker?

You can, and Warden's macOS fallback does. But Docker is heavyweight for
"run one npm script with a restricted home directory" — slow cold starts,
a daemon dependency, and a much bigger trust boundary than a namespace
sandbox needs. Warden aims to be a single static binary with near-zero
overhead, so sandboxing an MCP server is no harder than running it.

## Project docs

- [PRD.md](./PRD.md) — product requirements and scope
- [MVP.md](./MVP.md) — minimum viable product definition for M1
- [AGENTS.md](./AGENTS.md) — repository guidance for AI coding agents
- [CODESTYLE.md](./CODESTYLE.md) — Go style and package conventions
- [TESTING.md](./TESTING.md) — test strategy, fixtures, and CI expectations
- [REVIEWING.md](./REVIEWING.md) — reviewer checklist for security-sensitive changes

## Contributing

This project is just getting started — see [ROADMAP.md](./ROADMAP.md) for
the first milestones and [ARCHITECTURE.md](./ARCHITECTURE.md) for the design.
Good first contributions right now: fleshing out the policy schema, a
working Linux/bubblewrap prototype for milestone M1, or test MCP servers to
validate against.

## License

MIT
