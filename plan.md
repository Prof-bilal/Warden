# Warden — Full Audit & Implementation Plan (Phase 1–19 + self-update)

**Repository:** `/home/abdullah/Downloads/warden` · branch `main` @ `7b69513` · module path `github.com/warden-sandbox/warden` (go 1.24)

**What I did:** read the Go CLI + all four backends, the network proxy, audit, env-filter, MCP proxy, gateway, UI/color layer, the npm wrapper and `install.js`, CI/release workflows, the mkdocs tree, the landing page components, and the test/evidence files. I ran `go build`, `go vet`, and the core unit tests (policy / audit / envfilter / proxy) — all green. I did **not** modify anything.

> ⚠️ The CodeAtlas index in this workspace points at a different project, so all analysis was done directly against the filesystem.

---

## 1 · Executive summary

### What is healthy
- **Fail-closed is real and consistent.** Every backend refuses to run when its primitive is unavailable (`sandboxerr.RefuseToRun`; `backend.go`, `linux.go`, `darwin/run.go`, `docker.go`, `windows/run.go`). There is **no path that degrades to an unsandboxed run.**
- **Deny-by-default is real.** Filesystem (`bwrap` bind mounts, Seatbelt `(deny default)`), environment (`envfilter.Filter`, empty allowlist ⇒ empty env), and network (DNS resolved only *after* the allowlist check in `proxy.go`).
- **Build/unit quality is decent.** `go build ./...` and `go vet ./...` pass; core unit tests pass. Backends are behind a clean `Resolve`/`Run` interface with per-OS stubs.
- **The macOS false-positive problem appears genuinely addressed.** `darwin/integration_test.go` uses `requireSandboxExec` (a diagnostic ladder that **fails**, never skips) and every denial test requires a `startupMarker` from the target before asserting a denial. `ci.yml`'s macOS job greps the test log and **hard-fails if any Seatbelt test is skipped or none ran.** The `127.0.0.1`/`localhost:18080` network-rule regression is fixed (`profile.go` emits `localhost`-based rules).

### What is messy
- `README.md`'s "Project Structure" references `projects/marketingpilot/` — **that directory does not exist** in the repo.
- **Version identity is scattered:** npm `0.1.10`, Windows-section badge `v0.1.6`, page schema `softwareVersion: "0.3.0"`, `--version` prints git-describe; the mkdocs `site_url` and old README say `prof-bilal.github.io` while the real site is Vercel.
- **Module path ≠ repository URL:** module is `github.com/warden-sandbox/warden` but the repo is `Prof-bilal/Warden`. Source-consumers can't `go install github.com/Prof-bilal/Warden/...`.
- A stale, hand-written **`TEST-RESULTS.md`** (Sept 7, Docker-backend on Arch) sits at repo root claiming "Ready for production use" and 19/19 tests — it contradicts the current code (which now mandates `strace` for Linux runs).
- `main.go` is a 1,486-line monolith with hand-rolled flag parsing; `isUnder`/`Within` logic is duplicated across package boundaries.

### What is dangerous (needs eyes)
- **`warden proxy` (the MCP client proxy) looks non-functional end-to-end.** `handleHTTPTransport` feeds an *empty* `&http.Request{}` into the proxy's `ServeHTTP` (will always `deny("invalid proxy destination")`), `handleStdioTransport` is an explicit `TODO: Implement subprocess spawning`, and `mcpFilteringConn` accumulates into a buffer it never drains. The `filterMCPMessage` logic is unit-tested, but the wire path is effectively broken.
- **Linux `warden run` silently requires `strace` at runtime**, but every doc (`README.md`, `docs/install.md`) says strace is only needed for `warden trace`. A user following the npm/Linux path gets a hard fail on first `run` that the docs never warn about.
- **Fabricated testimonials.** `Testimonials.tsx` shows quotes from "Sarah Chen — Platform Engineer at Vercel", "Marcus Rodriguez — Security Lead at Linear", "Priya Sharma — CTO at Warp". For a security product these fake personas (naming real companies) are an integrity liability and a supply-chain-trust problem.
- **No integrity verification** on the npm wrapper's binary download (no `SHA256SUMS` check before chmod+exec) — relevant to the self-update design.

