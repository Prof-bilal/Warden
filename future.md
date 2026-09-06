# Warden — Future Growth Plan (Research-Grounded, Pre-1.0)

> **Status:** `future.md` is an untracked strategy sketch. It consolidates four domains — marketing, scaling, secrets guard, monetization — each researched independently with current (2026) web sources before drafting. Where a claim has no source, it is marked **Open question**. Grounded in what Warden actually is today: `M0–M8` shipped (Linux `bwrap`, macOS Seatbelt, Docker fallback, Windows AppContainer/WFP/ETW/Job Objects, egress proxy, `trace`/`init`/`logs`, limits, approval mode `--approve`, gateway integration, compatibility matrix 18 servers: 14 pass / 2 conditional / 2 fail), beta program defined, distribution via GitHub Releases + npm wrapper + Homebrew tap formula. See `warden-starter/warden/README.md:1`, `warden-starter/warden/ARCHITECTURE.md:1`, `warden-starter/warden/ROADMAP.md:1`, `warden-starter/warden/docs/compatibility.md:1`, `warden-starter/warden/REMAINING_WORK.md:1`.

**How to use:** This doc is **not** a commit to build everything below. It is the sequencing and evidence layer. Items tagged `Validated` have a source finding; `Speculative` are bets requiring beta signal. Most monetization work is `"don't build yet"`.

---

## 1 — Marketing & Content Strategy

### Positioning (one line)

> **Warden is a single-binary sandbox that lets you run any MCP server with the filesystem, network, and env it actually needs — nothing more — with `deny by default`, transparent `stdio`, and an auditable log.** For: a developer who installs third-party MCP servers locally and wants `policy.yaml` to be the trust boundary, not the host OS. Not for: platform-team multi-tenant gateways (explicit non-goal `ARCHITECTURE.md:182`).

Audience: technical, security-conscious, runs `npx`/`uvx` servers locally (`PRD.md:7`). Top adopters will be: solo OSS users with secrets on disk, teams standardizing 3–5 MCP servers, MCP gateway users (`docs/gateway.md:1`).

### Primary vs secondary launch channels (ranked by fit)

| Rank | Channel | Why for *this* audience | Current mechanics (2026) | Source |
|---|---|---|---|---|
| **1** | **Hacker News `Show HN`** | Highest trust, zero ads, rewards technical depth. Warden is a dev infra tool that can be tried (`warden run`) without signup — perfect `Show HN` object. | Title must be literal `Show HN: Warden – sandbox for MCP servers` ≤80 chars, no superlatives/`!`/emoji. **First comment mandatory** within ~1 min: what it is / why you built it / stack (bwrap/Seatbelt/WFP) / honest limits / how to try (`go build` or release binary). No vote solicitation — violates guidelines and nullifies votes. Gravity ≈1.8; early velocity in first 1–2 hours decides front-page escape from `/newest`. Best window weekday morning 7–10am ET (12–15 UTC) or Monday 00:00 UTC (Sunday 7pm ET) per 2021–2026 cohorts; post volume now ~200 Show HNs/day. | `hackmamba` Show HN pack `showhn.md:1` + HN guidelines `showhn.html` + `danfking.github.io/blog/2026/04/23/show-hn-by-the-numbers` (188k posts, median 2 pts, 1.4 stars/upvote, 24h half-life) |
| **2** | **Product Hunt** | Broader discover +press spillover, but generalist. Warden will underperform there unless pre-warmed. Pair as amplifier, not primary. | Day resets 12:01am PST; schedule 1 month ahead. Need tagline ≤60 chars, description ≤500 chars, 2+ gallery images (1270×760, <3MB), optional YouTube link, up to 3 launch tags, pricing (free). **70%** of Product of Day had maker first comment. Algorithm (2026) weights **comments/maker replies/time-on-page** over raw upvotes; bringing new-to-PH users helps. No pay-for-upvotes/bounty — banned. Hunter self-hunt is fine (no advantage to third-party hunter). No `?utm` in URL. Prepare 6 weeks: 2–3 wks audience + 1–2 wks assets + 24h live. | `producthunt.com/launch/how-product-hunt-works` + `producthunt.com/launch/preparing-for-launch` + `launchory.app/blog/how-to-launch-on-product-hunt` (2026-07-25) |
| **3** | **DevHunt** | Devtool-specific, complement to PH. Smaller, higher-signal for Warden. | **GitHub PR-based submissions** + GitHub-login voting (harder to farm). Open-source MIT, Next.js+Supabase, self-hostable. PR review latency (not instant). DR ~63, paid expedited $49. Volume is low — won't drive PH-level traffic alone. | `runany.dev/blog/devhunt-launch-platform` (2026-07-11) + `launchdirectories.com/directory/devhunt` + `devhunt.org/blog/tag/launch` |
| **4** | **BetaList / niche directories** | Cheap durable SEO/backlink footprint. Lower fit than DevHunt but useful as compounding layer after PH. | Listed as startup directory; schedule as post-launch spread across 100+ directories only after primary launches — “single launch fades in 48h; broad footprint compounds.” | `launchory.app` spillover checklist |
| — | **Reddit r/ClaudeAI, r/devops, Slack/Discord MCP communities, daily.dev/TLDR newsletters** | Where MCP adopters actually ask “how do you sandbox X?” — answer-first distribution. | Distribution must be value-first: tutorials, benchmark, failure analysis. Sponsored newsletter slots in TLDR/daily.dev convert narrowly; display/ads mostly wasted on devs. | `hackmamba.io/technical-content/technical-content-marketing` + `docsio.co/blog/developer-marketing` + `hackercontent.com/blog/cybersecurity-content-marketing` |

