# LinkedIn — Full Plan

**Page:** `Warden` (Company Page) + personal founder profile  
**Theme:** Dark `#0a0a0a` · Orange `#f97316` · Inter + JetBrains Mono  
**Link:** `github.com/Prof-bilal/Warden?utm_source=linkedin&utm_medium=social&utm_campaign=launch`

---

## 1. Profile Setup

### Company Page

| Item | Spec | Content |
|---|---|---|
| Logo | `300×300` | `warden-avatar-512.png` (orange shield on dark circle) — `brand/LOGO-PROMPTS.md` Prompt 3 |
| Cover | `1128×191` | `warden-banner-linkedin-1128x191.png` — Headline: `The boundary MCP never had` / Subtext: `Open source sandbox runtime · MIT licensed` — `brand/BANNER-PROMPTS.md` §2 |
| Tagline | 120 chars | `Sandbox runtime for MCP servers` |
| Description | ≤2000 chars | Use 100-word description from `WARDEN-LAUNCH-CONTENT.md` §1 (verified) |
| Website | Link | `github.com/Prof-bilal/Warden` |
| Industry | — | Computer & Network Security |
| Company size | — | 1-10 |
| Founded | — | 2026 |

### Personal Profile (Founder)

- Update headline to mention Warden: e.g., `Building Warden — sandbox runtime for MCP servers | Open source`
- Add Warden to Experience: `Warden — Sandbox runtime for MCP servers` with GitHub link

---

## 2. Founder Launch Post — Day 2 (Sept 11, 8:00 AM IST / 02:30 UTC)

> Post from **personal profile** (gets 5–10× more reach than Company Page). Then share to Company Page.

**Text (1,450 chars — LinkedIn sweet spot is 1,300–2,000):**
```
I kept installing MCP servers that had the same "feature": they could read my SSH keys, my .env, my whole home directory. The MCP protocol doesn't require process isolation, so the default for most servers is "run with the user's full permissions."

So I built Warden — an open-source sandbox runtime for MCP servers.

You declare what a server may touch in one YAML policy. A folder. A hostname. A few environment variables. Warden enforces that at the OS level — bubblewrap on Linux, Seatbelt on macOS, AppContainer + WFP + ETW on Windows — and audits every access attempt.

If it can't apply a real sandbox, it refuses to run the server. Never silently unsandboxed.

It's pre-1.0, MIT-licensed, and I'd genuinely value you trying to break it:

github.com/Prof-bilal/Warden

---

Worth noting what it does NOT do yet (all documented in docs/security.md):
• Proxy filters HTTP/HTTPS only (raw TCP/UDP relies on network namespace/WFP)
• No CPU limits
• macOS Seatbelt is deprecated upstream
• Windows needs an elevated shell

Tested against 18 real MCP servers (14 pass, 2 conditional, 2 fail by design — we publish why).

Try: warden trace -- <your-server> then warden run --policy policy.yaml

#MCP #Security #OpenSource #AIAgents #Sandbox #DeveloperTools
```

**Image:** `warden-post-linkedin-launch-1200x627.png` — Dark `#0a0a0a` card, orange shield top-left (80×80), white headline `The boundary MCP never had` (Inter Bold, 32px), muted subtext `Sandbox runtime for MCP servers — open source, pre-1.0` (Inter Regular, 16px, `#a1a1aa`), code snippet `policy.yaml` faint in background at 10% opacity.  
**Alt text:** `Dark card with orange shield icon, headline about MCP sandbox, and policy file code in background`

**Video alt:** Instead of image, upload `warden-explainer-landscape-1920x1080.mp4` natively (LinkedIn prefers native video over YouTube links — 3× more reach). Use `warden-explainer-poster-1280x720.jpg` as thumbnail if needed, add headline text in Figma.

---

## 3. Technical Post — Day 9 (Sept 16, 10:00 AM IST)

