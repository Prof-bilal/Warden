# CLI Reference

Every subcommand, flag, and exit behavior below matches the CLI source
(`cmd/warden`). With no subcommand, `warden` prints usage and exits 1.

## `warden run`

Run a server under a policy. The sandbox backend is auto-detected (native
first, Docker fallback, fail-closed when nothing is available).

```bash
warden run --policy <file> [--backend auto|linux|seatbelt|windows|docker]
           [--approve] [--approve-timeout <dur>] -- <command...>
```

| Flag | Meaning |
|---|---|
| `--policy <file>` | **Required.** Policy YAML. Relative `filesystem` grants resolve against this file's directory. |
| `--backend <name>` | Force a backend. Default `auto`. Aliases: `bwrap` → linux; `macos`, `darwin`, `sandbox-exec` → seatbelt; `appcontainer`, `lowbox` → windows. Unknown or unavailable backends fail closed. |
| `--approve` | Interactive approval: prompt on first out-of-policy access instead of hard-failing. Network approvals apply live; filesystem approvals are saved to the policy and the server restarts. Requires a terminal — fails closed without one. |
| `--approve-timeout <dur>` | Per-prompt timeout (e.g. `30s`, `2m`). Zero waits indefinitely; expiry denies. Requires `--approve`. |
| `-- <command...>` | Command to sandbox. Optional separator — everything after the flags is the command anyway; `--` lets the command itself start with a flag-like token. |

Command resolution: the CLI tail wins, otherwise the policy's `command:` is
used, otherwise the run fails with a usage error. Bare executable names
(`npx`, `uvx`, `node`) are resolved via `PATH` lookup and fail closed when
unresolvable; prefer absolute paths in checked-in policies. Exit code: the
sandboxed process's exit code, `1` on runtime failure, `2` on usage/policy
errors.

## `warden trace`

Run **unsandboxed** but instrumented; record every file/network access to the
trace log for `init` to consume:

```bash
warden trace -- <command...>
```

Prints the trace file path on stderr on exit. Exits with the command's exit
code.

## `warden init`

Generate a conservative starter policy from a trace log. Never overwrites an
existing file:

```bash
warden init [--log <file>] [--output <file>] [-- <command...>]
```

| Flag | Meaning |
|---|---|
| `--log <file>` | Trace log to read. Default: the latest trace. |
| `--output`, `-o <file>` | Where to write. Default `policy.yaml`. Refuses to overwrite. |
| `-- <command...>` | Recorded into the generated policy's `command:` field. |

Only successful observed accesses become grants; blocked events never do;
runtime paths (`/usr`, `/proc`, …) are omitted. Review and tighten the
output before relying on it.

## `warden logs`

Inspect the persistent JSONL audit log
(`${XDG_STATE_HOME:-~/.local/state}/warden/audit.jsonl`):

```bash
warden logs [--log <file>] [--tail <n> | -n <n>] [--follow | -f]
```

`--tail 0` prints nothing; `--follow` polls for appended events (survives
truncation). Prints "no audit events recorded yet" when the log doesn't exist.

## `warden version`

Print the stamped build version and exit 0:

```bash
warden version
warden --version
```

Release builds stamp the version at link time (the npm launcher downloads
that exact version's binary); unstamped source builds report `dev`.
Accepts `version`, `--version`, `-version`, and `-v`.

## `warden gateway`

Wrap gateway-registered servers so the gateway launches each one sandboxed:

```bash
warden gateway init --config <file> --policies <dir>
# Per-server deny-by-default policies (<name>.yaml). Remote (SSE/HTTP)
# servers are skipped — no local process to sandbox. Never overwrites.

warden gateway run --config <file> --policies <dir> --server <name> \
    [--backend ...] [--approve] [--approve-timeout <dur>] -- [extra args...]
# Run one registered stdio server sandboxed. Fails closed when the policy
# is missing or the server is remote. Bare launcher names resolve via PATH.

warden gateway wrap --config <file> --policies <dir> \
    [--warden-bin <path>] [--backend <name>] [--output <file>]
# Emit a gateway config whose stdio commands are prefixed with
# 'warden run --policy ...'. Point the gateway at the wrapped file.

warden gateway list --config <file> [--policies <dir>]
# Show registered servers: stdio vs remote, and policy presence.
```

Supported configs: Claude-style `mcpServers` JSON (Claude Desktop/Code,
Cursor, VS Code `servers`, Docker MCP Gateway clients) and gateway YAML
registries (`upstreams:` à la jonfairbanks/mcp-gateway, `backends:` à la
MikkoParkkola/mcp-gateway).

## Exit codes

| Code | Meaning |
|---|---|
| `0` | Success (for `run`: the sandboxed process exited 0) |
| other | For `run`/`trace`: the sandboxed/traced process's own exit code |
| `1` | Runtime failure (backend unavailable, enforcement error, approval restarts exhausted) |
| `2` | Usage or policy error (bad flags, unreadable/invalid policy, no command) |