**What *not* to do for this audience:** Generic SaaS tactics — LinkedIn display, gated whitepapers, AI-templated listicles — stopped ranking/converting in 2026. Security buyers sniff ghostwritten filler in seconds and punish credibility. See `otrenix.com/cybersecurity-content-marketing-guide` (4–8 substantial 2–4k word pieces/month, senior security writer 5–10× pipeline vs junior/AI).

### Pre-launch checklist (audience before assets)

- [ ] **Maker profile ≥30 days old**: real photo/bio, weekly genuine upvotes/comments so algorithm doesn't treat launch as drive-by (`launchory.app` cheapest leverage).
- [ ] **Warm list 200+**: waitlist / `notify me` page fed by newsletter, X/Twitter, Slack/Discord MCP channels, personal network. Goal = first-hour visitors who comment/ask.
- [ ] **GitHub proof**: `warden-starter/warden/README.md:15` quickstart must actually work (`warden run --policy …` + `trace`/`init`); polished README as landing page; issues triaged; `REMAINING_WORK.md:14` Windows ETW/WFP green so claim matches reality.
- [ ] **Demo-ability**: one-command install (`go build` or release binary), 5 example policies (`examples/`, `docs/compatibility.md:43`), `warden trace` replay on a popular server (filesystem/GitHub).
- [ ] **DevHunt PR ready** alongside PH assets (description, screenshots, founder story) so PR can be merged before live day.

### Launch-day checklist (24h playbook)

| Time (PST) | Action |
|---|---|
| **T-1d** | Assets final: ≤60-char tagline (“Sandbox for MCP servers — only the files, hosts, env you allow”), ≤500-char description, 240×240 thumbnail + ≥2× 1270×760 gallery, 30–60s Loom demo, first comment draft (story: “MCP runs with full user perms — SSH keys, exfil — nothing stops it; we built thin CLI over bwrap/Seatbelt/AppContainer”). |
| **12:01am PST Tue/Wed/Thu** | PH scheduled live. Immediately notify warm list: “We're live — check it out & leave a question” (never “upvote”). |
| **+0–1h** | Post `Show HN: Warden – sandbox for MCP servers` (literal, ≤80 chars) + first comment within 1 min. |
| **+0–4h** | Founder live in both threads: reply to every comment within minutes, AMA style. Time-on-page + reply depth are PH ranking signals (`launchory.app`). |
| **+2–18h** | Push to X thread, LinkedIn (insight not article), relevant subreddits/communities where self-promo allowed — link to PH/HN thread, not just site. |
| **+18–24h** | Last-chance nudge to warm list. |
| **+2d** | Add PH badge to README + landing, email onboarding sequence, list on DevHunt + niche directories for durable backlinks. |

### Recurring content plan — use what Warden already produces

No invented content from nothing. Cadence: **4–8 substantial pieces/month** max (security buyer model), each 2–4k words, technically correct, SME-interviewed, code examples validated.

| Source byproduct | Content piece | Funnel stage | Distribution |
|---|---|---|---|
| **Compatibility matrix** `docs/compatibility.md:17` + `testdata/compat/matrix.yaml:540` | “We tested 18 MCP servers — 14 pass, 2 conditional, 2 fail. Here's the exact policy each needs” (pillar) + 18 spoke deep-dives; “Playwright needs wildcard hosts — we don't have them yet” | Evaluation / Retention | HN + DevHunt + `hackmamba.io` pillar/spoke SEO, YouTube walkthrough, AI citation surface |
| **New backend** `ARCHITECTURE.md:66` | “Linux bwrap vs macOS Seatbelt vs Windows AppContainer: why one policy, three enforcers” (behind `Backend` interface) | Awareness | Blog + `hackercontent.com` SME interview (engineer explains tradeoffs) |
| **Beta friction** `docs/beta.md:31` | Monthly “Friction log” from compat reports (wnie `inherent` vs `schema-gap` vs `warden-bug`); close-the-loop comment on reporter issue | Evaluation | GitHub Discussions, Reddit, newsletter |
| **Roadmap milestone** `ROADMAP.md:155` | Milestone post per M9/M10 (what shipped, what failed, limits lie) | Retention | Changelog + release notes + docs site search |
| **`trace`/`init` byproduct** `ARCHITECTURE.md:140` | Tutorial: “Trace your MCP server in 5 min → `warden init` → tighten” | Activation | YouTube (2nd-largest search) + README quickstart |
| **Security review** `docs/security.md:142` | “7 threat categories + known limitations table” expanded into threat-report series | Awareness | Threat reports generate backlinks/press (`otrenix.com`) — repurpose one research cost into blog + talk + PDF + threads |

