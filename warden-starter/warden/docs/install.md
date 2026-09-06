# Install

## Day-one options

> **Note:** Homebrew (`warden-sandbox/warden`) and npm
> (`@warden-sandbox/mcp-warden`) distribution is coming soon. Until then,
> install from a GitHub Release or build from source — both give you the
> same single static binary.

### Option A — GitHub Release (easiest)

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

### Option B — Build from source

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
| Linux | `bwrap` (bubblewrap) for the native backend; `strace` for full file auditing | `command -v bwrap strace` |
| Linux (no bwrap) | Docker daemon — Warden falls back to `--backend docker` | `docker info` |
| macOS | `sandbox-exec` (ships with macOS) preferred; Docker as fallback | `command -v sandbox-exec` |
| Windows | AppContainer support (Windows 10+); fails closed without it | built-in |
| Anywhere without a native backend | Docker daemon | `docker info` |

Install `bwrap` on common distros:

```bash
sudo apt install bubblewrap strace      # Debian/Ubuntu
sudo dnf install bubblewrap strace      # Fedora
sudo pacman -S bubblewrap strace        # Arch
```

If **no** backend is available, Warden refuses to run — it never silently
falls back to unsandboxed execution. See the [FAQ](faq.md#warden-run-refuses-to-start)
if you hit that error.

## Verify your install

From the repo checkout, run a no-op command under the strictest fixture
policy (the `time` server needs nothing at all — adjust the binary path to
your OS, e.g. `/bin/true` on macOS):

```bash
warden run --policy testdata/compat/time/policy.yaml -- /usr/bin/true
echo "exit: $?"
warden logs --tail 5
```

Next: [Quickstart](quickstart.md).
