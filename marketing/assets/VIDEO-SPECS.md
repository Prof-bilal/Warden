# Video Specs — Every Platform, Every Format

**Theme:** Dark `#0a0a0a` · Orange `#f97316` · Burned-in captions (most platforms autoplay muted)  
**Source video:** `warden-landing/public/videos/warden-sandbox-explainer.mp4` (1920×1080, landscape)  
**Poster:** `warden-landing/public/videos/warden-sandbox-explainer-poster.jpg`

---

## 1. Specs per Platform

| Platform | Format | Resolution | Aspect | Duration | FPS | Codec | Max Size | Captions | Thumbnail |
|---|---|---|---|---|---|---|---|---|---|
| **X/Twitter** | Landscape | `1280×720` | 16:9 | ≤ 2:20 | 30 | H.264 MP4 | 512 MB | Burned-in + SRT | Auto or `1280×720` JPG |
| **Instagram Reel** | Portrait | `1080×1920` | 9:16 | 15–90s (best <60s) | 30 | H.264 MP4 | 4 GB | Burned-in (required, sound-off) | Cover frame `1080×1920` |
| **Instagram Feed** | Square/Portrait | `1080×1080` or `1080×1350` | 1:1 / 4:5 | ≤ 60s | 30 | H.264 MP4 | 4 GB | Burned-in | First frame |
| **Facebook Feed** | Landscape | `1280×720` | 16:9 | ≤ 240 min | 30 | H.264 MP4 | 4 GB | Burned-in + SRT upload | `1200×630` JPG |
| **Facebook Story** | Portrait | `1080×1920` | 9:16 | ≤ 60s | 30 | H.264 MP4 | 4 GB | Burned-in | — |
| **LinkedIn Feed** | Landscape | `1920×1080` | 16:9 | 3s–10 min | 30 | H.264 MP4 | 5 GB | SRT upload (can also burn in) | `1200×627` JPG |
| **YouTube** | Landscape | `1920×1080` | 16:9 | Any | 30 | H.264 MP4 | 256 GB | SRT upload + auto | `1280×720` JPG |
| **YouTube Shorts** | Portrait | `1080×1920` | 9:16 | ≤ 60s | 30 | H.264 MP4 | 256 GB | Burned-in | Auto |
| **Reddit** | Landscape | `1280×720` | 16:9 | ≤ 15 min | 30 | H.264 MP4 | 1 GB | Burned-in | Auto |
| **Hacker News** | — | — | — | — | — | — | — | — | — (link to YouTube) |

---

## 2. Which Video Goes Where

| Video | Source | Platforms | Notes |
|---|---|---|---|
| **Explainer (landscape, ~100s)** | `warden-sandbox-explainer.mp4` (existing) | YouTube (main), LinkedIn, Facebook Feed, X | Full walkthrough. Landscape 16:9. Burn captions. Link from HN/Reddit. |
| **Reel (portrait, ~60s)** | Re-cut from explainer OR new render from `marketing.md` Prompt 4 | Instagram Reel, YouTube Shorts, Facebook Story, TikTok | Tense "attack demo" format. 9:16. Kinetic type. Burn captions. See script below. |
| **Short cut (portrait, ~30s)** | Trimmed reel | X, Instagram Feed | Hook-only: problem → solution → CTA. For sound-off feeds. |

### Repurposing Map

```
warden-sandbox-explainer.mp4 (1920×1080, ~100s)
    │
    ├─→ YouTube (upload as-is, 1920×1080)
    ├─→ LinkedIn (upload as-is, or trim to 60–90s)
    ├─→ Facebook Feed (upload as-is)
    ├─→ X/Twitter (trim to ≤2:20, 1280×720)
    │
    └─→ CROP to 1080×1920 for portrait:
            ├─→ Instagram Reel (60s cut)
            ├─→ YouTube Shorts (60s cut)
            └─→ Facebook Story (60s cut)
```

**How to crop landscape → portrait:**
1. In Premiere / DaVinci / CapCut: set sequence to `1080×1920`, place `1920×1080` clip, scale to fill width, reposition to keep subject centered.
2. Or re-render from source (Remotion/HyperFrames) at `1080×1920` directly — better quality.

---

## 3. Reel Script (60s, Portrait — Use Verbatim)

This is `marketing.md` Prompt 4, adapted for verified claims (no 88ms/25K numbers unless re-verified):

