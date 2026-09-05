# Architecture

This document summarises how Warden works. For the full design including
non-goals and rationale, see [ARCHITECTURE.md](https://github.com/warden-sandbox/warden/blob/main/warden-starter/warden/ARCHITECTURE.md).

## Component overview

```
MCP Client (Claude / IDE / agent)
        │
        │  stdio
        ▼
┌───────────────────────────────────────────────────────┐
│                    warden CLI                         │
│  run  ·  trace  ·  init  ·  logs                      │
└───────────┬───────────────────────────────┬───────────┘
            │                               │
            ▼                               ▼
   ┌─────────────────┐           ┌───────────────────┐
   │  Policy Engine   │           │  Egress Proxy      │
   │  (parse, validate│           │  (hostname allow-  │
   │   resolve paths) │           │   list, DNS block) │
   └────────┬────────┘           └─────────┬─────────┘
            │                              │
            ▼                              │
   ┌─────────────────┐                     │
   │ Sandbox Backend  │◄────────────────────┘
   │ (bwrap /        │   Unix socket /
   │  Seatbelt /     │   loopback TCP
   │  Docker /       │
   │  AppContainer)  │
   └────────┬────────┘
            │
            ▼
   ┌─────────────────┐
   │ MCP Server       │  (sandboxed process)
   │ process          │
   └─────────────────┘
            │
            ▼
   ┌─────────────────┐
   │ Audit Logger     │  (JSONL: ~/.local/state/warden/audit.jsonl)
   └─────────────────┘
```

## Backend selection

`warden run` auto-selects the best available backend based on the host OS:

| OS | Preferred backend | Fallback |
|---|---|---|
| Linux | bubblewrap (`bwrap`) | Docker |
| macOS | `sandbox-exec` (Seatbelt) | Docker |
| Windows | AppContainer (planned M5) | Fail closed |
| Other | Docker | Fail closed |

Force a specific backend with `--backend linux|seatbelt|docker|windows`.
Docker is **never** preferred over a working native backend.

## How each backend works

### Linux — bubblewrap (`bwrap`)

Warden invokes `bwrap` with a carefully constructed argument list:

- **Namespaces:** `--unshare-user --unshare-ipc --unshare-pid --unshare-net`
  gives the process isolated user, IPC, PID, and network namespaces. The
  process runs as UID/GID 0 *inside* the namespace, but cannot escape.
- **Runtime base:** `/usr` and `/lib64` are bind-mounted read-only so
  dynamically-linked binaries can execute. On merged-`/usr` systems both
  must be bound explicitly because bwrap does not follow symlinks across
  bind mounts.
- **Pseudo-filesystems:** `/dev`, `/proc`, and a fresh `/tmp` (tmpfs) are
  mounted.
- **Policy grants:** Each `filesystem.read` path is `--ro-bind`'d; each
  `filesystem.write` path is `--bind`'d. Paths inside the runtime base
  cannot be granted write access.
- **Proxy bridge:** The Warden binary is bind-mounted to `/.warden/proxy-bridge`
  inside the namespace and run with `__proxy-bridge` to expose a loopback
  HTTP proxy on `127.0.0.1:18080`. The host-side egress proxy connects via
  a bind-mounted Unix socket at `/.warden/host-proxy/egress.sock`.
- **Environment:** Only names in `env.allow` are forwarded; all others are
  discarded. Proxy variables (`HTTP_PROXY`, `HTTPS_PROXY`, `ALL_PROXY`) are
  injected automatically.
- **Auditing (Linux only):** If a logger is configured, Warden wraps the
  bwrap invocation in `strace -f -e trace=%file,%network`, writing syscall
  traces to a temp file that is imported into the audit log after the
  sandbox exits.

### macOS — `sandbox-exec` (Seatbelt)

Apple's `sandbox-exec` enforces a generated Seatbelt profile. The profile
denies all filesystem and network access by default; grants are added for
each policy-allowed path and for loopback TCP to the egress proxy bridge.
File denies produce `EPERM`; structured audit events come from the egress
proxy. Seatbelt is deprecated by Apple but remains functional.

### Docker

When no native backend is available, Warden runs the command in a Docker
container with `--network none`. Host paths are bind-mounted deny-by-default
(only policy-granted paths are mounted). The in-container proxy bridge is
the current Warden ELF binary (Linux). On macOS/Windows hosts, set
`WARDEN_DOCKER_BRIDGE` to a cross-compiled Linux binary. The default image
is `alpine:3.20` (override with `WARDEN_DOCKER_IMAGE`).

### Windows — AppContainer (M5)

An early-stage backend using a restricted token, filesystem capabilities,
Windows Filtering Platform (WFP) rules, ETW audit events, and a Job Object
for process-tree limits. Warden refuses to run if any of these primitives
fail to install — it never falls back to an unrestricted process.

## Policy flow

```
policy.yaml
    │
    ▼
Load(path)                  // Read file, decode YAML with KnownFields(true)
    │
    ▼
ResolvePaths(dir)           // Resolve relative paths against policy file's dir
    │
    ▼
Validate()                  // Check semantic constraints (absolute paths, valid hosts, etc.)
    │
    ▼
ResolveCommand(cli)         // Merge CLI-specified command with policy.command
    │
    ▼
BuildBwrapArgs(cmd, p)      // Translate to backend-specific arguments
    │
    ▼
spawn(process)              // Execute with stdio passthrough
```

Key invariant: **unknown YAML keys are rejected** (`KnownFields(true)`), so
typos in policy files fail closed rather than being silently ignored.

## Egress proxy architecture

The egress proxy is the *only* network出口 from the sandbox:

1. On Linux, the sandbox gets a private network namespace — no default route,
   no external interfaces. `HTTP_PROXY`, `HTTPS_PROXY`, and `ALL_PROXY` point
   to `127.0.0.1:18080` (the bridge inside the namespace).
2. The bridge accepts loopback TCP connections and forwards them over a
   bind-mounted Unix socket to the host-side proxy.
3. The host-side proxy checks the hostname against `network.allow` *before*
   performing any DNS lookup or TCP connection. Blocked requests return 403.
4. Allowed connections are forwarded to the upstream host. For HTTP CONNECT
   (HTTPS/TLS), the proxy hijacks the connection and bidirectionally pumps
   data.
5. Every allow/deny decision is logged as a structured audit event.

DNS leakage is prevented because DNS resolution only happens inside the
proxy, after the allowlist check. Direct DNS queries from the sandbox have
no route.

## Audit log flow

- **Persistent log:** `${XDG_STATE_HOME:-~/.local/state}/warden/audit.jsonl`
  — append-only, mode `0o600`, written by the parent process.
- **Trace logs:** `${XDG_STATE_HOME:-~/.local/state}/warden/traces/<timestamp>.jsonl`
  — per-session logs created by `warden trace`, used as input to `warden init`.
- **Linux strace temp:** `/tmp/warden-strace-*.log` — deleted after import.

Each event is a JSON object with fields: `timestamp`, `type` (`file` or
`network`), `action`, `resource`, `allowed` (bool), `reason`.

View with `warden logs [--tail N] [--follow]`.

## Limits enforcement (Linux)

`limits.memory_mb` is enforced by sampling the resident memory of the
launcher process group every 25 ms. `limits.timeout_s` is a wall-clock timer.
On breach:

1. SIGTERM is sent to the process group.
2. After 750 ms, SIGKILL is sent if the process has not exited.
3. A structured `limit` audit event is logged.

Limits are **not** currently enforced on macOS or Docker backends.
