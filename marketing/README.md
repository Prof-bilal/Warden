# Warden — Marketing Hub

**Version:** 1.0 — Sept 2026  
**Theme:** Shield logo · Blue `#6E93E8` on dark `#10141A` (Linear-style)  
**Launch:** Sept 9 — Sept 30, 2026 (day-by-day plan inside `calendar/`)

> All claims in this folder are **verified** against source code/tests/registry. Any attack-simulation numbers (88ms, 25K, etc.) are **unverified** and must not be published until a reproducible harness exists. See `WARDEN-FACTS.md` §9.

---

## Folder Map

```
marketing/
├── README.md                    ← you are here
├── brand/
│   ├── GUIDELINES.md            Voice, tone, colors, typography, do/don't
│   ├── LOGO-PROMPTS.md          6 AI prompts for shield logo (DALL·E / Midjourney / SD)
│   └── BANNER-PROMPTS.md        Platform-specific banner prompts (X, LinkedIn, FB, YT)
├── platforms/
│   ├── x-twitter.md             X/Twitter — threads, singles, templates, image map
│   ├── linkedin.md              LinkedIn — founder + technical + short
│   ├── reddit.md                Reddit — r/MCP, r/netsec, r/commandline, r/LocalLLaMA
│   ├── hacker-news.md           Hacker News — Show HN + comments
│   ├── instagram.md             Instagram — 2 carousels, reel, stories
│   └── facebook.md              Facebook — Page + posts + group strategy
├── calendar/
│   └── LAUNCH-CALENDAR.md       Sept 9–30, day-by-day with checkboxes & time slots
├── assets/
│   ├── IMAGE-SPECS.md           Every image size, format, where it is used
│   └── VIDEO-SPECS.md           Video specs per platform, repurposing map
├── WARDEN-FACTS.md              Verified facts only (from root)
├── WARDEN-CONTENT-SOURCES.md    Evidence mapping (from root)
└── WARDEN-LAUNCH-CONTENT.md     Master content / safe-claims audit (from root)
```

---

## Quick Reference

### Key Links
| Asset | URL |
|---|---|
| GitHub | `github.com/Prof-bilal/Warden` |
| npm | `npm install -g warden-sandbox-cli` (v0.1.10) |
| Docs | `prof-bilal.github.io/Warden/` |
| Issues | `github.com/Prof-bilal/Warden/issues` |
| Releases | `github.com/Prof-bilal/Warden/releases` |

### Key Numbers (Verified Only)
| Number | Source |
|---|---|
| 18 MCP servers tested | `testdata/compat/matrix.yaml` |
| 14 pass / 2 conditional / 2 fail | same |
| 256 tests (all green on Sept 9, 2026) | `go test ./...` |
| 5 binaries (linux/darwin amd64+arm64, windows amd64) + SHA256SUMS | GitHub Releases v0.1.10 |
| 3 native backends + Docker fallback | `ARCHITECTURE.md` |
| MIT licensed | `LICENSE` |

### Key Phrases (Use Everywhere)
- "Deny by default"
- "Fail closed"
- "OS-level enforcement"
- "One YAML policy"
- "Transparent stdio"
- "Audit everything"

### What NOT to Say
- ❌ `88ms`, `25K`, `12.2M hashes`, `6 SSH keys`, `5/5 contained` — no reproducible harness
- ❌ "Production ready" — pre-1.0
- ❌ "Independently audited / certified" — not audited
- ❌ "Most secure / first MCP sandbox" — no basis
- ❌ "Blocks all attacks" — documented limitations exist

---

## How to Use This Folder

1. **Start with** `brand/GUIDELINES.md` — colors, voice, typography
2. **Generate visuals** with `brand/LOGO-PROMPTS.md` + `brand/BANNER-PROMPTS.md`
3. **Check specs** in `assets/IMAGE-SPECS.md` + `assets/VIDEO-SPECS.md` before exporting
4. **Pick a platform** in `platforms/` — each file is self-contained (setup → content → engagement)
5. **Follow the calendar** in `calendar/LAUNCH-CALENDAR.md` — Sept 9 = Day 1

Each `platforms/*.md` file lists **which image/video to attach to which post** (dimensions + file name). The calendar tells you **when** to post it.

---

## Color Reminder

| Token | Hex | Usage |
|---|---|---|
| Primary | `#6E93E8` | Logo, CTAs, highlights — marketing only |
| Background | `#10141A` | All marketing images |
| Surface | `#171C24` | Cards, code blocks |
| Surface 2 | `#1E242E` | Borders, dividers |
| Surface 3 | `#2A303C` | Secondary borders |
| Text | `#E8EBEF` | Headings |
| Muted | `#8D95A5` | Captions, secondary text |
| Grant | `#3FB27E` | Product UI only (not marketing) |
| Deny | `#E2604F` | Product UI only (not marketing) |
| Progress | `#D6A24A` | Product UI only (not marketing) |

> Marketing = blue/dark. Product = green/red/amber. They live in different contexts — don't mix.

---

## Existing Assets You Can Reuse Now

| Asset | Path | Use For |
|---|---|---|
| Explainer video | `warden-landing/public/videos/warden-sandbox-explainer.mp4` | LinkedIn, Facebook, YouTube (landscape) |
| Poster | `warden-landing/public/videos/warden-sandbox-explainer-poster.jpg` | Thumbnail |
| PH gallery HTML | `warden-landing/product-hunt-gallery/01-hero.html` … `06-install.html` | Screenshot for post images |
| PH gallery PNGs | `warden-landing/product-hunt-gallery/output/*.png` | Product Hunt + LinkedIn cards |

Regenerate PH gallery after color change:
```bash
cd warden-landing/product-hunt-gallery
node render.cjs
```