**Writer model:** Interview engineer 30 min → technical writer drafts → engineer accuracy-check (non-negotiable gate). Avoid content-agency docs drift.

### Measurement (not vanity)

Track sign-ups (not MQLs), **time-to-Hello-World** (first successful sandboxed run), 7-day active, 30-day retention. For content: qualified organic traffic (Search Console), engagement depth (time/scroll/return), backlinks/citations. Expect SEO compounding 6–12 months; research/threat reports spike faster (`hackercontent.com`).

### Reading list (cited)

- Product Hunt — `how-product-hunt-works`, `preparing-for-launch`, `how-to-post-a-product` (rules, scheduling, thumbnail/gallery spec)
- Launchory — “How to Launch a Startup on Product Hunt (2026)” (6-week prep, engagement-weighted algorithm)
- HN — `showhn.html` guidelines + reverse-engineered gravity 1.8 (`aclanthology` pack / `hn.algolia` 188k dataset) + `danfking` timing study
- DevHunt — `runany.dev` PR-based model + DR 63 + Supabase stack
- Hackmamba — Technical content marketing (PLG, pillar/spoke, YouTube as AI citation)
- Docsio — Developer Marketing pillar (docs are highest-converting surface; open core / open standards / OSS libraries)
- HackerContent — Cybersecurity Content Marketing (credibility gate, 30-min interview model, repurpose ratio)
- Otrenix — Cybersecurity strategy (4–8 pieces/month, senior writer 5–10× pipeline, 6–12 mo horizon)
- Hackmamba SEO/GEO/AEO — hub-and-spoke for topical authority, comparison pages cited 40.86% by LLMs

**Open questions (Marketing):**
- Whether PH audience overlaps enough with MCP adopters to justify mid-week slot contention vs HN-only launch. No source quantifies overlap.
- Current DevHunt maintainer PR review latency (no measured SLA found).
- Whether `warden-landing` Next.js docs should mirror new marketing posts for GEO.

---

## 2 — Scaling & Features — Post-M8 Roadmap

### Evidence base (2026)

- **MCP spec (2026-07-28)** says local servers **SHOULD** run sandboxed with minimal default privileges, using platform-appropriate tech (containers/chroot/app sandboxes), but **SHOULD not MUST** — `modelcontextprotocol.io/docs/2026-07-28/tutorials/security/security_best_practices`. This is the normative anchor for Warden.
- **NSA CSI (2026-06-02) `CSI_MCP_SECURITY.PDF`** reframes MCP risk as a **continuum** (protocol → agent → runtime → external services → long-term monitoring) and explicitly calls for OS-level isolation (AppContainer/seccomp/AppArmor/SELinux) + egress constraints, not just endpoint patches.
- **Academic/runtime work:** `aclanthology.org/2026.acl-industry.58.pdf` SHIELDMCP builds a **80+ technique / 14-tactic** threat taxonomy (SAFE-MCP/OpenSSF) + 3-stage runtime proxy (description integrity / param sanitization / response analysis) that cuts tool-poisoning 74%→9% and indirect prompt injection 47%→6% at <120ms median. Teaching: Warden's filesystem/network sandbox complements response-level defenses; neither subsumes the other.
- **Competitive sandbox approaches:** `sandboxreview.com` (2026-08-12) systematizes MCP isolation tiers: API-proxy servers may suffice with container + enforced network policy; **shell-execution/code-running servers need microVM (Firecracker/Kata/libkrun) or gVisor** kernel boundary; browser/file-manipulation servers are execution-capable regardless of name. Daytona delivers **<90 ms cold starts** for on-demand provisioning vs multi-second containers — relevance for agents spinning sandboxes per turn. Docker containerization alone adds restriction but lacks uniform policy, credential scoping, scaling controls.
- **MCP-SandboxScan (arxiv `2601.01241` WASM/WASI sandbox)** shows dynamic-only scanning can surface obfuscated filesystem access and runtime secret exfiltration missed by static scans — corroborates Warden's `trace`-without-enforcement + `strace`-outside-sandbox pattern (`ARCHITECTURE.md:124`).
- **MCP adoption signals:** Docker — `docker.com/blog/mcp-security-explained` (Sep 16 2025) reports MCP since Nov 2024 as “connective tissue”, **43% of analyzed servers had command injection flaws**, 150+ curated MCP servers in Docker MCP Catalog+Toolkit + policy gateway. Real threat: `SHIELDMCP` paper cites **437k dev envs compromised via malicious MCP package** (OAuth RCE) + 3 vulns CVE-2025-68143..68145 in Git MCP server — evidence the ecosystem is pre-app-store curation. **27.2% of servers expose exploitable tools** (`Zhao et al. 2025a` via SHIELDMCP). These justify prioritizing containment over polish pre-1.0.
- **Warden's own M8 snapshot:** 18 servers pinned (`testdata/compat/matrix.yaml:554`), known schema gaps documented as `schema-gap` not tech debt: no wildcard hosts (Playwright), no unix-socket grants (Docker daemon `/var/run/docker.sock`), no CPU cgroup, symlink unblocked (`docs/security.md:46`), `limits` ignored on macOS/Docker (`docs/security.md:118`), `bwrap --unshare-net` fallback not always fail-closed (`docs/security.md:64`). All validated by tests (`TestOverlapNormalizeIsWriteWins`, `TestResolveExecutable`, `TestDocumentedSchemaGaps`).

