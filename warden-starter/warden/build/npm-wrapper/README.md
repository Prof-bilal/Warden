# warden-sandbox-cli

npm wrapper for the [Warden](https://github.com/Prof-bilal/Warden) sandbox runtime.

Warden runs your MCP servers in a sandbox that only sees what you grant — a folder, a hostname, nothing more.

## Install

```bash
npm install -g warden-sandbox-cli
```

## Usage

```bash
# Run a server under a sandbox policy
warden run --policy policy.yaml -- node ./my-mcp-server/index.js

# Trace what a server accesses (unsandboxed)
warden trace -- node server.js

# Generate a starter policy from a trace
warden init -- log.json --output policy.yaml
```

## How it works

On first run, the wrapper downloads the correct Warden binary for your platform from GitHub Releases and caches it in `~/.cache/warden/`. Subsequent runs use the cached binary directly.

### Supported platforms

| OS | Architecture |
|---|---|
| Linux | amd64, arm64 |
| macOS | amd64, arm64 |
| Windows | amd64 |

## Policy example

```yaml
command: ["node", "server.js"]

filesystem:
  read: ["./data"]
  write: ["./output"]

network:
  allow: ["api.github.com"]

env:
  allow: ["GITHUB_TOKEN"]

limits:
  memory_mb: 512
  timeout_s: 300
```

## Links

- [GitHub](https://github.com/Prof-bilal/Warden)
- [Documentation](https://prof-bilal.github.io/Warden/)
- [Issue tracker](https://github.com/Prof-bilal/Warden/issues)

## License

MIT
