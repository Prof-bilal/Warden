# How to Install Warden

Pick one route. All three give you the same single `warden` binary.

> **Note:** A Homebrew tap is not available yet.

## Choose how to install

### 1. npm (recommended)

Install it globally so the `warden` command is on your `PATH`:

```bash
npm install -g warden-sandbox-cli@beta
warden --version
```

The first `warden` command downloads the binary for your platform from the matching GitHub Release. The download is cached in `~/.cache/warden/<version>/`, so later runs reuse it. `warden update` checks the published `SHA256SUMS` when it upgrades.

Installed from an older release that has no `warden update`? Upgrade with:

```bash
npm install -g warden-sandbox-cli@beta
```

A local (project) install puts `warden` only in that project, not on your global `PATH`. Use it with `npx warden --version` if you need that.

### 2. GitHub Release

Download the binary for your platform from [Releases](https://github.com/Prof-bilal/Warden/releases), check it against `SHA256SUMS`, and put it on your `PATH`. Linux example:

```bash
curl -LO https://github.com/Prof-bilal/Warden/releases/latest/download/warden-linux-amd64
curl -LO https://github.com/Prof-bilal/Warden/releases/latest/download/SHA256SUMS
sha256sum -c SHA256SUMS --ignore-missing
chmod +x warden-linux-amd64
sudo mv warden-linux-amd64 /usr/local/bin/warden
```

Running `warden` with no command prints usage and exits with code 1. That is expected.

### 3. Build from source

You need **Go 1.24 or newer**:

```bash
git clone https://github.com/Prof-bilal/Warden.git
cd Warden/warden-starter/warden
go build -o warden ./cmd/warden
./warden --version
```

## Check what your machine needs

Warden needs a sandbox tool from your operating system. Run this after installing:

```bash
warden doctor
```

It prints the sandbox status and whether this machine is ready. If it says **NOT READY**, fix the missing item it names before you run anything else.

### Per-OS prerequisites

| OS | What it needs | How to check |
|---|---|---|
| Linux | `bwrap` (bubblewrap) and `strace` | `command -v bwrap strace` |
| Linux without bwrap | Docker daemon. Warden falls back to `--backend docker` | `docker info` |
| macOS | `sandbox-exec` (included with macOS). Docker is the fallback | `command -v sandbox-exec` |
| Windows | AppContainer support (Windows 10 or newer) | Built in |
| Any other system | Docker daemon | `docker info` |

On Linux, `strace` is required for the native backend. Warden uses it for file and network auditing on every `warden run` and for `warden trace`. Without it, `warden run` refuses to start:

```
warden run: strace not found: required for complete file/network auditing on Linux
```

The Docker fallback does not need `strace`.

Install the Linux packages on common distros:

```bash
sudo apt install bubblewrap strace      # Debian/Ubuntu
sudo dnf install bubblewrap strace      # Fedora
sudo pacman -S bubblewrap strace        # Arch
```

If no sandbox backend is available, Warden refuses to run. It never falls back to running a program unsandboxed. See the [FAQ](faq.md#warden-run-refuses-to-start) if you hit that error.

## Confirm it works

Run the doctor check again:

```bash
warden doctor
```

You should see the sandbox status and **READY**. Then continue with the [Quickstart](quickstart.md), which walks through a first sandboxed run.
