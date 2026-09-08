# WARDEN-FACTS.md

**Purpose:** Verified facts only. Every claim below was cross-checked against
source code, tests, configuration, or an externally verifiable registry on
2026-09-08. Anything that could not be verified is marked `UNVERIFIED`.
Nothing here is inferred from marketing copy.

---

## 1. Identity

- Name: **Warden** — "a lightweight sandbox runtime for MCP servers"
  (`warden-starter/warden/README.md:3`).
- Repository: `https://github.com/Prof-bilal/Warden` (git remote `origin`,
  verified in workspace config).
- Go module: `module github.com/warden-sandbox/warden` / `go 1.22`
  (`warden-starter/warden/go.mod`).
- Language/toolchain: Go 1.22.12 (Wingo toolchain), verified by building and
  running the test suite locally.
- License: MIT (`warden-starter/warden/LICENSE`, footer states MIT).
- Version stamping source: `internal/version/version.go`; released binaries
  v0.1.3–v0.1.7 (GitHub Releases API), npm `warden-sandbox-cli` 0.1.0–0.1.9
  published (npm registry). Source-build stamps read `dev` when built
  unstamped (verified: `go build -o /tmp/warden-src ./cmd/warden` →
  `warden version dev`).

## 2. What is built (implemented in source)

| Component | Status | Evidence |
|---|---|---|
| CLI (`run/trace/init/logs/doctor/gateway/version/help`) | VERIFIED | `cmd/warden/main.go`; built and executed locally |
| Policy engine (parse/validate/normalize/resolve) | VERIFIED | `internal/policy/policy.go`, `grants.go`; `policy_test.go` passes |
| Linux backend (bubblewrap) | VERIFIED (code + escape tests pass on this host) | `internal/sandbox/linux/linux.go`; 7 integration tests PASS locally |
| macOS backend (sandbox-exec / Seatbelt) | VERIFIED (code + unit tests only) | `internal/sandbox/darwin/profile.go`, `run.go` |
| Windows backend (AppContainer/WFP/ETW/Job) | PARTIALLY VERIFIED | `internal/sandbox/windows/*` (~2,900 lines); unit/plan tests pass locally; escape tests require a Windows host/CI |
| Docker fallback backend | VERIFIED with CURRENT FAILING TEST | `internal/sandbox/docker/docker.go`; CLI reproduction works; `TestDockerBlocksUngrantedRead` fails under `go test` on this host |
| Egress proxy (hostname allowlist, pre-DNS) | VERIFIED | `internal/proxy/proxy.go`, `bridge.go`; `proxy_test.go` passes |
| Environment filtering (deny-by-default) | VERIFIED | `internal/envfilter/envfilter.go`; escape test asserts no leak |
| Audit log (JSONL, 0600) | VERIFIED | `internal/audit/audit.go`; Linux strace importer `internal/audit/strace.go` |
| Resource limits (memory RSS-sampled, wall-clock) | VERIFIED (Linux code + tests) | `internal/sandbox/linux/limits.go`; `limits_test.go`, `darwin/run.go` |
| Interactive approval mode (`--approve`) | VERIFIED | `internal/approve/*`, backend wiring; `approve_test.go` |
| Gateway integration (init/run/wrap/list) | VERIFIED | `cmd/warden/gateway.go`, `internal/gateway/*`; `gateway_test.go` |
| Compatibility matrix fixtures (18 servers) | VERIFIED (fixtures + manifest enforced by tests) | `testdata/compat/*` (18 `policy.yaml`), `matrix.yaml`, `internal/compat/compat_test.go` |
| Backend auto-selection & fail-closed refusal | VERIFIED | `internal/sandbox/backend.go`, `internal/sandbox/sandboxerr/sandboxerr.go` |
| Banners/first-run/doctor UX, `--version` | VERIFIED | `internal/ui/*`, `cmd/warden/main.go`; built and executed |

## 3. Test suite status (executed on 2026-09-08, Arch Linux x86_64)

- Command: `go build ./...` → OK. `go test -count=1 -v ./...` → **256 tests
  run, 255 PASS, 1 FAIL, 0 SKIP.**
- Passing packages (12 of 13): `cmd/warden`, `internal/approve`, `internal/audit`,
  `internal/compat`, `internal/envfilter`, `internal/gateway`, `internal/policy`,
  `internal/proxy`, `internal/sandbox`, `internal/sandbox/linux`,
  `internal/sandbox/windows`, `internal/ui`.
