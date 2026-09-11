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

On every `npm install`, the wrapper also sends one anonymous install ping
(package name, version, OS, CPU arch, Node version — nothing identifying) to
the telemetry endpoint so maintainers can estimate real install activity.
It never prints, never fails the install, and is skipped entirely with
`npm install --ignore-scripts` or `WARDEN_NO_TELEMETRY=1`. See
[Anonymous Installation Telemetry](#anonymous-installation-telemetry).

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

## Anonymous Installation Telemetry

The `postinstall` hook (`telemetry.js`, zero dependencies) sends one minimal
ping per `npm install` so maintainers can estimate real installation activity
beyond the npm download count.

Collected: Warden package version, OS/platform, CPU architecture, Node.js
version, and a server-side timestamp.

Not collected: source code, files, environment variables, secrets, MCP
contents, project paths, usernames, hostnames, or raw IP addresses (the API
never reads or stores them; events are one JSONL line each in a
server-local file, never served back over HTTP).

The ping is fully non-blocking: a 3-second timeout, all errors swallowed
silently, exit code always 0 — a dead endpoint, DNS failure, or 500 can never
break an install. Disable it with `npm install --ignore-scripts` (skips all
lifecycle scripts), `WARDEN_NO_TELEMETRY=1`, `DO_NOT_TRACK=1`, or
`WARDEN_TELEMETRY=0`. The endpoint is one constant (`TELEMETRY_URL`,
overridable via `WARDEN_TELEMETRY_URL`) and defaults to
`https://warden-six-rouge.vercel.app/api/install`.

## License

MIT
