# 0.2.0-beta.1 release preparation

Verified September 30, 2026. Candidate sources and artifacts are prepared locally;
no npm publish, GitHub release, tag push or website deployment has been performed.
This is a bounded preview, not universal agent or full MCP conformance certification.

## Release changes

- Real `npm pack` replaces the broken hand-built tarball. The workflow verifies
  launcher/runtime inclusion and publishes that exact tarball.
- The candidate is `0.2.0-beta.1`. Prereleases use the beta npm channel and GitHub
  prerelease status; stable tags use latest. Preview launchers pin their version.
- Downloads and cached launches require a matching SHA-256 release manifest.
  Download limits and trusted HTTPS redirects fail closed. Setup logs use stderr;
  signals and child exit codes propagate. Failed cache writes clean temporary files.
- Go self-updates understand prerelease ordering and save the verified manifest
  beside the binary for later offline npm launches. Unused install-hook code was
  removed; the shipped launcher owns download verification.
- MCP initialization offers the supported 2025-11-25 subset when a client requests
  another version, without forwarding unsupported client capabilities. Session
  HTTP requests require the chosen protocol header; JSON content types may include
  a charset. The real Codex check caught the previous initialization rejection.
- Release builds depend on reusable CI, including npm tests on three OSes, website
  and docs builds, real Linux server/Codex scenarios, native gateway lifecycle jobs
  on macOS/Windows, existing backend/attack tests and repository hygiene.
- The compatibility website now records the scoped Codex Linux host result.

## Executed checks

| Check | Result and scope |
| --- | --- |
| Go build, vet, full tests | Passed, in that order, with Go 1.24.13 |
| Race checks | Passed: MCP bridge, updater, attestation and client configuration |
| Native Linux backend | 21 tests passed; zero skips, including startup/escape controls |
| npm launcher | Nine regression tests passed; real six-file npm tarball verified |
| Official filesystem MCP server | `@modelcontextprotocol/server-filesystem@2026.8.31`, actual native sandbox, SDK stdio and HTTP workflows passed |
| Real Codex host | CLI 0.159.2 app-server, both installed CLI and independently installed pinned npm package: startup, filtered discovery, allowed read, OS-denied write/outside read passed |
| Creator and gateway | Raw client and official SDK over both transports; signed checks, independent test-key verification, artifact-drift refusal and stop passed |
| Native gateway lifecycle | Linux fixture: stdio startup/EOF, read/denials, HTTP session deletion and authenticated stop passed |
| Cross-builds | Five stamped assets: Linux amd64/arm64, macOS amd64/arm64, Windows amd64; cross-building does not prove native behavior |
| Website/docs | Next production build passed (91 pages); strict MkDocs build passed |
| Release workflow validation | Both workflows passed actionlint 1.7.7; local artifacts passed SHA-256 checks |

The packed npm scenarios also checked first download with pending MCP input,
offline cached execution, preview channel isolation, corrupted cache refusal,
wrong checksum, truncated HTTPS response and untrusted redirect refusal.
They performed preview/apply/idempotence, policy-change refusal and exact rollback
on a disposable Cursor-shaped configuration. That is not a Cursor GUI workflow.

The release is unpublished, so downloads used a local TLS fixture with a
test-only Node preload that redirected transport. Production code retains fixed
GitHub URLs. The upgrade scenario used current wrapper source packed under a
legacy version, not a historical released launcher. Codex used isolated homes,
an offline provider and explicit MCP calls; no model turn or production credentials.
Creator checks used an ephemeral test issuer, not an enrolled maintainer identity.

## Reproduce on Linux

Install Node, Go, bubblewrap, strace and OpenSSL. From `warden-starter/warden`:

```bash
go build ./...
go vet ./...
go test ./...
go build -ldflags '-X github.com/warden-sandbox/warden/internal/version.Version=0.2.0-beta.1' -o /tmp/warden-preview ./cmd/warden
/tmp/warden-preview packs prepare filesystem --output /tmp/warden-prepared-new --backend linux
WARDEN_BIN=/tmp/warden-preview WARDEN_SMOKE_RUNTIME=/tmp/warden-prepared-new node testdata/ecosystem/smoke-npm-release.mjs > /tmp/warden-npm-report.json
INPUTS="$(node -e "const r=require(process.argv[1]); console.log(require('node:path').join(r.evidenceDirectory,'private','scenario-inputs.json'))" /tmp/warden-npm-report.json)"
WARDEN_SCENARIO_INPUTS="$INPUTS" node testdata/ecosystem/smoke-codex-host.mjs
WARDEN_BIN=/tmp/warden-preview node testdata/ecosystem/smoke-native-gateway.mjs
WARDEN_BIN=/tmp/warden-preview WARDEN_MCP_SDK=/tmp/warden-prepared-new/node_modules/@modelcontextprotocol/sdk node testdata/ecosystem/smoke-growth.mjs
```

Use an unused preparation output path. Codex 0.159.2 must be installed or supplied
through `WARDEN_CODEX_BIN`. The harnesses retain private diagnostic inputs outside
the repository; do not publish their full directories. The CI workflow pins the
Codex npm dependency. Publishing is triggered by a pushed version tag; a manual
Release workflow run builds without publishing.

## Remaining release gates and limits

The edited GitHub workflows have not been executed remotely. Run required CI on
the final commit before publishing. Native macOS/Windows execution and GUI-host
workflows remain unverified here; browser inspection was denied by automatic
approval review in this session, so visual UI QA was not performed. Docker attack
tests remain a remote CI gate because Docker is unavailable in this workspace.

Checksums rely on the HTTPS GitHub release manifest, without independent publisher
signatures. The gateway remains a 2025-11-25 request/response subset: no tasks,
pagination, subscriptions, sampling, elicitation, roots or 2026-07-28/MRTR support.
Dynamic OAuth brokerage, hosted issuer enrollment/revocation and broad host/user
pilots remain separate work. Remote provider processes are not OS-sandboxed.
Candidate pack labels remain conservative; one server workflow does not establish
every pack or every agent's compatibility. All earlier research gates are not complete.

## Feature introduction draft

Warden's 0.2 preview brings candidate policy packs, reversible MCP client setup,
filtered stdio and authenticated HTTP connections, signed creator checks and local
policy maintenance to the CLI and website. Linux workflows have been exercised
with the official filesystem MCP server and Codex CLI. Each host and platform has
its own evidence level, and remote providers' processes remain outside the sandbox.

After publication, install with `npm install -g warden-sandbox-cli@beta`. Start
with `warden packs list`, `warden clients` and the setup guide. Review the policy
and preview configuration changes before applying them with `--yes`.

References used for implementation: [npm packing](https://docs.npmjs.com/cli/v11/commands/npm-pack/),
[npm publishing](https://docs.npmjs.com/cli/v11/commands/npm-publish/),
[MCP lifecycle](https://modelcontextprotocol.io/specification/2025-11-25/basic/lifecycle),
[Codex app-server](https://learn.chatgpt.com/docs/app-server).