- Failing package: `internal/sandbox/docker` —
  `TestDockerBlocksUngrantedRead` (integration_test.go:50). Docker daemon is UP
  and image `alpine:3.20` present on this host. A faithful manual reproduction
  of the same policy through the real `warden` CLI **blocks correctly** (exit 1),
  so the failure is specific to the test's harness path and needs triage before
  the Docker fallback can be called fully verified.
- Linux escape tests that RUN and PASS on this host (bwrap installed):
  `TestSandboxReadGrantAccessible`, `TestSandboxUnlistedPathInvisible`,
  `TestSandboxWriteGrantWritable`, `TestSandboxWriteOutsideGrantDenied`,
  `TestSandboxExitCodePropagated`, `TestSandboxEnvPassthrough`,
  `TestRunFailsLoudWithoutBwrap`, plus unit tests for bwrap arg building,
  limits (timeout, memory), env filter.
- Windows escape tests exist but execute only on Windows (`//go:build windows`);
  the CI workflow asserts they run (`<root>/.github/workflows/ci.yml` windows job).
  On this Linux host the windows package executes only cross-platform tests
## 4. Backends — verified behavior

### Linux (bubblewrap)
- Builds bwrap args: `--unshare-user --unshare-ipc --unshare-pid --unshare-net
  --uid 0 --gid 0`, `--dev /dev --proc /proc --tmpfs /tmp`, ro-bind `/usr`
  and `/lib64` (merged-/usr fix), ro-bind `filesystem.read`, rw-bind
  `filesystem.write`, command parent dir auto bound ro
  (`internal/sandbox/linux/linux.go:40–100`).
- Env: only `env.allow` names forwarded, empty allowlist = empty env
  (`envfilter.go` + `TestSandboxEnvPassthrough`).
- Limits: RSS of launcher + descendants sampled every 25 ms →
  SIGTERM then SIGKILL after 750 ms; wall-clock timer
  (`internal/sandbox/linux/limits.go`, `darwin/run.go:155–204`).
- Fails loudly when bwrap is missing (`TestRunFailsLoudWithoutBwrap`).
- **Documented gap** (`docs/security.md:60–65`): if `--unshare-net` cannot be
  applied because unprivileged user namespaces are disabled, the fallback may
  run with host network access rather than failing closed — the doc itself says
  Warden "currently does not always detect it."

### Network (all backends)
- Hostname allowlist enforced by an in-process HTTP proxy; DNS resolution
  happens only AFTER the allowlist check (`proxy.go:1–4`, `security.md:58–59`).
- On Linux, the sandbox network namespace has no default route; the proxy is
  injected via `HTTP_PROXY`/`HTTPS_PROXY`/`ALL_PROXY`/`NO_PROXY=`.
- Windows: WFP sublayer hard-permits only loopback to the local proxy and
  hard-blocks other outbound IPv4/IPv6 including direct DNS (`plan.go:10–12`,
  `wfp.go`).
- **Documented limitation** (`security.md:152`): the proxy filters HTTP/HTTPS
  only; raw TCP/UDP is not proxied. Blocking raw traffic relies on the network
  namespace (Linux/macOS) or WFP filters (Windows).
- Docker backend: `--network none` (`docker.go:95`).

### Filesystem
- Linux/macOS/Docker: granted paths only; everything else invisible
  (bind mount) not permission-denied (`linux.go`, `docs/security.md:16–17`).
- Windows: policy-derived capability SIDs + DACL ACE grants on granted paths;
  token boundary denies everything else (`appcontainer.go`,
  `plan.go:4–9`). Runtime read paths (`C:\Windows\System32` etc.) always
  readable, never writable; system roots never writable (`plan.go:42–59`).
- **Documented gap** (`security.md:42–45`): symlinks inside a granted path are
  not resolved/blocked.

### Environment
- Only `env.allow` names are forwarded; values come from the parent
  environment at launch, never stored in the policy file
  (`policy.go:37–40`, `envfilter.go`).
- Gateway env templates: `${NAME}` (required) and `${NAME:-default}` (fallback)
  (`docs/gateway.md:56–66`).

### Fail-closed
- No backend available → `RefuseToRun` error: e.g. "no sandbox backend is
  available … refusing to run unsandboxed" (`backend.go:82,90,99,104`,
  `sandboxerr.go`). The Linux integration test `TestRunFailsLoudWithoutBwrap`
  asserts the error text contains "fails closed by design".
