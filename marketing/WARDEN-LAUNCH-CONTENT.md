## 1. Product Summary

### What exactly is Warden?
Warden is a single-binary command-line tool that runs MCP servers inside an
OS-native sandbox. A small YAML policy declares exactly which filesystem
paths, network hostnames, and environment variables a server may touch;
Warden translates that policy into OS enforcement primitives — bubblewrap
namespaces on Linux, Seatbelt profiles on macOS, an AppContainer token with
WFP/ETW/Job-Object enforcement on Windows, or a Docker container as a
fallback — and refuses to run unsandboxed if no backend can be applied.
`VERIFIED`: `internal/sandbox/backend.go`, `internal/sandbox/*`.

### One-liner
> A sandbox runtime for MCP servers — deny by default, grant a folder and a
> hostname, audit everything else.

### Short description (~50 words)
Warden runs MCP servers inside an OS-level sandbox. You declare what a server
is allowed to touch — files, hosts, environment variables — in a small YAML
policy. Everything else is blocked at the OS boundary, and every allowed and
blocked access is written to an audit log. Linux, macOS, and Windows backends;
never falls back to an unsandboxed run.

### 100-word description
MCP servers from npm, PyPI, or GitHub run as plain processes with your full
user permissions — they can read your SSH keys, phone home, or rewrite your
files, and the MCP protocol does nothing to stop it. Warden puts a boundary
underneath them: a `policy.yaml` grants the exact paths, hostnames, and
environment variables a server needs. On Linux it uses bubblewrap
(unprivileged namespaces); on macOS, Seatbelt; on Windows, an AppContainer
token with WFP egress filters, Job-Object limits, and ETW auditing; Docker is
the fallback when no native primitive is available. If no backend can be
applied, Warden refuses to run rather than running unsandboxed. Every access
attempt is recorded in a JSONL audit log.

### Technical description
Warden is a Go 1.22 CLI (`cmd/warden/main.go`) around a policy engine
(`internal/policy/policy.go`) and four backends behind one interface
(`internal/sandbox/backend.go`). The egress proxy (`internal/proxy/proxy.go`)
checks hostnames against the allowlist before any DNS lookup. Environment
filtering (`internal/envfilter/envfilter.go`) forwards only allowlisted names.
Linux uses `bwrap --unshare-net --unshare-pid --unshare-user --unshare-ipc`
with read-only bind mounts for `/usr`/`/lib64` and policy grants; the sandbox
network namespace has no default route. Windows derives capability SIDs and
DACL ACE grants from the policy, installs a WFP sublayer that permits only the
loopback proxy, attaches a Job Object for memory/time limits, and starts a
real-time ETW session for audit — each step fails closed. Docker fallback uses
`--network none --read-only` with host-path binds. `VERIFIED` throughout.

---

## 2. Verified Product Facts

1. All four backends exist in source; Linux escape tests pass; Windows and
   macOS real-hardware runtime verification is still pending. `VERIFIED` /
   `PARTIALLY VERIFIED`.
2. The full test suite on the reference host: **256 tests, 255 pass, 1 fail
   (Docker-backend integration test), 0 skip**. `VERIFIED` (executed).
3. Compatibility matrix: **18 real MCP servers — 14 pass, 2 conditional, 2
   fail** — each with a pinned fixture policy enforced by tests.
   `VERIFIED` (fixtures); live-server verdicts from documented manual runs.
4. Network allowlisting happens **before DNS resolution** for denied hosts.
   `VERIFIED`.
5. Deny-by-default filesystem: unlisted paths are invisible, not merely
   permission-denied. `VERIFIED` + escape test.
6. Empty `env.allow` ⇒ empty environment. `VERIFIED` + escape test.
7. Fail-closed: no usable backend ⇒ Warden refuses to run; the refusal error
   is tested. `VERIFIED`.
8. Distribution: GitHub Releases v0.1.3–v0.1.7, npm `warden-sandbox-cli`
   0.1.0–0.1.9, docs site on GitHub Pages. `VERIFIED` (externally).
9. M0–M8 roadmap complete (per ROADMAP.md); `REMAINING_WORK.md` tracks open
   items including Homebrew tap not published.
