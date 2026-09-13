# Brand Guidelines — Warden

**Theme:** Shield logo · Blue `#6E93E8` on dark `#10141A`  
**Inspiration:** Linear (dark premium), Vercel (editorial clarity), Github (developer trust)

---

## 1. Color Palette

### Marketing Palette (Social, Banners, Website, Slides)

| Token | Hex | Usage |
|---|---|---|
| `primary` | `#6E93E8` | Logo shield, CTAs, highlights, underlines, bullet dots |
| `primary-hover` | `#8DACF0` | Button hover, link hover |
| `bg` | `#10141A` | Page background, post backgrounds, banner backgrounds |
| `surface` | `#171C24` | Cards, code blocks, carousel slides |
| `surface-2` | `#1E242E` | Borders, dividers, code-block borders |
| `surface-3` | `#2A303C` | Secondary borders, dividers |
| `surface-4` | `#3A4150` | Tertiary borders, subtle structure |
| `text` | `#E8EBEF` | Headings, primary text on dark |
| `text-muted` | `#8D95A5` | Captions, secondary text, timestamps |

### Product Palette (CLI, Logs, Terminal — Do NOT Use for Marketing)

| Token | Hex | Usage |
|---|---|---|
| `grant` | `#3FB27E` | Checkmarks, "allowed" in product UI only |
| `deny` | `#E2604F` | Crosses, "blocked" in product UI only |
| `progress` | `#D6A24A` | Progress indicators, warnings in product UI only |

> **Rule:** Marketing = blue/dark. Product = green/red/amber. They never appear together in the same image — otherwise the image looks like a traffic light.

### Why Blue on Dark Works

- Blue on `#10141A` feels **technical and trustworthy** — like a developer tool, not a warning.
- Blue on dark = premium, modern. Linear, Vercel, and Github use this pattern.
- Most security tools use red/black or orange — blue **stands out** without feeling alarming.
- Cool tone = professional for a security tool (developer-focused, not consumer).

---

## 2. Typography

| Role | Font | Weight | Usage |
|---|---|---|---|
| Display/Headings | **Newsreader** or **GT Sectra** | 400–600 | Post headlines, banner headlines, carousel covers |
| Body | **Geist Sans** | 400–500 | Captions, slide body, post copy |
| Code | **Geist Mono** | 400 | `policy.yaml`, terminal commands, file paths |

- Display headline tracking: `-0.02em` (tight, editorial style)
- Body tracking: normal
- Code blocks: `0.875rem`, rounded corners `6px`, background `#0D1017`, border `#2A303C`

---

## 3. Logo Usage

### Primary Lockup
- **Icon:** Geometric shield (two angular lines meeting at a point, subtle gap in center = boundary)
- **Wordmark:** `WARDEN` — Geist Sans Bold, uppercase, tracking `0.08em`
- **Spacing:** Icon height = wordmark cap-height, gap = 0.4× icon width
- **Clear space:** At least 1× icon width on all sides

### Variations

| Variant | When to Use |
|---|---|
| Shield + WARDEN (horizontal) | Headers, banners, website, slides |
| Shield only (square) | Profile pictures, favicon, app icon, watermarks |
| Shield outline (stroke) | Light backgrounds, print, merch |
| Monochrome white on orange | Stamps, stickers, loading states |

### Don'ts
- Don't rotate the shield
- Don't add gradients, shadows, or 3D effects
- Don't place orange shield on orange background
- Don't stretch or change proportions
- Don't add taglines inside the logo lockup

See `LOGO-PROMPTS.md` for AI generation prompts.

---

## 4. Voice & Tone

| Principle | Do | Don't |
|---|---|---|
| **Honest** | "Pre-1.0, tested against 18 servers, 14 pass" | "Production ready", "Most secure" |
| **Technical** | "bwrap with unshared namespaces, WFP sublayer" | "AI-powered", "Game-changing" |
| **Humble** | "Still early — go find where it breaks" | "The only tool you need" |
| **Direct** | "Deny by default. Grant a folder." | "Leverage our cutting-edge solution" |
| **Evidence-pinned** | "See `docs/security.md:42-45`" | "Trust us, it's secure" |

### Sentence Style
- Short sentences. One idea per sentence.
- Active voice: "Warden blocks" not "Access is blocked by Warden"
- No superlatives: no "most", "best", "only", "first"
- Numbers only if verified (see `WARDEN-FACTS.md`)

---

## 5. Imagery Style

### Photography / Illustration
- **No stock photos** of hackers in hoodies, padlocks, or binary rain
- Prefer: terminal screenshots, policy code cards, boundary diagrams
- Diagram language: dark background, orange structural lines, white labels, muted gray secondary

### Diagram Language (Same as Landing Page)

| Element | Color | Style |
|---|---|---|
| Sandbox boundary | `#6E93E8` (blue) dashed or `#2A303C` solid | 2px dashed for boundary, 1px solid for structure |
| Granted path | `#3FB27E` with `✓` | Green + checkmark |
| Denied path | `#E2604F` with `✗` or dashed | Red + cross |
| Progress path | `#D6A24A` with `~` | Amber + tilde |
| Server box | `#171C24` fill, `#1E242E` border | Rounded 8px |
| Code card | `#0D1017` fill, `#2A303C` border | Rounded 8px, Geist Mono |

---

## 6. Content Rules

### Always Include
- Link to `github.com/Prof-bilal/Warden` (or UTM-tagged variant)
- One verified claim (from `WARDEN-FACTS.md` SAFE list)
- CTA: "Try `warden trace -- <your-server>`" or "File a compatibility report"

### Never Include
- Attack-simulation numbers (88ms, 25K, 6 SSH keys, etc.) — unverified
- "Production ready" / "Ready (macOS/Windows)" — verification pending
- "Audited / certified" — not audited
- "Blocks all attacks" — documented limitations exist (see `docs/security.md`)

### Hashtag Cap
- X: 2–3 max
- LinkedIn: 3–5
- Instagram: 5–10
- Reddit/HN: none