- Windows: every enforcement layer (token, DACL grants, WFP, Job Object,
  ETW session) must initialize or the run refuses (`windows/run.go:55–136`,
  `plan.go:18–19`): "A plain CreateProcess fallback is never acceptable."
- Windows WFP DLL missing → `wfpSupported()` returns fail-closed `errWFPDLLMissing`
  (`REMAINING_WORK.md:60–68`, `internal/sandbox/windows/integration_test.go:181–197`).
- ETW: if the trace session cannot start (e.g. another controller holds the
  kernel providers), the run refuses — auditing is part of the security gate
  (`ROADMAP.md` M5 completion section, `REMAINING_WORK.md`).
## 5. CLI — verified behavior

- `warden` (no args): branded usage, exit 1 (source HEAD also prints a large
  `WARDEN` banner on a TTY).
- `warden --help` / `warden help`: usage; `warden <cmd> --help`: per-command help
  (source HEAD has polished help system, commit `a3672ca`).
- `warden --version` / `warden version` / `-v` / `-version`: prints stamped
  build version; `dev` if unstamped (`cmd/warden/main.go:84–89`).
- `warden run --policy <file> [--backend auto|linux|seatbelt|windows|docker]
  [--approve] [--approve-timeout <dur>] -- <cmd...>`
- `warden trace -- <cmd...>` (unsandboxed, strace-instrumented)
- `warden init [--log <file>] [--output <file>] [-- <cmd...>]`
- `warden logs [--tail <n>] [--follow] [--log <file>]`
- `warden doctor` (backend/namespace/proxy/policy readiness)
- `warden gateway init|run|wrap|list`
- Internal `warden __proxy-bridge` (not user-facing).
- Color/unicode controls: `NO_COLOR`, `TERM=dumb`, `WARDEN_NO_UNICODE=1`,
  `FORCE_COLOR`, `CI` detection, `WARDEN_NO_SPINNER` (`docs/cli.md` CLI
  Experience section).

## 6. Compatibility matrix — verified evidence

- `testdata/compat/matrix.yaml` lists 18 servers: **14 pass, 2 conditional
  (fetch, kubernetes), 2 fail (docker — inherent; playwright — schema-gap)**.
- 18 matching fixture `policy.yaml` files exist under `testdata/compat/`.
- `internal/compat/compat_test.go` enforces that every manifest entry has a
  fixture and that fixture grants match the manifest probes, using the policy
  engine only (no bwrap/network) — verified passing.
- `compat.go:8–9`: "End-to-end backend runs remain manual; see
  testdata/compat/README.md." → The "verified against real servers" verdicts
  come from documented manual/internal runs, not from an automated live-server
  test in CI.
- Failure classification for docker-mcp: granting the Docker daemon socket
  would void the sandbox (inherent). Playwright: no wildcard hosts and no
  unix-socket grant in the schema (schema gap).
- Evidence: `testdata/compat/matrix.yaml`, `docs/compatibility.md`.

## 7. Distribution — verified

- GitHub Releases: `v0.1.3` … `v0.1.7` (API verified 2026-09-08; latest
  `v0.1.7` published 2026-09-08T03:01:34Z). Assets: 5 binaries
  (`warden-{darwin,linux}-{amd64,arm64}`, `warden-windows-amd64.exe`) +
  `SHA256SUMS`.
- npm: `warden-sandbox-cli` 0.1.0–0.1.9 published (npm registry verified).
- GitHub Pages docs site: `https://prof-bilal.github.io/Warden/` serves
  (HTTP 200 verified).
- Build from source: `go build -o warden ./cmd/warden` (docs/install.md; verified
  locally).
- Homebrew: formula generator exists (`build/brew.sh`) but the tap is **not
  published** (`REMAINING_WORK.md:199–204`).
- Note: `docs/install.md` says npm is "coming soon" and uses the name
  `@warden-sandbox/mcp-warden`, while the real published package is
## 8. Roadmap / status tracking — verified

- `ROADMAP.md`: M0–M8 all marked `[x]` (repo groundwork, Linux prototype,
  network enforcement, usability/trace/init/limits, macOS, Windows, stretch
  [approval + gateway], M8 compat matrix + beta program + fixtures).
- `REMAINING_WORK.md`: P0 items fixed (`eadca83` ETW proc bindings →
  advapi32; `f2232c2` WFP DLL probe → fail-closed). Open items: Homebrew tap
  not published, npm rerun-safety cosmetic issues, stale wrapper version in
  repo, confirm P0 fixes green on next Windows CI rerun.
