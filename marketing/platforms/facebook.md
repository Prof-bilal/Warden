# Facebook — Full Plan

**Page:** `Warden Sandbox` (Category: Software · Developer Tools)  
**Theme:** Dark `#10141A` · Blue `#6E93E8` · Geist Sans + Geist Mono  
**Link:** `github.com/Prof-bilal/Warden?utm_source=facebook&utm_medium=social&utm_campaign=launch`

---

## 1. Page Setup

| Item | Spec | Content |
|---|---|---|
| Profile picture | `170×170` | `warden-avatar-512.png` (orange shield on dark circle) — `brand/LOGO-PROMPTS.md` Prompt 3 |
| Cover photo | `820×312` | `warden-banner-facebook-820x312.png` — Headline: `Run MCP servers safely` / Subtext: `One policy. OS-level enforcement.` — `brand/BANNER-PROMPTS.md` §3 |
| Page name | — | `Warden Sandbox` |
| Username | — | `@wardensandbox` (if available) |
| Category | — | `Software` → `Developer Tools` |
| About (short) | 255 chars | `Sandbox runtime for MCP servers. One YAML policy grants files, hosts, and env vars — everything else is blocked at the OS level. Open source (MIT).` |
| About (long) | — | Full description: see Launch Post §2 below (first paragraph) |
| Website | Link | `github.com/Prof-bilal/Warden` |
| Action button | — | `Learn More` → GitHub URL |
| Messenger | — | Enable for community support (auto-reply: "Thanks for reaching out! For bug reports, please use GitHub Issues: github.com/Prof-bilal/Warden/issues") |

---

## 2. Launch Post — Day 5 (Sept 14, 1:00 PM IST / 07:30 UTC)

**Text:**
```
I just open-sourced Warden — a sandbox runtime for MCP servers.

Most MCP servers run with your full user permissions (SSH keys, env vars, every host). Warden fixes this with a YAML policy that grants only what's needed, enforced at the OS level.

Key features:
• Deny-by-default filesystem, network, and environment
• OS-native backends (Linux bwrap, macOS Seatbelt, Windows AppContainer + WFP)
• Fail-closed: refuses to run if no backend available
• Audit log of every access attempt
• Tested against 18 real MCP servers (14 pass, 2 conditional, 2 fail by design)

It's pre-1.0, MIT-licensed, and I'd love your feedback.

Try it:
  npm install -g warden-sandbox-cli
  warden doctor
  warden trace -- node server.js
  warden run --policy policy.yaml

GitHub: github.com/Prof-bilal/Warden
Docs: prof-bilal.github.io/Warden/

#OpenSource #Security #MCP #AIAgents #DeveloperTools
```

**Image:** `warden-post-facebook-launch-1200x630.png` — Dark `#10141A` card, blue shield top (80×80), white headline `Run MCP servers safely` (Geist Sans Bold, 32px, centered), muted subtext `One YAML policy. OS-level enforcement. Deny by default.` (Geist Sans Regular, 16px, `#8D95A5`), code snippet `policy.yaml` faint bg at 10% opacity.  
**Alt text:** `Dark card with blue shield icon, headline about running MCP servers safely, and policy file code in background`

**Video alt:** Instead of image, upload `warden-explainer-landscape-1920x1080.mp4` natively (Facebook prefers native video — 3× more reach than YouTube links). Use `warden-explainer-poster-1280x720.jpg` as thumbnail.

---

## 3. Share Post — Day 8 (Sept 17, 10:00 AM IST)

> Share the GitHub release or a terminal screenshot.

**Text:**
```
Just released Warden v0.1.10 — open source sandbox for MCP servers.

If you run MCP servers (for Claude, IDEs, or AI agents), they have full access to your system by default. Warden fixes that with OS-level sandboxing.

What's new in v0.1.10:
• Linux bubblewrap backend (verified)
• macOS Seatbelt + Windows AppContainer
• Docker fallback
• `warden trace` → `warden init` → `warden run` workflow

GitHub: github.com/Prof-bilal/Warden
npm: npm install -g warden-sandbox-cli

#OpenSource #Security #MCP
```