### Post-M8 feature roadmap

> Each item tagged `Validated` (research finding justifies it) or `Speculative` (bet, requires beta signal). Sequenced by risk-reduction / adoption-unblock.

#### M9 — Usable by more real servers (schema expressiveness + learn mode) — **Validated**

Targets the two `fail` + two `conditional` rows directly. Without this, Warden caps at 14/18.

| # | Feature | Evidence / why next | Spec status | Acceptance |
|---|---|---|---|---|
| M9.1 | **Wildcard / domain-suffix allowlist** (`*.example.com`, `**` opt-in with audit warning) | Playwright `schema-gap` fail: browser visits arbitrary domains (`docs/compatibility.md:88`). `sandboxreview.com` tiers browser-automation as inherently execution-capable; no honest policy today. | Validated (gap) | `network.allow` accepts `*.host` + docs note; `docs/compatibility.md:88` Playwright moves to `conditional`; regression `TestDocumentedSchemaGaps` flips |
| M9.2 | **Unix-socket grants** (`unix:/var/run/docker.sock` scoped, deny by default) | Docker-mcp `inherent` fail but socket pattern recurs; spec says sandbox granularity is open. Today “granting socket = handing host control” (`docs/compatibility.md:98`), so scoped grant preserves both function and isolation where possible; where not, move to explicit `inherent` with reason. | Validated | Policy field `filesystem.sockets: [path]` or `network.unix_allow`; Linux bind-mount + macOS Seatbelt profile generation |
| M9.3 | **Learn mode** (`--learn` / policy `learn: true` → `trace` → interactive approve without hand-writing YAML first) | `ARCHITECTURE.md:176` open question (lean yes for low-friction onboarding). M3 `trace`/`init` proved generation works but still separate step; reduces policy-authoring friction the beta program is specifically measuring (`docs/beta.md:8`). | Validated (architecture open q + beta goal) | One command generates starter policy, prompts approvals, writes non-overwriting file; tested without overwriting existing policy |

#### M10 — Hardening & parity (close documented limitations) — **Validated**

Addresses `docs/security.md:140` Known Limitations and `docs/compatibility.md:86` gaps.

| # | Feature | Evidence | Spec status |
|---|---|---|---|
| M10.1 | **Fail-closed net namespace** — if `bwrap --unshare-net` disabled on host, refuse to run (never fall back to host net) | `docs/security.md:64` egress bypass gap: Warden currently not always fail-closed. NSA CSI says constrain execution is lifecycle, not optional. | Validated |
| M10.2 | **CPU / cgroup limits** on Linux; explicit `ENOTSUP` warning → fail-closed on macOS/Docker until implemented | No CPU throttling (`docs/security.md:145`), `limits` silently ignored on macOS/Docker; isolation tiers expect resource quotas per spec. | Validated |
| M10.3 | **Symlink containment audit** — log symlink-resolved canonical path + warn when granted-path symlink escapes grant | `docs/security.md:46` symlink gap. | Validated |
| M10.4 | **Audit parity** — macOS/Windows file-deny as structured events (not just EPERM), Windows Kernel-Network ETW decode (`docs/security.md:150`) | Current: Seatbelt file denies as EPERM, proxy records nets; Windows network payloads captured but not decoded. | Validated |
| M10.5 | **Limit enforcement parity** — timeout/memory on macOS (Seatbelt + `limits`) + Docker (`--memory`, `--stop-timeout`) or documented fail-closed | `docs/security.md:118` silently ignored. | Validated |

#### M11 — Gateway & approval polish (stretch, post-parity) — **Mixed**

| # | Feature | Evidence | Spec status |
|---|---|---|---|
| M11.1 | **Gateway auto-discovery** — wrap servers registered in gateway config automatically (extend `docs/gateway.md:1` `wrap` beyond single-server `run`) | Ecosystem: Docker MCP Toolkit 150+ curated servers, gateway is emerging distribution shape (`docker.com` Fig 4). Warden already has `warden gateway wrap/init/run`. | Validated (ecosystem) |
| M11.2 | **Approval persistence UX** — `once` / `session` (in-memory) / `persistent` (append to policy) prompts over `/dev/tty` with 750ms escalation preserved (`ARCHITECTURE.md:118`) | M7 approval shipped but filesystem prompts require respawn (`ARCHITECTURE.md:134` bind-mounts fixed at spawn). Needs stabilization from beta friction reports. | Speculative — needs 5+ beta reports (`docs/beta.md:109`) |
| M11.3 | **Policy registry & pinning** — published policy for popular servers, digest-pinned, pulled via `warden policy get` | Mirrors Docker Catalog pin-by-digest pattern; reduces copy-paste drift. | Speculative — validated only if beta shows copy-paste as top friction |

#### M12 — Ecosystem & distribution (polish after reality-check) — **Validated (timing validated, scope speculative)**

