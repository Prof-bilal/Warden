# Brand Guidelines — Warden

**Theme:** Shield logo · Orange `#f97316` on dark `#0a0a0a`  
**Inspiration:** Brave (orange security), Reddit (warm community), Linear (dark premium)

---

## 1. Color Palette

### Marketing Palette (Social, Banners, Website, Slides)

| Token | Hex | Usage |
|---|---|---|
| `primary` | `#f97316` | Logo shield, CTAs, highlights, underlines, bullet dots |
| `primary-hover` | `#ea580c` | Button hover, link hover |
| `primary-soft` | `#fff7ed` | Light background washes (if needed for print) |
| `bg` | `#0a0a0a` | Page background, post backgrounds, banner backgrounds |
| `surface` | `#1a1a1a` | Cards, code blocks, carousel slides |
| `surface-2` | `#27272a` | Borders, dividers, code-block borders |
| `text` | `#ffffff` | Headings, primary text on dark |
| `text-muted` | `#a1a1aa` | Captions, secondary text, timestamps |
| `text-dim` | `#71717a` | Footnotes, disclaimers |

### Product Palette (CLI, Logs, Terminal — Do NOT Use for Marketing)

| Token | Hex | Usage |
|---|---|---|
| `granted` | `#22c55e` | Checkmarks, "allowed" in product UI only |
| `denied` | `#ef4444` | Crosses, "blocked" in product UI only |

> **Rule:** Marketing = orange/dark. Product = green/red. They never appear together in the same image — otherwise the image looks like a traffic light.

### Why Orange on Dark Works

- Brave proved orange can feel **security-grade** when paired with dark, not white.
- Orange on white = warning/caution. Orange on `#0a0a0a` = premium, like Netflix, Brex, Linear.
- Most dev tools use blue/purple — orange **stands out** in the feed without shouting.
- Warm tone = approachable for a security tool (less intimidating than red/black).

---

## 2. Typography

| Role | Font | Weight | Usage |
|---|---|---|---|
| Headings | **Inter** or **Geist Sans** | 700–800, uppercase for "WARDEN" | Post headlines, banner headlines, carousel covers |
| Body | **Inter** | 400–500 | Captions, slide body, post copy |
| Code | **JetBrains Mono** or **Geist Mono** | 400 | `policy.yaml`, terminal commands, file paths |

- Headline tracking: `0.02em` (slightly loose for uppercase)
- Code blocks: `0.9em`, rounded corners `8px`, background `#1a1a1a`, border `#27272a`

---

## 3. Logo Usage

### Primary Lockup
- **Icon:** Geometric shield (two angular lines meeting at a point, subtle gap in center = boundary)
- **Wordmark:** `WARDEN` — Inter Black, uppercase, tracking `0.08em`
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
| Sandbox boundary | `#f97316` (orange) dashed or `#27272a` solid | 2px dashed for boundary, 1px solid for structure |
| Granted path | `#ffffff` with `✓` | White + checkmark |
| Denied path | `#71717a` with `✗` or dashed red | Muted + cross |
| Server box | `#1a1a1a` fill, `#27272a` border | Rounded 8px |
| Code card | `#1a1a1a` fill, `#27272a` border | Rounded 8px, JetBrains Mono |

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
