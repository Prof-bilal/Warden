# Logo Prompts — Warden Shield

**Concept:** Minimalist geometric shield — two angular lines meeting at a point, subtle vertical gap in center (the "boundary"). No detail, no texture, no gradients. Flat, vector-style.  
**Colors:** Orange `#f97316` icon on dark `#0a0a0a` or transparent. White `#ffffff` variant for dark backgrounds if needed.  
**Tools:** Works in Midjourney v6, DALL·E 3, Stable Diffusion XL, Ideogram. After generation, vectorize with **Vectorizer.AI** or **Illustrator Image Trace**.

> Replace `[ORANGE]` with `#f97316` and `[DARK]` with `#0a0a0a` if the tool supports hex. Otherwise use "vibrant orange" and "near-black".

---

## Prompt 1 — Primary Shield Icon (Core Mark)

**Use:** Profile pictures (X, Instagram, Facebook, LinkedIn, GitHub), favicon, app icon, watermark

```
Minimalist geometric shield icon, flat vector logo, two sharp angular lines forming a shield shape meeting at a bottom point with a subtle vertical gap in the center representing a boundary, solid vibrant orange (#f97316) on transparent background, no text, no gradients, no shadows, no 3D, clean and bold, suitable for a developer security tool, works at 32x32 pixels, centered, symmetrical --style raw --ar 1:1
```

**Negative prompt (if supported):** `no padlock, no key, no lock, no gradients, no shadows, no 3D, no text, no letters, no complex details, no illustration, no mascot`

**Export:** `warden-shield-primary-1024.png` (1024×1024), then `warden-shield-primary-512.png`, `warden-shield-primary-32.png` (favicon)

---

## Prompt 2 — Shield + Wordmark Lockup (Horizontal)

**Use:** Website header, X/Twitter header left, LinkedIn cover left, slide decks, README header

```
Minimalist tech logo lockup, left: geometric shield icon in vibrant orange (#f97316) — two angular lines forming a shield with subtle center gap, flat vector — right: the word "WARDEN" in bold uppercase sans-serif (Inter Black style), white (#ffffff) text, generous letter spacing, on near-black (#0a0a0a) background, horizontal layout, icon height equals text cap-height, clean modern flat design, no gradients, no shadows, centered
```

**Negative:** `no padlock, no gradients, no tagline, no extra text, no 3D`

**Export:** `warden-lockup-horizontal-2400x600.png` (2400×600), `warden-lockup-horizontal-1200x300.png`

**Layout spec:** Icon 1×, wordmark cap-height = icon height, gap = 0.4× icon width, clear space = 1× icon width all sides.

---

## Prompt 3 — Favicon / Social Avatar (Circular)

**Use:** X/Instagram/Facebook/LinkedIn profile pictures, GitHub avatar, favicon

```
Circular app icon, geometric shield centered inside a dark circle (#0a0a0a), shield in vibrant orange (#f97316) — same angular shield with center gap as before, minimal and bold, no text, flat vector, perfect symmetry, works at 32x32 pixels, high contrast, centered, circular background
```

**Negative:** `no text, no letters, no gradients, no shadows, no border`

**Export:** `warden-avatar-512.png` (512×512 circle), `warden-avatar-320.png`, `warden-favicon-32.png`

**Safe area:** Shield occupies 60% of circle diameter.

---

## Prompt 4 — Outline Variant (For Light Backgrounds / Print)

**Use:** Light-mode docs, print, merch, watermarks on light photos

```
Minimalist geometric shield icon, outline version — thin stroke (2px) forming the same angular shield shape with center gap, vibrant orange (#f97316) stroke on white background, no fill, no text, flat vector, clean and light, suitable for print
```

**Negative:** `no fill, no solid, no gradients, no text`

**Export:** `warden-shield-outline-1024.png` (transparent), `warden-shield-outline-whitebg-1024.png`

---

## Prompt 5 — Monochrome Stamp (White on Orange)

**Use:** Stickers, stamps, loading spinner, monochrome contexts, merch

```
Geometric shield icon, solid white (#ffffff) shield on vibrant orange (#f97316) square background with rounded corners (12px radius), same angular shield with center gap, minimal flat vector, high contrast, bold, no text
```

**Negative:** `no gradients, no shadows, no text, no outline`

**Export:** `warden-stamp-orange-1024.png` (1024×1024, rounded square), `warden-stamp-orange-512.png`

---

## Prompt 6 — Animated Intro (For Video)

**Use:** First 2 seconds of explainer video, reels intro

```
Description for motion: Dark background (#0a0a0a), orange geometric shield icon fades in at center (scale 0.8 → 1.0, 0.4s ease-out), then the word "WARDEN" fades in to the right of the shield (opacity 0 → 1, 0.3s, 0.1s delay after shield). No other elements. Hold 0.5s. Total 1.2s. Flat, no gradients, no camera movement.
```

**Not an image prompt** — give this description to your motion tool (After Effects, Remotion, or HyperFrames). Use the static shield from Prompt 1 as the source asset.

---

## Post-Generation Checklist

- [ ] Vectorize the chosen shield with Vectorizer.AI (or Illustrator Image Trace → Expand)
- [ ] Check at 32×32 — is it still recognizable?
- [ ] Check on both `#0a0a0a` and `#ffffff` — does it work on both?
- [ ] Export all sizes listed above
- [ ] Save source `.svg` as `warden-shield.svg` in `marketing/brand/assets/` (create folder if needed)

## Recommended Tool Stack

| Task | Tool |
|---|---|
| Generate | Midjourney v6 (`--style raw --ar 1:1`) or DALL·E 3 |
| Vectorize | Vectorizer.AI (free) or Adobe Illustrator |
| Edit | Figma (free) — for lockup spacing, color tweaks |
| Favicon | RealFaviconGenerator.net — from 512px PNG |

## File Naming

```
warden-shield-primary-1024.png
warden-shield-primary-512.png
warden-shield-primary-32.png
warden-lockup-horizontal-2400x600.png
warden-avatar-512.png
warden-avatar-320.png
warden-favicon-32.png
warden-shield-outline-1024.png
warden-stamp-orange-1024.png
warden-shield.svg          # vector source
```
