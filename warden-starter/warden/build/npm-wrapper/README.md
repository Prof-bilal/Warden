# warden-sandbox-cli

npm wrapper for the [Warden](https://warden-six-rouge.vercel.app/) sandbox runtime.

Warden runs your MCP servers in a sandbox that only sees what you grant — a folder, a hostname, nothing more.

## Install (global CLI)

Warden is intended to be installed as a **global** CLI. This puts the `warden` command on your `PATH`:

```bash
npm install -g warden-sandbox-cli
```

Then verify:

```bash
warden --version
warden help
```

`npm install -g warden-sandbox-cli` installs the `warden` CLI globally. After a successful global install you can run `warden` from any directory.

### Advanced: local (project) install

A local install is supported for development/CI pin scenarios:

```bash
npm install warden-sandbox-cli
```

A local install does **not** put `warden` on your global shell `PATH`. Use one of:

```bash
npx warden --version
./node_modules/.bin/warden --version
```

Do not expect this sequence to work as a global command:

```bash
npm install warden-sandbox-cli   # local install only
warden --version                # fails: command not found
```

## Usage

```bash
# Run a server under a sandbox policy
warden run --policy policy.yaml -- node ./my-mcp-server/index.js

# Trace what a server accesses (unsandboxed)
warden trace -- node server.js

# Generate a starter policy from a trace log
warden init --log log.json --output policy.yaml

# Check for / apply updates (requires a release that includes `warden update`)
warden update --check
warden update
```

Older releases published before `warden update` existed will not recognize that
command. Upgrade those installs manually:

```bash
npm install -g warden-sandbox-cli@latest
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

- [Homepage](https://warden-six-rouge.vercel.app/)
- [GitHub](https://github.com/Prof-bilal/Warden)
- [Documentation](https://prof-bilal.github.io/Warden/)
- [Issue tracker](https://github.com/Prof-bilal/Warden/issues)

## License

MIT
