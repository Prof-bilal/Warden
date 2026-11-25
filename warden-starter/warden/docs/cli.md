# Warden CLI Reference

Every subcommand, flag, and exit behavior below matches the CLI source
(`cmd/warden`). With no subcommand, `warden` prints usage and exits 1.

## Commands you will use most

| Command | What it does |
|---|---|
| `warden doctor` | Checks whether this machine can run programs in the sandbox |
| `warden grant` | Reviews and adds one host or filesystem read grant |
| `warden guard` | Reviews and installs an npm or PyPI release in a sandbox |
| `warden hook` | Checks Claude Code shell and file tool requests against a policy |
| `warden learn` | Observes a Linux sandbox and reviews a starter policy |
| `warden run --policy <file>` | Runs a program inside the sandbox |
| `warden trace -- <command>` | Runs a program once without the sandbox and records what it touches |
| `warden init` | Writes a draft `policy.yaml` from a trace |
| `warden logs` | Shows allowed and blocked accesses |
| `warden autopilot plan` | Inspect a project before enrolling a Linux agent ([guide](autopilot.md)) |
| `warden agent --policy <file> -- <agent>` | Lets a coding agent work on a sanitized copy of your project ([guide](agent-workspace.md)) |
| `warden add <server> to <client>` | Connects a catalog server to an MCP client through the sandbox |

## All commands

| Command | What it does |
|---|---|
| `agent` | Edit a sanitized workspace with protected secret fields |
| `init` | Create a policy from a trace |
| `run` | Run an MCP server in the sandbox |
| `trace` | Record access attempts for policy generation (unsandboxed) |
| `logs` | Inspect the audit log |
| `doctor` | Check sandbox readiness |
| `grant` | Review and add a host or filesystem read grant |
| `guard` | Review and install an npm package or PyPI wheel |
| `hook` | Check or configure the advisory Claude Code hook |
| `learn` | Observe a Linux sandbox and review a starter policy |
| `migrate` | Preview or apply a version 2 policy migration |
| `validate` | Check policy files without running a command |
| `add` | Connect an MCP server to a client, sandboxed |
| `wrap` / `unwrap` | Preview or apply a client launcher wrapper, and undo it |
| `clients` | List supported client configuration adapters |
| `inventory` | Inspect configured stdio connections and drift |
| `packs` | Browse and generate policy packs |
| `gateway` | Wrap servers registered in a gateway file |
| `stop` | Stop managed gateway connections |
| `proxy` | Run an MCP client proxy with filtering |
| `connect` | Filter a sandboxed stdio MCP connection |
| `serve` | Serve authenticated MCP over HTTP |
| `mcp` | Expose Warden's tools to MCP clients over stdio |
| `policy-diff` | Review policy changes before an upgrade |
| `report` | Record and summarize local pilot events |
| `k8s` | Generate container and Kubernetes manifests from a policy |
| `creator` | Create and verify signed creator evidence |
| `version` | Show the version |
| `update` | Update Warden to the latest release |
| `help` | Show help for a command, for example `warden help run` |

Run `warden help <command>` for the flags of any command. The sections below cover the most common commands in detail.

## `warden run`

Run a server under a policy. The sandbox backend is auto-detected (native
first, Docker fallback, fail-closed when nothing is available).

```bash
warden run [--policy <file>] [--backend auto|linux|seatbelt|windows|docker]
           [--dry-run] [--approve] [--approve-timeout <dur>] -- <command...>
```

| Flag | Meaning |
|---|---|
| `--policy <file>` | Select a policy YAML file explicitly. Relative filesystem grants resolve against the policy file’s directory. |
| `--dry-run` | Print the effective policy, its SHA-256, and the selected backend without launching the command. Use an explicit policy or a local `warden.yaml`. |
| `--backend <name>` | Force a backend. Default `auto`. Aliases: `bwrap` → linux; `macos`, `darwin`, `sandbox-exec` → seatbelt; `appcontainer`, `lowbox` → windows. Unknown or unavailable backends fail closed. |
| `--approve` | Interactive approval: prompt on first out-of-policy access instead of hard-failing. Network approvals apply live; filesystem approvals are saved to the policy and the server restarts. Requires a terminalfails closed without one. |
| `--approve-timeout <dur>` | Per-prompt timeout (e.g. `30s`, `2m`). Zero waits indefinitely; expiry denies. Requires `--approve`. |
| `-- <command...>` | Command to sandbox. Optional separatoreverything after the flags is the command anyway; `--` lets the command itself start with a flag-like token. |

