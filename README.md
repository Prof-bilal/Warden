<p align="center">
  <img src="warden-landing/public/warden-logo.svg" width="120" alt="Warden Logo">
</p>

<h1 align="center">Warden</h1>

<p align="center">
  <strong>Sandbox runtime for MCP servers</strong>
</p>

<p align="center">
  <a href="https://www.npmjs.com/package/warden-sandbox-cli">
    <img src="https://img.shields.io/npm/v/warden-sandbox-cli?style=flat-square&color=blue" alt="npm version">
  </a>
  <a href="https://www.npmjs.com/package/warden-sandbox-cli">
    <img src="https://img.shields.io/npm/dm/warden-sandbox-cli?style=flat-square&color=green" alt="npm downloads">
  </a>
  <a href="https://github.com/Prof-bilal/Warden/blob/main/LICENSE">
    <img src="https://img.shields.io/npm/l/warden-sandbox-cli?style=flat-square" alt="license">
  </a>
  <a href="https://github.com/Prof-bilal/Warden/releases">
    <img src="https://img.shields.io/github/v/release/Prof-bilal/Warden?style=flat-square&color=orange" alt="GitHub release">
  </a>
</p>

<p align="center">
  MCP servers run with full access to your machine. Warden runs them in a sandbox so they only see what you grant.
</p>

---

## Why Warden?

MCP servers (Claude Desktop, Cursor, VS Code Copilot) run as plain processes with **full access** to your filesystem, network, and environment variables. That MCP server you just installed from GitHub? It can read your SSH keys, access your AWS credentials, and connect to any host.

**Warden fixes this** by running MCP servers in OS-native sandboxes with deny-by-default access control.

## Quick Start

```bash
# Install
npm install -g warden-sandbox-cli

# Create a policy
cat > policy.yaml << 'EOF'
command: ["node", "server.js"]
filesystem:
  read: ["./data"]
  write: ["./output"]
network:
  allow: ["api.github.com"]
env:
  allow: ["GITHUB_TOKEN"]
EOF

# Run sandboxed
warden run --policy policy.yaml -- node server.js
```

The server only sees `./data` (read), `./output` (write), `api.github.com` (network), and `GITHUB_TOKEN` (env). Everything else is **invisible**.

## How It Works

```
┌─────────────────────────────────────────────────┐
│                  Your Machine                    │
├─────────────────────────────────────────────────┤
│                                                  │
│  ┌──────────────┐      ┌──────────────────┐     │
│  │  MCP Client  │──────│  Warden Sandbox  │     │
│  │  (Claude)    │ stdio│                  │     │
│  └──────────────┘      │  ┌────────────┐  │     │
│                        │  │ MCP Server │  │     │
│                        │  └────────────┘  │     │
│                        │                  │     │
│                        │  ✓ ./data (read) │     │
│                        │  ✓ api.github.com│     │
│                        │  ✗ everything else│     │
│                        └──────────────────┘     │
│                                                  │
└─────────────────────────────────────────────────┘
```

## Features

| Feature | Description |
|---------|-------------|
| **Deny by default** | Paths don't exist unless granted — not "permission denied" |
| **OS-native** | bubblewrap (Linux), Seatbelt (macOS), AppContainer (Windows) |
| **No Docker required** | Native sandboxing first, Docker only as fallback |
| **Auto-generate policies** | `warden trace` + `warden init` watches and generates policies |
| **Audit trail** | Logs every blocked access attempt |
| **Interactive approval** | `--approve` mode prompts on first blocked access |

## Commands

```bash
warden run --policy policy.yaml -- node server.js    # Run sandboxed
warden trace -- node server.js                       # Record access patterns
warden init                                          # Generate policy from trace
warden logs                                          # View audit log
warden doctor                                        # Check sandbox readiness
warden update                                        # Update to latest version
```

## Platform Support

| OS | Backend | Status |
|----|---------|--------|
| Linux | bubblewrap | ✅ Verified |
| macOS | Seatbelt | ✅ CI verified |
| Windows | AppContainer + WFP | ✅ CI verified |
| Any | Docker (fallback) | ✅ Works |

## Install Options

```bash
# npm (recommended)
npm install -g warden-sandbox-cli

# Manual download
# https://github.com/Prof-bilal/Warden/releases

# From source
go build -o warden ./cmd/warden
```

## Documentation

- [Install Guide](warden-starter/warden/docs/install.md)
- [Schema Reference](warden-starter/warden/docs/schema.md)
- [Example Policies](warden-starter/warden/examples/)
- [Architecture](ARCHITECTURE.md)

## Contributing

See [CONTRIBUTING.md](warden-starter/warden/CONTRIBUTING.md) for development setup and guidelines.

## License

MIT
