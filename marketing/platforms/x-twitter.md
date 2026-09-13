# X (Twitter) — Full Plan

**Handle:** `@warden_sandbox` (or `@wardenmcp`)  
**Theme:** Dark `#10141A` · Blue `#6E93E8` · Geist Sans + Geist Mono  
**Link:** `github.com/Prof-bilal/Warden` (add `?utm_source=x&utm_medium=social&utm_campaign=launch` for tracking)

---

## 1. Profile Setup

| Item | Spec | Content |
|---|---|---|
| Profile picture | `400×400` | `warden-avatar-512.png` (orange shield on dark circle) — see `brand/LOGO-PROMPTS.md` Prompt 3 |
| Header | `1500×500` | `warden-banner-x-1500x500.png` — Headline: `Sandbox your MCP servers` / Subtext: `Deny by default. Fail closed.` — see `brand/BANNER-PROMPTS.md` §1 |
| Bio (160 chars) | Text | `Sandbox runtime for MCP servers. Deny by default. Fail closed. Open source (MIT). ↓ github.com/Prof-bilal/Warden` |
| Pinned tweet | — | Launch thread (Tweet 1–10 below) |
| Website | Link | `github.com/Prof-bilal/Warden` |

---

## 2. Launch Thread — Day 1 (Sept 10, 9:00 AM IST / 03:30 UTC)

> Post as a thread: publish Tweet 1, then reply to it with Tweet 2, etc. Each tweet ≤280 chars. Attach image only to Tweets 1, 2, 6, 9 (X throttles threads with too many images).

### Tweet 1 — Hook (with image)

**Text:**
```
The MCP servers you `npx` every week run as plain processes with your full user permissions.

Your SSH keys. Your .env. Every host you can reach.

Nothing in the protocol stops them.

This is the problem Warden exists for. 🧵
```

**Image:** `warden-post-x-launch-1200x675.png` — Dark bg, blue shield left, white headline `YOUR MCP SERVERS CAN READ EVERYTHING` (Geist Sans Bold, 36px), muted subtext `MCP has no concept of a boundary.` Alt text: `Dark card with blue shield icon and headline about MCP servers having full system access`

**Video alt:** None — image only.

### Tweet 2 — Solution (with image)

**Text:**
```
The fix: one YAML policy.

filesystem:
  read: ["./data"]
network:
  allow: ["api.github.com"]
env:
  allow: ["GITHUB_TOKEN"]

Deny by default. Nothing is reachable unless you grant it.
```

**Image:** `warden-policy-example-800x600.png` — Code card: `policy.yaml` with filesystem/network/env/limits blocks highlighted in blue as named. Background `#171C24`, border `#1E242E`, Geist Mono.

### Tweet 3 — Backends

**Text:**
```
Under the hood it's OS primitives, not magic:

• Linux: bubblewrap with unshared namespaces
• macOS: sandbox-exec (Seatbelt)
• Windows: AppContainer + WFP + ETW
• Docker: --network none fallback

Auto-selected. Never silent.
```

**Image:** None (keep thread lightweight). Or attach `warden-post-x-backends-1200x675.png` — 3-row backend table (Linux Ready / macOS Ready / Windows Code-complete).

### Tweet 4 — Network

**Text:**
```
Network: a local proxy checks hostnames against your allowlist BEFORE DNS.

Denied hosts are never resolved. No DNS leak channel.

On Linux, the sandbox has no default route — the only way out is the proxy.
```

**Image:** None. Alt if used: diagram server → proxy → allowed host (green) / blocked DNS (red X).

### Tweet 5 — Filesystem

**Text:**
```
Filesystem: granted paths only.

Ungranted paths don't exist inside the sandbox — not "permission denied", actually invisible.

Your server can't even see what it's not allowed to touch.
```

**Image:** None.

### Tweet 6 — Environment (with image)

**Text:**
```
Environment: only what you list in `env.allow` crosses the boundary.

Empty list = empty environment.

Your shell stays yours. No accidental AWS_KEY leakage.

env:
  allow: ["GITHUB_TOKEN"]  # only this passes through
```

**Image:** `warden-terminal-env-1200x675.png` — Terminal showing `env` inside sandbox: only `GITHUB_TOKEN` listed, rest empty.

### Tweet 7 — Fail Closed

**Text:**
```
If Warden can't apply a sandbox it refuses to run.

The refusal is a tested feature, not an error.

"A plain CreateProcess fallback is never acceptable." — windows/run.go:18
```

**Image:** None. Or terminal card: `✗ Warden refused to start: sandbox backend unavailable — fails closed by design.`

### Tweet 8 — Compatibility

**Text:**
```
Real-server reality check: 18 MCP servers in the compat matrix.

14 pass out of the box.
2 need per-deployment hosts (fetch, kubernetes).
2 can't be sandboxed honestly (docker-mcp, playwright) — we publish why.

github.com/Prof-bilal/Warden/blob/main/docs/compatibility.md
```

**Image:** None.

### Tweet 9 — CTA (with image + video option)

**Text:**
```
Try it in 3 commands:

warden doctor
warden trace -- node server.js
warden run --policy policy.yaml

PRs and breakage reports welcome. That's the signal we want.

github.com/Prof-bilal/Warden
```

**Image:** `warden-terminal-run-1200x675.png` — Terminal: `warden run --policy policy.yaml -- node server.js` with green `✓ Sandbox active` line.  
**Video alt:** Upload `warden-explainer-landscape-1920x1080.mp4` trimmed to 60s (1280×720) — native video gets 3× reach vs image.

### Tweet 10 — Pinned Reply (reply to Tweet 1)

