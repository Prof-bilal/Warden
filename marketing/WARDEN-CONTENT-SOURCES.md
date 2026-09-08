# WARDEN-CONTENT-SOURCES.md

Maps every important claim used in Warden launch content to its evidence.
Paths are relative to the repository root `/home/abdullah/Downloads/warden`.
`[V]` = verified by reading code/tests/config on 2026-09-08. `[Ex]` =
externally verified (registry/API).

---

## A. Product identity & positioning

| Claim | Source | Verification |
|---|---|---|
| "A lightweight sandbox runtime for MCP servers" | `warden-starter/warden/README.md:1–9` | [V] |
| Deny-by-default: path/host/env denied unless granted | `internal/policy/policy.go:1–7`; `ARCHITECTURE.md:7–9` | [V] |
| Transparent stdio — client sees a normal server | `ARCHITECTURE.md:5–6`; `internal/sandbox/windows/run.go:17–22` | [V] |
| Single static binary, near-zero overhead goal | `README.md:76–77` (root) and `docs/prd.md` NFR section | [V] (stated goal; not benchmarked) |
| MIT license | `warden-starter/warden/LICENSE` | [V] |
| Version v0.1.x series | GitHub Releases API (`v0.1.3`–`v0.1.7`) [Ex]; npm `warden-sandbox-cli` 0.1.0–0.1.9 [Ex] | [Ex] |

## B. Features — verified

| Claim | Source | Verification |
|---|---|---|
| Linux backend = bubblewrap, unshared namespaces, bind mounts, no root | `internal/sandbox/linux/linux.go:40–100` | [V] + runs |
| Network = hostname allowlist proxy, DNS check before resolution | `internal/proxy/proxy.go:1–4,243–278` | [V] |
| Env = allowlist only, empty list = empty env | `internal/envfilter/envfilter.go:15–39` | [V] + test |
| Filesystem = granted paths only, others invisible | `internal/sandbox/linux/linux.go`; escape tests | [V] + tests pass |
| Resource limits = RSS sampling 25 ms, wall-clock, SIGTERM→SIGKILL | `internal/sandbox/linux/limits.go`; `darwin/run.go:155–204` | [V] + tests |
| Windows = AppContainer token + capability DACLs + WFP + Job Object + ETW | `internal/sandbox/windows/plan.go:1–19`, `run.go:55–136` | [V] (code); real-hardware run UNVERIFIED |
| Windows fails closed on any layer failure | `internal/sandbox/windows/plan.go:18–19` | [V] |
| Backend auto-select, Docker fallback, never unsandboxed | `internal/sandbox/backend.go:73–151` | [V] + test |
| Audit log JSONL at `${XDG_STATE_HOME}/warden/audit.jsonl` mode 0600 | `README.md:78–79`; `internal/audit/audit.go` | [V] |
| `--approve` interactive mode (network live, filesystem save+restart) | `docs/approve.md`; `internal/approve/*`; `ARCHITECTURE.md:117–137` | [V] |
| Gateway init/run/wrap/list for Claude-style & 2 gateway YAML families | `docs/gateway.md:1–38`; `cmd/warden/gateway.go` | [V] |
## C. Test evidence

| Claim | Source | Verification |
|---|---|---|
| 256 tests run / 255 pass / 1 fail / 0 skip on reference host | `go test -count=1 -v ./...` executed 2026-09-08 | [V] — reproduce with `go test ./...` |
| Docker backend integration test currently fails | `internal/sandbox/docker/integration_test.go:24–52` | [V] — FAIL on this host; CLI repro blocks correctly |
| Linux escape tests pass (read/write/env/fail-closed) | `internal/sandbox/linux/integration_test.go` | [V] — PASS on this host |
| Windows escape tests exist (read deny, network block, timeout tree-kill) | `internal/sandbox/windows/integration_test.go:43–179` | [V] — require Windows |
| CI runs Linux + Windows jobs; Windows job asserts ETW/escape tests ran | `<root>/.github/workflows/ci.yml` | [V] |
| Release pipeline auto-publishes GitHub Release + npm + Pages | `<root>/.github/workflows/release.yml` | [V] |

## D. Compatibility matrix

| Claim | Source | Verification |
|---|---|---|
| 18 servers: 14 pass / 2 conditional / 2 fail | `testdata/compat/matrix.yaml`; `docs/compatibility.md:17–40` | [V] fixtures; live-server runs manual |
| Every server has a pinned policy fixture | `testdata/compat/<name>/policy.yaml` ×18 | [V] |
| Fixtures enforced by tests (schema-level) | `internal/compat/compat.go:8–9`; `internal/compat/compat_test.go` | [V] — passes |
| No wildcard hosts in schema (validated) | `internal/policy/policy.go` (validateHost); `docs/faq.md:77–85` | [V] |
| Docker-mcp failure inherent; Playwright schema-gap | `testdata/compat/matrix.yaml` (docker, playwright rows) | [V] |

## E. Distribution

| Claim | Source | Verification |
|---|---|---|
| Binaries for linux/darwin amd64+arm64 + windows amd64 | `warden-starter/warden/dist/` (5 files + SHA256SUMS) | [V] |
## F. Security-model claims — use with disclosure

| Claim | Source | Verification |
|---|---|---|
| Blocked hosts never resolved via DNS (allowlist before lookup) | `proxy.go:1–4`; `docs/security.md:57–59` | [V] |
| Ungranted files don't exist inside sandbox | `docs/security.md:16–17`; linux escape tests | [V] |
| Credential exposure if `HOME` + broad read grant | `docs/security.md:160–163`; `docs/faq.md:59–66` | [V] — user footgun |
| No CPU limits; no cgroups; HTTP/HTTPS-only proxy; strace 2–5×; Seatbelt deprecated; userns-disabled gap | `docs/security.md:140–153` | [V] — documented by project |
| Attack-simulation numbers (88ms, 27ms, 25K, 6 SSH keys, 59 env, 12.2M hashes) | `warden-landing/components/Proof.tsx` only | **NO SOURCE** — no harness/logs/fixtures in repo; treat as UNVERIFIED |

## G. Docs inconsistency audit (found while verifying)

| Inconsistency | Evidence |
|---|---|
| Root `README.md` + `warden-README.md` say "pre-alpha / design skeleton" | vs `warden-starter/warden/README.md:11–13` "All backends implemented" |
| `docs/install.md` says npm "coming soon", name `@warden-sandbox/mcp-warden` | npm has `warden-sandbox-cli` 0.1.0–0.1.9 live |
| `PRD.md` lists Windows support as non-goal | Windows backend implemented (M5 done, `ROADMAP.md`) |
| Checked-in `warden` binary reports `v0.1.6-2-g67d3403` | HEAD is `2d65e9d` (npm 0.1.9); source build reports `dev` |
| Backends.tsx says all three platforms "Ready" | Windows/macOS real-hardware verification still pending (see §10 facts) |
| GitHub Releases v0.1.3–v0.1.7 | `https://api.github.com/repos/Prof-bilal/Warden/releases` | [Ex] |
| npm `warden-sandbox-cli` published | `npm view warden-sandbox-cli versions` | [Ex] |
| Docs site on GitHub Pages (HTTP 200) | `https://prof-bilal.github.io/Warden/` | [Ex] |
| Homebrew tap NOT published | `REMAINING_WORK.md:199–204` | [V] |
| `warden trace` + `warden init` starter-policy workflow | `docs/cli.md`; `internal/audit/trace.go`; `internal/policy/starter.go` | [V] |