| # | Feature | Evidence | Spec status |
|---|---|---|---|
| M12.1 | **Homebrew tap publication** `build/brew.sh` already generates formula but `REMAINING_WORK.md:139` P3.5 says no tap repo set up | Need 5+ external beta reports + matrix green before polish per `ROADMAP.md:149` why-M8-before-polish rule. | Validated (prematurity) — do not ship before M9/M10 |
| M12.2 | **npm publish idempotency** — skip if version exists (fix v0.1.4 403 rerun `REMAINING_WORK.md:153`) | Direct bug. | Validated |
| M12.3 | **Windows CI robustness** — ETW/WFP bindings already fixed (`eadca83`, `REMAINING_WORK.md:15`) but need binding-fail-closed without panic (`syscalls.go:1` → `advapi32`/`iphlapi`/`fwpuclnt` probing), POSIX-path test skips (`REMAINING_WORK.md:92`) | First CI run `34028064382` exposed panic; acceptance pending rerun. | Validated |

### How to append to `ROADMAP.md` without collision

Next available is **M9**. Since this doc already reserves `M9 = usability + schema`, `M10 = hardening`, the **Secrets Guard** domain ( §3 ) must take **M11** (or be merged as `M11 = Secrets Guard`). If you keep this future.md untracked, add to `ROADMAP.md:155`:

```
## M9 — Compatibility unclog (Validated — Playwright/Docker gaps)
- [ ] M9.1 wildcard hosts  …  ## M10 — Hardening & parity …
```

Do not renumber existing M0–M8. Do not duplicate `REMAINING_WORK.md:64` P0/P1 into roadmap as milestone — keep as fix checklist.

**Open questions (Scaling):**
- Whether microVM/gVisor isolation should be a Warden-native backend vs documented recommendation + `wrap` via gateway (Daytona pattern) — no Warden-team position found.
- Whether HCP-style hourly cluster cost shapes user willingness to adopt per-server sandbox vs expectation of free-local sandbox.
- Current `api.github.com`-style host vs path-based filtering demand — no beta data yet.

---

## 3 — Secrets Guard (Proposal 3 Integration)

> This domain had its own prompt (`secrets-guard-domain-prompt.md`) — file not found in checkout. Below is the research-first proposal built from Warden gaps + current secrets tooling sources, so the work does not duplicate assumed patterns.

### Problem Warden already documents

- `env.allow` is allowlist-only, values from parent env not stored in policy (`ARCHITECTURE.md:159`), but `docs/security.md:96` warns: `HOME` + `filesystem.read` containing `~/.ssh` or parent leaks keys by policy-authoring mistake; `trace` runs unsandboxed so sensitive env is visible in trace log until policy tightened.
- Secrets can enter via: (a) env vars, (b) files (`config/credentials.json`, `~/.netrc`, `.pgpass`), (c) tool output → prompt injection → exfil (SHIELDMCP 74% tool-poisoning success pre-mitigation). Warden today sandboxes *what a server can reach*, not *which bytes are secrets*.

### Research findings (2026)

| Area | Current state | Source |
|---|---|---|
| **Detection patterns** | Signature + entropy + validation (“semantic analysis”) + historical scanning + pre-commit hook; `gitleaks`, `trufflehog`-style API-key pattern matching & validation heuristics. | Semgrep Secrets docs (entropy/validation/historical/pre-commit) + API Radar HN scanner (`news.ycombinator.com/item?id=44720248`) |
| **Open-source guard models** | `trufflehog`/`gitleaks` as OSS scanners with live GitHub scanning; `doppler` CLI/K8s operator OSS but control plane closed; `infisical` MIT core, `ee/` enterprise directory phones-home `LICENSE_KEY` to validate, 67 `ee/services` includes `audit-log-stream`, `secret-rotation-v2`, `saml`, `kmip`, `hsm` (2026-08-23). | `merginit.com/blog/23062026-free-secrets-env-management-comparison` + `nomadlab.cc/blog/2026/04/hashicorp-vault-vs-doppler-vs-infisical-2026` |
| **Pricing shape for guard-adjacent products** | Infisical Cloud free up to 5 identities / 3 projects / 3 envs / 10 integrations; Pro $18–$20/identity/mo, Advanced $40, Enterprise custom. Doppler Developer free 3 users (+$8/extra), Team $21/user/mo (machines free), Enterprise custom; caps 500 users/250 projects. Vault Community free (BSL 1.1, not OSI), HCP Dedicated Dev $0.03/hr (~$22/mo), Standard $1.58–$7.49/hr, Plus $1.84–$9.4/hr + per-client $27–$112/mo. | `infisical.com/pricing` + `doppler.com/pricing` + `envmanager.com/blog/hashicorp-vault-pricing` + `nomadlab.cc` |
| **Runtime guard pattern** | SHIELDMCP 3-stage proxy (structural anomaly → semantic intent classifier τd=0.72 → cross-call correlation) ; MCP spec says `tool descriptions` are untrusted unless from trusted server. | `aclanthology.org/2026.acl-industry.58.pdf` + `modelcontextprotocol.io` |

### Proposed scope

#### `docs/secrets-guard-prd.md` (what Warden will do)

