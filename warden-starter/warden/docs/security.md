# Security Review

Warden is designed with a deny-by-default threat model: nothing is accessible
inside the sandbox unless the policy explicitly grants it. This document
outlines the known threats, mitigations, gaps, and limitations.

## Threat Model

### 1. Sandbox escape — access outside the granted boundary

**Threat.** A compromised or buggy sandboxed process attempts to read files,
write to disk, or reach the network outside the paths and hosts granted in
the policy.

**Mitigation.**
- Filesystem: bwrap bind-mounts only declared paths; everything else is
  invisible inside the sandbox (not merely permission-denied).
- Network: `bwrap --unshare-net` gives the process a private network
  namespace with no default route. All external traffic must traverse the
  egress proxy, which checks the hostname allowlist before DNS resolution.
- Environment: `envfilter.Filter` discards all non-allowlisted variables.

**Gaps.**
- `bwrap` itself is trusted external code; a vulnerability in bwrap could
  undermine the namespace isolation.
- The Docker fallback has a larger attack surface (Docker daemon, image, etc.)
  than the native bwrap backend.

### 2. Policy confusion / path traversal

**Threat.** A malformed or malicious policy file grants overly broad access
(e.g., path traversal to `/` or the host root).

**Mitigation.**
- `KnownFields(true)` on the YAML decoder rejects unknown keys, catching
  typos and accidental misconfiguration.
- Relative paths are resolved against the policy file's own directory; a
  path resolving to the filesystem root (`/`) is rejected.
- Write grants inside the runtime base (`/usr`, `/lib64`) are rejected to
  prevent making system directories writable.

**Gaps.**
- Symlinks inside a granted path could redirect the sandboxed process to
  other locations on the host. Warden does not currently resolve or block
  symlinks.

### 3. Egress proxy bypass

**Threat.** The sandboxed process bypasses the egress proxy and reaches the
network directly, or leaks DNS queries outside the proxy.

**Mitigation.**
- The private network namespace has no default route; direct connections to
  any external host have no route and fail.
- `HTTP_PROXY`, `HTTPS_PROXY`, and `ALL_PROXY` are injected into the
  sandbox environment pointing to the local bridge; direct DNS queries go
  nowhere.
- The egress proxy performs DNS lookup only *after* the hostname passes the
  allowlist, preventing DNS leakage for blocked hosts.

**Gaps.**
- If `bwrap --unshare-net` fails because unprivileged user namespaces are
  disabled on the host, the fallback falls open: the proxy is present but
  the process may have host network access. Warden should fail closed in
  this scenario but currently does not always detect it.

### 4. Audit log tampering

**Threat.** A sandboxed process modifies or deletes its own audit log to
cover up access attempts.

**Mitigation.**
- The persistent audit log at `${XDG_STATE_HOME:-~/.local/state}/warden/audit.jsonl`
  is written by the parent Warden process *outside* the sandbox.
- On Linux, `strace` runs outside the sandbox and writes trace output to a
  temp file that is imported into the audit log after the sandbox exits.
- The log file is created with mode `0o600` (owner-only).

**Gaps.**
- strace temp files (`warden-strace-*.log`) are created under `/tmp` and
  imported later; while Warden cleans them up, a race condition could
  theoretically allow observation of unprocessed trace data.

### 5. Credential exposure

**Threat.** Sensitive environment variables (SSH keys, tokens, secrets) leak
into the sandbox.

**Mitigation.**
- `env.allow` uses a strict allowlist; variables not explicitly listed are
  discarded. An empty allowlist produces an empty environment.
- Variable names are validated to reject malformed entries that could
  interfere with the filter.

**Gaps.**
- If `HOME` is passed through `env.allow` **and** `~/.ssh` (or a parent
  directory) is granted in `filesystem.read`, the sandboxed process can
  read SSH keys. This is a policy-authoring mistake, not a Warden bug, but
  it is easy to make inadvertently.
- The `warden trace` flow runs unsandboxed, so sensitive env vars are
  visible in the trace log until the generated policy removes them.

### 6. Resource exhaustion

**Threat.** A sandboxed process consumes unlimited CPU or memory, starving
other workloads or causing OOM kills on the host.

**Mitigation.**
- `limits.memory_mb` is enforced on Linux by sampling the process tree's
  resident memory every 25 ms; breach triggers SIGTERM, then SIGKILL after
  750 ms.
- `limits.timeout_s` enforces a wall-clock cap with the same termination
  sequence.
- Both breaches are recorded as structured audit events.

**Gaps.**
- macOS (Seatbelt) and Docker backends do not enforce memory or timeout
  limits; the values are parsed but silently ignored.
- No platform currently throttles CPU usage or uses cgroups for resource
  isolation.

### 7. Proxy bridge privilege

**Threat.** The proxy bridge (running inside the sandbox) has broader
network access than the sandboxed process and could be exploited to forward
traffic to unauthorized destinations.

**Mitigation.**
- The bridge only receives connections from the loopback interface; the
  sandboxed process cannot reach it from any other namespace.
- The host-side proxy enforces the same `network.allow` allowlist before
  performing any DNS lookup or upstream connection.
- The bridge binary is bind-mounted read-only into the sandbox.

**Gaps.**
- A bug in the bridge or proxy forwarding logic could theoretically allow
  traffic to non-allowlisted hosts. The bridge's trust boundary is
  inherently larger than the sandboxed process's.

## Known Limitations

| Area | Limitation |
|---|---|
| Network enforcement | Relies on `bwrap --unshare-net`; if user namespaces are disabled on the host, Warden falls back to Docker or runs with broader access. |
| CPU limits | No CPU throttling or cgroup-based limits on any platform. |
| Process-tree visibility | Limits apply to the direct child process tree only; grandchildren spawned via `fork()` + `exec()` outside the tracked tree may escape limits. |
| macOS Seatbelt | Deprecated by Apple; may be removed in a future macOS release with no Warden fallback on native. |
| Docker fallback | Requires a running Docker daemon. The in-container proxy bridge must be a Linux ELF binary — on macOS this requires cross-compilation. |
| Windows AppContainer | Early-stage implementation; edge cases with process groups and WFP rule ordering are possible. |
| strace auditing | Adds ~2–5x overhead on Linux; `strace` must be installed. Auditing is optional — runs without a logger fall back to syscall-level denials only. |
| Egress proxy scope | Only intercepts HTTP/HTTPS traffic. Raw TCP and UDP connections cannot be filtered by the proxy. |

## Security Best Practices for Users

1. **Run `warden trace` before writing policy by hand.** It records actual
   syscalls and generates a conservative starter policy that you can
   review and tighten.

2. **Never pass `HOME` in `env.allow` unless you also restrict
   `filesystem.read` appropriately.** The combination of `HOME` + a broad
   read grant (especially `~/.ssh`) leaks credentials.

3. **Review the audit log after each run.** Use `warden logs --tail 50` to
   check for blocked attempts that may indicate a misconfigured policy or
   unexpected server behavior.

4. **Keep Warden updated.** Security fixes land in new releases — check
   [ROADMAP.md](https://github.com/warden-sandbox/warden/blob/main/warden-starter/warden/ROADMAP.md) for the current status and the issue
   tracker for disclosed vulnerabilities.

5. **Use absolute paths for `command`.** Relative executable paths may
   resolve to unexpected locations inside the sandbox.

6. **Minimize `filesystem.write` grants.** Write access lets the server
   modify files on disk — grant write only where absolutely necessary and
   prefer read-only mounts elsewhere.