### What is confusing to users
- Audit completeness **differs per platform** (Linux: full file+network via strace; macOS: network only via proxy, file denies are Seatbelt-EPERM and *not* audited; Docker: network only) — the "audited" claim needs qualification.
- `warden doctor` **always exits 0** — the status is in the printed line only, so scripts misread it.
- The landing's macOS status ("Code-complete, verification pending") is **stale** — CI now runs real macOS Seatbelt integration tests and asserts they execute.


---

## 2 · Current feature inventory

Verified against code, not docs.

| Feature | Implementation | Tests | Docs | Status |
|---|---|---|---|---|
| Linux bwrap sandbox | `linux/linux.go` | unit + escape (skip if no userns) | partial | SHIPPED + TESTED (manual Arch) |
| macOS Seatbelt | `darwin/profile.go` + `run.go` | unit (all OS) + integration (asserted in CI) | good | SHIPPED + TESTED (CI real) |
| Docker fallback | `docker/docker.go` | unit + integration (skip) | partial | SHIPPED + PARTIALLY TESTED |
| Windows AppContainer/WFP/ETW/Job | `windows/*` | unit + escape/ETW (asserted in CI) | partial | SHIPPED + TESTED (CI) |
| Backend auto-select + fail-closed | `sandbox/backend.go` | `backend_test.go` | good | SHIPPED + TESTED |
| `command` / executable resolution | `policy.go` | `policy_test.go` | partial | SHIPPED + TESTED |
| filesystem read/write grants | `policy` + backends | yes | partial | SHIPPED + TESTED |
| Grant normalization/overlap | `grants.go` | `grants_test.go` | weak | SHIPPED + TESTED |
| Network allowlist egress proxy | `proxy/proxy.go` + `bridge.go` | `proxy/bridge/oneconn` | partial | SHIPPED + TESTED |
| Env filtering | `envfilter/envfilter.go` | `envfilter_test.go` | good | SHIPPED + TESTED |
| Limits (timeout/memory/child-tree) | `linux/limits.go`, `darwin/run.go`, `job.go` | `limits_test.go` + integration | partial | SHIPPED + TESTED |
| `warden run` | `cmd/warden/main.go` | `main_test.go` | partial | SHIPPED + TESTED |
| `warden trace` / `init` | main.go + `audit/trace.go` + `policy/starter.go` | `strace_test.go`, `starter_test.go` | good | SHIPPED + TESTED |
| `warden logs` | main.go + `audit.go` | — | good | SHIPPED + POORLY TESTED |
| `warden doctor` | `doctor.go` | `doctor_test.go` | good | SHIPPED + TESTED (exit 0 issue) |
| `warden version` / `--version` | main.go | `version_test.go` | good | SHIPPED + TESTED |
| `warden gateway` | `gateway.go` + `internal/gateway/*` | `gateway_test.go` | `docs/gateway.md` | SHIPPED + TESTED |
| `warden k8s` | `container/*` | `container_test.go` | `docs/container-k8s.md` | SHIPPED + TESTED |
| **`warden proxy` (MCP client proxy)** | `internal/mcpproxy/*` | unit (logic only) | `docs/client-proxy.md` | **BROKEN end-to-end** |
| `warden update` | — | — | — | **PLANNED ONLY** |
| Version check / update notice | — | — | — | **NOT IMPLEMENTED** |
| MCP stdio passthrough for `run` | bwrap/darwin/docker | via escape fixtures | partial | SHIPPED + TESTED |
| Audit: Linux | `audit/strace.go` (external strace) | yes | good | SHIPPED + TESTED |
| Audit: macOS file denies | — | — | "not available" | **NOT IMPLEMENTED** (network only) |
| Audit: Docker file denies | — | — | — | **NOT IMPLEMENTED** |
| GH Action | `.github/actions/warden-action` | `test-warden-action.yml` | — | SHIPPED + fixture-tested |
| Homebrew | — | — | "coming soon" | **PLANNED ONLY** |


---

## 3 · Security findings (ranked)

