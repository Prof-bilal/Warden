# Instagram — Full Plan

**Handle:** `@warden.sandbox`  
**Theme:** Dark `#0a0a0a` · Orange `#f97316` · Inter + JetBrains Mono  
**Link in bio:** `github.com/Prof-bilal/Warden?utm_source=instagram&utm_medium=social&utm_campaign=launch` (or Linktree)  
**Content type:** Visual-first — carousels (2–3× more engagement than singles) + reels (<60s best)

---

## 1. Profile Setup

| Item | Spec | Content |
|---|---|---|
| Profile picture | `320×320` (circle crop) | `warden-avatar-512.png` (orange shield on dark circle) — `brand/LOGO-PROMPTS.md` Prompt 3 |
| Name | 30 chars | `Warden` |
| Bio (150 chars) | Text | `Sandbox runtime for MCP servers\nDeny by default · Fail closed · Audit everything\nOpen source (MIT) ⬇️` |
| Link in bio | Link | `github.com/Prof-bilal/Warden` or Linktree with: GitHub / npm / Docs / YouTube |
| Category | — | `Software Company` or `Science & Technology` |
| Account type | — | Business or Creator (for analytics + scheduling) |
| Story highlights | 3 covers `1080×1920` | `How it works` / `Install` / `Security` — see `brand/BANNER-PROMPTS.md` §5 |

---

## 2. Carousel 1 — "The Problem" — Day 5 (Sept 14, 9:00 AM IST)

> 6 slides, `1080×1080` each (square). Swipeable. Cover must hook in 1 second.

### Slide 1 — Cover (Hook)

**Visual:** Dark `#0a0a0a` bg, large orange shield icon (200×200) centered top, white headline below.  
**Text:**
```
YOUR MCP SERVERS
CAN READ EVERYTHING.

[Small orange line below, 60px wide, 3px thick, centered]
```

**Font:** Inter Black, 48px, `#ffffff`, line-height 1.1, centered. Orange line: `#f97316`.

### Slide 2 — What is an MCP Server?

**Visual:** Dark bg, white text, orange bullet dots.  
**Text:**
```
What is an MCP server?

A process that gives AI tools
— Claude, IDEs, agents —
access to your files,
network, and data.

You install them from
npm, PyPI, or GitHub.
```

**Font:** Inter Regular, 28px, `#ffffff` for headings, `#a1a1aa` for body.

### Slide 3 — The Problem

**Visual:** Dark bg, orange `✗` icons.  
**Text:**
```
The problem:

MCP has no concept
of a boundary.

Servers run with your
FULL user permissions:

• Every file you can read
• Every host you can reach
• Every env variable you have
```

**Font:** Headline Inter Bold 32px `#ffffff`, bullets Inter Regular 24px `#a1a1aa`, orange dots `#f97316`.

### Slide 4 — What They Can Access

**Visual:** Dark bg, code-style list with orange file icons.  
**Text:**
```
What a server can access
(by default):

~/.ssh/id_rsa  →  READ
.env           →  READ
/etc/passwd    →  READ
api.internal   →  CONNECT
AWS_SECRET_KEY →  READ

Nothing stops them.
```

**Font:** JetBrains Mono, 20px, `#ffffff` for paths, `#71717a` for arrows, orange `#f97316` for `READ`/`CONNECT`. Last line Inter Bold 24px `#f97316`.

### Slide 5 — Why It Matters

**Visual:** Dark bg, white text, subtle orange warning triangle (outline, not filled).  
**Text:**
```
This isn't theoretical.

Supply chain attacks happen.
Buggy code happens.

You don't know what
a server does
until it does it.
```

### Slide 6 — CTA

**Visual:** Dark bg, large orange shield (120×120) centered top, white CTA below, muted link at bottom.  
**Text:**
```
Warden fixes this.

One YAML policy.
OS-level enforcement.
Deny by default.

Link in bio →

github.com/Prof-bilal/Warden
```

**Font:** Headline Inter Black 36px `#ffffff`, CTA Inter Bold 20px `#f97316`, link Inter Regular 16px `#71717a`.

**Caption:**
```
Your MCP servers run with your full user permissions. Nothing in the protocol stops them from reading your SSH keys, your .env, or calling any host.

Warden fixes that with OS-level sandboxing and a deny-by-default policy.

Open source (MIT) → link in bio

#MCP #Security #OpenSource #AIAgents #DeveloperTools #CyberSecurity #TechStartup #Programming
```

**Alt text (per slide):** Write 1-sentence description per slide for accessibility (Instagram → Advanced settings → Write alt text).

