# About Warden

## Mission

Modern AI tooling runs third-party code with first-party trust. MCP servers
are installed with a curl pipe or an `npx` one-liner and immediately inherit
everything the user can do. Warden exists to make the safe path the easy
path: **sandbox every MCP server by default, with a policy file small enough
to read in one sitting.**

## Why not just use Docker?

You can — and Warden uses it as a fallback. But Docker is heavyweight for
"run one script with a restricted home directory": slow cold starts, a daemon
dependency, and a far bigger trust boundary than a namespace sandbox needs.
Warden is a single static binary over OS-native primitives
([bubblewrap](https://github.com/containers/bubblewrap) on Linux,
`sandbox-exec`/Seatbelt on macOS with a Docker fallback, AppContainer + WFP
on Windows), so sandboxing a server costs almost nothing.

## Project status

Warden is in **beta**. Implemented and tested:

| Area | State |
|---|---|
| Linux sandbox (bubblewrap) | ✅ Filesystem, network egress proxy, audit, limits |
| macOS sandbox (Seatbelt) | ✅ With Docker fallback |
| Windows sandbox (AppContainer/WFP) | ✅ Fail-closed, no unsandboxed fallback |
| `trace` / `init` / `logs` | ✅ Observe, generate, inspect |
| Approval mode (`--approve`) | ✅ Prompt instead of hard-fail |
| Gateway integration | ✅ Wrap gateway-registered servers |
| Compatibility matrix | ✅ 18 servers, 14 pass — see [Compatibility](compatibility.md) |

Coming soon: Homebrew tap, npm wrapper publishing, and the external beta
program ([join it](beta.md)). Track milestones in
[ROADMAP.md](https://github.com/Prof-bilal/Warden/blob/main/warden-starter/warden/ROADMAP.md).

## Security posture

Deny-by-default on filesystem, network, and environment. No silent fallback
to unsandboxed runs — a missing backend fails loudly. Every blocked access
is logged. Known limitations (no CPU throttling, no wildcard hosts,
unix-socket grants, Seatbelt deprecation) are documented honestly in the
[Security Review](security.md) and [Compatibility](compatibility.md) pages —
not buried.

## License and links

- **License:** MIT — see
  [LICENSE](https://github.com/Prof-bilal/Warden/blob/main/warden-starter/warden/LICENSE)
- **Repository:**
  [Prof-bilal/Warden](https://github.com/Prof-bilal/Warden)
- **Contributing:** see
  [CONTRIBUTING.md](https://github.com/Prof-bilal/Warden/blob/main/warden-starter/warden/CONTRIBUTING.md)
- **Compatibility reports:** file one via the [Beta Program](beta.md)