**In scope (deny-by-default, fail-closed):**
1. **Pre-flight secret audit** in `trace`/`init`: run existing OSS patterns (high-entropy + regex for GitHub, AWS, GCP, Slack tokens) over captured env + filesystem grants before writing policy; flag candidates, never auto-allow.
2. **Env hygiene check** on `warden run --policy`: warn/error when `env.allow` contains `HOME` while `filesystem.read` grants parent of `~/.ssh`, `~/.aws`, `~/.netrc`, `~/.git-credentials`; require `--allow-unsafe` to proceed (matches `docs/security.md:96` best-practice).
3. **Secret-aware audit redaction** — structured audit log (`warden logs` JSONL) redacts values of allowlisted env + high-entropy file snippets before printing/persisting; raw log remains `0600` (`docs/security.md:76`) but `logs --tail` is safe to paste.
4. **`warden secrets scan <policy|trace-log>`** — offline scanner using pattern set; outputs findings as `approval`-type audit events so `--approve` UI can prompt on secret-typed access.

**Explicit non-goals (follow-up, not this milestone):**
- Centralized vaulting / rotation / dynamic secrets (Infisical/Doppler/Vault do this; Warden is local runtime, not control plane).
- Full SHIELDMCP semantic classifier — would add model dependency + latency (>120ms) that conflicts with single-binary, near-zero-overhead goal.

**Source mapping:** (1)(2) derived from Warden's own documented gaps; (3)(4) from Infisical/Doppler secrets model + Semgrep Secrets capability description. No invented crypto.

#### `docs/secrets-guard-patterns.md`

- Catalog of patterns Warden will scan for (GitHub `ghp_`, AWS `AKIA`, GCP, Slack `xoxb`, generic high-entropy 32+ chars, `GITHUB_TOKEN`, `SLACK_BOT_TOKEN`, etc. from `testdata/compat/*/policy.yaml` env lists).
- Decision tree: env allowlist → filesystem grant → audit event → redaction.
- Mapping to `trace` capture vs `run` enforcement.

#### `ROADMAP.md`, `AGENTS.md`, `TESTING.md` additions

