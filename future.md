# Warden — Future Directions (Post-2026 MCP Shift)

> **Status:** Research-grounded strategy sketch, 2026-09-08. Consolidates Warden's actual shipped surface (`M0–M8`: Linux `bwrap` `warden-starter/warden/internal/sandbox/linux/linux.go:40`, macOS Seatbelt, Docker fallback `warden-starter/warden/internal/sandbox/docker/docker.go:1`, Windows AppContainer/WFP/ETW/Job Objects, egress proxy `warden-starter/warden/internal/proxy/proxy.go:1`, `trace`/`init`/`logs`, limits, `--approve`, gateway integration, compatibility matrix 18 servers 14 pass / 2 conditional / 2 fail `warden-starter/warden/docs/compatibility.md:1`) with independent web verification of the claims that motivated this doc. Where a claim has no source it is marked **Open question**. This doc is **not** a commitment to build all items — it is sequencing and evidence. See `warden-starter/warden/ARCHITECTURE.md:1`, `warden-starter/warden/ROADMAP.md:1`.

**How to use:** Three production-facing directions ordered by lift and by how well they reuse what Warden already does, plus one flagged long-term bet that is explicitly **not** a near-term roadmap item. Each item is tagged `Validated` (source found) or `Speculative` (bet needing signal). Sections cite the file + line for the Warden mechanism they would reuse.

---

## 0 — Landscape shift: what changed since Warden's original design

### Original assumption

`ARCHITECTURE.md:182` non-goal: *"Multi-tenant / server-side deployment — Warden targets a developer running MCP servers locally, not a hosted gateway."* The CLI `warden-starter/warden/internal/sandbox/linux/linux.go:65` (`--unshare-user --unshare-ipc --unshare-pid --unshare-net`) + egress proxy `internal/proxy/proxy.go:1` (DNS only after allowlist) + `policy.yaml` (`internal/policy/policy.go:27` `command` / `filesystem` / `network.allow` / `env.allow` / `limits`) were designed for `stdio` subprocess wrapping on a single laptop.

### What is true in mid-2026 (verified)