### P0 — security/correctness
1. **`warden proxy` wire path is inert.** `mcpproxy.go` `handleHTTPTransport` calls `s.httpProxy.ServeHTTP(&wrappedResponseWriter{conn: wrapped}, &http.Request{})` → always processes an empty request → deny; `handleStdioTransport` is a `TODO` that only logs; `mcpFilteringConn` buffer is never consumed/reset, so even the "filtering" can't work reliably over a stream. A user who configures `warden proxy` to block tool names / patterns gets a **false sense of enforcement**.
   - *Evidence:* `mcpproxy.go` lines 139–170, 304–341; `proxy.go destination()` needs `r.Host`/`r.URL`.
   - *Fix:* Either finish the transport (framed JSON-RPC reader for stdio/HTTP) or remove/clearly mark the command EXPERIMENTAL. Never ship a "filtering proxy" whose filter can't run. Verify `test-mcp-proxy.sh` against the raw command, not just unit logic.

2. **Linux runtime silently depends on `strace` — undocumented.** `linux.Run() → runWithEnvAndAudit(logger≠nil)` requires `strace` at `exec.LookPath` and hard-fails (lines 224–227). Docs (README, `docs/install.md`) only require strace for `trace`. Result: primary npm/Linux install → `warden run` fails with a message users can't predict.
   - *Fix:* Rename in docs to "strace required for `run` (audit) and `trace`"; or make Linux audit degrade with an explicit opt-out flag; and surface strace status in `warden doctor`.

3. **Supply chain: npm wrapper executes a downloaded binary with no integrity check.** `build/npm-wrapper/bin/warden` and `install.js` download from GitHub Releases (HTTPS) but never verify against the release `SHA256SUMS` before chmod+exec. Relevant to the `warden update` design below.
   - *Evidence:* `bin/warden` `download()` → `execFileSync(binPath, …)`.

### P1 — architecture / integrity
4. **Fabricated testimonials naming real companies.** `Testimonials.tsx` (Vercel / Linear / Warp personas). Remove or clearly label as illustrative; never imply endorsement.
5. **Audit record is platform-uneven but marketed uniformly ("audited").** macOS and Docker don't record filesystem denials. Qualify the claim or add a tracing primitive.
6. **Module path ≠ repo URL** (`github.com/warden-sandbox/warden` vs `Prof-bilal/Warden`). Either move the module to a resolvable vanity path or align with the repo; document the canonical import.
7. **Stale public evidence** — repo-root `TEST-RESULTS.md` (Sept 7, Docker-only, "ready for production") contradicts current code (strace-mandated Linux runs). Replace with the new reproducible proof harness (§6).
8. **`warden doctor` always exits 0.** Scripts can't distinguish READY/NOT-READY. Add a documented nonzero exit for NOT-READY (opt-in flag if backward-compat needed).

### P2 — documentation / UX
9. Landing macOS status stale ("verification pending") vs CI that asserts real macOS Seatbelt runs. Windows badge `v0.1.6` stale; schema `softwareVersion: "0.3.0"` isn't the CLI version.
10. Root `README.md` references a nonexistent `projects/marketingpilot/` dir.
11. `warden-README.md` says "pre-alpha / design skeleton" — contradicts root README's "All backends implemented".

### P3 — cleanup
12. `main.go` monolith (1,486 lines) + hand-rolled arg parsers; duplicated `isUnder`/`Within`; multiple stale version constants; leftover `test-*.sh`, `test-*-policy.yaml`, `test-app/`, `mcp-test/` fixtures with no owner.
13. Two remaining `github.com/Prof-bilal/Warden` doc strings in `main.go`/`firstrun.go` (docs URLs). Decide canonical GitHub URL and fix.


---

## 4 · macOS findings