- **ROADMAP M11 (next after scaling's M9/M10)** — “Secrets Guard (scan + hygiene + redaction)”. Avoids collision by taking M11 if scaling took M9/M10.
- **AGENTS.md:18** — Add invariant: “Secrets handling: never persist env var values in policy; audit log redacts secrets by default.”
- **TESTING.md:7** — Escape tests for secrets: (a) env allow with `HOME` + ssh path granted → must warn; (b) trace log containing `ghp_` value → scanner flags; (c) audit log tail → no raw token leaked.

**Open questions (Secrets Guard):**
- Whether to vendor `gitleaks`/`trufflehog` rules or maintain own minimal set (license / update cadence not evaluated).
- Whether redaction should be on-by-default for `warden logs` vs opt-in (usability vs leak risk).

---

## 4 — Monetization

### Current traction signal (honest)

- Releases `v0.1.3–v0.1.5` published, Linux/macOS binaries on GitHub, npm `@warden-sandbox/cli` live, docs site `prof-bilal.github.io/Warden/` serving. `REMAINING_WORK.md:201` says first real CI (run `34028064382`) exposed Windows panic; job now must be green before claiming Windows verified.
- **No public traction signal measured here beyond that.** Stars, installs, compatibility-matrix contributors have not been shown to be material for pricing yet. Monetization plan must be sequenced as “after evidence, not before.” Any comp that assumes traction distorts.

### Comparable companies (4+ data points, 2026 sources)

| Company | License / posture | Packaging | Published pricing | Takeaway for Warden |
|---|---|---|---|---|
| **Semgrep** | OSS Community LGPL (Opengrep fork), paid Platform | Code (SAST), Supply Chain, **Secrets ($15/contrib/mo)**, AI credits | Community free; Platform free ≤10 contributors; **Team $30–$35/contrib/mo** (bundled), Enterprise custom, startup discounts | Bundle wins: Secrets at $15/mo is the most direct comp to Warden's secrets guard. Free ≤10 is generous on-ramp that Warden could mirror. |
| **Snyk** | Commercial (closed) | Code, Open Source, Container, IaC (separate products) | **Free** tight limits (200 OS /100 Code tests/mo); **Team $25/dev/mo** (5–10 devs cap), **Ignite $1,260/yr/dev** (≤50 devs), Enterprise custom. Vendr median: **$34.9k/yr @ 50 devs, $67.5k/yr @ 100 devs**, 38–42% renewal discount. SSO gated behind Ignite (common complaint). | Per-dev seat scales painfully; bundling vs splitting matters. Warden should avoid SSO-behind-high-tier anti-pattern. |
| **Infisical** | MIT core + `ee/` enterprise (not OSI), cloud + self-host | Secrets, PAM, certs; identity-based | **Free 5 identities/3 projects**, **Pro $18–$20/identity/mo**, **Advanced $40**, Enterprise custom with `LICENSE_KEY` phoning home. | Per-identity (human+machine) vs per-seat vs per-client are **different units** — picking wrong one taxes automation. |
| **Doppler** | Tooling OSS, platform closed | Config sync, rotation, operator | **Developer free 3 users** (+$8/extra), **Team $21/user/mo** (“AI agents ride free”), Enterprise custom | Explicit “machines free” headline is a pricing **position** — Warden could mirror “no agent fees” for sandbox runners. |
| **HashiCorp Vault** | **BSL 1.1** (not OSI since Aug 2023), free license but not free to run | Community free, **HCP Dedicated Dev $0.03/hr (~$22/mo)**, Standard $1.58–$7.49/hr, Plus $1.84–$9.4/hr + per-client $27–$112/mo; Enterprise custom, often 6-figures | Per-cluster-hour + per-client is opaque; “free OSS cheapest but highest ops cost”. Warden must stay single-binary, not hourly-managed. |
| **Tailscale** | Freemium + `headscale`/`netbird` OSS alternatives | Networking | **Personal free 6 users**, **Standard $8/seat/mo**, **Premium $18/seat/mo**, 2026 switch from usage-based to **seat-based** for predictability; break-even vs self-hosted at few-dozen seats | Seat-based predictability vs usage-based is the 2026 meta-lesson; commit to one unit and keep it. |

*Sources: `semgrep.dev/pricing` + `aicodereview.cc/blog/semgrep-pricing` + `konvu.com/compare/snyk-vs-semgrep` + `snyk.io/plans` + `vendr.com/marketplace/snyk` (Snyk medians) + `infisical.com/pricing` + `doppler.com/pricing` + `merginit.com` + `nomadlab.cc/blog/2026/04/hashicorp-vault-vs-doppler-vs-infisical-2026` + `envmanager.com/blog/hashicorp-vault-pricing` + `tailscale.com/pricing` + `tailscale.com/blog/pricing-v4`.*

### Tiering proposal (reasoned from comps, names invented but mapped)

| Tier | Who it’s for | What’s in it | Price shape (testable) | Comp anchor |
|---|---|---|---|---|
| **Community — self-hosted, MIT** | Solo dev, OSS, every MCP user today | Single binary, all backends, `trace`/`init`/`logs`/`approve`, community policies (`testdata/compat/`), file/net/env sandboxes | **Free forever**; self-host, no account. Optional sponsor/donate. | Semgrep free ≤10, Infisical free 5 identities, Vault Community |
| **Team — cloud-tuple or paid self-host** | Small team 5–20 devs standardizing 5–20 MCP servers | Everything in Community + **policy registry** (publish/pull/pin `warden policy get`), **audit retention + streaming** (90-day), **SAML SSO**, **role-based `env.allow` reviews**, **gateway auto-wrap** for pooled servers | **Per-seat** `~$15–$20/seat/mo` or **per-identity** if machines counted — **but machines (MCP server runtimes) ride free** (Doppler model), so teams aren't taxed for adding sandboxes | Doppler $21 “agents free” + Tailscale $8/$18 + Semgrep $30 bundle |
| **Enterprise — self-managed + negotiated** | Regulated org, 100+ devs, compliance | Team + **custom audit retention** (365+), **SCIM**, **LDAP**, **SIEM stream** (like Infisical `audit-log-stream`), **KMIP/HSM**, **dedicated support / SLA 99.95%**, **Windows WFP/ETW attestation** for air-gapped hosts | **Custom quote, per-seat** with volume bands (27→$112/client analogue from Vault Flex at high scale, but per-seat for Warden). Published list anchors negotiation. | Infisical Enterprise `ee/` + Vault Enterprise (quote-only but scoped by client count) |

**Why per-seat with machines-free:** `nomadlab.cc` shows 3 units in market — per-seat (Tailscale, Snyk, Doppler humans), per-identity (Infisical humans+machines), per-cluster+client (Vault). Counting each sandboxed MCP server as a billable “identity” would penalize adoption of the core value prop (sandbox per server). Doppler's explicit “AI agents ride free” is the cleanest precedent to copy.

**What *not* to do from comps:** Gate SSO behind top tier (Snyk complaint), require host-wide usage-based metering (Vault client explosion), or split Code/Secrets into separate SKUs that force bundle math (Semgrep actually bundles; Snyk inventory pain).

### Sequencing: what has to be true first

Monetization is **docs + positioning now, not code**. Build triggers:

1. **M9 + M10 shipped** — wildcard + unix-socket + fail-closed parity. Without 14→17+ pass, the paid tier has nothing to sell (“will it work with my server?” must be yes).
2. **Beta exit criteria hit** — 5+ external reports, highest-impact gaps fixed or scheduled (`docs/beta.md:109`). This is the real usability proof.
3. **Windows CI green + distribution stable** — `REMAINING_WORK.md:64` P0 WFP + P2 POSIX-path debt resolved; Homebrew tap published, npm idempotent — otherwise paid users hit P3 papercuts.
4. **≥ ~50–100 weekly active self-hosted installs** (measured via anonymized `warden version --check` or registry pulls) **or** ≥10 orgs running ≥5 sandboxes continuously for 30 days. *If this has not happened, price pages are still draft.*

Until then, keep **Community free** and invest in content funnel ( §1 ) not billing.

### “Don't build yet” list (pre-1.0)

- [ ] No paywall, no license server that phones home (Infisical `LICENSE_KEY` pattern rejected for local runtime).
- [ ] No billing infra, no Stripe webhook, no usage metering beyond anonymous install ping.
- [ ] No RBAC/SCIM/SIEM features beyond fielding design docs — code only after sequencing gate 1–4.
- [ ] No per-client or per-server hourly metering; document chosen unit (per-seat, machines-free) but do not implement enforcement.
- [ ] No feature-gating of core sandbox primitives — `bwrap`/Seatbelt/AppContainer, egress proxy, audit, trace stay in Community. Gate only **registry + retention + governance**.
- [ ] What *to* build now: **pricing page draft** (`docs/pricing.md` or landing `/pricing`), **comparison page** (Warden vs Docker-only vs microVM vs gateway), and **registry spec** (OpenAPI for `warden policy get`) — zero enforcement code.

**Open questions (Monetization):**
- Which unit actually predicts Warden cost-to-serve: per-seat (dev count), per-sandbox (server count), or per-org flat? Need metering of sandbox-hours (proxy + strace overhead ~2–5× on Linux per `docs/security.md:150`) to decide. No Warden runtime-cost data found.
- Whether Infisical's `ee/` phoning-home model is acceptable for any future on-prem governance tier — conflicts with `single static binary, no daemon` promise.

---

## Final Pass — Cross-Doc Tensions to Flag (not silently pick)

1. **Build vs sell timing:** Scaling (§2 M9/M10 hardening) says *build now* (wildcard, fail-closed, limits parity are validated gaps blocking adoption). Monetization (§4) says *sell later* (don't gate until beta + installs). **Resolution:** Do M9/M10 as Community (no tier split); keep monetization as docs-only until sequencing gates. No conflict if work lands in the same Community binary.
2. **Wildcard openness vs security:** Scaling's M9.1 wildcard expands attack surface vs Secrets Guard's hygiene (scope egress + env). **Flag:** Wildcard must require explicit `--allow-wildcard` audit warning + redaction; policy registry should discourage wildcards in published policies.
3. **Content volume vs credibility:** Marketing wants 4–8/mo depth; scaling ships 3 milestones quickly. **Flag:** Do not publish triage posts until fix has regression test (`TestDocumentedSchemaGaps` etc.) — credibility depends on pinned fixtures (`testdata/compat/`).

---

## Where to Land These Docs (since `future.md` is untracked)

If you promote this to tracked work, split into:
- `docs/marketing-strategy.md` ← §1
- `docs/scaling-roadmap.md` ← §2 (and append `M9`/`M10`/`M11`/`M12` to `ROADMAP.md:155` + `docs/roadmap.md:1`, no renumber; add nav to `mkdocs.yml:8`)
- `docs/secrets-guard-prd.md` + `docs/secrets-guard-patterns.md` ← §3 (and patch `AGENTS.md:18`, `TESTING.md:7`)
- `docs/monetization-strategy.md` ← §4

Add all four to `README.md:147` “Project docs” / Documentation section. Keep milestone numbers distinct: assign **M9=Scaling usability, M10=Hardening, M11=Secrets Guard, M12=Polish** — no collisions.

## Sources (full)

Product Hunt launch guides (`producthunt.com/launch/how-product-hunt-works`, `…/preparing-for-launch`, `help.producthunt.com/en/articles/479557-how-to-post-a-product`, Launchory 2026-07-25), DevHunt (`runany.dev/blog/devhunt-launch-platform` 2026-07-11, BetaList `betalist.com/startups/devhunt`, `firsto.co/projects/devhunt`), HN (`news.ycombinator.com/showhn.html`, Show HN pack `showhn.md` `gravity≈1.8`, Dan King 2026-04-23 188k posts dataset), MCP spec 2026-07-28 (`modelcontextprotocol.io`), NSA CSI 2026-06-02 (`media.defense.gov … CSI_MCP_SECURITY.PDF`), SHIELDMCP 80+ techniques (`aclanthology.org/2026.acl-industry.58.pdf`), SandboxReview tiering + Daytona 90ms (`sandboxreview.com`), MCP-SandboxScan WASM/WASI (`arxiv.org/pdf/2601.01241v1`), Docker MCP guide + 43% injection 150+ catalog (`docker.com/blog/mcp-security-explained`), Hackmamba technical content + SEO/GEO/AEO (`hackmamba.io`), Docsio dev marketing (`docsio.co`), HackerContent (`hackercontent.com`), Otrenix 2026 guide (`otrenix.com`), Semgrep pricing (`semgrep.dev/pricing`, `aicodereview.cc`), Snyk plans + Vendr medians (`snyk.io/plans`, `vendr.com/marketplace/snyk`, `konvu.com`), Infisical/Doppler/Vault pricing + identity vs seat analysis (`infisical.com/pricing`, `doppler.com/pricing`, `merginit.com`, `nomadlab.cc`, `envmanager.com`, `talescale.com`/`blog/pricing-v4`), API Radar Show HN (`news.ycombinator.com/item?id=44720248`), Gecko Launch HN (`…44747204`).

---

*Generated 2026-09-06 from current web research, separate per domain. Claims without source are marked open questions; do not treat as fact.*