**Text (1,600 chars):**
```
How Warden's network block actually works — and what it doesn't do:

A local proxy checks the destination hostname against your policy's allowlist BEFORE any DNS lookup. So a denied host is never even resolved — there's no DNS-leak channel. The sandbox's network namespace has no default route (Linux/macOS) or WFP blocks everything but the proxy (Windows). The only way out is the proxy.

On Windows that's a WFP sublayer permitting exactly one loopback endpoint and blocking everything else outbound. On Linux it's bwrap --unshare-net with no default route and HTTP_PROXY/HTTPS_PROXY/ALL_PROXY injected.

Worth noting what it does NOT do yet:
• The proxy filters HTTP/HTTPS only — raw TCP/UDP relies on no-route/WFP
• No CPU throttling or cgroup limits on any platform
• If unprivileged user namespaces are disabled, the network boundary can widen (documented gap, not always detected)
• macOS Seatbelt is deprecated by Apple

All of this is in the repo's security review: github.com/Prof-bilal/Warden/blob/main/docs/security.md

It's pre-1.0, and the limitations are the point — a security tool's credibility is the parts it refuses to claim.

Try: warden trace -- <your-server>

#MCP #NetworkSecurity #OpenSource #Security
```

**Image:** `warden-post-linkedin-network-1200x627.png` — Diagram: server box → orange proxy box → `✓ api.github.com` (green) / `✗ evil.com` (red, label `never resolved`). Dark `#0a0a0a` bg, orange `#f97316` proxy, white labels.  
**Alt text:** `Network diagram showing Warden's proxy checking hostnames before DNS resolution`

---

## 4. Short Post — Day 14 (Sept 19, 4:00 PM IST)

**Text (280 chars — for engagement, not reach):**
```
Your MCP servers don't need your whole filesystem.

Warden gives them a boundary, an audit log, and a policy file.

Open source, pre-1.0, fail-closed.

github.com/Prof-bilal/Warden

#MCP #Security
```

**Image:** `warden-post-linkedin-short-1200x627.png` — Minimal: dark bg, large white text `Your MCP servers don't need your whole filesystem.` (Inter Black, 36px, centered), small orange shield below, `github.com/Prof-bilal/Warden` muted at bottom.

---

## 5. Image / Video Usage Map

| Post | Asset | Dimensions | Source |
|---|---|---|---|
| Founder launch | `warden-post-linkedin-launch-1200x627.png` OR `warden-explainer-landscape-1920x1080.mp4` | 1200×627 / 1920×1080 | Figma: shield + headline OR existing video |
| Technical | `warden-post-linkedin-network-1200x627.png` | 1200×627 | Figma: network diagram |
| Short | `warden-post-linkedin-short-1200x627.png` | 1200×627 | Figma: headline + shield |
| Company Page share | Same as founder post (re-share, don't duplicate) | — | — |

See `assets/IMAGE-SPECS.md` and `assets/VIDEO-SPECS.md`.

---

## 6. Company Page Re-Share

After posting from personal profile, share to Company Page:

**Share text:**
```
Warden is now on LinkedIn.

Sandbox runtime for MCP servers — deny by default, OS-level enforcement, fail closed.

Open source (MIT): github.com/Prof-bilal/Warden

#MCP #Security #OpenSource
```

---

## 7. Posting Tips

| Tip | Detail |
|---|---|
| Best times (IST) | 8–10 AM, 12–1 PM (lunch), 5–6 PM (commute) |
| Length | 1,300–2,000 chars gets most engagement — don't be too short |
| Hashtags | 3–5 max — LinkedIn doesn't favor many |
| Video | Native upload > YouTube link (3× reach) |
| Engagement | Reply to every comment within 4 hours; comment on 3–5 other posts that day to boost visibility |
| Personal vs Company | Personal profile gets 5–10× more reach — always post personally first, then share to Company Page |
| UTM | `?utm_source=linkedin&utm_medium=social&utm_campaign=launch` |