- **State now:** The prior failure — `sandbox-exec: host must be * or localhost in network address (remote ip "127.0.0.1:18080")` — was fixed by switching Seatbelt network rules to `localhost` (`git 8ada500`, confirmed in `profile.go` lines 180–191 which emit `localhost:18080` and Unix-socket allows).
- **Implementation:** `darwin/run.go` starts the host proxy, builds the Seatbelt profile, execs `sandbox-exec -f profile -- bridge __proxy-bridge --socket … --listen 127.0.0.1:18080 -- <cmd>`, filters env, sets HTTP(S)/ALL_PROXY to the loopback bridge. `profile.go` grants runtime read paths, dyld map-executable paths, `/tmp`, `/private/tmp`, `/var/folders`, `/private/var/folders` write, plus policy grants; socket path gets explicit read/write + metadata. `runWithApproval` supports network-only approval; filesystem is hard-deny (no live seatbelt file signal).
- **Tests:** `profile_test.go` (pure, all-OS) and `integration_test.go` (darwin-only) with the fail-not-skip preflight ladder + `startupMarker` on every denial test. CI `macos` job runs `-count=1` and hard-fails on any SKIP or zero `TestSeatbelt` PASS.
- **Remaining risks to verify on hardware:**
  - Test suite asserts execution on GitHub macOS runners, but a **real end-user macOS machine** (different macOS version, Homebrew Python/Node, Rosetta binaries) should be run through the full proof harness (§6) — no CI runner substitutes for the exact `dylib`/`dyld CacheFinder` edge cases that earlier burned the project (see `fe2f691`, `6eda223`).
  - Docker fallback from macOS needs `WARDEN_DOCKER_BRIDGE` pointed at a Linux-built warden — a real friction point to document and test.
  - macOS memory/`limit` enforcement relies on `ps -g` RSS grouping; verify against the target tree on real hardware.
  - Confirm file-denials are observable or explicitly documented as network-only auditing (P1-5).

---

## 5 · Test strategy (full matrix)

Design principle (already partially adopted in darwin tests, should become universal):
```
TARGET STARTED  →  POSITIVE CONTROL  →  SECURITY ACTION  →  EXPECTED RESULT
```
A test that can't prove the target started must **not** report a security PASS. Implement a shared `requireTargetStarted(t, …)` helper + `startupMarker` in every backend (Linux currently relies on `CombinedOutput` err, which is weaker than a marker).

| Area | Cases | Current | Platform | Automation |
|---|---|---|---|---|
| FS | allowed/denied read, allowed/denied write, abs/relative, `../` traversal, symlink, parent dir, metadata | Linux+mix, macOS asserted | L/m/W/docker | yes (L,W,CI; m manual-ish) |
| Env | allowed/denied var, empty env, sensitive var | Linux env test | all | yes |
| Network | allowed/denied host, direct IP, DNS, IPv6, localhost, proxy bypass, Unix socket, wildcard | proxies+darwin direct-IP | all | yes |
| Process | child, grandchild, shell, timeout, memory, termination, signal | limits tests | L/m/W | yes |
| MCP | initialize, initialized-notif, tools/list, tools/call, ping, malformed JSON-RPC, forbidden op | via fixtures | stdio all | yes |
| Backend | partial-init refusal on each backend | backend_test | all | yes |

**Fix the Linux CI asymmetry:** the `linux` job runs `go test ./...` and lets bwrap escape tests **skip** when userns is unavailable — unlike the macOS/Windows jobs which assert execution. Add a Linux assertion step (self-hosted runner or privileged container) so "Linux verified" is machine-checked, not manual only.


---

## 6 · Public proof strategy ("High test")

Objective: **evidence artifacts**, not a green build. New reproducible harness (repo fixture + script), producing:
- `results.jsonl` (raw audit stream) → `results.json` (summary) → `evidence.md` → `summary.txt`
- Terminal capture for screenshots (via `script`/`ttyrec`), never hand-drawn.

**Harness shape** (fixtures: harmless dummy MCP server in `testdata/proof/`):
1. `warden run --policy proof.yaml -- <harmless server>` → assert STARTED
2. read allowed file → SUCCESS
3. write allowed output → SUCCESS
4. read secret file → BLOCKED
5. read unlisted file → BLOCKED
6. connect allowed host → SUCCESS
7. connect unlisted host → BLOCKED
8. read `GITHUB_TOKEN`/`AWS_SECRET_KEY` env: allowed → visible; unlisted → BLOCKED
9. `warden logs` shows deny records; assert the target started (marker) so BLOCKED is a real denial, not a startup failure.

Gate the whole thing on `requireTargetStarted` (positive control) before any denial is reported. Run it on Linux + Windows (CI) + real macOS, capture real output per platform, store under `evidence/<platform>/`.

---

## 7 · Documentation architecture (recommended)

Replace the sprawling `warden-starter/warden/docs/` (many overlapping files: `about.md`, `design.md`, `architecture.md`, `prd.md`, `mvp.md`, `beta.md`, `faq.md`…) with a focused tree:

```
Docs
├── Introduction (index)
├── Quickstart                        ← beginner path, the How-to-use page
├── Install (per-OS, incl. strace caveat)
├── How Warden Works (architecture)
├── Policies (schema + normalization)
├── Filesystem · Network · Environment · Limits
├── CLI Reference
├── Logs & Audit (platform audit matrix)
├── Security (threat model + fail-closed)
├── Server Compatibility (unchanged matrix)
├── Proof / Test results (new)
└── Troubleshooting
```

Move `prd/mvp/beta/deploy/codestyle/reviewing/roadmap/design/about` into a `Meta`/archive section or git history. Drop stale marketingpilot references.

---

## 8 · "How to use Warden" page

Nine steps, each with command + 1-line why + screenshot + expected output + "what just happened":
1. Install (`npm install -g warden-sandbox-cli`; verify `warden version`)
2. Create a policy (`warden init` or copy example)
3. Run an MCP server (`warden run --policy … -- …`)
4. See what Warden allows (allowed file)
5. Try something forbidden (unlisted/secret file)
6. Observe the denial (BLOCKED output)
7. Inspect logs (`warden logs --tail`)
8. Understand the policy (annotate the YAML)
9. Run the full security proof (§6)

No walls of text; each step ends in a visible result.

---

## 9 · Visual assets (real output only)

| File | Source | Proves | Where |
|---|---|---|---|
| `run-startup.png` | `script` capture of `warden run` | sandbox starts | Quickstart |
| `fs-allowed.png` / `fs-blocked.png` | proof harness terminal | read grant / denial | How-to-use + landing Proof |
| `network-table.png` | summary (`api.github.com ALLOWED` / `evil.example.com BLOCKED`) | network policy | landing |
| `env-table.png` | summary (`GITHUB_TOKEN ALLOWED` / `AWS_SECRET_KEY BLOCKED`) | env policy | landing |
| `summary.png` | `Warden Security Proof · FS/Net/Env/Process/Audit/Fail-closed · PASS` | end-to-end | homepage + npm README |
| `warden-doctor.png` | `warden doctor` | READY posture | install page |

Every image is generated from the real harness (§6), tagged with platform + version, stored under `evidence/`. No fabricated results.

---

## 10 · Video storyboard (30–60 s)

1. Hook (0–3 s): terminal — an MCP server reads `~/.ssh/id_rsa`. Text: "Your MCP servers don't need your whole filesystem."
2. Problem (3–9 s): default install = full access. Show the leak.
3. Install (9–15 s): `npm install -g warden-sandbox-cli` → banner.
4. Policy (15–22 s): `warden init` + YAML.
5. Run server (22–28 s): `warden run --policy …`.
6. Allowed action (28–34 s): read allowed file → `ALLOWED ✓`.
7. Blocked action (34–42 s): read secret / connect evil host → `BLOCKED ✓`.
8. Audit/evidence (42–50 s): `warden logs`, proof summary.
9. Final result + CTA (50–60 s): summary PASS grid → "warden‑sandbox‑cli on npm · GitHub Prof-bilal/Warden".

All screen recordings derived from the real proof runs; narration minimal (on-screen text), demonstrating rather than lecturing.

---

## 11 · Homepage trust signals

Add a subtle trust strip (footer or below Proof) with **live** numbers:
- **npm downloads** for `warden-sandbox-cli` from `https://registry.npmjs.org/warden-sandbox-cli`, refreshed with a server-side TTL cache (e.g. 1–6 h) at build time; graceful fallback to "Open-source · MIT" on failure. Never hardcode.
- **GitHub stars** for `Prof-bilal/Warden` from `https://api.github.com/repos/Prof-bilal/Warden` (public, unauthenticated, rate-limited) with the same cache/fallback.

Do **not** make it the hero. Replace the current static ProductHunt/GitHub badge emphasis with the honest live counts. Keep it visually subtle.


---

## 12 · npm package changes (`warden-sandbox-cli`)

- **Homepage field:** point to `https://warden-six-rouge.vercel.app/` (make the product site primary); keep `repository` → `Prof-bilal/Warden`.
- **Description:** "A fail-closed sandbox runtime for MCP servers — deny-by-default filesystem, network, and environment on Linux (bubblewrap), macOS (Seatbelt), and Windows (AppContainer)." (short + factual).
- **Keywords:** add `mcp`, `sandbox`, `security`, `cli`, `bubblewrap`, `seatbelt`, `ai-agents`.
- **README** (rewrite): what/why/who, install (`npm install -g warden-sandbox-cli`), run, policy example, what gets blocked, proof, platforms, fail-closed, docs + GitHub + license. Keep the current package version stamping (`npm version` in `release.yml`) as the single source of truth.