- Source HEAD includes: npm 0.1.9 bump (`2d65e9d`), polished help (`a3672ca`),
  CLI identity/branding (`e5e89d9`), landing-page Proof section + unified
  fail-closed errors (`669b79b`).
- README at repo root (`README.md`) and `warden-README.md` are **outdated**:
  they still say "early development / pre-alpha", "design skeleton",
  "quickstart not yet implemented", and describe no Windows backend. The
  authoritative README is `warden-starter/warden/README.md`, which says all
  backends implemented.

## 9. Attack-simulation claims — NOT verified

The following numbers appear ONLY in landing-page/marketing content
(`warden-landing/components/Proof.tsx`,
`warden-landing/product-hunt-gallery/04-proof.html`, `marketing.md`):
"12.2M hashes in 6.5s", "5/5 files encrypted in 88ms", "1 token + 6 SSH keys
stolen", "25K connection attempts", "1 file destroyed", "3 secrets from 59
env vars", block timings "38ms/33ms/27ms/29ms/27ms", "5/5 (100%) contained,
0 bypasses", "ten for ten".

**Verification result:** No attack harness, no reproduction scripts, no audit
log fixtures, and no committed test data exist anywhere in the repository or
its git history to support these numbers. They were introduced in a single
landing-page commit (`669b79b`) with no accompanying evidence. They are
**DOCUMENTED BUT UNVERIFIED** and must not be published as measured results
until a reproducible harness and fixtures are committed. The landing page's
own honesty note ("Run internally against Warden's Windows backend. Not yet
independently audited") is the correct framing if these are ever used, but the
numbers themselves still lack a source.

## 10. What runs where (platform support matrix — evidence basis)

| Platform | Backend | Evidence | Real-machine verification |
|---|---|---|---|
| Linux (amd64/arm64) | bwrap; Docker fallback | Code + 7 escape tests PASS on this host; binary in dist/ | VERIFIED on Arch x86_64 today |
| macOS | sandbox-exec (Seatbelt); Docker fallback | Code (`darwin/*`), unit tests, binary built | NOT verified on real macOS hardware (no macOS CI) |
| Windows | AppContainer + WFP + ETW + Job Object | Code, cross-platform unit tests, Windows CI job present | NOT verified on real Windows here; first CI run caught 2 P0 bugs, fixes landed, rerun acceptance open |

## 11. Honest limitations documented by the project itself

(`docs/security.md` "Known Limitations")

- Network enforcement depends on `bwrap --unshare-net`; if unprivileged user
  namespaces are disabled, Warden may fall back to Docker or run with broader
  access (documented gap).
- No CPU throttling or cgroup limits on any platform.
- Process-tree limits: only the direct child tree is tracked; grandchildren
  detached by fork+exec outside the tracked tree may escape limits.
- macOS Seatbelt is deprecated by Apple; may be removed in a future macOS
  release with no Warden native fallback.
- Docker fallback needs a running daemon; on macOS the in-container proxy
  bridge must be a Linux-built binary (`WARDEN_DOCKER_BRIDGE`).
- Windows AppContainer needs an elevated (admin) process for WFP + ETW;
  Warden fails closed without it. The kernel WPP providers accept one enabling
  session, so another controller (PerfView, an EDR) also fails the run closed.
- Windows audit visibility: a denied file open is enforced by the token before
  it reaches the filesystem, so there is no blocked record — denial shows as an
  absent allowed operation. Kernel-Network payloads not yet decoded.
- Linux strace auditing adds ~2–5x overhead; strace must be installed; auditing
  is optional for runs (denials still enforced at syscall level).
- The egress proxy only intercepts HTTP/HTTPS.

## 12. Repo hygiene observations (verified)

- Working tree has uncommitted changes: `branding_test.go`, `firstrun_test.go`,
  `ui_test.go` (test fixes, +34/−1), untracked `marketing.md` and
  `warden-landing/product-hunt-gallery/`.
- The checked-in `warden-starter/warden/warden` binary is stale compared to
  source HEAD (reports `v0.1.6-2-g67d3403`; HEAD is `2d65e9d`).
- Root `README.md`/`warden-README.md` contradict the authoritative
  `warden-starter/warden/README.md`.
- `docs/install.md` npm references are stale (name + "coming soon").
- `PRD.md` still lists "Windows support in v1" as a non-goal, contradicting the
  implemented M5 Windows backend.
  `warden-sandbox-cli` → documentation inconsistency.
  (plan logic, ETW decode, etc.).