Warden checks the explicit policy, then `warden.yaml` in the working directory,
then a reviewed pack for the exact artifact version. The bare Filesystem
launcher selects the bundled version. For other commands, an interactive
terminal can review a starter policy. Without a matching policy or a terminal,
the command exits before launching.

Command resolution: the CLI tail wins, otherwise the policy's `command:` is
used, otherwise the run fails with a usage error. Bare executable names
(`npx`, `uvx`, `node`) are resolved via `PATH` lookup and fail closed when
unresolvable; prefer absolute paths in checked-in policies. Exit code: the
sandboxed process's exit code, `1` on runtime failure, `2` on usage/policy
errors.

**Run output:** `warden run` never prints the large banner. It prioritizes
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

Only information from the loaded policy is shown. No permissions are
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

Use `warden logs --tail 50 --explain` to see reasons for denied or failed
accesses in the selected entries and which policy section to check.

## `warden learn`

```bash
warden learn --output ./warden.learned.yaml -- ./my-command
```

Run this from the project directory on Linux. The observation reads the project
through a read-only mount with secret-file protection, uses a disposable
`HOME`, and blocks egress. The observation runs for up to 30 seconds. Warden
offers exact observed files as read grants for individual approval and lists
requested hosts in the report for separate review.

The final confirmation saves the policy and its `.report.json` sidecar,
including the SHA-256 digest. Run with that policy and pass the printed digest
to `warden run --policy-sha256` to pin the reviewed bytes.

## `warden guard`

```bash
warden guard npm install is-number@7.0.0 --output ./reviewed-npm
warden guard pip install idna==3.10 --output ./reviewed-wheel
```

