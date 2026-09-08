# CLI Reference

Every subcommand, flag, and exit behavior below matches the CLI source
(`cmd/warden`). With no subcommand, `warden` prints usage and exits 1.

## CLI Experience

Warden has a consistent, security-focused terminal identity. Colors and
banners are semantic and degrade gracefully.

- **Banner:** A large `WARDEN` ASCII banner is shown for `warden` (no args),
  `warden init`, and `warden doctor` when stderr is a TTY. It is never shown
  for `warden --version`, `warden --help`, or `warden run` so that
  scripted and sandboxed invocations stay clean. On narrow terminals
  (`COLUMNS<60`) the banner falls back to a compact header; on CI or when
  output is piped it is omitted entirely.

- **Header:** `warden doctor` and `warden init` show a compact header
  (`WARDEN` + `MCP Server Sandbox Runtime` + `Version …`) when the large
  banner is not used.

- **Colors:** Green marks success/allowed/sandbox-active, red marks
  blocked/denied/errors, cyan is used for structural headings/paths/metadata,
  and normal foreground is used for body text. Colors are semantic and never
  the sole signal — every state also has a text marker (`✓`/`✗` + `ALLOWED`/
  `BLOCKED` where applicable) for accessibility.

- **Color control:** Honors `NO_COLOR=1` (https://no-color.org),
  `FORCE_COLOR`/`CLICOLOR_FORCE`, and `TERM=dumb`. Set `NO_COLOR=1` or
  `WARDEN_NO_COLOR=1` to disable all ANSI. Use `WARDEN_NO_UNICODE=1` to
  force ASCII fallbacks (`OK`/`x` instead of `✓`/`✗`).

- **Non-TTY / CI:** When `CI` is set or stderr is not a TTY, Warden avoids
  animations and uses deterministic bracketed progress (`[1/4] … OK`) so
  logs stay clean. Spinners are short, never leave stray characters, and
  are disabled in CI. Set `WARDEN_NO_SPINNER=1` to force static output.

- **First-run:** The first time Warden is invoked interactively it shows a
  one-time welcome (`Welcome to Warden` + capabilities + `warden init`/`run`/
  `doctor` hints) and then never again. The marker lives at
  `${XDG_STATE_HOME:-~/.local/state}/warden/welcomed` (`0600`). It is
  suppressed in CI, when `WARDEN_NO_FIRST_RUN=1` is set, or when neither
  stdout nor stderr is a TTY. It never blocks `warden run` and never
  requires interactive input.

- **Installation:** The npm wrapper (`build/npm-wrapper/install.js`) shows
  polished progress with TTY spinner vs CI static fallback, matching the
  Go banner. See [Install](install.md) for the expected output.

- **Fail-closed messaging:** When a backend cannot be initialized, Warden
  prints `✗ Warden refused to start: sandbox backend unavailable` with the
  specific reason and `fails closed by design — it will never run your
  MCP server without a working sandbox`. This is not just UI — it reflects
  the `sandboxerr.RefuseToRun` error that prevents any unsandboxed fallback.

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

**Run output:** `warden run` never prints the large banner — it prioritizes
runtime information. When stderr is a TTY (and `CI` is not set and
`WARDEN_QUIET` is empty) it prints a compact pre-launch summary to stderr
only (so MCP stdio is untouched):

```
WARDEN
──────────────────────────────────────
Policy     policy.yaml
Backend    linux
Command    /usr/bin/node server.js

Filesystem
  ✓ /workspace/project (read)
  ✗ everything else

Network
  ✓ api.github.com
  ✗ everything else

Environment
  ✓ GITHUB_TOKEN
  ✗ all unspecified variables

──────────────────────────────────────
✓ Sandbox active
```

Only information from the loaded policy is shown — no permissions are
invented. In CI / piped / `WARDEN_QUIET=1` the summary is suppressed so
`warden run` stays silent apart from the sandboxed process's own stdio.
Security events are recorded to the audit log (`warden logs`) as structured
`allowed`/`blocked` JSON; the CLI formats them as `✓ ALLOWED` / `✗ BLOCKED`
when displayed (color reinforces but never replaces the text marker).

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

## `warden doctor`

Check that the host can actually enforce a sandbox. It probes the real
backends (bubblewrap/Seatbelt/AppContainer/Docker), `strace` availability on
Linux, the egress proxy, and the policy engine, and prints a
security-oriented report:

```bash
warden doctor
```

Example output (TTY):

```
WARDEN DOCTOR

Environment
────────────────────────────────
✓ Operating system        linux/amd64
✓ Sandbox backend         linux
✓ Namespace support       available (bwrap)
✓ Network proxy           available (egress allowlist)
✓ Policy engine           ready (YAML + validation)
✓ Fail-closed             enabled (never runs unsandboxed)

Security posture
────────────────────────────────
✓ Filesystem isolation    explicit paths only
✓ Environment filtering   explicit vars only
✓ Network policy          explicit hosts only
✓ Fail-closed behavior    enforced

Status: READY
```

When the backend is unavailable it prints `Status: NOT READY` in red,
the underlying `RefuseToRun` reason, and `Warden fails closed when
sandboxing is unavailable`. Exit code is always `0`; the status line is
the signal for scripts.

## `warden version`

Print the stamped build version and exit 0. This stays script-friendly and
never prints a banner:

```bash
warden version
warden --version
```

Single line `warden version <semver>` (or `dev` for unstamped builds).
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