---

## 3. Carousel 2 — "How It Works" — Day 7 (Sept 16, 9:00 AM IST)

### Slide 1 — Cover

**Visual:** Dark bg, orange shield + code brackets `{}` icon.  
**Text:**
```
HOW WARDEN WORKS
in 60 seconds.

[Orange line below]
```

### Slide 2 — Step 1: Policy

**Visual:** Code card: `policy.yaml` with filesystem/network/env blocks, orange highlight on each block. Background `#1a1a1a`, border `#27272a`, JetBrains Mono.  
**Text:**
```
Step 1: Write a policy

filesystem:
  read: ["./data"]
network:
  allow: ["api.github.com"]
env:
  allow: ["GITHUB_TOKEN"]

Grant what's needed.
Nothing more.
```

### Slide 3 — Step 2: Run

**Visual:** Terminal card: `warden run --policy policy.yaml -- node server.js` with green `✓` output. Dark terminal, JetBrains Mono.  
**Text:**
```
Step 2: Run your server

warden run --policy policy.yaml
  -- node server.js

Warden picks the best
backend automatically:
bwrap / Seatbelt / AppContainer
```

### Slide 4 — What Happens

**Visual:** Dark bg, two-column: `✓ Granted` (green) / `✗ Denied` (muted red/gray).  
**Text:**
```
What happens:

✓ Granted paths → accessible
✗ Ungranted    → invisible
✓ Allowed hosts → connected
✗ Blocked hosts → never resolved
✓ Listed env    → passed through
✗ Unlisted env  → empty
```

**Font:** JetBrains Mono 18px, green `#22c55e` for `✓`, muted `#71717a` for `✗`.

### Slide 5 — Fail Closed

**Visual:** Dark bg, terminal card with orange `✗` and white text: `Warden refused to start: sandbox backend unavailable — fails closed by design.`  
**Text:**
```
Fail closed:

If Warden can't apply
a sandbox, it refuses
to run.

No silent fallback.
No "sandbox disabled."
Never unprotected.
```

### Slide 6 — CTA

**Visual:** Same as Carousel 1 Slide 6 but with 3-step flow.  
**Text:**
```
Try it:

warden doctor
warden trace -- node server.js
warden run --policy policy.yaml

Link in bio →

github.com/Prof-bilal/Warden
```

**Caption:**
```
How Warden sandboxes an MCP server in 3 steps:

1. Write a policy — grant a folder, a hostname, env vars
2. Run — Warden picks the OS-native backend
3. Everything else is blocked + logged

Fail closed: if it can't sandbox, it refuses to run.

Link in bio → github.com/Prof-bilal/Warden

#MCP #Sandbox #Security #OpenSource #DeveloperTools #HowItWorks #TechExplained
```

---

## 4. Reel — 60s Portrait — Day 6 (Sept 15, 6:00 PM IST)

> `1080×1920`, 30fps, H.264 MP4, burned-in captions (sound-off viewing). Tense "attack demo" format.

### Script (Use Verbatim — Verified Claims Only)

```
[0:00-0:06] Visual: dark, tense red-alert styling — glitch/countdown aesthetic, no logo yet.
Narration: "Your MCP servers can read everything on your machine."
Caption: YOUR MCP SERVERS CAN READ EVERYTHING

[0:06-0:15] Visual: rapid list appearing: "~/.ssh/id_rsa", ".env", "AWS_SECRET_KEY", "api.internal.com" — each with red "READ" / "CONNECT" stamp.
Narration: "SSH keys. Env vars. Every host. Nothing in the protocol stops them."
Caption: SSH KEYS · ENV VARS · EVERY HOST

[0:15-0:22] Visual: hard cut to black. Orange shield fades in.
Narration: "Warden fixes that."
Caption: WARDEN FIXES THAT.

[0:22-0:40] Visual: policy.yaml code card animating in, then terminal: "warden run --policy policy.yaml -- node server.js" → green "✓ Sandbox active"
Narration: "One YAML policy. Grant a folder, a hostname, env vars. Everything else is blocked at the OS level."
Caption: ONE POLICY. OS-LEVEL ENFORCEMENT.

[0:40-0:52] Visual: blocked attempts with green "BLOCKED" stamps: "read ~/.ssh/id_rsa → BLOCKED (invisible)" / "connect evil.com → BLOCKED (never resolved)"
Narration: "Ungranted files are invisible. Denied hosts are never even resolved."
Caption: INVISIBLE · NEVER RESOLVED

[0:52-0:60] Visual: Warden wordmark (orange shield + WARDEN), install command "npm install -g warden-sandbox-cli", GitHub link.
Narration: "Warden. Free, open source, on GitHub right now."
Caption: WARDEN — LINK IN BIO
```