Choose a new output directory with an existing parent. The default release
cooldown is 48 hours. User settings come from
`$XDG_CONFIG_HOME/warden/guard.yaml` or `~/.config/warden/guard.yaml`.
Use `--strict` to block missing release metadata, or pass
`--policy ./warden.yaml` to apply the additional restrictions from a
[version 2 guard section](schema.md#guard).
An explicit `--override-age` applies to one invocation and is audited.
For a reviewed npm archive without runtime dependencies or a published
shrinkwrap, Warden downloads and verifies the archive first, then runs npm
offline inside the sandbox. On Windows, installs that require sandboxed
registry access depend on Windows Filtering Platform (WFP).
The pip command installs a verified binary wheel. Review its dependencies
individually.

## `warden grant`

```bash
warden grant host docs.example.org --policy warden.yaml
warden grant path ./data/report.json --policy warden.yaml
warden grant host docs.example.org --policy warden.yaml --for 30m
```

The command previews one exact host or read grant and waits for `y` at the
terminal. The default policy is `./warden.yaml`. Write grants, recognized
credential hosts, and protected secret paths require direct policy review.
Restart the sandbox after saving. Regenerate any policy pin that’s meant to
reference the updated policy.

`--for` saves an expiry instead of permanent access. The native Linux and
Windows backends end the sandbox when its first active temporary permission
expires. Start it again to use the remaining grants.

## `warden hook`

Install an advisory PreToolUse hook in Claude Code 2.1.295 or later.

```bash
warden hook install --policy ./warden.yaml
warden hook uninstall
```

The default scope is the current project's `.claude/settings.local.json`.
Use `--scope user` for `~/.claude/settings.json`, or `--config <file>` for
an explicit settings path. Use the same scope and settings path for uninstall.

The hook checks Bash and PowerShell invocations against the exact literal words
in `command` and applies read, write, and deny rules to the matched file tools.
Use a dedicated policy with the shell command and paths you want to grant.
The check returns exit 2 on a denial or invalid input, leaving Claude Code's
permissions in effect on success. Use `warden run` for OS-level containment
of processes launched outside these advisory checks.

Undo restores the original settings bytes when the installed settings still
match the saved snapshot. If you have since edited the settings, review the
`PreToolUse` groups in the file and remove the entry calling `warden hook check`.
See the [Claude Code hooks reference](https://code.claude.com/docs/en/hooks).

## `warden doctor`

Check that the host can actually enforce a sandbox. It probes the real
backends (bubblewrap/Seatbelt/AppContainer/Docker), `strace` availability on
Linux, the egress proxy, and the policy engine, and prints a
security-oriented report:

```bash
warden doctor
```

Run `warden doctor --prove` on Linux, macOS, or Windows to exercise the native
sandbox with disposable fixtures. It verifies granted reads and writes, then
checks that the sandbox blocks an SSH decoy, an outside write, a network
destination, and an environment sentinel. Linux and macOS check a denial
from the egress proxy for a reserved hostname. Windows checks access to a
live host loopback endpoint and keeps the SSH decoy outside the read grant.
Any failed check or unavailable native backend exits with code 1.

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
sandboxing is unavailable`. The exit code is machine-readable so scripts
and CI can gate on it:

- `0`ready to run
- `1`NOT READY (sandbox backend or a required tool such as `strace` is missing)
- `2`usage error (unexpected arguments)

## `warden migrate`

```bash
warden migrate ./warden.yaml
warden migrate --write ./warden.yaml
warden migrate --client cursor --config ./.cursor/mcp.json ./warden.yaml
```

Preview shows the schema-version and SHA-256 changes and reports MCP launchers
that need review. With no policy paths, Warden checks `./warden.yaml` if it exists.
It scans existing client configs in known user and project locations. Use
`--client` and `--config` together to inspect an explicit config instead.

Apply verifies that the effective policy is identical across the schema change.
It updates inspected `warden run` digest pins that match the original bytes,
including the corresponding undo records. Each changed file gets an owner-only
`.bak` next to it. Keep backups private because they can contain credentials.

Stale pins, unknown fields, and launcher policies whose paths depend on the client
working directory need manual review. An empty MCP allowlist denies every tool.
Legacy stdio proxies require separate review before changing their launcher.

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

## `warden add`

Connect one MCP server to one MCP client in a single command, sandboxed behind
Warden:

```bash
warden add slack to claude-desktop
warden add notion to claude-desktop
warden add @scope/mcp-server@1.2.3 to claude-code \
    --allow-host api.example.com --allow-env SCOPE_KEY
warden add --list-clients
```

The first argument is a catalog pack id (`warden packs list`) or any npm MCP
package (`name[@version]`). `add` prepares the pinned npm release inside the
sandbox (registry-only egress, install scripts disabled), generates a
deny-by-default policy, and registers a wrapped launcher in the client's
config after a preview + confirmation prompt.

| Flag | Meaning |
|---|---|
| `to <client>` / `--client <id>` | Target client adapter: `claude-desktop`, `claude-code`, `cursor`, `codex`, `vscode`, `gemini`, `cline`, `cascade`, `generic`. |
| `--name <name>` | Entry name in the client config (default: pack id / package name). |
| `--scope project\|user` | Config scope (default `project`; `claude-desktop` defaults to `user`). |
| `--config <file>` | Explicit configuration path. |
| `--runtime <dir>` | Prepared runtime directory (default `~/.local/share/warden/runtimes/...`, honors `XDG_DATA_HOME`). |
| `--data <dir>` | Data directory for packs that need one. |
| `--policy <file>` | Use your own reviewed policy; its `command` becomes the upstream. |
| `--allow-host <host>` | Generic npm servers: grant network egress (repeatable). |
| `--allow-env <name>` | Generic npm servers: forward an env var name (repeatable). |
| `--allow-read <path>` / `--allow-write <path>` | Generic npm servers: grant a read/write path (repeatable). |
| `--backend <name>` | Sandbox backend (default `auto`). |
| `--yes` | Apply without the confirmation prompt. |
| `--dry-run` | Preview only; writes nothing, starts no process. |
| `--list-clients` | Show every detected client config and its current entries (read-only; cannot be combined with other arguments). |

Behavior: catalog packs use their reviewed pack grants; any other npm package
gets a deny-by-default starter policy (no network, no env, no paths beyond
the runtime) extended only by the `--allow-*` flags. Credentials are reported
by **name only** and never written into config files. Before writing, `add`
probes the real sandbox with an inert Warden process and refuses to modify
the config if the backend is unavailable. Undo anytime with
`warden unwrap --client <client> --server <entry>`. See `warden help add`.

## `warden gateway`

Wrap gateway-registered servers so the gateway launches each one sandboxed:

```bash
warden gateway init --config <file> --policies <dir>
# Per-server deny-by-default policies (<name>.yaml). Remote (SSE/HTTP)
# servers are skippedno local process to sandbox. Never overwrites.

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
| `3` | For `migrate`, pending changes or findings requiring manual review |

## Terminal output

- **Colors:** Set `NO_COLOR=1` to turn off colors. `TERM=dumb` also turns them off.
- **ASCII only:** Set `WARDEN_NO_UNICODE=1` to replace the check marks with plain text.
- **First-run welcome:** The first interactive run shows a short welcome. Set `WARDEN_NO_FIRST_RUN=1` to hide it. It is never shown in CI or for `warden run`.
- **Refusing to run:** If no sandbox backend is available, `warden run` stops with an error and does not run the program. It never falls back to running it unsandboxed. See the [FAQ](faq.md#warden-run-refuses-to-start).
