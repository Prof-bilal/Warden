# Install

## Day-one options

> **Note:** Homebrew (`warden-sandbox/warden`) distribution is pending. Today,
> install via npm, GitHub Release, or build from sourceall give you the
> same single static binary.

### Option Anpm (easiest)

Warden is a **global CLI**. Install it with `-g` so the `warden` command is
on your `PATH`:

```bash
npm install -g warden-sandbox-cli
warden --version
warden help
```

`npm install -g warden-sandbox-cli` installs the `warden` CLI globally.

A local (project) install is an advanced/development option and does **not**
put `warden` on your global shell `PATH`:

```bash
npm install warden-sandbox-cli          # local only
npx warden --version                    # use via npx
# warden --version                      # will fail: command not found
```

The platform binary is
downloaded lazily on the first `warden` invocation: the npm launcher prints
`Downloading warden v<version>...`, fetches the matching GitHub Release
binary (5 binaries + `SHA256SUMS` per release) into
`~/.cache/warden/<version>/`, and then runs it. Later invocations reuse the
cached binary. Checksum verification against the published `SHA256SUMS` is
performed by `warden update` when upgrading.

Releases that predate `warden update` do not include that commandupgrade
those installs with:

```bash
npm install -g warden-sandbox-cli@latest
```

### Option BGitHub Release

Download the binary for your platform from
[Releases](https://github.com/Prof-bilal/Warden/releases), verify the
checksum in `SHA256SUMS`, and put it on your `PATH`:

```bash
# Linux example
curl -LO https://github.com/Prof-bilal/Warden/releases/latest/download/warden-linux-amd64
sha256sum -c SHA256SUMS  # from the same release page
chmod +x warden-linux-amd64
sudo mv warden-linux-amd64 /usr/local/bin/warden
warden  # prints usage; exit code is 1 with no subcommand, that's normal
```

### Option CBuild from source

Requires **Go 1.24+** (see `go.mod`):

```bash
git clone https://github.com/Prof-bilal/Warden.git
cd Warden/warden-starter/warden
go build -o warden ./cmd/warden
./warden  # prints usage; exit code is 1 with no subcommand, that's normal
```

## Per-OS prerequisites

The binary alone isn't enougheach platform needs its sandbox primitive:

| OS | Needs | Check |
|---|---|---|
| Linux | `bwrap` (bubblewrap) for the native backend; **`strace` required for `warden run` auditing and `warden trace`** | `command -v bwrap strace` |
| Linux (no bwrap) | Docker daemonWarden falls back to `--backend docker` | `docker info` |
| macOS | `sandbox-exec` (ships with macOS) preferred; Docker as fallback | `command -v sandbox-exec` |
| Windows | AppContainer support (Windows 10+); fails closed without it | built-in |
| Anywhere without a native backend | Docker daemon | `docker info` |

> **⚠️ Important:** `strace` is required on Linux for the native (bubblewrap)
> backendWarden uses it for the file/network audit on **every `warden run`**
> and for `warden trace`. Without it, `warden run` refuses to start (fail-closed):
> ```
> warden run: strace not found: required for complete file/network auditing on Linux
> ```
> The Docker fallback does not require `strace` (`--backend docker`).

Install dependencies on common distros:

```bash
sudo apt install bubblewrap strace      # Debian/Ubuntu
sudo dnf install bubblewrap strace      # Fedora
sudo pacman -S bubblewrap strace        # Arch
```

If **no** backend is available, Warden refuses to runit never silently
falls back to unsandboxed execution. See the [FAQ](faq.md#warden-run-refuses-to-start)
if you hit that error.

## What a successful install looks like

`npm install -g warden-sandbox-cli` itself prints only npm's standard
outputthere is no Warden banner or progress screen at install time. The
binary is fetched on the first `warden` invocation:

```
$ warden --version
Downloading warden v0.1.17...
warden version 0.1.17
```

Bare `warden` (no subcommand) then shows usage and exits 1. In an
interactive terminal it starts with the large ASCII banner:

```
██     ██  █████  ██████  ██████  ███████ ███    ██
██     ██ ██   ██ ██   ██ ██   ██ ██      ████   ██
██  █  ██ ███████ ██████  ██   ██ █████   ██ ██  ██
██ ███ ██ ██   ██ ██   ██ ██   ██ ██      ██  ██ ██
 ███ ███  ██   ██ ██   ██ ██████  ███████ ██   ████
               MCP SERVER SANDBOX

WARDEN
Secure execution for MCP servers.

Usage:
  warden <command> [options]

Commands:
  init       Create a security policy
  run        Run an MCP server in the sandbox
  ...

Get started:

  warden init
  warden run --policy policy.yaml -- <server>
  ...
```

When output is piped or `CI` is set, the large banner is omitted and only
the compact `WARDEN` header + usage is printed. Colors respect `NO_COLOR`
and `TERM=dumb`; set `WARDEN_NO_UNICODE=1` for ASCII fallbacks.

First-run after install, `warden` (no args) in an interactive terminal
shows a one-time welcome with the banner, capabilities, and next steps
(`warden init` / `warden doctor`). It is never shown in CI, never shown
for `warden run`, and can be disabled with `WARDEN_NO_FIRST_RUN=1`. See
[CLI Reference](cli.md#cli-experience).

## Verify your install

From the repo checkout, run a no-op command under the strictest fixture
policy (the `time` server needs nothing at alladjust the binary path to
your OS, e.g. `/bin/true` on macOS):

```bash
warden run --policy testdata/compat/time/policy.yaml -- /usr/bin/true
echo "exit: $?"
warden logs --tail 5
warden doctor   # shows Environment + Security posture + Status: READY
```

Next: [Quickstart](quickstart.md).