**Audio:** Piper TTS or similar free narration (see `marketing.md` prompts). Or trending audio at low volume + captions.

**Caption:**
```
Your MCP servers have full access to your system. Warden sandboxes them with OS-level enforcement.

One policy. Deny by default. Fail closed.

Link in bio → github.com/Prof-bilal/Warden

#MCP #Security #Reels #TechReels #DeveloperTools #OpenSource #CyberSecurity #AIAgents
```

**Cover frame:** `1080×1920` — Dark bg, orange shield, white headline `YOUR MCP SERVERS CAN READ EVERYTHING` (Inter Black, 48px). This is what shows in grid.

**Hashtags (Reels):** 5–8 max. Use: `#MCP #Security #DeveloperTools #OpenSource #TechReels #AIAgents`

---

## 5. Stories — Daily During Launch Week

### Story 1 — Poll (Day 1)

**Visual:** `1080×1920`, dark bg, white text, orange poll sticker.  
**Text:**
```
Quick question:

Do you know what your
MCP servers can access
on your machine?
```

**Sticker:** Poll — `Yes` / `No`

### Story 2 — Reveal (Day 1, 4 hours later)

**Visual:** `1080×1920`, dark bg, orange `✗` list.  
**Text:**
```
Most people don't.

MCP servers run with FULL
user permissions by default.

~/.ssh  •  .env  •  every host
```

### Story 3 — CTA (Day 1, next day)

**Visual:** `1080×1920`, dark bg, orange shield, white CTA.  
**Text:**
```
I built Warden to fix this.

One YAML policy. OS-level
sandbox. Deny by default.

Swipe up → github.com/Prof-bilal/Warden
```

**Sticker:** Link sticker → GitHub URL

### Story 4 — Behind the Scenes (Day 3)

**Visual:** Photo/screenshot of terminal with `warden doctor` output.  
**Text:**
```
Building in the open.

Every claim → pinned to a test.

github.com/Prof-bilal/Warden
```

---

## 6. Image / Video Usage Map

| Post | Asset | Dimensions | Source |
|---|---|---|---|
| Carousel 1 (6 slides) | `warden-carousel-problem-01` … `06-1080x1080.png` | 1080×1080 ×6 | Figma: dark/orange, Inter + JetBrains Mono |
| Carousel 2 (6 slides) | `warden-carousel-how-01` … `06-1080x1080.png` | 1080×1080 ×6 | Figma: code cards + terminal |
| Reel | `warden-reel-portrait-1080x1920.mp4` | 1080×1920, 60s | CapCut / Remotion — script above |
| Reel cover | `warden-reel-cover-1080x1920.jpg` | 1080×1920 | Figma: shield + headline |
| Stories (×4) | `warden-story-01` … `04-1080x1920.png` | 1080×1920 ×4 | Figma: dark/orange, short text |
| Highlights (×3) | `warden-highlight-*-1080x1920.png` | 1080×1920 ×3 | `brand/BANNER-PROMPTS.md` §5 |

See `assets/IMAGE-SPECS.md` and `assets/VIDEO-SPECS.md` for export specs.

---

## 7. Hashtag Strategy

| Post Type | Hashtags (5–10) |
|---|---|
| Carousels | `#MCP #Security #OpenSource #AIAgents #DeveloperTools #CyberSecurity #TechStartup #Programming #HowItWorks` |
| Reels | `#MCP #Security #DeveloperTools #OpenSource #TechReels #AIAgents #Reels #TechTok` |
| Stories | No hashtags (use poll + link stickers) |

**Don't:** Use more than 10, use irrelevant tags, use `#viral`/`#fyp` (hurts reach for tech content).

---

## 8. Posting Schedule & Tips

| Tip | Detail |
|---|---|
| Best times (IST) | 9 AM, 12 PM, 6 PM — carousels at 9 AM, reels at 6 PM (peak scroll) |
| Carousel > single | Carousels get 2–3× more engagement — always prefer carousel over single image |
| Reel length | <60s best — algorithm favors completion rate |
| Captions | Always burned-in for reels (80% watch muted) |
| Alt text | Write for every carousel slide (accessibility + SEO) |
| Stories | Post 1–2 per day during launch week, use polls/questions |
| Engagement | Reply to every comment within 2 hours; like + reply to DMs |
| Cross-post | Share reel to Facebook automatically (toggle in Instagram) |