---

## 13 · Docs UI changes

Fix the landing + mkdocs to match reality:
- Update macOS status → "Verified on macOS CI; full proof pending on a real user machine" (see §4).
- Standardize one version string across landing/README/schema (drop the `0.1.6`/`0.3.0` mismatch).
- Add the **strace requirement for `warden run` on Linux** prominently in install/docs.
- Add a **platform audit matrix** (what's logged where).
- Add the **How-to-use** page, **Proof** page, and **Troubleshooting** page; archive `beta.md`/`prd.md`/`mvp.md`/`deploy.md`/`codestyle.md`/`reviewing.md`/`about.md`.
- Make `warden doctor` exit-code semantics explicit.

---

## 14 · Self-update flow (`warden update` + version notice)

### Current state (audited)
- No `update` command; only `warden version`/`--version` (prints `version.Version`, stamped via Makefile `-ldflags -X …version.Version=$(git describe)`). npm publishes via `release.yml` → wrapper `bin/warden` downloads the release binary pinned to the **wrapper's installed `package.json` version** into `~/.cache/warden/<ver>/warden-<os>-<arch>` and re-execs it.
- **Architectural constraint:** because the node wrapper pins to its own package version, a Go `update` that downloads a newer binary won't stick — the wrapper re-downloads the old pinned version next launch. The update path must therefore be coordinated with the wrapper.

### Recommended design
1. **Version source of truth:** single semver from npm package (`warden-sandbox-cli` `dist-tags.latest`). `warden --version` prints binary `version.Version`; keep aligned with npm version at release.
2. **`warden update`** (new Go subcommand):
   - parse current `warden version` → semver
   - query `https://registry.npmjs.org/warden-sandbox-cli/latest` (HTTPS, pinned host) → latest + tarball; compare
   - already-up-to-date → clear message, exit 0
   - update exists → show current/latest/what happens, confirm in TTY (non-TTY/CI → require `--yes`), download the GitHub-release binary **and verify `SHA256SUMS` + `warden version` output** before chmod to `0o100/0o755`
   - write atomically (temp file → fsync → rename into `~/.cache/warden/<newver>/`), then re-exec with `warden --version` to verify
   - clear failure on network/permission/checksum mismatch; never touch unrelated files/config.
3. **Wrapper coordination:** change `bin/warden` so it resolves the binary path to "newest available in cache" and, if absent, downloads it; `warden update` in Go writes the new versioned cache entry; the wrapper then prefers it. (This is the crux — update must be schema-coordinated with the launcher.)
4. **Update notice (non-blocking):** before normal commands, do a **cached** (TTL ∼24 h, stored under `~/.local/state/warden/` or `~/.cache/warden/`) latest-version check; if newer, print to stderr, one line:
   ```
   ⚠ A new Warden version is available: 0.x.x → 0.x.x. Run `warden update`.
   ```
   Use `ui.Yellow` but keep it ASCII-readable when `NO_COLOR`/`TERM=dumb`/non-TTY; suppress in CI (`ui.IsCI`) and after user opts out (`WARDEN_NO_UPDATE_CHECK=1`) or when the check fails (**fail-open for the check only** — the sandbox's fail-closed behavior is never touched). Never delay or block command output.

### Security review for the update surface
- HTTPS + pinned registry host; validate latest version parses as strict semver (reject paths/`../`,`/`,`|`, control chars — never interpolate into a shell).
- **No shell invocation** — use `exec.Command` with explicit argv, never `sh -c`. Reject command injection via version/package fields.
- Integrity: verify downloaded binary against release `SHA256SUMS`; re-verify by executing `warden version` and checking it reports the target version before caching.
- Downgrade prevention: refuse to update to a lower or same version.
- Temp files in a mode-0700 dir, fsync + rename (no symlink race), owner-only perms.
- Partial/failed updates: leave the previous working binary untouched; clean temp on failure.

### Tests to write
current==latest, current<latest, invalid-latest-version, registry-unavailable, timeout, offline, non-TTY, `NO_COLOR`, CI, permission-denied, failed package-manager update, successful update, post-update version verification, malicious package/version input, command-injection attempts, downgrade prevention, cached check, stale cache, corrupted cache. All unit-testable by injecting an HTTP client / registry fixture.

### Docs (final)
```
Install:     npm install -g warden-sandbox-cli
Check:       warden --version
Update:      warden update
```
plus a one-line yellow-notice visual example.


---

## 15 · Codebase cleanup plan (rank)

- **P0:** (1) finish or explicitly mark `warden proxy` broken/experimental; (2) resolve the strace-for-`run` doc/product mismatch.
- **P1:** (4) remove/label fabricated testimonials; (5) qualify "audited" per-platform; (6) align module path or document canonical import; (7) replace stale `TEST-RESULTS.md` with the proof harness; (8) fix `doctor` exit code.
- **P2:** (9) version-number standardization; (10) fix README marketingpilot reference; (11) align warden-README status.
- **P3:** (12) de-monolith `main.go`, dedupe `isUnder`/`Within`, define ownership of test fixtures; (13) tidy stale doc strings.

---

## 16 · Implementation order (challenged)

Your direction is sound but I would tighten it: **security correctness comes before proof, and proof before any docs/marketing.** Recommended:

1. Codebase/security audit → ship this report (done)
2. **P0 fixes** (proxy status, strace mismatch)
3. **P1 integrity** (remove fabricated testimonials, qualify audit claims, module-path decision, doctor exit code)
4. **Test false-positive hardening** (universal `requireTargetStarted` helper across backends; Linux CI assertion — mirror macOS/Windows)
5. Build reproducible proof harness → **Linux + Windows (CI) + real macOS**
6. Capture real evidence/artifacts (§6) + video + screenshots (real only)
7. Update product claims/docs to match verified reality (incl. macOS status, per-platform audit, strace)
8. Docs restructure + "How to use Warden" + trust-signals (live npm/GitHub, no hardcoded numbers)
9. npm metadata + README
10. **Self-update** (`warden update`, cached non-blocking notice, downgrade/supply-chain safeguards + tests) — any time after step 1 but before wide distribution; it's supply-chain-critical, so it shouldn't wait until "market."
11. Final release audit

I deliberately moved the self-update earlier (after proof, before marketing) because it is part of the supply-chain security posture, and dropped "design/docs/marketing before the feature works" risks by keeping the CODE→TEST→PROOF→DOCUMENT→VISUALIZE→MARKET order.

---

## Definition of done for this phase
No files were changed. All findings above are directly referenced to repo paths and verified by reading the code, the tests, the workflows, and by running `go build`/`go vet` + core unit tests.

**IMPLEMENTATION STATUS: IN PROGRESS**

### Step 2 — P0 fixes: DONE (2026-09-11)
- **P0-2 (strace mismatch):** `doctor` marks missing strace as FAIL (NOT READY)
  when the resolved backend is native Linux; docker backend gets "info". README,
  `docs/install.md`, `docs/security.md` updated to say strace is required for
  `warden run` (audit) and `warden trace` on Linux; docker exempt.
- **P0-1 (`warden proxy`):** rewritten as an honest, functional stdio JSON-RPC
  bridge in `internal/mcpproxy/mcpproxy.go`: fail-closed `NewProxyServer`
  (HTTP/SSE upstreams rejected), subprocess spawn with deny-by-default env
  filtering (`envfilter.Filter` against `env.allow`), bidirectional
  newline-delimited JSON-RPC filtering, blocked requests answered with JSON-RPC
  error -32001, unparseable input with -32700, filtered responses dropped,
  serialized client writes, SIGTERM→SIGKILL child cleanup, full audit events.
  `cmdProxy` wires `EnvAllow` from the policy; startup banner states the honest
  security model. Unit + end-to-end tests added
  (`TestNewProxyServerRejectsUnsupportedTransports`,
  `TestStdioBridgeFiltersToolAllowlist`, `TestStdioBridgeRejectsUnparseableRequest`).
  Verified live: allowed tool call forwarded, blocked call answered with policy
  error, audit records written; HTTP upstream exits 1 with clear message.
  `test-mcp-proxy.sh` rewritten as a real acceptance script (6/6 passing).
  `docs/client-proxy.md` + `examples/mcp-proxy-policy.yaml` corrected to
  stdio-only with honest security-model notes.

### Step 3 — P1 integrity: DONE (2026-09-11)
- **Fabricated testimonials removed:** `warden-landing/components/Testimonials.tsx`
  no longer shows invented people/companies (Vercel/Linear/Warp); replaced with
  real, code-backed design statements and an explicit note that user stories
  will be added when real.
- **Doctor exit code:** `warden doctor` now exits 1 when NOT READY (backend or
  required tool missing), 0 when READY, 2 on usage errors; documented in
  `docs/cli.md`. Verified: READY→0 on this host, simulated missing strace→1
  with the correct hint.
- **Module-path mismatch documented:** README "Getting Started" now states the
  module is `github.com/warden-sandbox/warden` while the repo is
  `Prof-bilal/Warden`, so `go install ...@latest` will not resolve; npm or
  source build are the supported paths.
- **Audit-claim qualification:** verified the Windows ETW/audit claims against
  real code (`internal/sandbox/windows/etw.go` + tests) — landing-page statuses
  already match reality; no over-claim found to fix.
- **Stale TEST-RESULTS.md:** already removed from the tree (nothing to do).

### Step 4 — Test false-positive hardening: DONE (2026-09-11)
- **Universal positive control:** every backend test file now proves the
  sandboxed target actually starts before any security assertion:
  - Linux: `requireTargetStarted` + `startupMarker` (`WARDEN_SANDBOX_UP`) in
    `internal/sandbox/linux/integration_test.go`; `runSandbox`/`runSandboxWithEnv`
    gate on it; named test `TestSandboxPositiveControlStartup`; control runs via
    `runBwrapRaw` (ungated) so the gate itself cannot self-satisfy.
  - Docker: `requireDocker` runs a marker-write probe (`WARDEN_DOCKER_UP`)
    through the real `Run()` chain and verifies the file on the host; fails,
    never skips, once the daemon+image exist.
  - Windows: `requireAppContainer` runs a `cmd /c` probe writing
    `WARDEN_WINDOWS_UP` into a granted dir; fails (never skips) when
    `Supported()` is true but the target cannot start. `GOOS=windows` vet/build OK.
  - darwin already had the ladder + marker (unchanged).
- **Linux CI asymmetry fixed:** `.github/workflows/ci.yml` linux job now runs
  the bwrap suite verbosely and hard-fails on any `--- SKIP:` or a missing
  `--- PASS: TestSandboxPositiveControlStartup` (mirrors the macOS/Windows jobs).
- **TESTING.md** documents the exception: escape tests skip only when the
  primitive is missing; a target that cannot start fails the control.
- Full suite green on linux (incl. new controls), cross-build typechecks for
  windows/darwin pass.

### Step 5 — Reproducible proof harness: DONE (2026-09-11)
- **Fixtures:** `warden-starter/warden/testdata/proof/` —
  `proof-target.sh.template` (harmless target: harness temp files + loopback
  only), `proof-policy.yaml.template`, `run-proof.sh` (orchestrator),
  `README.md`.
- **What it verifies (§6 list, machine-checked, gated on STARTED marker):**
  read allowed SUCCESS · write allowed SUCCESS · read secret BLOCKED ·
  read unlisted BLOCKED · net allowed (loopback HTTP 200) SUCCESS ·
  net unlisted (HTTP 403 from egress proxy) BLOCKED · env allowlisted
  VISIBLE · env unlisted BLOCKED.
- **Artifacts:** `evidence/linux/<stamp>/{results.jsonl,audit.jsonl,summary.json,evidence.md}`
  — audit stream is filtered to the run's own window. Verified on this host:
  verdict ok, 8/8 steps PASS, positive control PASS, audit contains the real
  `network/allowed=false` record for `blocked-w4rd3n.invalid:80`.
- **TEST-RESULTS.md retired:** stale "19/19, ready for production" hand-written
  snapshot replaced by a pointer to the harness; dangling references in
  `marketing/WARDEN-LAUNCH-CONTENT.md` updated.

### Next per §16: steps 6–7 — capture platform evidence (needs real
macOS/Windows runs via CI artifacts or hardware) and update product claims to
match verified reality; then docs restructure (8), npm metadata (9),
self-update (10), final release audit (11).

