# Warden

**A lightweight sandbox runtime for MCP servers.**

MCP servers routinely run as a plain Node or Python process on your machine
with full filesystem and network access — even ones you just cloned from
GitHub five minutes ago. Warden runs them in a restricted sandbox instead,
so a server only ever gets the files, network hosts, and environment
variables you explicitly grant it.

> **Status:** All backends implemented (Linux, macOS, Windows); npm and GitHub
> Releases distribution is live. Verification state: Linux verified on real
> hardware, Windows verified via CI escape tests, macOS CI-green and pending a
> real-hardware harness run. See
> [warden-starter/warden/TESTING.md](./warden-starter/warden/TESTING.md) and
> [warden-starter/warden/REMAINING_WORK.md](./warden-starter/warden/REMAINING_WORK.md)
> for the exact state.

---

## Project Structure

```
warden/
├── warden-starter/warden/    # Warden CLI — Go backend (the actual sandbox runtime)
├── warden-landing/           # Landing page — Next.js 14 + TypeScript + Tailwind
├── docs/                     # Container/k8s mode, MCP client proxy
├── examples/                 # Example policies (container, MCP proxy)
├── testdata/proof/           # Proof-harness target fixture
├── test-mcp-proxy.sh         # MCP proxy acceptance script
├── ARCHITECTURE.md           # System architecture and design
├── ROADMAP.md                # Milestones and timeline
└── README.md                 # This file
```

---

## Getting Started

### 1. Warden CLI (Go backend)

The core sandbox runtime. Builds a single static binary.

**Prerequisites:**
- Go 1.22+
- `bubblewrap` (Linux), `sandbox-exec` (macOS), or Docker (fallback)
- `strace` (Linux native backend — required by `warden run` for file/network auditing)

> **Module path note:** the Go module is declared as
> `github.com/warden-sandbox/warden`, but the repository lives at
> `github.com/Prof-bilal/Warden`. Until the paths are aligned, `go install
> github.com/warden-sandbox/warden/cmd/warden@latest` will **not** resolve —
> install via npm (`npm install -g warden-sandbox-cli`) or build from source
> below.

```bash
cd warden-starter/warden

# Build
make build          # produces ./warden binary

# Run tests
make test

# Build for all platforms
make build-all      # outputs to dist/

# Development
make vet            # static analysis
make fmt            # format code
```

**Quick usage after build:**

```bash
# Run a server under a policy
./warden run --policy policy.yaml -- node server.js

# Generate a starter policy by tracing
./warden trace -- node server.js
./warden init

# Check sandbox readiness
./warden doctor
```

See [warden-starter/warden/README.md](./warden-starter/warden/README.md) for
full CLI reference, [docs/](./warden-starter/warden/docs/) for detailed guides,
and [examples/](./warden-starter/warden/examples/) for sample policies.

---

### 2. Warden Landing Page (Next.js frontend)

The marketing/landing site deployed to Vercel.

**Prerequisites:**
- Node.js 18+

```bash
cd warden-landing

# Install dependencies
npm install

# Start dev server (http://localhost:3000)
npm run dev

# Build for production
npm run build

# Lint
npm run lint
```

See [warden-landing/README.md](./warden-landing/README.md) for deployment notes.

---

## How Warden Works

See [ARCHITECTURE.md](./ARCHITECTURE.md) for the full design. In short:
Warden is a thin CLI over OS-native sandboxing primitives — [bubblewrap](https://github.com/containers/bubblewrap)
on Linux, `sandbox-exec`/Seatbelt on macOS (with a Docker fallback), and a
policy engine that translates a simple YAML file into the low-level
namespace/seccomp/network rules each platform actually needs.

## Why not just use Docker?

You can, and Warden's macOS fallback does. But Docker is heavyweight for
"run one npm script with a restricted home directory" — slow cold starts,
a daemon dependency, and a much bigger trust boundary than a namespace
sandbox needs. Warden aims to be a single static binary with near-zero
overhead, so sandboxing an MCP server is no harder than running it.

## Prerequisites

### Linux

`strace` is required on Linux for the native (bubblewrap) backend: Warden
uses it **both** for `warden trace` (policy generation) **and** for the
file/network audit on every `warden run` — without it, `warden run` refuses
to start (fail-closed). The Docker fallback (when `bwrap` is unavailable)
does not require `strace`.

```bash
# Arch Linux
sudo pacman -S strace

# Ubuntu/Debian
sudo apt install strace

# Fedora
sudo dnf install strace
```

Other Linux dependencies:
- `bubblewrap` — sandbox backend
- `Node.js` or `Python` — to run MCP servers

### macOS

- `sandbox-exec` (built-in) — sandbox backend
- Xcode Command Line Tools: `xcode-select --install`

### Windows

- `AppContainer` support (Windows 10+)
- Elevated shell (Administrator) for network filtering

## Contributing

See [warden-starter/warden/CONTRIBUTING.md](./warden-starter/warden/CONTRIBUTING.md)
for how to get involved, and
[warden-starter/warden/REMAINING_WORK.md](./warden-starter/warden/REMAINING_WORK.md)
for current priorities. Good first contributions: example policies for new
MCP servers, testing the backends on your platform, or improving the docs.

## License

MIT