**Image:** `warden-terminal-run-1200x630.png` — Terminal: `warden run --policy policy.yaml -- node server.js` with green `✓ Sandbox active`. Dark terminal, Geist Mono.  
**Alt text:** `Terminal showing Warden running an MCP server in a sandbox`

---

## 4. Video Post — Day 12 (Sept 21, 6:00 PM IST)

**Text:**
```
How Warden sandboxes an MCP server in 60 seconds:

1. Write a policy — grant a folder, a hostname, env vars
2. Run — Warden picks the OS-native backend
3. Everything else is blocked + logged

Fail closed: if it can't sandbox, it refuses to run.

Full explainer: youtu.be/[VIDEO_ID]
GitHub: github.com/Prof-bilal/Warden

#MCP #Security #OpenSource #DeveloperTools
```

**Video:** `warden-reel-portrait-1080x1920.mp4` OR `warden-explainer-landscape-1920x1080.mp4` — upload natively. For portrait, Facebook will show it as a Story/Reel.

---

## 5. Group Strategy (Optional)

### Relevant Facebook Groups

| Group | Why | How to Post |
|---|---|---|
| `MCP Developers` (if exists) | Direct audience | Share launch post with context |
| `AI Agents & Tools` | Broader AI audience | Technical post + discussion question |
| `Open Source Developers` | OSS community | Share GitHub release |

### Group Post Template

```
Hey everyone — I built Warden, an open-source sandbox runtime for MCP servers.

MCP servers run with your full user permissions by default. Warden fixes that with a YAML policy enforced at the OS level (bwrap/Seatbelt/AppContainer).

• Deny by default — grant a folder, hostname, env vars
• Fail closed — refuses to run if no backend available
• Tested against 18 real servers

It's pre-1.0 and I'd love feedback from anyone running MCP servers.

GitHub: github.com/Prof-bilal/Warden

Happy to answer questions!
```

> **Rule:** Read each group's rules before posting. Don't spam. One group per day max. Always add value, not just a link.

---

## 6. Image / Video Usage Map

| Post | Asset | Dimensions | Source |
|---|---|---|---|
| Launch post | `warden-post-facebook-launch-1200x630.png` OR `warden-explainer-landscape-1920x1080.mp4` | 1200×630 / 1920×1080 | Figma: shield + headline OR existing video |
| Share post | `warden-terminal-run-1200x630.png` | 1200×630 | Terminal screenshot |
| Video post | `warden-reel-portrait-1080x1920.mp4` or `warden-explainer-landscape-1920x1080.mp4` | 1080×1920 / 1920×1080 | CapCut / existing video |
| Page cover | `warden-banner-facebook-820x312.png` | 820×312 | `brand/BANNER-PROMPTS.md` §3 |

See `assets/IMAGE-SPECS.md` and `assets/VIDEO-SPECS.md`.

---

## 7. Response Templates

**Technical question:**
```
Thanks for asking! [Answer]. Full architecture: github.com/Prof-bilal/Warden/blob/main/ARCHITECTURE.md
```

**"Is it production ready?":**
```
Pre-1.0 right now (M0-M8 shipped, no third-party audit). Security model + known limitations are in docs/security.md. We're collecting compatibility reports to prioritize before 1.0 — your feedback helps!
```

**Bug report:**
```
Thanks for testing! Could you file a report? github.com/Prof-bilal/Warden/issues/new?template=compatibility-report.md
```

---

## 8. Posting Tips

| Tip | Detail |
|---|---|
| Best times (IST) | 9 AM, 1 PM, 6 PM — test and see |
| Image vs video | Posts with images get 2.3× more engagement; native video gets 3× more than links |
| Questions | Ask a question to drive comments (e.g., "What would you grant a server you just installed?") |
| Groups | Share in relevant groups but follow group rules — one group per day max |
| Insights | Check Facebook Insights → Posts → When Your Fans Are Online |
| Reply window | Within 2 hours |
| Cross-post | Instagram reel auto-shares to Facebook if toggled (Instagram → Settings → Sharing) |
| UTM | `?utm_source=facebook&utm_medium=social&utm_campaign=launch` |