**Text:**
```
MIT licensed. Single binary. No daemon.

Built in the open — every claim is pinned to a test in the repo.

If something doesn't fit the schema, file a compatibility report:
github.com/Prof-bilal/Warden/issues/new?template=compatibility-report.md
```

**Image:** None.

---

## 3. Single Launch Post (Alt Format — For Algorithm)

If you want a single-post version instead of/in addition to thread:

**Text (278 chars):**
```
MCP servers run with your full user permissions right now.

Warden sandboxes them: one YAML policy, OS-level enforcement (bwrap / Seatbelt / AppContainer), audit log on every attempt, fail closed if it can't sandbox.

Free & open source.

github.com/Prof-bilal/Warden
```

**Image:** `warden-post-x-launch-1200x675.png` (same as Tweet 1)  
**Post at:** 12:00 PM IST (different time than thread — don't compete with yourself)

---

## 4. Follow-Up Posts (Week 2+)

### Post 2 — Filesystem Deep-Dive (Sept 12, 10 AM)

**Text:**
```
When an MCP server tries to read ~/.ssh/id_rsa under Warden:

• If path is granted → access allowed (logged)
• If NOT granted → path doesn't exist (not "permission denied")

The server can't even see it exists. That's the difference between a sandbox and a permission check.
```

**Image:** `warden-terminal-blocked-1200x675.png` — Terminal: `read ~/.ssh/id_rsa → BLOCKED (invisible)`  
**Alt text:** `Terminal showing blocked filesystem access attempt`

### Post 3 — Network Deep-Dive (Sept 13, 10 AM)

**Text:**
```
How Warden's network block actually works:

1. Server tries to connect to evil.com
2. Local proxy intercepts BEFORE DNS
3. Hostname checked against policy allowlist
4. Not allowed → DNS never resolves → impossible

No DNS leak channel. The sandbox has no default route.
```

**Image:** Diagram: server → proxy (orange) → `✓ api.github.com` (green) / `✗ evil.com` (red, "never resolved")

### Post 4 — Fail-Closed Emphasis (Sept 14, 10 AM)

**Text:**
```
The most important result in Warden's test suite isn't a block.

It's a refusal.

When the backend can't initialize, Warden doesn't fall back to running unsandboxed. It refuses to start.

Fail closed is the whole point. The test asserts the exact wording: "fails closed by design."
```

**Image:** Terminal card: `✗ Warden refused to start: sandbox backend unavailable` (orange `✗`, white text, dark bg)

### Post 5 — Community Call (Sept 15, 10 AM)

**Text:**
```
Looking for MCP server maintainers to test Warden.

Run your server:
  warden trace -- <your-server>
  warden init
  warden run --policy policy.yaml

File a compatibility report if something breaks — that's the feedback we need:
github.com/Prof-bilal/Warden/issues
```

**Image:** `warden-post-x-cta-1200x675.png` — 3-step terminal flow: `trace → init → run`

---

## 5. Image / Video Usage Map

| Post | Asset | Dimensions | Source |
|---|---|---|---|
| Tweet 1 | `warden-post-x-launch-1200x675.png` | 1200×675 | Figma: shield + headline (orange/dark) |
| Tweet 2 | `warden-policy-example-800x600.png` | 800×600 | Screenshot PH gallery `03-policy.html` |
| Tweet 3 (opt) | `warden-post-x-backends-1200x675.png` | 1200×675 | Screenshot PH gallery `05-backends.html` |
| Tweet 6 | `warden-terminal-env-1200x675.png` | 1200×675 | Terminal screenshot |
| Tweet 9 | `warden-terminal-run-1200x675.png` + `warden-explainer-landscape-1920x1080.mp4` (trimmed) | 1200×675 / 1280×720 | Terminal + existing video |
| Post 2 | `warden-terminal-blocked-1200x675.png` | 1200×675 | Terminal screenshot |
| Post 3 | Network diagram | 1200×675 | Figma: server→proxy→hosts |
| Post 4 | Refusal card | 1200×675 | Figma: terminal card |
| Single post | `warden-post-x-launch-1200x675.png` | 1200×675 | Same as Tweet 1 |

See `assets/IMAGE-SPECS.md` and `assets/VIDEO-SPECS.md` for export specs.

---

## 6. Response Templates

**Technical question:**
```
Thanks for asking! [Answer]. Full architecture: github.com/Prof-bilal/Warden/blob/main/ARCHITECTURE.md — happy to go deeper.
```

**Bug report:**
```
Thanks for testing! Could you file a compatibility report? github.com/Prof-bilal/Warden/issues/new?template=compatibility-report.md — include server name, OS, policy, and error output.
```

**"Why not Docker?":**
```
Docker is one of Warden's fallbacks. But for "run one npm script with a restricted home" — namespace sandboxing has a smaller trust boundary, no daemon, and near-zero startup. Warden prefers OS-native; Docker is the fallback.
```

**"When production ready?":**
```
Pre-1.0 right now (M0-M8 shipped, no third-party audit yet). Security model + limitations are in docs/security.md — collecting compatibility reports to prioritize before 1.0. Your feedback helps!
```

---

## 7. Posting Schedule & Tips

| Tip | Detail |
|---|---|
| Best times (IST) | 9 AM, 12 PM, 5 PM — test and double down on what works |
| Thread length | 7–10 tweets max — engagement drops after 7 |
| Reply window | Within 4 hours during launch week |
| Hashtags | 2–3 max: `#MCP #Security #OpenSource` (no more) |
| Alt text | Write for every image (accessibility + SEO) |
| UTM | Add `?utm_source=x&utm_medium=social&utm_campaign=launch` to GitHub links |
| Pin | Pin Tweet 1 for 7 days, then replace with best-performing single |
