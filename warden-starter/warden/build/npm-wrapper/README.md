# warden-sandbox-cli

npm wrapper for the [Warden](https://warden-six-rouge.vercel.app/) sandbox runtime.

Warden runs your MCP servers with explicit filesystem, network and environment grants.

## 0.2.0 preview

This checkout prepares `0.2.0-beta.2`. It is not published by editing this file.
After the preview has been published, install it explicitly:

```bash
npm install -g warden-sandbox-cli@beta
warden --version
```

The preview adds candidate policy packs, reversible MCP host configuration,
filtered stdio and authenticated HTTP connections, creator evidence and local
policy maintenance. Start with `warden packs list`, `warden clients` and
`warden wrap --help`. Preview a configuration change before using `--yes`.
Adapters cover multiple MCP hosts; compatibility still requires a host workflow
check. Local servers require an available sandbox backend. Remote providers'
processes are outside Warden's OS sandbox.

The gateway offers a bounded 2025-11-25 protocol subset during negotiation.
Tasks, subscriptions, sampling, elicitation, roots and 2026-07-28/MRTR are
unsupported. Creator signatures require independently trusted issuer keys and
fresh revocation data; a badge alone does not establish safety.

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

On first run, the wrapper downloads the matching binary and `SHA256SUMS` from
the versioned GitHub release over HTTPS. It verifies SHA-256 before installation
and before every cached launch. Download messages go to stderr so MCP stdout
stays clean. Corruption, unsupported platforms and failed downloads stop launch.
Downloads have size, time and redirect limits; redirects require trusted GitHub
HTTPS hosts. Checksums authenticate downloaded bytes against the release
manifest, not an independent release signing key.

Linux and macOS cache under `~/.cache/warden/`; Windows uses
`%LOCALAPPDATA%\\warden` (or the user's `AppData\\Local\\warden`). Older caches
without a manifest need a one-time online checksum fetch. Verified caches work
offline afterward.

Preview packages use their exact version. Stable packages can use a newer cached
stable binary, including one installed by `warden update`, but never select a
cached preview. For an exact package-version pin or rollback, set
`WARDEN_PIN_VERSION=1`. `warden update` follows stable by default; an explicit
preview update requires `--version 0.2.0-beta.2` after that release exists.

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