```
[0:00-0:06] Visual: dark, tense red-alert styling — glitch/countdown aesthetic, no logo yet.
Narration: "We ran five real attacks against an unprotected MCP server."

[0:06-0:22] Visual: rapid-fire red-stamped result cards landing one after another:
  "Ransomware — files encrypted" / "Credential harvester — SSH keys stolen" /
  "Data wiper — file destroyed" / "Exfiltration — data sent out"
[Small honesty caption: "Internal tests against Warden's sandbox — not a third-party audit."]
Narration: "Ransomware encrypted files in milliseconds. A harvester stole SSH keys with zero detection."

[0:22-0:28] Visual: hard cut to black. One line of text only.
Narration: "Then we turned on Warden."

[0:28-0:50] Visual: same four attacks replay, now green-stamped "BLOCKED".
Narration: "Every attack, blocked — before it could do anything. Filesystem, network, environment — deny by default."

[0:50-0:58] Visual: large on-screen text — "Fail closed. 0 bypasses."
Narration: "If it can't sandbox, it refuses to run. Fail closed, always."

[0:58-0:65] Visual: Warden wordmark (orange shield + WARDEN), install command.
Narration: "Warden. Free, open source, on GitHub right now."
```

**Render targets:** `1080×1920`, 30fps, H.264, burned-in word-level captions (sound-off viewing).

---

## 4. Caption / Subtitle Specs

| Platform | Caption Style | File |
|---|---|---|
| X, Instagram Reel, FB Story, Shorts | **Burned-in** (required — autoplay muted) | Baked into video |
| LinkedIn | **SRT upload** + burned-in optional | `warden-captions-en.srt` |
| YouTube | **SRT upload** + auto-captions | `warden-captions-en.srt` |
| Facebook Feed | **SRT upload** (Facebook auto-generates too) | `warden-captions-en.srt` |

**Burned-in specs:**
- Font: Inter Bold, white `#ffffff` with `#0a0a0a` outline (2px stroke)
- Position: bottom 15% safe area, centered, max 2 lines, 32px on 1080×1920
- Word-level highlight: current word in `#f97316` (optional, for kinetic reels)

**SRT specs:**
- Max 2 lines, 42 chars/line, 1.5s minimum display

---

## 5. Thumbnail Specs

| Platform | Dimensions | Text | Background |
|---|---|---|---|
| YouTube | `1280×720` | `Your MCP servers don't need your whole filesystem` | Dark `#0a0a0a` + orange shield accent |
| LinkedIn | `1200×627` | Same | Same |
| X | `1200×675` | Same | Same |

**Thumbnail prompt (AI):**
```
YouTube thumbnail, dark near-black (#0a0a0a) background, large bold white text "Your MCP servers don't need your whole filesystem" in Inter Black, small orange (#f97316) geometric shield icon in corner, minimal tech style, no faces, no stock, 1280x720, high contrast
```
*Re-typeset text in Figma — don't rely on AI text.*

---

## 6. File Naming

```
warden-explainer-landscape-1920x1080.mp4    # YouTube, LinkedIn, FB
warden-explainer-poster-1280x720.jpg        # Thumbnail
warden-reel-portrait-1080x1920.mp4          # Instagram Reel, Shorts, FB Story
warden-reel-cover-1080x1920.jpg             # Reel cover
warden-captions-en.srt                     # Subtitles
```

## 7. Tool Stack

| Task | Tool |
|---|---|
| Edit landscape | DaVinci Resolve (free) / Premiere |
| Edit portrait / kinetic | CapCut (free) / After Effects / HyperFrames |
| Re-render from code | Remotion (from `marketing.md` prompts) |
| Captions | CapCut auto-captions → manual fix → export SRT + burn in |
| Thumbnail | Figma + AI background |
| Compress | HandBrake (H.264, 30fps, CRF 22) |

---

## 8. Existing Video — How to Use Now

You already have `warden-sandbox-explainer.mp4` (2.9 MB, 1920×1080):

- [ ] Upload to **YouTube** (unlisted → public on launch day)
- [ ] Upload to **LinkedIn** (native upload, not YouTube link — LinkedIn prefers native)
- [ ] Upload to **Facebook** (native)
- [ ] Trim to 60s portrait for **Instagram Reel** / **Shorts** (crop or re-render)
- [ ] Generate thumbnail from poster JPG (add headline text in Figma)