| Claim in prompt | Verification 2026-09-08 | Verdict |
|---|---|---|
| **Remote MCP over HTTP is now default for production** — GitHub, Stripe, Linear, Notion, Cloudflare publish hosted endpoints rather than pointing users at a repo | **Validated.** `openhelm.ai/blog/best-remote-mcp-servers-2026` (2026-07-10) "Remote MCP servers connect with just a URL — no install" + lists GitHub/Notion/Linear/Stripe/Cloudflare hosted endpoints; `apiscout.dev/guides/top-apis-with-mcp-endpoints-2026` (2026-04-09) "16+ production APIs now ship native MCP endpoints" with table `Stripe mcp.stripe.com / GitHub HTTP / Notion / Linear / Cloudflare` all `Transport: HTTP (remote) Auth: OAuth`; `hidekazu-konishi.com/entry/mcp_server_ecosystem_reference_2026.html` catalog confirms `stdio+HTTP` as de-facto new-server pattern, Cloudflare Workers as reference scaffolding; `mcpplaygroundonline.com/blog/awesome-mcp-servers` "30 remote HTTP servers by July 2026" plus 2026-07-28 spec dropping sticky sessions; `mcpservers.org/remote-mcp-servers` directory counts 293 official remote servers | **Holds.** "Default for production" is fair shorthand — official-vendor guidance is remote-first; `stdio` local remains common for reference/community servers |
| **Azure MCP Server 2.0 — self-hosted remote deployment as headline** | **Validated.** `devblogs.microsoft.com/azure-sdk/announcing-azure-mcp-server-2-0-stable-release/` (2026-04-10): *"The defining advancement in 2.0 is the self-hosted, remote MCP server support … so you can deploy it exactly where your team builds and operates … centrally managed policy, security controls"*; 276 tools / 57 services; `learn.microsoft.com/en-us/azure/developer/azure-mcp-server/how-to/deploy-remote-mcp-server-copilot-studio` (2026-07-10) Azure Container App + Managed Identity + Entra App Registration pattern; `nerova.ai/news/azure-mcp-server-2-0-stable-agentic-cloud-automation-april-2026` corroborates same | **Exact headline confirmed** |
| **Competitor: MintMCP Gateway handling OAuth/governance at scale** | **Validated.** `mintmcp.com/docs/architecture` — unified authentication, per-user OAuth/SSO token handling, RBAC, request logging, hosted connectors on Fly.io Machines; `jenova.ai/en/resources/enterprise-mcp-infrastructure-how-mintmcp-solves-ai-tool-governance-at-scale` (2025-10-22) + `druce.ai/governance/wiki/vendors/mintmcp` (2026-06-28) = venture-backed (Coatue/Hustle/Maven/WVV), public launch 2026-02-05, legal name Dependable AI Inc., SOC 2 Type II, "OAuth/SSO protection and governance for MCP at scale"; `businesswire.com/news/home/20260205079173` launch PR same date | **Real, monetized competitor — not hypothetical** |
| **Agentic coding tools in CI pipelines** (GitHub Action triggering Claude Code against a repo, not on a laptop) | **Validated.** `aiforanything.io/blog/claude-code-github-actions-cicd-integration-guide-2026` (2026-05-15) + `baeseokjae.github.io/posts/claude-code-github-actions-2026/` (2026-04-24): official `anthropics/claude-code-action@v1`, headless `claude -p` mode, PR review / auto-fix loops, `CLAUDE.md` as versioned policy; `fast.io/resources/claude-code-github-action-setup-guide` (2026-06-30) coverage; `code.claude.com/docs/en/github-actions` official docs | **Category is real and growing** (1.3M repos using AI code review per baeseokjae guide — unverified independently but direction matches Microsoft's June 2026 security blog) |
| **AI browser / computer-use agents mainstream security problem** — Atlas, Comet, Chrome/Edge agent features; employees already using them; Atlas bypassing encryption practice exposing auth data | **Partially validated with correction.** `techtimes.com/articles/318528/20260616/ai-browser-comparison-2026-atlas-vs-comet-vs-dia-ranked-security-use-case.htm` (2026-06-17): Atlas launched macOS Oct 2025, Comet free Oct 2025, Dia; three agents reached broad availability 2025-early 2026. Security issues documented: prompt injection via Omnibox (Oct 2025), Brave research on Comet screenshot injection, Felou bypass; `axis-intelligence.com/browser-agent-security-risk-guide/` (2026-03-31) catalog of data-leakage via AI context transmission, extension token theft (Socket.dev Jan 2026, 2,300 installs), CVE-2025-47241 whitelist bypass (1,500+ projects affected), session token hijacking. **Atlas specifics need correction:** `tech-insider.org/ca/chatgpt-atlas-vs-perplexity-comet-vs-gemini-chrome-2026` (2026-09-04) — *"ChatGPT Atlas stopped working as standalone product on Aug 9, 2026 … never received another security patch"* and was **macOS-only, never shipped Windows/iOS/Android** before being folded into ChatGPT desktop + Codex superapp; `piunikaweb.com/2026/06/25/chatgpt-atlas-perplexity-comet-ai-browsers-leaking-credentials/` documents BioShocking puzzle-game credential leak affecting both Atlas and Comet (2026-06-25) — this matches "exposing authentication data" but the mechanism was **prompt-injection exfil**, not "bypassing standard encryption practices" as worded. `dope.security/post/ai-browser-governance-2026` (2026-07-14): *"AI browsers turn the browser itself into an agent … extension policies and browser isolation cannot govern them"* + shadow-IT framing ("employees almost certainly already using them") corroborated | **Core security urgency holds; Atlas is already sunset as standalone (not an expanding market for a new integration), encryption-bypass framing is imprecise — actual documented class is prompt-injection / context exfiltration / session-token theft** |
| **Positioning: Warden vs MintMCP complementary layers** — MintMCP controls *who can call* the server; Warden would control *what the server can do once called* at OS/kernel level | **Validated as architecture.** MintMCP docs position gateway at auth/routing/logging; Warden's `ARCHITECTURE.md:1` + `internal/sandbox/*` position at namespace/seccomp/WFP/Job Object enforcement. No source contradicts complementarity. | **Keep — accurate differentiation** |

**Net assessment:** 4 of 5 claims verify cleanly; browser-agent section needs the nuance above. The macro shift away from "people who don't use MCP locally" being a small edge case is supported: hosted endpoints are now the vendor-recommended path for GitHub/Stripe/Linear/Notion/Cloudflare/Atlassian + Azure self-hosted remote pattern.

---

## 1 — Warden for CI/CD — the easiest real win (build first)

### Why this fits Warden best

- **Smallest lift.** CI runners are Linux containers. Warden's Linux `bwrap` backend `warden-starter/warden/internal/sandbox/linux/linux.go:40` (`BuildBwrapArgs` is pure, unit-tested without `bwrap` installed) already applies almost as-is. No new sandbox primitive; the egress proxy `internal/proxy/proxy.go:56` (Unix socket + `HTTP_PROXY`/`HTTPS_PROXY` bridge) and `internal/policy/policy.go:27` `policy.yaml` schema are reused unchanged.
- **Direct tie to Proposal 3 (Secrets-Leak Scanner).** CI is where that scanner and this action share an audience: the moment an agent has `contents: write` + access to `GITHUB_TOKEN`/`ANTHROPIC_API_KEY` inside the runner is the moment exfil risk peaks.
- **Category is proven.** `anthropics/claude-code-action@v1` is a real GitHub Action with headless `-p` mode, auto-fix `workflow_run` triggers, and path-filter / max-turns cost controls. It is the reference for "agentic coding inside CI."

### What to build

**`warden-action` — a GitHub Action wrapping an agentic CI job with the same `policy.yaml` model.**

```
# .github/workflows/review.yml — target UX
jobs:
  review:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: warden-sandbox/warden-action@v1
        with:
          policy: ./policy.yaml
          # run the agent *inside* the Warden sandbox, not beside it
          run: claude -p "Review this PR …" --max-turns 4
```

- Implementation is a composite action: install `warden` static binary (`dist/warden-linux-amd64` already in `warden-starter/warden/dist/`), assert `bwrap` + `strace` available (fail closed — `RUN` `linux.go:141` already refuses without `bwrap`; `strace` required for audit), then exec `warden run --policy <file> -- <agent-argv>`. Same fail-closed guarantee as local `warden run` (`internal/sandbox/sandboxerr`).
- **Policy authoring path:** commit `policy.yaml` + `CLAUDE.md` side-by-side. `warden trace -- <agent dry-run>` → `warden init` locally produces starter policy; CI enforces it. Wildcard / learn-mode gaps from `docs/compatibility.md:86` are tracked separately — CI policies for review bots are narrow (repo checkout write + `api.github.com` + `ANTHROPIC_API_KEY`), so 14/18 pass rate is not a blocker.
- **Secrets hygiene:** `env.allow` never stores values (`internal/policy/policy.go:60` comments *values from parent env not stored*); action Docs must show `${{ secrets.ANTHROPIC_API_KEY }}` only in `env:` / `with:` blocks (GitHub masks them), never in `prompt:` (log leak — see `fast.io` secret-exposure warning; `microsoft.com/security/blog/2026/06/05/securing-ci-cd-in-agentic-world-claude-code-github-action-case/` Figure 2 attack flow: Read-tool exfil of `/proc` + `sk-ant-` reconstruction). Warden's deny-by-default env (`internal/envfilter`) + audit log surface for blocked file/network attempts gives the evidence trail the scanner (Proposal 3) needs to flag an exfil.
- **Runner compatibility note:** GitHub `ubuntu-latest` supports unprivileged user namespaces (`bwrap` tested in Flatpak). Validate with one `warden doctor` step in the action; if `--unshare-net` disabled on the host, fail closed per `docs/security.md:64` (M10.1 target), don't silently fall back to host net.

### What stays out of v1

No BuildKit/Docker-in-Docker modes, no Windows runner. Linux container path only. No "Warden-as-service" for CI — it's a per-job wrapper, not a sidecar deployment.

### Acceptance / sequencing

- Milestone: place after whatever `M9`/`M10` scaling chooses — call it **M9a CI Action** or fold as `M9` if scaling's wildcard/unix-socket work is deferred. It does not require M9/M10 to ship, but shipping after `M10.1` fail-closed-net makes the CI story stronger.
- Tests: `policy.Load` KnownFields fail-closed, `BuildBwrapArgs` overlap coalescing, plus one E2E workflow against a fixture repo (`testdata/compat/` style) asserting blocked `/etc/shadow` and `registry.npmjs.org` never reached.
- Docs: single page `docs/ci-action.md` + example workflow + secrets-scanner cross-link.

**Evidence tag:** `Validated` — runner shape, `bwrap` reuse, and action pattern all have sources; secrets tie-in derives from Warden's own `docs/security.md:96` env hygiene warning + Microsoft's June 2026 CI exfil research.

---

## 2 — Warden Client Proxy — for people who never run a server at all

### Why this is the most literal answer to "people who don't use MCP locally"

Someone whose Claude Desktop is configured to talk to `https://mcp.github.com/mcp` or `https://mcp.stripe.com` over Streamable HTTP (2026-03-26 / 2026-06-18 / 2025-11-25 transports per `hidekazu-konishi.com` catalog) has **no local subprocess for Warden to wrap** — `warden run … -- node ./my-mcp-server/index.js` does not apply.

But the same worry persists at a different layer: *what is my AI client actually sending to that remote server, and does it match what I intended?*

### What to build

**A local proxy sitting between the client and any MCP server (local or remote) — logging every outbound tool call, optionally blocking payloads matching sensitive patterns before they leave the machine.**

```
Claude Desktop / Cursor / ChatGPT  --HTTP/SSE/stdio-->  warden proxy (localhost:8765)  --HTTP/SSE-->  github.mcp.com / stripe.mcp.com / linear.mcp.com
                                         |
                                    audit.jsonl + optional block (regex / entropy / gitleaks set)
```

- **Interposition point:** MCP's Streamable HTTP transport is plain HTTP + JSON-RPC. The proxy speaks both sides: it terminates the client's SSE/HTTP session and re-opens the upstream session with the same OAuth token (forwarded header) or API key (`env.allow` forwarded). For `stdio` servers it can still wrap: client points at `warden proxy --upstream stdio: npx …` and Warden injects the allowlist check around the JSON-RPC frame, not around the subprocess's syscalls.
- **Reuse:** the egress proxy's allowlist + audit pattern `internal/proxy/proxy.go:56` (`map[string]struct{}` + `Approver` callback + `audit.Logger`) generalizes to an MCP-aware proxy: current proxy filters by destination hostname before DNS; new proxy filters by **tool name + payload shape** before forwarding. `internal/audit/audit.go` structured JSON-lines + `warden logs` already exist; `internal/policy/policy.go:54` `Network.Allow` semantics extend to `mcp.allow_tools` / `mcp.deny_patterns` (see schema note below).
- **Positioning:** This turns Warden from "a tool for people who self-host MCP servers" into **"a tool for anyone whose AI client talks to MCP servers, period."** That's the addressable-market unlock the prompt identified — and it is real: `mcpservers.org` 293 official remote servers, 16→25→30 count growth Jan→July 2026, and `openhelm.ai` framing of "remote-first is the sensible default."
- **Sensitive-pattern blocking (opt-in):** reuse the same pattern set Proposal 3 (Secrets Guard) already proposes — GitHub `ghp_`, AWS `AKIA`, GCP, Slack `xoxb`, `GITHUB_TOKEN`/`SLACK_BOT_TOKEN` env names already in `testdata/compat/*/policy.yaml`, plus high-entropy 32+ heuristic from Semgrep Secrets docs. Block is **pre-egress** (payload never leaves loopback), logged as `type: mcp_block`. Redaction for `warden logs --tail` follows the same `0600` raw-log / redacted-tail split proposed for audit logs.
- **OAuth story:** do **not** re-implement OAuth. For remote servers the proxy is a pass-through that forwards the client's existing OAuth 2.1 Authorization header (PKCE/`audience`/`resource` binding per `hidekazu-konishi.com` §9). For self-hosted remote (Azure pattern) where Warden *is* the server, auth is Entra/Managed Identity in front of the Container App — the proxy does not need to mint tokens.

### What it is not

Not a gateway. Not MintMCP. Warden Client Proxy is a **single-user localhost process** with no control plane, no multi-tenant SSO, no Fly.io data plane. The mental model is `mitmproxy` for MCP, not a team gateway.

### Schema sketch (non-breaking extension)

```yaml
# policy.yaml — forwarded compat, new top-level `mcp:` only for proxy mode
mcp:
  upstream: "https://mcp.github.com/mcp"  # or "stdio: npx -y @modelcontextprotocol/server-github"
  allow_tools: ["list_pull_requests", "get_issue"]  # deny-by-default, like filesystem/network
  deny_patterns: ["ghp_[A-Za-z0-9]{36}", "AKIA[0-9A-Z]{16}"]  # optional, pre-egress block
filesystem: { read: [], write: [] }  # still enforced when upstream is stdio
network: { allow: ["mcp.github.com"] }
env: { allow: ["GITHUB_TOKEN"] }
```

Fail closed: unknown `mcp.*` keys → `KnownFields(true)` error (`internal/policy/policy.go:83`) just like any other typo'd policy field.

### Effort and risk

- Heavier than CI action (new transport handler + MCP JSON-RPC framing), but lighter than Container/K8s mode (no seccomp/NetworkPolicy compilation). Main product risk is client compatibility: Streamable HTTP spec revs (`2025-03-26` / `2025-06-18` / `2025-11-25`) differ on SSE vs pure HTTP; test against Claude Desktop Custom Connectors + Anthropic Messages API + AgentCore Gateway (per `hidekazu-konishi.com` §9.4/9.5).
- Must prove value before adding ML classifiers — keep to regex + entropy + host allowlist. SHIELDMCP's 3-stage proxy (structural → semantic classifier τ=0.72 → cross-call correlation, 74%→9% tool-poisoning at <120ms) is the research ceiling; Warden's local proxy should ship as stage 1 only.

### Acceptance

- Modes: `warden proxy --policy policy.yaml --listen :8765 --upstream https://…` + `warden proxy --upstream stdio:…` (stdio bridge).
- Audit: every tool call is a structured event; blocked payloads never traverse loopback.
- Tests: replay fixture capturing GitHub `list_pull_requests` + Stripe `create_customer` → assert allowed; fixture containing `ghp_` in `notes` field → assert blocked + audit `mcp_block`.

**Evidence tag:** `Validated` — remote endpoint existence + OAuth pattern + proxy reuse are sourced; blocking-by-pattern is speculative until evaluated against real payloads (mark as opt-in).

---

## 3 — Warden Container / K8s Mode — for teams hosting their own remote server

### Who this is for

Teams who **do** run their own MCP server, but as a shared cloud service (Cloudflare Workers, Vercel, EC2, a Kubernetes pod) rather than a local `bwrap` process — e.g. `github.com/microsoft/mcp` self-hosted remote template `Azure-Samples/azmcp-foundry-aca-mi` (Azure Container App) or a Cloudflare Worker remote MCP (`developers.cloudflare.com/agents/model-context-protocol/`).

### What to build

**The same `policy.yaml` a developer tested locally compiles down to container-native enforcement for that server's production deployment.**

```
policy.yaml  ──translate──▶  Dockerfile snippet + seccomp profile + readOnlyRootFilesystem + NetworkPolicy / EgressFirewall
   (filesystem.read/write  →  readOnlyRootFilesystem:true + explicit emptyDir / PVC mounts)
   (network.allow          →  NetworkPolicy egress allowlist / OVN EgressFirewall FQDN where supported)
   (env.allow               →  envFrom filtered + no Automatic Inheritance)
   (limits.memory_mb/timeout →  resources.limits.memory + liveness/readiness)
```

- **Reuse:** `policy.Load` + `Normalize` + `Validate` stay canonical; new emitter `internal/policy/k8s.go` (not yet existing — new file) renders Kubernetes manifests or Docker `--read-only --tmpfs --security-opt seccomp=…` flags. The Linux runtime base logic `internal/sandbox/linux/linux.go:32` (`/usr` + `/lib64` read-only, `/tmp` tmpfs) maps directly to `securityContext: { readOnlyRootFilesystem: true }` + `capabilities: { drop: [ALL] }` + `seccompProfile: { type: RuntimeDefault }` (per `kubernetes.io/docs/tutorials/security/seccomp/` and `fips-agents/code-sandbox/docs/sandbox-egress-networkpolicy-vs-opa.md`).
- **Network nuance:** `internal/policy/policy.go:210` validates bare hostnames; K8s `NetworkPolicy` with `egress: []` (deny-all) plus per-host `to:` blocks is IP/CIDR-based, not FQDN. Production teams on OVN-Kubernetes can use `EgressFirewall` for FQDN (`k8s.ovn.org/v1` `allow docs.openshift.com`) but with known DNS-TTL race (30-min poll) — not recommended for deny-critical FQDN control per `fips-agents` decision doc. Recommendation: emit strict `NetworkPolicy egress: []` default + explicit `to` blocks by resolved IP where the MCP server's upstream hosts are known and stable (`api.github.com` is CloudFront-fronted — advise pinning + `egress: []` review rather than fragile FQDN). For zero-egress sandboxes, `NetworkPolicy` alone is sufficient; OPA/Rego sidecar proxy is overkill per `fips-agents/code-sandbox` analysis (adds 2-3 containers, `NET_ADMIN` init, Rego bundle sync, no material new layer for `egress: []`).
- **Distribution signal this is worth doing:** Azure MCP 2.0 docs explicitly target "centrally managed deployments with consistent policy, security controls, and configuration" — policy-as-manifest is the enforcement story those teams need after the gateway has handled auth.

### Positioning (keep, don't blur)

> **MintMCP's gateway controls *who* can call the server (OAuth, RBAC, observability). Warden's container mode controls *what the server can technically do once called*, at the OS/kernel level. Complementary layers, not head-on competition.**

The `mintmcp.com/docs/architecture` page sells authentication/routing/logging on Fly.io Machines; `druce.ai` positions MintMCP as "managed-and-compliant first" vs `ibm-contextforge` (self-hosted open source) vs `docker-mcp-gateway` (developer isolation). Warden has no managed data plane — its value is the *strongest local technical boundary* for a given deployment target. Keep the line sharp in copy.

### What stays out

No hosted control plane. No multi-tenant auth. No CRD/operator in v1 — just a manifest emitter (`warden k8s render --policy policy.yaml --out k8s/`) plus a docs page referencing the `fips-agents/code-sandbox` pattern (Landlock + NetworkPolicy + seccomp) as the hardening baseline.

### Acceptance

- `warden k8s render` emits `NetworkPolicy` + `Deployment` `securityContext` hardenings that `go test` validates against a fixture `policy.yaml`; `docker build` snippet for non-K8s hosts.
- Tests assert FQDN limitations are surfaced as warnings, not silently emitted as ineffective policy.

**Evidence tag:** `Validated` for core motivation (Azure remote docs + K8s seccomp/NetworkPolicy sources exist); FQDN expressiveness limits are a known gap — mark emitter as `conditional` where upstream hosts are CDN-fronted.

---

## 4 — Longer-term bet — AI browser & computer-use agents (flagged as stretch, companion product)

### Why it's on the radar

- Employees are almost certainly already using Atlas/Comet/Dia whether IT approved or not (`dope.security` shadow-IT framing + `axis-intelligence.com` risk guide). Browser agents sit at the intersection of identity, data, and app access — the natural enforcement point shifting from network-perimeter to browser-level controls (`layerxsecurity.com/learn/best-agentic-browser-security-platforms/` 2026-03-20 trend: agentic-identity detection as baseline).
- Documented harm class is real: prompt-injection → exfil (Brave Comet screenshot attack Oct 2025, Felou bypass; Axis Risk 4 data leakage via AI context transmission; CVE-2025-47241 whitelist bypass across 1,500+ projects; Socket.dev enterprise-extension token harvest; PiunikaWeb BioShocking puzzle-game credential leak 2026-06-25).

### Why it's genuinely different engineering

- Warden's current architecture is **namespace-based process sandboxing** (`bwrap` `ARCHITECTURE.md:1` flow: `CLI → Policy → Backend → Server` with loopback proxy bridge `/.warden/host-proxy/egress.sock` `internal/sandbox/linux/linux.go:206`). Isolating a **whole graphical browser session** is a different boundary: existing approaches use **disposable microVMs (Firecracker/Kata/libkrun) or gVisor** kernel boundary, not `--unshare-*` (`sandboxreview.com` 2026-08-12 tiers: *"shell-execution/code-running servers need microVM or gVisor"*, browser/file-manipulation servers are execution-capable regardless of name; Daytona <90ms cold starts vs multi-second containers for per-turn agent sandboxes).
- ChatGPT Atlas as a standalone is already sunset (Aug 9 2026) — the browser-agent surface going forward is Comet (free worldwide + Android), Gemini in Chrome (3B install base, Auto Browse Jan 28 2026), and Dia. Chasing Atlas integration specifically is chasing a shipped-and-folded product.

### Recommendation

Treat as a **possible future companion product under the same trust/brand, not a near-term feature to bolt onto the existing codebase.** Keep on the radar, don't dilute the current roadmap. If pursued, companion would be a browser-session isolator (microVM per session, not a policy.yaml extension) and would need a separate threat model from Warden's filesystem/network/env boundary.

**Evidence tag:** `Speculative` — market urgency is sourced, technical mismatch is sourced, but Warden-team position on microVM vs gVisor vs docs-only recommendation is open.

**Open questions (Browser agents):**
- Whether microVM/gVisor isolation should be a Warden-native backend vs documented recommendation + `wrap` via gateway (Daytona pattern) — no team position found.
- Whether enterprise willingness to pay for browser isolation (vs free Comet + Gemini) supports a companion product — no pricing signal found.
- Whether the `Ninth Circuit Amazon v. Perplexity` (heard 2026-06-11) precedent enabling/disabling autonomous shopping/account access shifts demand for browser-agent controls — pending ruling.

---

## Sequencing & where to fold this

Prioritized in the order requested (easiest real win → market expansion → team deployment → flagged bet):

```
Now:   1. Warden for CI/CD (warden-action)         ← smallest lift, bwrap reuse, secrets-scanner tie-in
Next:  2. Warden Client Proxy (localhost MITM)       ← biggest TAM unlock, MCP JSON-RPC proxy
Then:  3. Warden Container/K8s mode (policy→manifest)← team remote-hosted, complementary to MintMCP
Later: 4. Browser agents — flagged future bet        ← companion product, microVM boundary, do not bolt on
```

**Milestone mapping (no collision with existing M0–M8):**

- If `docs/scaling-roadmap.md` exists, fold sections 1–3 there in this order and append to `warden-starter/warden/ROADMAP.md:155` as **M9 CI Action → M10 Client Proxy → M11 Container/K8s** (or keep scaling's `M9 usability + M10 hardening` and slot CI as `M9a`). Reserve `M12` for Polish/distribution (Homebrew tap publication `warden-starter/warden/build/brew.sh` already generates formula but per `REMAINING_WORK.md:139` P3.5 no tap repo yet).
- Keep browser agents as an **appendix / future-bet** section — not a numbered milestone.
- Competitive positioning sentence (Warden *what it can do* vs MintMCP *who can call it*) belongs in `README.md` + landing CTA row, not buried in a doc.

**Collateral to ship alongside:**

- `docs/ci-action.md` + example `.github/workflows/` using `testdata/compat` fixtures
- `docs/client-proxy.md` + `mcp:` schema extension + `warden proxy --help` output
- `docs/container-k8s.md` + `warden k8s render` reference + `fips-agents/code-sandbox` attribution
- `docs/browser-agents.md` one-pager explicitly marked *"Companion product, not on current roadmap"*

**What not to build yet (from research):**

- No paywall / license server that phones home (Infisical `LICENSE_KEY` pattern conflicts with `single static binary, no daemon` promise).
- No multi-tenant gateway / SSO control plane — that's MintMCP's lane; Warden wins on the kernel boundary.
- No semantic classifier proxy requiring a model dependency (>120ms) — SHIELDMCP is the ceiling, Warden ships regex+entropy only.
- No FQDN NetworkPolicy presented as deny-critical enforcement — surface the DNS-TTL race warning.

---

## Cross-cutting tensions to keep explicit

1. **Wildcard openness vs secrets hygiene.** Client Proxy's `deny_patterns` and CI's `env.allow` filtering must apply before egress; Container mode's strict `egress: []` vs FQDN gap must not silently widen. Wildcard hosts (Playwright `schema-gap` `docs/compatibility.md:86`) if later added require explicit `--allow-wildcard` audit warning + redaction — policy registry should discourage wildcards in published policies.
2. **Build vs sell timing.** CI Action and Client Proxy can ship as Community (no tier split); monetization (if any) remains docs-only until beta + install signals — same discipline as the prior monetization draft.
3. **Content vs credibility.** Ship docs that cite the compatibility matrix `testdata/compat/matrix.yaml:540` and the actual `BuildBwrapArgs`/`BuildDockerArgs`/`policy.Validate` behaviour; do not publish triage posts until regression fixtures exist.

---

## Sources (full)

- **Warden internals:** `warden-starter/warden/ARCHITECTURE.md:1`, `warden-starter/warden/ROADMAP.md:1`, `warden-starter/warden/docs/compatibility.md:1`, `warden-starter/warden/internal/policy/policy.go:27`, `warden-starter/warden/internal/sandbox/linux/linux.go:40`, `warden-starter/warden/internal/proxy/proxy.go:56`, `warden-starter/warden/internal/sandbox/docker/docker.go:1`, `warden-starter/warden/REMAINING_WORK.md:1`
- **Remote MCP as default (2026):** `openhelm.ai/blog/best-remote-mcp-servers-2026` (2026-07-10), `apiscout.dev/guides/top-apis-with-mcp-endpoints-2026` (2026-04-09), `hidekazu-konishi.com/entry/mcp_server_ecosystem_reference_2026.html` (catalog 2026, transports + auth + spec rev table), `mcpplaygroundonline.com/blog/awesome-mcp-servers` (2026-01-13 updated 2026-07-18, 30 remote count), `mcpservers.org/remote-mcp-servers` (293 directory), `reskilll.com/remote-mcp-servers-2026-deploy-once-use-anywhere` (2026-08-07 shift table), `linear.app/docs/mcp` + `developers.notion.com/docs/mcp` + `docs.stripe.com/mcp` + `developers.cloudflare.com/agents/model-context-protocol/` + `github.com/github/github-mcp-server` vendor sources
- **Azure MCP Server 2.0 (2026-04-10):** `devblogs.microsoft.com/azure-sdk/announcing-azure-mcp-server-2-0-stable-release/`, `learn.microsoft.com/en-us/azure/developer/azure-mcp-server/how-to/deploy-remote-mcp-server-copilot-studio` (2026-07-10), `learn.microsoft.com/en-us/azure/developer/azure-mcp-server/how-to/deploy-remote-mcp-server-microsoft-foundry` (2026-02-27), `nerova.ai/news/azure-mcp-server-2-0-stable-agentic-cloud-automation-april-2026` (2026-07-13), `chatforest.com/reviews/azure-mcp-servers/` (Build 2026 review)
- **MintMCP Gateway:** `mintmcp.com/docs/architecture`, `mintmcp.com/mcp-gateway`, `mintmcp.com/blog/mintmcp-vs-portkey` (2026-05-14), `jenova.ai/en/resources/enterprise-mcp-infrastructure-how-mintmcp-solves-ai-tool-governance-at-scale` (2025-10-22), `druce.ai/governance/wiki/vendors/mintmcp` (2026-06-28), `businesswire.com/news/home/20260205079173/en/MintMCP-Launches-Enterprise-Governance-Platform-for-AI-Agents-and-MCP-Servers` (2026-02-05)
- **CI / agentic coding (2026):** `aiforanything.io/blog/claude-code-github-actions-cicd-integration-guide-2026` (2026-05-15), `baeseokjae.github.io/posts/claude-code-github-actions-2026/` (2026-04-24), `fast.io/resources/claude-code-github-action-setup-guide` (2026-06-30), `code.claude.com/docs/en/github-actions`, `microsoft.com/en-us/security/blog/2026/06/05/securing-ci-cd-in-agentic-world-claude-code-github-action-case/` (CI exfil vuln, HackerOne disclosure Apr 29 → mitigation May 5 2026, Agents Rule of Two)
- **Sandbox primitives:** `github.com/containers/bubblewrap` (user namespaces, `PR_SET_NO_NEW_PRIVS`, `--new-session` CVE-2017-5226, seccomp limitations), `kubernetes.io/docs/tutorials/security/seccomp/` + `kubernetes.io/docs/reference/node/seccomp/` (seccomp stable since v1.19), `fips-agents/code-sandbox/docs/sandbox-egress-networkpolicy-vs-opa.md` (2026-04-14 NetworkPolicy `egress: []` vs OPA sidecar, OVN ACL enforcement, FQDN TTL race), `fips-agents/code-sandbox` (Landlock LSM, NetworkPolicy, Kata/gVisor tiers)
- **Browser / computer-use agents (2026):** `techtimes.com/articles/318528/20260616/ai-browser-comparison-2026-atlas-vs-comet-vs-dia-ranked-security-use-case.htm` (2026-06-17), `axis-intelligence.com/browser-agent-security-risk-guide/` (2026-03-31 + CVE-2025-47241, extension token harvest Jan 2026), `piunikaweb.com/2026/06/25/chatgpt-atlas-perplexity-comet-ai-browsers-leaking-credentials/` (2026-06-25 BioShocking), `dope.security/post/ai-browser-governance-2026` (2026-07-14), `layerxsecurity.com/learn/best-agentic-browser-security-platforms/` (2026-03-20), `tech-insider.org/ca/chatgpt-atlas-vs-perplexity-comet-vs-gemini-chrome-2026` (2026-09-04 Atlas sunset Aug 9 2026), `sandboxreview.com` (2026-08-12 isolation tiers + microVM/gVisor vs containers), `openai.com/index/continuously-hardening-chatgpt-atlas-against-prompt-injection-attacks/` + `openai.com/index/introducing-chatgpt-atlas/` (Oct 2025)
- **MCP ecosystem context:** `modelcontextprotocol.io/docs/2026-07-28/tutorials/security/security_best_practices` (SHOULD sandbox, minimal privileges), `docker.com/blog/mcp-security-explained` (2025-09-16, 43% command injection), `aclanthology.org/2026.acl-industry.58.pdf` SHIELDMCP (80+ techniques, 3-stage proxy)
