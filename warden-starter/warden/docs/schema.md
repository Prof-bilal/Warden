# Policy Schema Reference

This document describes the complete policy YAML schema used by Warden.
See [examples/policy.example.yaml](https://github.com/Prof-bilal/Warden/blob/main/warden-starter/warden/examples/policy.example.yaml) for a
fully-worked example.

## Top-level fields

| Field | Type | Required | Default | Description |
|---|---|---|---|---|
| `command` | `[]string` | No* | CLI-specified via `--` | How to start the sandboxed process. If omitted, you must supply the command after `--` on the CLI. |
| `filesystem` | `Filesystem` | Yes | — | Filesystem access grants. |
| `network` | `Network` | Yes | — | Network egress allowlist. |
| `env` | `Env` | Yes | — | Environment variable passthrough list. |
| `limits` | `Limits` | No | — | Resource limits enforced on Linux. |

\* A policy file without `command:` is valid as long as the CLI supplies the
command via `warden run --policy <file> -- <cmd...>`.

## `command`

```yaml
command: ["/usr/bin/node", "server.js"]
```

The executable should be an **absolute path** inside the sandbox — the sandbox
binds the parent directory read-only, so only an absolute path is guaranteed
visible. For convenience, `warden run` (and `warden gateway run`) also accept
a bare name on `PATH` (`npx`, `uvx`, `node`) and resolve it via `LookPath` at
launch, failing closed when the name is not found. Prefer the absolute path
in checked-in policies so the grant is explicit and reproducible.

Values after the executable are passed as-is — relative paths here are
resolved against the policy file's directory by `ResolvePaths`.

## `filesystem`

```yaml
filesystem:
  read: ["./data/cache"]
  write: ["./output"]
```

Each entry is a path (relative to the policy file, resolved by
`ResolvePaths`). After resolution:

- `read` paths are bind-mounted **read-only**.
- `write` paths are bind-mounted **read-write**.
- Paths inside the runtime base (`/usr`, `/lib64`) cannot be granted write —
  that would make part of the system writable and widen the sandbox.
- A path listed in both `read` and `write` is coalesced to a single `write`
  grant at load time (`Normalize`), since a write grant already subsumes
  reads underneath it. The backends keep a fail-closed ambiguity check for
  policies constructed programmatically without going through `Load`.

**Validation rules:**

- Paths must not be empty.
- After resolution, paths must be absolute.
- Paths resolving to the filesystem root (`/`) are rejected.
- Relative paths are resolved against the directory containing the policy
  file, not the current working directory.

> **Enforced** — Yes. The Linux (bwrap) backend mounts only these paths;
> everything else is invisible to the sandboxed process.

## `network`

```yaml
network:
  allow:
    - "api.github.com"
    - "10.0.0.5"
```

A list of hostnames or IP literals the sandboxed process may reach. Ports
are **not** part of the grant — the egress proxy forwards all ports for an
allowlisted host.

**Validation rules:**

- Entries must not be empty.
- Must be a bare hostname or IP literal — paths (`/foo`), query strings
  (`?q=1`), fragments (`#anchor`), and colons (interpreted as ports) are
  rejected.
- Hostnames must not exceed 253 characters; each label must be 1–63 chars,
  alphanumeric or hyphen, and must not start or end with a hyphen.
- IPv6 literals are accepted via `net.ParseIP`.

> **Enforced** — Linux only. The sandbox gets a private network namespace
> with no default route; all external traffic goes through the egress proxy
> which checks this allowlist. On macOS the Seatbelt backend denies all
> network except loopback to the proxy bridge; the proxy enforces the list.
> Docker runs with `--network none` and similarly uses the proxy bridge.

See [Architecture – Egress Proxy](architecture.md#egress-proxy-architecture) for the
proxy design.

## `env`

```yaml
env:
  allow:
    - "GITHUB_TOKEN"
    - "NODE_ENV"
```

A list of environment variable **names** to forward from the parent process.
Values are **never** stored in the policy file — they are read from the
parent at runtime. An empty allowlist results in an empty environment for the
sandboxed process.

**Validation rules:**

- Variable names must not be empty.
- Names must not contain spaces, tabs, newlines, or `=` characters.

> **Enforced** — All platforms. `envfilter.Filter` selects only the
> allowlisted names from `os.Environ()` before spawning the child.

> **Security note:** If you pass `HOME` through `env.allow`, the sandboxed
> process sees your real home directory. Combined with a `filesystem.read`
> grant for `~/.ssh`, this leaks your SSH keys. See [Security Review](security.md) for details.

## `limits`

```yaml
limits:
  memory_mb: 512
  timeout_s: 300
```

| Sub-field | Type | Required | Default | Description |
|---|---|---|---|---|
| `memory_mb` | `int` | No | `0` (no limit) | Resident memory cap for the entire process tree. Sampled every 25 ms; exceeded processes receive SIGTERM then SIGKILL after 750 ms. |
| `timeout_s` | `int` | No | `0` (no limit) | Wall-clock timeout. Breach triggers the same SIGTERM → SIGKILL sequence. |

**Validation rules:**

- Values must not be negative.
- `memory_mb` must not overflow when converted to bytes.
- `timeout_s` must not overflow a `time.Duration`.

> **Enforced** — Linux (bwrap) backend. macOS and Docker backends do not
> currently enforce these limits; the values are parsed for schema
> compatibility but silently ignored. No CPU throttling is available on any
> platform.

## Cross-reference

- For a complete working policy, see [examples/policy.example.yaml](https://github.com/Prof-bilal/Warden/blob/main/warden-starter/warden/examples/policy.example.yaml).
- For backend-specific enforcement details, see [Architecture – Sandbox Backends](architecture.md#backend-selection).
- For security implications of each section, see [Security Review](security.md).
