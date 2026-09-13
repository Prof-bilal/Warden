# Banner Prompts — Platform-Specific

All banners: dark `#10141A` background, blue `#6E93E8` accents, white `#E8EBEF` text, Geist Sans Bold for headlines, Geist Sans Regular for subtext. No gradients, no photos, no stock imagery.

Each prompt is for an AI image generator (Midjourney, DALL·E 3, Ideogram). Generate at the exact dimensions, then add text in **Figma** (don't rely on the generator for text — it will misspell).

---

## 1. X (Twitter) Header — 1500×500

**Text:**
- **Headline:** `Sandbox your MCP servers`
- **Subtext:** `Deny by default. Fail closed.`

**Safe area:** Keep text inside center 1260×330 — edges are cropped on mobile.

**AI Prompt (background only):**
```
Dark near-black (#10141A) abstract tech background, subtle geometric grid pattern in very dark gray (#171C24), faint blue (#6E93E8) horizontal line near bottom third, minimal and premium, no text, no icons, no gradients, flat, suitable for a developer tool header, 1500x500 banner, dark mode
```

**Figma text layout:**
- Left side (80px from left, vertically centered):
  - Headline: Geist Sans Bold, 42px, `#E8EBEF`, line-height 1.1
  - Subtext: Geist Sans Regular, 18px, `#8D95A5`, 16px below headline
- Right side: small shield icon (80×80) at 80px from right, vertically centered, `#6E93E8`

**Export:** `warden-banner-x-1500x500.png`

---

## 2. LinkedIn Cover — 1128×191

> LinkedIn crops aggressively. Keep all text in center 900×140.

**Text:**
- **Headline:** `The boundary MCP never had`
- **Subtext:** `Open source sandbox runtime · MIT licensed`

**AI Prompt (background only):**
```
Dark near-black (#10141A) minimal tech banner, very subtle dot grid in dark gray (#1E242E), thin blue (#6E93E8) vertical accent line on left edge (4px wide, full height), premium and clean, no text, no icons, flat, 1128x191 banner
```

**Figma text layout:**
- Left side (32px from left, vertically centered):
  - Headline: Geist Sans Bold, 24px, `#E8EBEF`
  - Subtext: Geist Sans Regular, 14px, `#8D95A5`, 6px below headline
- Right side: shield icon (48×48) at 32px from right

**Export:** `warden-banner-linkedin-1128x191.png`

---

## 3. Facebook Cover — 820×312

> Facebook cover is cropped to 640×312 on mobile. Keep text centered.

**Text:**
- **Headline:** `Run MCP servers safely`
- **Subtext:** `One policy. OS-level enforcement.`

**AI Prompt (background only):**
```
Dark near-black (#10141A) tech banner, subtle blue (#6E93E8) geometric shield watermark in background at 10% opacity, centered, large but faint, minimal, premium, no text, flat, 820x312 banner
```

**Figma text layout:**
- Centered (both axes):
  - Headline: Geist Sans Bold, 32px, `#E8EBEF`
  - Subtext: Geist Sans Regular, 16px, `#8D95A5`, 8px below
- Shield icon (40×40) above headline, centered, 12px gap

**Export:** `warden-banner-facebook-820x312.png`

---

## 4. YouTube Banner — 2560×1440

> Safe area for text: 1235×338 centered. Everything outside is cropped on TV/mobile.

**Text:**
- **Headline:** `WARDEN`
- **Subtext:** `Sandbox runtime for MCP servers — open source`

**AI Prompt (background only):**
```
Dark near-black (#10141A) wide tech banner, subtle grid and faint blue (#6E93E8) horizontal accent line across center, minimal premium dark theme for a developer tool YouTube channel, no text, no characters, flat, 2560x1440
```

**Figma text layout:**
- Centered in safe area:
  - Headline: Geist Sans Bold, 56px, `#E8EBEF`, tracking 0.08em
  - Subtext: Geist Sans Regular, 18px, `#8D95A5`, 12px below
- Shield icon (64×64) centered above headline, 16px gap

**Export:** `warden-banner-youtube-2560x1440.png`

---

## 5. Instagram Story Highlight Covers — 1080×1920 (per highlight)

Create 3 highlight covers (1080×1080 circle crop, but design at 1080×1920):

| Highlight | Icon | Text |
|---|---|---|
| `How it works` | Shield + code brackets `{}` | — (icon only) |
| `Install` | Terminal `>` | — (icon only) |
| `Security` | Shield + checkmark `✓` | — (icon only) |

**AI Prompt (per cover, background only):**
```
Circular icon design, dark near-black (#10141A) circle background, centered blue (#6E93E8) minimal icon — [shield / terminal / checkmark], flat vector, no text, 1080x1920 story format, icon in upper center (1080x1080 safe area)
```

**Figma:** Place orange icon (400×400) centered in 1080×1080 top area. No text — highlight title is set in Instagram app.

**Export:** `warden-highlight-how-1080x1920.png`, `warden-highlight-install-1080x1920.png`, `warden-highlight-security-1080x1920.png`

---

## After Generation

1. **Don't use AI-generated text** — re-typeset all headlines in Figma with Geist Sans
2. **Check safe areas** — preview on each platform (mobile + desktop)
3. **Export as PNG** (not JPG) to keep dark blacks true
4. **File naming:**
   ```
   warden-banner-x-1500x500.png
   warden-banner-linkedin-1128x191.png
   warden-banner-facebook-820x312.png
   warden-banner-youtube-2560x1440.png
   warden-highlight-how-1080x1920.png
   ```

## Tool Stack

| Task | Tool |
|---|---|
| Generate background | Midjourney / DALL·E 3 / Ideogram |
| Add text & layout | Figma (free) |
| Safe-area check | Figma frame with safe-area guides |
| Export | Figma → Export → PNG 2× |