10. **Attack-simulation numbers on the landing page (88ms, 25K connections,
    etc.) have NO reproducible source in the repo.** `DOCUMENTED BUT
## 3. Target Users

| Audience | Problem | Why they care | Warden's value | Evidence | Confidence |
|---|---|---|---|---|---|
| MCP / AI-agent developers | Servers run unsandboxed with full user rights | They install `npx`/`uvx` servers daily | One YAML policy per server, transparent stdio | `PRD.md:7–10`, README quickstart | High |
| Security-conscious developers | `~/.ssh`, `.env`, AWS creds exposed to third-party code | Real credential loss | Deny-by-default + audit log | `TEST-RESULTS.md`, `docs/security.md` | High |
| Devs running third-party MCP servers from many sources | Trusting code they didn't write | Supply-chain risk is the norm | Works with the ecosystem, no fork+patch | 18-server matrix | High |
| Companies standardizing on 3–5 MCP servers for teams | Consistent, repeatable policy per server | Repeatable security posture | Gateway wrap + policy files in git | `docs/gateway.md`, `examples/` | Medium |
| MCP gateway users | Gateway orchestrates but doesn't confine | Sandbox every registered stdio server | `warden gateway init/run/wrap/list` | `docs/gateway.md` | Medium |
| Platform / infra engineers | Evaluate sandbox runtimes (bwrap, container, OS sandboxing) | Prefer OS-native, small trust boundary | Architecture documented and tested | `ARCHITECTURE.md` | Medium |

Not targeted (explicit non-goals): multi-tenant/hosted gateways, agent-identity
audit trails. `VERIFIED`: `ARCHITECTURE.md:181–188`.

---

## 4. Core Features

| Feature | Status | Evidence | User value | Marketing priority |
|---|---|---|---|---|
| Filesystem isolation (deny-by-default, invisible paths) | `VERIFIED` | `linux.go`, escape tests pass | Stops SSH-key/credential theft, data wiping | P0 |
| Network restrictions (hostname allowlist, pre-DNS) | `VERIFIED` | `proxy.go`, Windows WFP, escape tests | Stops exfil and callbacks | P0 |
| Environment filtering (allowlist only) | `VERIFIED` | `envfilter.go` + escape test | Stops env-secret theft | P0 |
| Resource limits (memory, timeout, tree-kill) | `VERIFIED` (Linux/macOS/Docker); `PARTIALLY VERIFIED` (Windows) | `limits.go`, Job Object, tests | Stops DoS/runaway servers | P1 |
| Policy-based permissions (YAML, deny by default) | `VERIFIED` | `policy.go`, `grants.go` | The core UX | P0 |
| OS-level sandboxing (bwrap/Seatbelt/AppContainer) | `VERIFIED` (code); real-hardware Windows/macOS pending | backend code + tests | Small trust boundary | P0 |
| MCP server execution (transparent stdio) | `VERIFIED` | ARCHITECTURE goals, tests | Client-agnostic | P0 |
| Fail-closed behavior | `VERIFIED` | `sandboxerr.go`, tests, Windows gates | The headline trust property | P0 |
| Backend auto-selection + explicit `--backend` | `VERIFIED` | `backend.go` | Works everywhere or not at all | P1 |
| CLI (`run/trace/init/logs/doctor/gateway`) | `VERIFIED` | `main.go`, executed | Full sandbox workflow | P0 |
| Audit log (JSONL, 0600, strace on Linux) | `VERIFIED` | `audit.go`, `strace.go` | See what a server tried | P1 |
| Interactive approval (`--approve`) | `VERIFIED` | `approve/*`, docs | Low-friction policy widening | P1 |
| Gateway integration (wrap registered servers) | `VERIFIED` | `gateway.go`, docs | Adopts existing setups | P2 |
| Compatibility matrix + fixtures | `VERIFIED` (fixtures); verdicts manual | `testdata/compat/*`, `compat_test.go` | "Will this work for me?" | P1 |
| Playwright wildcards / unix-socket grants | `NOT IMPLEMENTED` (schema gaps) | matrix.yaml rows | n/a — documented limit | n/a |
| Windows real-hardware verification | `PARTIALLY VERIFIED` | CI + REMAINING_WORK | Unblocks "Windows Ready" claim | P0 blocker for that claim |
| Attack-simulation proof numbers | `DOCUMENTED BUT UNVERIFIED` | Proof.tsx only | n/a until rebuilt | BLOCKED |

---

## 5. Security Model

### Filesystem
- **Linux/macOS/Docker:** bind-mount only the granted paths, read-only or
  read-write. Everything else does not exist inside the sandbox. Runtime base
  (`/usr`, `/lib64`) is read-only. `VERIFIED`: `linux.go`, `darwin/profile.go`,
  `docker.go`, escape tests (`TestSandboxUnlistedPathInvisible`).
- **Windows:** AppContainer (LowBox) token denies all filesystem access by
  default; the policy grants become capability SIDs + DACL ACE entries on the
  granted paths. System paths are readable-but-never-writable. `VERIFIED`
  (code): `plan.go`, `appcontainer.go`.
- **Known gaps (project-documented):** symlinks inside granted paths are not
  resolved/blocked (`docs/security.md:42–45`).

### Network
- Hostname allowlist proxy; the hostname is checked **before** any DNS lookup,
  so denied destinations aren't even resolved. `VERIFIED`: `proxy.go:1–4`.
- **Linux/macOS:** the sandbox has its own network namespace with no default
  route; `HTTP_PROXY`/`HTTPS_PROXY`/`ALL_PROXY`/`NO_PROXY=` are injected.
  Direct non-loopback connections have no route. `VERIFIED`: `linux.go`,
  `darwin/run.go`.
- **Windows:** WFP sublayer hard-permits only the loopback proxy and blocks all
  other outbound IPv4/IPv6 including direct DNS. `VERIFIED` (code): `wfp.go`,
  `plan.go:10–12`.
- **Project-documented limits:** the proxy filters HTTP/HTTPS only — raw
  TCP/UDP traffic relies on no-route/WFP for blocking (`security.md:152`); if
  unprivileged user namespaces are disabled, the Linux fallback "may have host
  network access" and is "not always detected" (`security.md:60–65`). Say
  exactly this much — no stronger.

### Environment
- Only `env.allow` names pass through; empty allowlist ⇒ empty environment.
- Values come from the parent environment at launch; policy files store names
  only. `VERIFIED`: `envfilter.go`, `policy.go:37–40`, escape test.
- Footgun documented: passing `HOME` plus a broad `filesystem.read` grant can
  expose `~/.ssh` — Warden's docs call this a user policy bug
  (`security.md:160–163`, `faq.md:59–66`).

### Fail closed
- No backend / requested backend unavailable ⇒ `RefuseToRun`, e.g. *"no
  sandbox backend is available … refusing to run unsandboxed"*; a test asserts
  the wording stays ("fails closed by design").
- Windows: every layer (token, DACL grants, WFP, Job Object, ETW) must
  initialize; "A plain CreateProcess fallback is never acceptable."
- ETW single-session kernel providers: if another controller holds them
  (PerfView, an EDR), Warden refuses to run unaudited.
All `VERIFIED`: `backend.go`, `sandboxerr.go`, `windows/run.go`, `ROADMAP.md`.
## 6. Platform Matrix

| OS | Backend | Run today? | Required deps | Known limitations | Test coverage | Docs |
|---|---|---|---|---|---|---|
| Linux (amd64/arm64) | bubblewrap (native), Docker (fallback) | Yes | `bwrap`; `strace` for full audit; Docker optional | userns-disabled gap; strace overhead 2–5× | Escape tests PASS on reference host; CI Linux job | `docs/install.md`, `TESTING-PLATFORMS.md` |
| macOS | sandbox-exec / Seatbelt (native), Docker (fallback) | Code ready; real-hardware verification pending | `sandbox-exec` (ships with macOS); Linux-built bridge for Docker fallback | Seatbelt deprecated by Apple; no file-deny audit events on macOS | Unit tests only; **no macOS CI** | same |
| Windows | AppContainer + WFP + ETW + Job Object | Code ready (v0.1.7 binary); real-hardware verification pending | Windows 10/11 + Admin/elevation for WFP+ETW; WFP DLL must exist (probe now fail-closed) | Admin required; ETW single-session; WFP-based network block; denied opens leave no block record | Cross-OS unit tests + Windows CI job (first run caught 2 P0 bugs, fixed) | `docs/install.md`, `REMAINING_WORK.md` |
| Other/unknown OS | Docker | Only with Docker | Docker daemon + Linux-ELF bridge host | Larger trust surface | — | `docs/install.md` |

**Compatibility claim rank:**
- "Linux: Ready" — `VERIFIED` (this host today).
- "macOS: Ready" — `PARTIALLY VERIFIED` (do not ship "Ready" without a macOS
  machine/CI run).
- "Windows: Ready" — `PARTIALLY VERIFIED` (do not ship "Ready" until the
  Windows CI rerun after `eadca83`/`f2232c2` is green and recorded).
- Docker fallback — `PARTIALLY VERIFIED`: CLI behavior verified locally, but
  the repo's own `TestDockerBlocksUngrantedRead` currently fails under
  `go test` — must be triaged/fixed before launch.

---

## 7. Attack Test Evidence

**What exists and is verified:**
- Linux escape tests (fail-closed + boundary): read grant works, unlisted
  path invisible, write-outside denied, env passthrough filters, bwrap-missing
  refuses. All PASS on reference host.
  `internal/sandbox/linux/integration_test.go`.
- Windows escape tests (code): un-granted read denied, granted read works,
  network blocked + audited (curl must get 403/exit 22), timeout terminates
  tree, WFP-DLL probe resilient. Execute on Windows only.
  `internal/sandbox/windows/integration_test.go`.
- Windows unit tests for WFP failure modes (`TestWFPDLLProbeResilient`,
  `TestWFPDLLNameNonEmpty`).
- Credential-threat tests are boundary tests, not attack-scenario audits:
  `~/.ssh/id_rsa` blocked (`TEST-RESULTS.md`), `/etc/shadow` probe denies in
  every compat fixture (`testdata/compat/matrix.yaml`).

**What is claimed on the landing page but NOT in the repo:**
- "Ransomware — 5/5 files encrypted in 88ms", "Credential Harvester — 1 token
  + 6 SSH keys", "DDoS — 25K connection attempts", "Data Wiper — 1 file
  destroyed", "CPU Exhaustion — 12.2M hashes in 6.5s", "env-stealer — 3
  secrets from 59 env vars", block timings "38/33/27/29/27ms", "5/5 contained.
  0 bypasses" / "ten for ten".
- **No harness, scripts, logs, or fixtures exist for any of these numbers.**
  They were added in one landing-page commit (`669b79b`). `DOCUMENTED BUT
  UNVERIFIED`. Do not repeat them externally; if the internal test data truly
  exists elsewhere, commit a reproducible harness + fixtures first.

**Real count of automated security tests in the repo:** the 256-test suite
contains boundary/escape tests across backends (Linux: 7 live escape tests;
Windows: 6 escape/lifecycle tests on Windows-only; plus policy-schema
attack-surface tests). Automated *scenario* attacks (ransomware, DDoS, etc.)
do **not** exist in the repository. State that honestly.

---

## 8. Competitive Positioning

**Category:** MCP security / sandbox runtime for local AI-agent servers —
closest peers are MCP-gateway products with policy layers (e.g. MCP gateways
with allow/deny), general sandbox runtimes (Firejail, bwrap, sandbox-exec,
containers), and AI-security platforms.

What alternatives would a user consider:
1. Run MCP servers in Docker directly (`docker run --network none ...`).
2. Firejail / bwrap manually.
3. MCP gateways with per-server permission files.
4. macOS Seatbelt / Windows AppContainer used directly.
5. Nothing (the current default).

Why choose Warden (differentiation, all verified):
- **Policy → enforcement translation** for MCP workloads, not raw sandbox
  flags ("a policy.yaml is the trust boundary").
- **Deny-by-default + fail-closed** as a hard product rule with tests
  asserting the refusal message.
- **Transparent stdio** — your Claude config/IDE keeps talking to the server
  unchanged.
- **`warden trace` → `warden init`** starter-policy workflow (no blind YAML).
- **18-server compatibility matrix with pinned fixtures.**
- **Egress proxy with pre-DNS allowlist** — denied hostnames are never
  resolved, so there's no DNS-leak channel.
- **Cross-platform native backends** instead of Docker-everywhere.
- **Single static binary, no daemon.**

Where Warden is weaker (be honest):
- No CPU-throttling/cgroup limits; proxy is HTTP/HTTPS-only; unprivileged-
  userns-disabled case can broaden access; Seatbelt is deprecated upstream;
  Windows needs admin; the schema can't express wildcard hosts or unix sockets
  (blocks Playwright-class servers); real-hardware Windows/macOS proof is
  pending; one Docker integration test currently fails.
- No "first to market / only tool" claim is made here — the ecosystem is
  moving fast and we have not audited every adjacent product for this package.

---

## 9. Product Hunt

**Name:** Warden

**Taglines (ranked; ≤60 chars):**
1. `Sandbox your MCP servers with one YAML policy` (47)
2. `MCP servers get exactly the access you grant. Nothing more` (58)
3. `A deny-by-default sandbox for MCP servers` (44)
4. `Run MCP servers in an OS-level sandbox, not with full access` (60)
5. `Warden — the boundary MCP never had` (39)

Pick #1 as primary; it names the action, the artifact, and the audience.
Use #5 as the brand line everywhere else.

**Short descriptions (≤500 chars each):**
1. "Warden runs MCP servers inside an OS-native sandbox. A small policy.yaml
   grants the exact files, hosts, and environment variables a server needs —
   everything else is blocked at the OS boundary and logged to an audit file.
   Linux bwrap, macOS Seatbelt, Windows AppContainer, Docker fallback. If no
   backend applies, Warden refuses to run. Free, open source, single binary."
2. "Every MCP server you install runs with your full user permissions today —
   your SSH keys, your network, your files. Warden puts a sandbox underneath
   them: grant a folder, a hostname, a few env vars; deny everything else by
   default; audit everything. No daemon, no containers required, works with
   any MCP client, tested against 18 real servers."
3. "Warden is a lightweight sandbox runtime for MCP servers. Policy in →
   boundary out: bubblewrap on Linux, Seatbelt on macOS, AppContainer + WFP +
   ETW on Windows, Docker as fallback. Fail closed by design — if it can't
   sandbox, it refuses to run. Open source (MIT), single static binary."

**Long description (1 strong version):**
> MCP put AI tools inside your editor — and inside your `~/.ssh`.
>
> Most MCP servers ship as a plain Node or Python process that runs with your
> user's full permissions. Nothing in the MCP protocol stops a buggy or
> malicious server from reading your keys, calling home, or rewriting your
> files.
>
> Warden gives MCP the boundary the protocol never had. You write one small
> YAML policy that grants exactly what a server needs — a data folder, an API
> hostname, a couple of environment variables. Warden translates that into
> real OS enforcement: unprivileged namespaces via bubblewrap on Linux,
> Seatbelt on macOS, and on Windows an AppContainer token with WFP egress
> filters, Job-Object limits, and an ETW audit trail. Docker is the fallback
> when no native primitive exists.
>
> Everything not granted is blocked before it can happen — denied hostnames
> are never even resolved, unlisted files simply don't exist, and your shell
> environment is never inherited. Every access attempt, allowed or blocked,
> lands in a JSONL audit log.
>
> And if Warden can't apply a sandbox, it refuses to run the server. Fail
> closed is the whole point.
>
> Warden is pre-1.0, MIT-licensed, single-binary, and tested against 18 real
> MCP servers (14 pass out of the box, 2 need per-deployment hosts, 2 can't be
> sandboxed honestly — we publish why). Try it against a server you didn't
> write; that's the real test.

**Product Hunt topics:** Developer Tools · Artificial Intelligence ·
Security · Open Source · Command Line · MCP · AI Agents

**Maker story (built only from repo facts, no invented biography):**
"The MCP ecosystem exploded in the last year: reference servers, community
servers, npx one-liners. What stayed the same: nothing sandboxes them. The
protocol doesn't, the installers don't, and the docs never mention it. I
started Warden as a design skeleton and kept it honest by pinning every claim
to a test — the compatibility matrix, the escape tests, the fail-closed error
messages. The roadmap is M0–M8, all shipped: Linux, macOS, Windows, Docker
fallback, egress proxy, trace/init policy generation, approval mode, and a
beta program that asks the community to file friction reports. It's pre-1.0,
freely licensed, and the security review in the repo lists what it does not
do yet — because a security tool's credibility is the parts it refuses to
claim." (All elements traceable to `ROADMAP.md`, `ARCHITECTURE.md`,
`docs/security.md`, `docs/beta.md`.)

**Product Hunt gallery (8 images)** — all visuals must use repo-verified
claims. The existing `warden-landing/product-hunt-gallery/` renders 01–06;
07–08 are additions.

```
01 — Hero
  Headline: "Secure execution for MCP servers."
  Supporting: "Give AI tools only the access they actually need."
  Visual: sandbox boundary around an MCP server (dark UI, granted=green,
          denied=red) — exists as 01-hero.html
  CTA: "Install" (npm / GitHub Releases)

02 — Problem
  Headline: "MCP has no concept of a boundary."
  Supporting: "A server you cloned five minutes ago runs with your full
               user permissions. ~/.ssh, $ENV, every host."
  Visual: dashed red lines from a server box to ~/.ssh, $ENV, /etc, *.internal
  CTA: "Run `warden trace` on your server"

03 — Policy
  Headline: "One policy file. That's the trust boundary."
  Supporting: "Grant a folder, a hostname, a handful of env vars. Everything
               else doesn't exist." — literal policy.yaml annotated
  Visual: code card with highlighted blocks (filesystem/network/env/limits)
  CTA: "Copy the example policy"

04 — Proof (REBUILT from verified evidence)
  Headline: "The boundary is tested, not assumed."
  Supporting: "Fail-closed refusal: 'no sandbox backend available…
               refusing to run unsandboxed'. Escape tests in CI: unlisted
               paths invisible, denied hosts never resolved, empty env list
               = empty environment." — REPLACE the current 88ms/25K table
  Visual: test-pass table + refusal terminal card
  CTA: "Run the test suite yourself" (go test ./...)

05 — Backends
  Headline: "One policy, three native backends."
  Supporting: "Linux bubblewrap · macOS Seatbelt · Windows AppContainer +
               WFP + ETW · Docker fallback. Auto-selected, never silent."
  Visual: backend status table (Linux Verified / macOS+Windows verification
          pending / Docker fallback)
  CTA: "Read the architecture doc"

06 — Install
  Headline: "Thirty seconds to a sandboxed server."
  Supporting: "npm install -g warden-sandbox-cli" / GitHub Releases /
               build from source
  Visual: terminal install + `warden run --policy policy.yaml`
  CTA: "Install now"

07 — Compatibility (add)
  Headline: "Tested against 18 real MCP servers."
  Supporting: "14 pass, 2 conditional, 2 fail — and we publish why."
  Visual: compat matrix table with pinned policy links
  CTA: "Check your server"

08 — Developer loop (add)
  Headline: "Trace it. Review it. Ship it sandboxed."
  Supporting: "`warden trace` records what a server touches; `warden init`
               writes a conservative starter policy; `warden logs` shows
               what it tried."
  Visual: three terminal panes (trace → init → logs)
  CTA: "Start with `warden doctor`"
```

---

## 10. Landing Page

**Hero**
- Headline: *Your MCP servers don't need your whole filesystem.*
- Subheadline: Warden runs them in a sandbox that only sees what you grant —
  a folder, a hostname, nothing more. Everything else fails, and Warden writes
  down that it tried.
- CTA primary: `npm install -g warden-sandbox-cli` (copy button)
- CTA secondary: `View on GitHub`
- Proof strip: "Deny by default · Fail closed · Audit everything · 18 servers
  tested"

**Problem**
MCP has no concept of a boundary. A server you installed five minutes ago
runs with your full user permissions by default — every file you can read,
every host you can reach. Nothing in the protocol stops it.
(Existing copy is accurate; keep `Problem.tsx`.)

**How Warden works** (six beats, all verified):
1. **Policy** — `policy.yaml` (`command`, `filesystem.read/write`,
   `network.allow`, `env.allow`, `limits.memory_mb/timeout_s`); unknown keys
   are rejected; no wildcard hosts. (`policy.go`, `schema.md`)
2. **Sandbox** — auto-selected backend: bubblewrap / Seatbelt / AppContainer /
   Docker. No usable backend ⇒ refuse to run. (`backend.go`)
3. **Filesystem** — granted paths only, read-only or read-write; everything
   else invisible. (`linux.go`, escape test)
4. **Network** — hostname allowlist enforced by a local proxy before DNS;
   denied hosts are never resolved; no default route out of the sandbox.
   (`proxy.go`)
5. **Environment** — only `env.allow` names forwarded; empty list = empty
   environment. (`envfilter.go`)
6. **Fail closed** — every backend layer must initialize or Warden refuses.
   (`sandboxerr.go`, windows gates)

**Proof** — use ONLY verified results:
- Test suite: 256 tests run, 255 pass, 1 known failure (Docker integration)
  on the reference host — or phrase as "Linux escape tests pass; Windows and
  macOS escape suites run in CI" until the suite is all-green.
- Linux escape tests pass: read grant OK, unlisted path invisible, write
  outside grant denied, env passthrough filters, bwrap-missing refuses.
- 18-server matrix with pinned fixtures.
- **Remove the current attack-simulation tables (88ms/25K/etc.) or
  re-verify them first.** A security tool cannot show unverifiable numbers.

**Compatibility** — see §6 matrix. Mark Linux Ready; macOS/Windows "code
complete, verification pending"; Docker "fallback, integration test under
triage" until fixed.

**Developer experience** — install (3 commands), `warden doctor`, `warden
trace -- …`, `warden init`, `warden run --policy policy.yaml`, `warden logs`.

**Open source** — MIT; single repo; docs site on GitHub Pages; issue
templates including a compatibility-report template.

**Roadmap** — only what ROADMAP.md + REMAINING_WORK.md say: M0–M8 shipped;
open: Homebrew tap, Windows CI rerun acceptance, schema gaps (wildcards,
unix sockets), hardening milestones in `future.md` (uncommitted).

---

## 11. LinkedIn

**Founder-style launch post**
> I kept installing MCP servers that had the same "feature": they could read
> my SSH keys, my .env, my whole home directory. The MCP protocol doesn't
> require process isolation, so the default for most servers is "run with the
> user's full permissions."
>
> So I built Warden — an open-source sandbox runtime for MCP servers.
>
> You declare what a server may touch in one YAML policy. A folder. A
> hostname. A few environment variables. Warden enforces that at the OS level
> — bubblewrap on Linux, Seatbelt on macOS, AppContainer + WFP + ETW on
> Windows — and audits every access attempt.
>
> If it can't apply a real sandbox, it refuses to run the server. Never
> silently unsandboxed.
>
> It's pre-1.0, MIT-licensed, and I'd genuinely value you trying to break it:
> → [repo link]

**Technical post**
> How Warden's network block actually works: hostname allowlist enforced by a
> local proxy that checks the destination BEFORE any DNS lookup — so a denied
> host is never even resolved, and there's no DNS-leak channel. The sandbox's
> network namespace has no default route; the only way out is the proxy.
> On Windows that's a WFP sublayer permitting exactly one loopback endpoint
> and blocking everything else outbound.
> Worth noting what it does NOT do yet: the proxy filters HTTP/HTTPS, CPU
> limits don't exist, and unprivileged-userns-disabled hosts can widen the
> network boundary. All of this is documented in the repo's security review.

**Short post**
> Your MCP servers don't need your whole filesystem. Warden gives them a
> boundary, an audit log, and a policy file. Open source, pre-1.0,
> fail-closed. [repo]

**CTA:** Try `warden trace -- <your-server>` and see what it actually touches.
File a compatibility report if something breaks — that's the signal we want.

---

## 12. X (Twitter)

**Single launch post**
> MCP servers run with your full user permissions right now.
> Warden sandboxes them: one YAML policy, OS-level enforcement
> (bwrap / Seatbelt / AppContainer), audit log on every attempt,
> fail closed if it can't sandbox. Free & open source.
> github.com/Prof-bilal/Warden

**Thread (7–10 tweets)**
1. The MCP servers you `npx` every week run as plain processes with your
   user's permissions. Nothing in the protocol stops them. This is the
   problem Warden exists for.
2. The fix: a small YAML policy. `filesystem.read`, `network.allow`,
   `env.allow`. Deny by default — nothing is reachable unless you grant it.
3. Under the hood it's OS primitives, not magic: bubblewrap namespaces on
   Linux, sandbox-exec on macOS, an AppContainer token + WFP + ETW on Windows.
4. Network: a local proxy checks hostnames against the allowlist BEFORE DNS.
   Denied hosts are never resolved. No leakage via DNS queries.
5. Filesystem: granted paths only. Ungranted paths don't exist inside the
   sandbox — not "permission denied", actually invisible.
6. Environment: only what you list in `env.allow` crosses the boundary.
   Empty list = empty env. Your shell stays yours.
7. If Warden can't apply a sandbox it refuses to run. The refusal is a
   tested feature, not an error.
8. Real-server reality check: 18 MCP servers in the compat matrix — 14 pass,
   2 need per-deployment hosts, 2 can't be sandboxed honestly. We publish
   the why.
9. Try it in 3 commands: `warden doctor` → `warden trace -- node server.js`
   → `warden run --policy policy.yaml`. PRs and breakage reports welcome.
## 13. Reddit

**Title (r/MCP, r/LocalLLaMA, r/netsec, r/commandline):**
`I sandboxed every MCP server I run — here's the tool I'm building (open source, pre-1.0)`

**Technical post (show, don't shill):**
> I got tired of the gap between "MCP is great" and "an MCP server is just a
> process with my user's permissions". The protocol doesn't isolate anything,
> so I've been building Warden — a sandbox runtime for MCP servers.
>
> How it actually works (no marketing): you write a `policy.yaml` listing
> `filesystem.read`, `filesystem.write`, `network.allow` (bare hostnames only,
> no wildcards), `env.allow`, and `limits`. Warden translates that into OS
> primitives: bubblewrap with unshared namespaces on Linux, `sandbox-exec` on
> macOS, and on Windows an AppContainer token + WFP egress filters + Job
> Object + ETW audit. Docker `--network none --read-only` is the fallback.
>
> Details I think are worth discussing:
> - Denied hostnames are checked against the allowlist before DNS, so they're
>   never resolved at all (no DNS leak channel).
> - Ungranted filesystem paths are invisible, not "permission denied".
> - Empty `env.allow` = empty environment. Nothing inherited.
> - If no backend can be applied, it refuses to run. The refusal message is in
>   a tested path.
>
> Honest caveats (docs/security.md lists these): the egress proxy only filters
> HTTP/HTTPS; no CPU limits; if unprivileged userns are disabled the network
> boundary can widen; macOS Seatbelt is deprecated upstream; Windows needs an
> elevated shell.
>
> I also published the failures: the compat matrix covers 18 servers — 14 pass,
> fetch/kubernetes are conditional, docker-mcp and playwright are not
> sandboxable without schema changes (docker needs the daemon socket, which
> voids the sandbox).
>
> I'd like honest breakage reports more than stars: `warden trace -- <your
> server>` then `warden run --policy policy.yaml`. If something doesn't fit
> the schema, that's exactly the feedback the project is collecting.

**Discussion question:**
> MCP servers run with your full user permissions today. What would you grant
> a server you just installed from npm — and does a YAML allowlist feel like
> the right trust boundary, or is per-API-key scoping (OAuth-level) the only
> thing that actually fixes this? Where do you draw the line between sandbox
> and gateway?

---

## 14. Hacker News

**Show HN title (≤80 chars, no superlatives):**
`Show HN: Warden – sandbox for MCP servers`

**First comment (post immediately):**
> Warden is a single-binary CLI that runs MCP servers inside an OS-native
> sandbox. You write a policy granting the paths, hostnames, and env vars a
> server may use; everything else is denied at the OS layer.
>
> Stack: Go. Linux = bubblewrap (unshare user/ipc/pid/net, ro-bind /usr
> /lib64 + policy paths). macOS = sandbox-exec. Windows = AppContainer token
> with WFP egress filters, Job-Object limits, ETW audit (fails closed if the
> DLLs/session can't initialize). Docker --network none is the fallback.
> Egress = local proxy that checks hostnames before DNS.
## 15. YouTube

**Title options:**
1. `Sandboxing MCP servers, properly (Warden)`
2. `Warden: the MCP sandbox you should try before your next npx`
3. `How I sandbox untrusted MCP servers on Linux, macOS, and Windows`

**Description:**
Your MCP servers run as plain processes with your full user permissions.
Warden is an open-source sandbox runtime that changes that: a policy.yaml
grants the exact files, hosts, and environment variables a server needs, and
everything else is blocked at the OS boundary. This video walks through the
mechanism — bubblewrap namespaces, the pre-DNS egress proxy, environment
filtering, and fail-closed behavior — and the limits the project documents
openly. Repo: github.com/Prof-bilal/Warden. MIT, pre-1.0.

**Chapters:**
0:00 The problem / 0:30 What Warden enforces / 1:20 Filesystem + network
mechanics / 2:40 Environment + fail-closed / 3:20 Compatibility matrix and
known limits / 4:00 Try it

**Thumbnail text:** `Your MCP servers don't need your whole filesystem`
**Tags:** MCP, Model Context Protocol, sandbox, security, AI agents,
bubblewrap, WFP, egress proxy, developer tools, open source
**Pinned comment:** Which MCP servers are you running unsandboxed? The honest
answers are the ones this project needs — file a compatibility report if your
server doesn't fit.

---

## 16. Discord

**Launch announcement (#mcp #ai-agents #dev-tools):**
## 17. Video Scripts

**Rule: no unverifiable numbers.** Replacement proof points below are all
repo-verifiable. If the attack harness is later committed, re-add those
numbers with the harness.

### Product Hunt / Landing page — ~90–100s
- [0–8s] Gate-mark logo resolves. "Your MCP servers don't need your whole
  filesystem."
- [8–22s] Diagram: server box, dashed lines to `~/.ssh`, `$ENV`, `/etc`.
  "Right now most MCP servers run with your full permissions — every file you
  can read, every host you can reach. Nothing in the protocol stops them."
- [22–45s] `policy.yaml` card; blocks highlight as named. "Warden fixes that
  with one file. Grant a folder, a hostname, a handful of environment
  variables. Everything else simply doesn't exist to it."
- [45–62s] Terminal: `warden run --policy policy.yaml -- node server.js`; then
  a denied attempt card "read ~/.ssh/id_rsa → BLOCKED". "An unlisted file
  isn't permission-denied — it's invisible. A denied hostname isn't even
  resolved, so there's no DNS to leak."
- [62–78s] Refusal card: "Warden refused to start: sandbox backend
  unavailable… fails closed by design." "If it can't sandbox a server
  properly, it refuses to run it at all."
- [78–92s] Backend strip: Linux bwrap · macOS Seatbelt · Windows
  AppContainer+WFP+ETW · Docker fallback. "OS-native enforcement chosen
  automatically."
- [92–100s] "Free, open source, pre-1.0. Go break it." + install command.

### LinkedIn — ~60–90s (founder tone, sound-off captions)
- Same mechanics, but framed as the builder's story; final card:
  "It's free, open source, and honestly early. If you run MCP servers, I'd
  value you trying to break it." No invented history — the repo's own story
  (design skeleton → escape tests → compat matrix) is the narrative.

### YouTube / Hacker News — ~100–120s (technical, skeptical audience)
- Cover: mechanism (bwrap flags, bind mounts, no-route netns), egress proxy
  check-before-DNS, envfilter, fail-closed windows stack, and the "known
  limits" list from docs/security.md verbatim. End: "Read the security review
  before you trust it. That's the point."
- Show the compat matrix (18 servers, honest failures).

### Reels / TikTok / Shorts — ~45–65s
- Format without attack numbers: "3 things your MCP server can do right now"
  (read your SSH keys / call any host / read any env var) → "Warden: one
  policy file" → "deny by default" → "audit log" → "refuses to run if it
  can't sandbox". Kinetic type, captions burned in. Do NOT show the 88ms/25K
  numbers unless re-verified.

---
## 18. CLI Content — Ideal First-Run Experience

| Command | Purpose | Current behavior (verified) | Recommended presentation |
|---|---|---|---|
| `warden` | Landing/identity | Branded usage, exit 1; banner on TTY (source HEAD) | Keep; banner + 3 commands (`doctor`, `run`, `trace`) |
| `warden help` | Discovery | Usage text (HEAD) | Keep; group Commands / Backends / Examples |
| `warden --help` | Same as help | Usage (HEAD) | Same output as `help` |
| `warden --version` | Build identity | `warden version v0.1.x`; `dev` unstamped (HEAD) | Keep; add `(pre-release)` suffix when `dev` |
| `warden init` | Scaffold policy | Generates non-overwriting starter policy (HEAD banner) | Show "next": `warden run --policy policy.yaml` |
| `warden doctor` | Readiness check | Checks backend/namespaces/proxy/policy (HEAD banner) | Show a table + a copy-paste fix line per red row |
| `warden run` | The core verb | Runs sandboxed, per-command help (HEAD) | Show policy path + backend chosen + audit path on exit |

Note: published release binary (`v0.1.7`) predates the polished help system
in source HEAD — release v0.1.8+ before onboarding copy goes live.
> Warden v0.1 — sandbox runtime for MCP servers. Deny-by-default policy.yaml
> (files / hosts / env), OS-native backends (bwrap, Seatbelt, AppContainer),
> pre-DNS hostname allowlist, audit log, and fail-closed refusal when no
> backend applies. Tested against 18 real servers (14 pass). Pre-1.0, MIT.
> github.com/Prof-bilal/Warden

**Tester recruitment message (#beta):**
> Looking for MCP server maintainers/users to run their own servers under
> Warden and file friction reports: `warden trace -- <your server>` →
> review the generated policy → `warden run --policy policy.yaml` → open a
> GitHub issue with the "Compatibility report" template. Reports are public
> and that's the point — the schema needs testing by people who didn't design
> it. (docs/beta.md has the full program.)

**Technical discussion message:**
> Thought experiment for the room: Warden's network allowlist is hostname-only
> with no wildcards, which makes browsers (Playwright) unsandboxable and
> makes the "grant" model dead simple. What's the right trade for AI agents —
> scoped wildcard hosts with an audit warning, or a different primitive like
> per-request approval in the gateway? The repo tracks this as an open schema
> gap (`matrix.yaml` → playwright).
>
> Honest limits (docs/security.md): proxy filters HTTP/HTTPS only; no CPU
> throttling; userns-disabled hosts can widen the network boundary; Windows
> and macOS backends are code-complete but my real-hardware verification
> covers Linux so far; one Docker integration test currently fails under the
> test harness (CLI path verified). I'm not publishing unverifiable attack
> numbers — if a claim isn't in the repo with a test, it isn't on the page.
>
> Compat: 18 servers in the matrix, 14 pass, 2 conditional, 2 fail by design.
>
> Try: `warden doctor` → `warden trace -- node server.js` → `warden run
> --policy policy.yaml`. Build from source (Go 1.22) or grab the release
> binaries. Pre-1.0, MIT.
    UNVERIFIED` — remove or rebuild with a committed harness.
## 19. README (advisory rewrite, root README)

Current root README is stale ("pre-alpha / design skeleton"). Recommended
replacement structure (all claims verified / from `warden-starter/warden`):

**Headline:** `# Warden — A lightweight sandbox runtime for MCP servers`
**One-line:** *Deny by default. Grant a folder, a hostname, a few env vars.
Audit everything else.*
**Quick demo:** `warden run --policy policy.yaml -- node server.js`
**Install:** GitHub Releases (5 binaries + SHA256SUMS) · npm
`warden-sandbox-cli` · build from source (Go 1.22). Note Homebrew tap PENDING.
**Quick start:** 4 commands: `warden doctor` → `warden trace -- node server.js`
→ `warden init` → `warden run --policy policy.yaml`; then `warden logs`.
**Policy example:** filesystem read/write, network.allow, env.allow, limits
(no wildcard hosts; read/write overlap coalesces to write).
**Architecture:** diagram + one-paragraph mechanism per backend; link
`ARCHITECTURE.md`.
**Security model:** four bullets (filesystem invisible / network pre-DNS /
env allowlist / fail-closed) + link `docs/security.md` known limitations.
**Testing:** `go test ./...` (reference host: 256 tests, 1 known Docker
integration failure — triage in progress); escape tests per backend; CI.
**Platform support:** matrix with honest statuses (Linux verified; macOS/
Windows code-complete, verification pending; Docker fallback).
**Limitations:** the exact list from `docs/security.md` §Known Limitations.
**Roadmap:** link `ROADMAP.md` (M0–M8 shipped) + `REMAINING_WORK.md`.
**Contributing:** link CONTRIBUTING.md; first tasks: compat reports, policy
fixtures, backend testing on macOS/Windows.

---

## 20. Website Copy (content structure)

- **Hero** — headline: *Your MCP servers don't need your whole filesystem.*
  Subhead + install CTA + GitHub CTA (see §10).
- **Problem** — "MCP has no concept of a boundary." Body: default install =
  full user permissions; nothing in the protocol stops a buggy/malicious
  server. Visual: boundary diagram (exists: `Problem.tsx`).
- **Solution** — "A boundary, in one YAML file." 3 cards: Files / Network /
  Environment, each "granted, not guessed" + audit.
- **How it works** — 6-step: policy → backend → filesystem → network → env →
  fail-closed, each step annotated with the enforcing file (see §5).
- **Policy example** — literal `policy.yaml` with annotations; "try it" box.
- **Attack demonstration** — **replaced by verified evidence**: live
  `warden run` terminal capture + audit log showing blocked reads/connects;
  "escape tests in CI" callout. No unverified numbers.
- **Architecture** — backend diagram + mechanism table.
- **Compatibility** — 18-server matrix (14/2/2) with honest statuses.
- **Proof** — test suite status, escape tests, pre-DNS block, fail-closed
  refusal screenshot.
- **CLI** — command reference with copy buttons.
- **Open source** — MIT, repo, issue templates (incl. compat report).
- **FAQ** — §21.
- **CTA** — "Try it against a server you didn't write."
- **Footer** — MIT · GitHub · Docs · Roadmap · About.

---

## 21. FAQ

1. **What is Warden?** A sandbox runtime for MCP servers — a single binary
   that runs a server inside an OS-enforced boundary defined by a YAML policy.
2. **What is an MCP server?** A process that implements the Model Context
   Protocol so AI clients (Claude, IDEs, agents) can expose tools/data. It
   runs locally, usually as `node`/`python`, spawned by the client config.
3. **Why sandbox MCP servers?** Because they run with your full user
   permissions and nothing in the protocol restricts them — a buggy or
   malicious server can read `~/.ssh`, call any host, or rewrite files.
   (`README.md` "The problem".)
4. **Does Warden replace Docker?** No. Docker is one of Warden's backends
   (fallback). On Linux/macOS/Windows Warden prefers OS-native sandboxing so
   the trust boundary is smaller and there's no daemon dependency.
5. **How does filesystem isolation work?** Linux: bwrap bind-mounts only
   granted paths; unlisted paths are invisible. macOS: Seatbelt profile.
   Windows: AppContainer token denies everything; policy grants become
   capability DACL entries. (`docs/security.md`.)
6. **How does network isolation work?** A local proxy checks the destination
   hostname against `network.allow` before any DNS lookup; the sandbox has no
   default route (Linux/macOS) or WFP blocks everything but the proxy
   (Windows). Limitation: the proxy filters HTTP/HTTPS only. (`proxy.go`,
   `docs/security.md:152`.)
7. **What happens when sandboxing fails?** Warden refuses to run — "no sandbox
   backend … refusing to run unsandboxed". Every Windows layer must initialize
   or the run is refused. (`backend.go`, `windows/run.go`.)
8. **What platforms are supported?** Backends exist for Linux (bwrap),
   macOS (Seatbelt), Windows (AppContainer/WFP/ETW), and Docker. Verified on a
   real machine today: Linux. macOS/Windows verification is pending. See §6.
9. **Is Warden production-ready?** No — it's pre-1.0 (M0–M8 shipped). One
   Docker integration test currently fails under the test harness; Windows and
   macOS need real-hardware verification before "Ready" claims.
10. **Has Warden been independently audited?** No. Security is documented in
    `docs/security.md`, and the test suite includes escape tests, but there is
    no third-party audit or certification.
11. **Can a malicious MCP server bypass Warden?** We can't claim an absolute.
    The threat model and known gaps are documented (`docs/security.md`):
    e.g., symlinks inside grants, HTTP/HTTPS-only proxy, userns-disabled
    hosts, no CPU limits. The fail-closed path is tested; the boundary is not
    formally proven.
12. **Is Warden open source?** Yes — MIT, single repo, GitHub.
13. **How do I contribute?** Compatibility reports (template in repo),
    policy fixtures for more servers, backend testing on macOS/Windows, and
    docs. `CONTRIBUTING.md`.
14. **Which MCP servers work?** 18 tested — 14 pass, 2 conditional (fetch,
    kubernetes), 2 fail by design (docker, playwright). Full matrix with
    exact policies in `docs/compatibility.md`.
15. **Why can't Warden block everything?** Wildcard hostnames and unix-socket
    grants don't exist in the schema yet (Playwright-class servers); granting
    the Docker socket would void the sandbox. These are documented schema
    gaps, and the matrix shows them honestly.

---

## 22. Public Claims Audit

### SAFE CLAIMS (publish as-is)
| Claim | Evidence | Confidence |
|---|---|---|
| Warden runs MCP servers inside an OS-enforced sandbox | `backend.go`, backends | High |
| Deny-by-default: unlisted paths/hosts/env are blocked | `policy.go`, escape tests | High |
| Denied hostnames are not resolved (checked before DNS) | `proxy.go:1–4` | High |
| Ungranted paths are invisible, not permission-denied | Linux escape test | High |
| Empty env allowlist ⇒ empty environment | `envfilter.go` + test | High |
| Warden refuses to run when no backend applies (fail closed) | `backend.go`, `sandboxerr.go`, test | High |
| Enforces memory + wall-clock limits (Linux) | `limits.go` + tests | High |
| Tested against 18 real MCP servers; 14 pass / 2 conditional / 2 fail | `matrix.yaml`, fixtures | High (fixtures); Medium (live verdicts manual) |
| MIT-licensed, open source, single binary | LICENSE, dist/ | High |
| Linux backend verified today; macOS/Windows code-complete, verification pending | execution + code | High |
| Audit log of allowed/blocked attempts | `audit.go` | High |

### QUALIFIED CLAIMS (need context/disclosure)
| Claim | Required disclosure |
|---|---|
| "256 tests, 255 pass" | Add "on reference Arch Linux host, 2026-09-08; 1 known Docker integration failure in triage" |
| "Windows AppContainer backend" | "Code-complete; escape tests execute in Windows CI; real-hardware verification in progress (see REMAINING_WORK.md)" |
| "Blocks ransomware/credential theft" (the Proof tables) | Only allowed IF a committed harness + fixtures reproduce them; otherwise DO NOT USE (see below) |
| "No DNS leakage" | True for proxy path; qualify "for HTTP/HTTPS egress; raw sockets rely on no-route/WFP" |

### DO NOT CLAIM
- Any attack-simulation number (88ms, 33ms, 25K attempts, 6 SSH keys, 59 env
  vars, 12.2M hashes, "5/5 contained", "ten for ten") — no reproducible
  evidence exists in the repo.
- "Independently audited", "certified", "pen-tested by a third party", "the
  most secure", "the first MCP sandbox" — none supported.
- "Production ready" / "Ready (macOS/Windows)" — verification pending; also
  one test currently fails.
- "Warden is a firewall for all traffic" / "blocks raw TCP/UDP via the proxy"
  — proxy is HTTP/HTTPS only.
- "Works on any Linux without setup" — it needs bwrap (or Docker), strace for
  full audit, and fails loudly otherwise.

---

## 23. Launch Asset Checklist

**P0 (blocking):**
- [ ] Triage + fix `TestDockerBlocksUngrantedRead` (or document a clean skip);
      make `go test ./...` green.
- [ ] Remove unverified attack numbers from `Proof.tsx` /
      `product-hunt-gallery/04-proof.html`/`marketing.md`, or commit a
      reproducible harness + fixture data.
- [ ] Ship release ≥ v0.1.8 with the polished help system + CLI identity
      (source HEAD is ahead of published binaries).
- [ ] Update root `README.md` (stale pre-alpha claims) to match
      `warden-starter/warden/README.md`.
- [ ] Windows: land green CI rerun after `eadca83`+`f2232c2` and record it.
- [ ] Landing page backend statuses: Linux Ready; macOS/Windows
      "verification pending"; Docker fallback "triage in progress".

**P1 (should):**
- [ ] macOS smoke test on real hardware (Seatbelt backend + Docker fallback).
- [ ] Homebrew tap decision (generate + publish, or remove from copy).
- [ ] Fix `docs/install.md` npm references (package name is
      `warden-sandbox-cli`, already live).
- [ ] Commit `marketing.md`/gallery behind a tracked `docs/` home or delete.
- [ ] First-run experience pass per §18 (banner, doctor table, next hints).

**P2 (nice):**
- [ ] Benchmark overhead honestly (bwrap start time vs docker) to replace
      "near-zero overhead" with a number.
- [ ] Publish the 18-server matrix as a rendered page on GitHub Pages.
- [ ] Add a "known limits" page on the landing site mirroring `docs/security.md`.
- [ ] Screenshot/terminal-cast of `warden run` + `warden logs` for gallery.

---

## 24. Recommended Launch Strategy

1. **Fix, then launch (sequential, ~2–3 weeks):** (a) all-green test suite,
   (b) evidence-backed Proof section, (c) fresh release with the current
   source HEAD, (d) macOS + Windows smoke results recorded.
2. **Channel order:** Hacker News `Show HN` first (technical audience, honest
   = rewarded, no signup), then a LinkedIn founder post ~48h later, Discord
   communities and Reddit for breakage reports, Product Hunt only after the
   landing page and evidence are solid (it's an amplifier, not a primary
   channel for dev infra).
3. **Everywhere:** repeat the same 6 claims (§22 SAFE), link the security
   review, and ask for breakage reports rather than stars. Never paste an
   attack number into a channel-specific post that isn't verified on the
   landing page.
5. **Post-launch loop:** file every incompatibility as a public issue with
   the compat template; publish the triage (warden-bug vs schema-gap vs
   inherent) — that cadence is what the roadmap's M8 process was built for.

---

## Final Recommendation

**PRODUCT READINESS: 6/10** — Real, multi-backend implementation with passing
escape tests on Linux and hundreds of tests, but one failing test, no recent
release of HEAD, and uncommitted changes in the working tree.
**MARKETING READINESS: 4/10** — Strong brand/positioning drafts exist, but
the Proof section contains unverifiable numbers that must be removed or
rebuilt, and docs are internally contradictory.
**SECURITY-EVIDENCE READINESS: 5/10** — Excellent honest threat model and
fail-closed design; weak on reproducible attack evidence (none in repo) and
missing real-hardware verification for 2 of 3 platforms.
**DOCUMENTATION READINESS: 5/10** — Deep docs and a docs site, but the root
README and install docs are stale and contradictory.
**LAUNCH READINESS: 4/10**

1. **Should Warden launch now?** Not in its current state. The product is
   real and the story is good — but a security tool with a failing test in
   its own suite and unverifiable proof numbers will get eaten alive by the
   exact audience that would adopt it. Two to three weeks of tidying turns a
   risky launch into a strong one.
2. **Strongest thing about Warden:** The fail-closed discipline — a tested
   product rule ("refuse to run rather than run unsandboxed") enforced across
   four backends, plus a strikingly honest `docs/security.md` that publishes
   its own gaps. Those two together are more credible than any attack table.
3. **Biggest weakness:** Unverifiable attack-simulation claims presented as
   proof, while the only reproducible security evidence is boundary tests —
   and one of the integration tests currently fails. The gap between claim
   and evidence is the risk.
4. **Claim to absolutely NOT make:** Any attack-simulation number (88ms, 25K,
   6 SSH keys, 59 env vars, "0 bypasses"). There is no harness or data in the
   repository to back it, and on Hacker News that's a reputation-ender.
5. **Fix before Product Hunt:** Remove/re-verify Proof numbers; make the test
   suite green; publish a fresh release from source HEAD; align landing page
   backend statuses with reality.
6. **Fix before Hacker News:** All of the above plus a genuinely honest first
   comment (windows/macOS verification status, Docker test status, known
   limits). HN rewards that posture; a "Ready on all platforms" claim without
   evidence will be shredded.
7. **Fix before asking strangers to attack it:** Stand up a committed,
   reproducible attack-test harness with fixtures (even a simple one: block
   reads/connects/env-thief scripts + recorded results in CI), plus the macOS
   and Windows real-hardware runs. You cannot ask "try to break it" and
   simultaneously have no reproducible attack suite in the repo.
8. **Single highest-leverage improvement:** Get `go test ./...` green and
   publish a release from current HEAD, then replace the Proof section with
   the *real* verified story (fail-closed refusal, pre-DNS blocking, invisible
   files, 18-server matrix). Those four fix the credibility gap that
   currently blocks everything else.
