# Install

## Day-one options

> **Note:** Homebrew (`warden-sandbox/warden`) distribution is pending. Today,
> install via npm, GitHub Release, or build from source — all give you the
> same single static binary.

### Option A — npm (easiest)

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

On install, the package runs an anonymous telemetry ping (`telemetry.js`) that
can never fail the install and is skipped with `npm install --ignore-scripts`
(see [Anonymous Installation Telemetry](telemetry.md)). The platform binary is
downloaded lazily on first `warden` invocation from GitHub Releases (5 binaries
+ `SHA256SUMS` per release).

Releases that predate `warden update` do not include that command — upgrade
those installs with:

```bash
npm install -g warden-sandbox-cli@latest
```

### Option B — GitHub Release

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

### Option C — Build from source

Requires **Go 1.22+**:

```bash
git clone https://github.com/Prof-bilal/Warden.git
cd Warden/warden-starter/warden
go build -o warden ./cmd/warden
./warden  # prints usage; exit code is 1 with no subcommand, that's normal
```

## Per-OS prerequisites

The binary alone isn't enough — each platform needs its sandbox primitive:

| OS | Needs | Check |
|---|---|---|
| Linux | `bwrap` (bubblewrap) for the native backend; **`strace` required for `warden run` auditing and `warden trace`** | `command -v bwrap strace` |
| Linux (no bwrap) | Docker daemon — Warden falls back to `--backend docker` | `docker info` |
| macOS | `sandbox-exec` (ships with macOS) preferred; Docker as fallback | `command -v sandbox-exec` |
| Windows | AppContainer support (Windows 10+); fails closed without it | built-in |
| Anywhere without a native backend | Docker daemon | `docker info` |

> **⚠️ Important:** `strace` is required on Linux for the native (bubblewrap)
> backend — Warden uses it for the file/network audit on **every `warden run`**
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

If **no** backend is available, Warden refuses to run — it never silently
falls back to unsandboxed execution. See the [FAQ](faq.md#warden-run-refuses-to-start)
if you hit that error.

## What a successful install looks like

When installed via `npm` (`npm install -g warden-sandbox-cli`) the first
`warden` invocation downloads the platform binary with a polished,
non-blocking progress sequence. In a TTY it animates briefly with a braille
spinner; in CI or when piped it falls back to deterministic bracketed lines
so logs stay clean. No spinner is left behind on exit.

TTY (interactive):

```
██     ██  █████  ██████  ██████  ███████ ███    ██
██     ██ ██   ██ ██   ██ ██   ██ ██      ████   ██
██  █  ██ ███████ ██████  ██   ██ █████   ██ ██  ██
██ ███ ██ ██   ██ ██   ██ ██   ██ ██      ██  ██ ██
 ███ ███  ██   ██ ██   ██ ██████  ███████ ██   ████
              MCP SERVER SANDBOX

        Secure execution for MCP servers

  Installing Warden...

  ✓ Checking platform  (linux/amd64)
  ✓ Installing runtime  (warden-linux-amd64)
  ✓ Installing CLI
  ✓ Verifying installation

  ─────────────────────────────────────

  ✓ Warden v0.1.0 installed successfully.

  Get started:

    warden init
    warden run --policy policy.yaml -- <server>
    warden doctor  — check sandbox readiness

  Security:
    Warden fails closed when sandboxing is unavailable.
```

CI / non-TTY fallback:

```
WARDEN — MCP Server Sandbox  v0.1.0
[1/4] Checking platform... OK
[2/4] Installing runtime... OK
[3/4] Installing CLI... OK
[4/4] Verifying installation... OK

Warden v0.1.0 installed successfully.
```

The same fail-closed guarantee applies: if the platform is unsupported or
the binary cannot be downloaded/verified, the installer exits non-zero
and prints `You can manually install from: https://github.com/Prof-bilal/Warden/releases`
without leaving a half-installed state. Colors respect `NO_COLOR` and
`TERM=dumb`; set `WARDEN_NO_UNICODE=1` for ASCII fallbacks.

First-run after install, `warden` (no args) shows a one-time welcome with
the banner, capabilities, and next steps (`warden init` / `warden doctor`).
It is never shown in CI, never shown for `warden run`, and can be disabled
with `WARDEN_NO_FIRST_RUN=1`. See [CLI Reference](cli.md#cli-experience).

## Verify your install

From the repo checkout, run a no-op command under the strictest fixture
policy (the `time` server needs nothing at all — adjust the binary path to
your OS, e.g. `/bin/true` on macOS):

```bash
warden run --policy testdata/compat/time/policy.yaml -- /usr/bin/true
echo "exit: $?"
warden logs --tail 5
warden doctor   # shows Environment + Security posture + Status: READY
```

Next: [Quickstart](quickstart.md).
