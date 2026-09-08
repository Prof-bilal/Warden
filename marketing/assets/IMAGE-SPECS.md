# Image Specs — Every Platform, Every Size

**Theme:** Dark `#0a0a0a` background · Orange `#f97316` accents · White `#ffffff` text · Inter + JetBrains Mono  
**Source for screenshots:** `warden-landing/product-hunt-gallery/01-hero.html` … `06-install.html` → `node render.cjs` → `output/*.png`

---

## 1. Size Reference (All Platforms)

| Platform | Image Type | Dimensions | Aspect | Format | Max Size | Where Used |
|---|---|---|---|---|---|---|
| **X/Twitter** | Post image | `1200×675` | 16:9 | PNG | 5 MB | Attached to launch thread, singles |
| **X/Twitter** | Profile picture | `400×400` | 1:1 | PNG/JPG | 2 MB | `@warden` avatar |
| **X/Twitter** | Header | `1500×500` | 3:1 | PNG/JPG | 2 MB | Profile header — see `brand/BANNER-PROMPTS.md` |
| **Instagram** | Post (square) | `1080×1080` | 1:1 | PNG/JPG | 30 MB | Carousel slides (preferred) |
| **Instagram** | Post (portrait) | `1080×1350` | 4:5 | PNG/JPG | 30 MB | Carousel slides (alt) |
| **Instagram** | Story / Reel cover | `1080×1920` | 9:16 | PNG/JPG | 30 MB | Stories, Reel thumbnails, highlight covers |
| **Instagram** | Profile picture | `320×320` | 1:1 | JPG | 2 MB | `@warden.sandbox` avatar (circle crop) |
| **Facebook** | Post image | `1200×630` | 1.91:1 | PNG/JPG | 8 MB | Launch post, shared posts |
| **Facebook** | Profile picture | `170×170` | 1:1 | PNG | 2 MB | Page avatar |
| **Facebook** | Cover | `820×312` | 2.63:1 | PNG/JPG | 2 MB | Page cover — see `brand/BANNER-PROMPTS.md` |
| **LinkedIn** | Post image | `1200×627` | 1.91:1 | PNG/JPG | 5 MB | Founder + technical posts |
| **LinkedIn** | Company logo | `300×300` | 1:1 | PNG | 2 MB | Company Page logo |
| **LinkedIn** | Cover | `1128×191` | 5.9:1 | PNG/JPG | 2 MB | Company Page cover — see `brand/BANNER-PROMPTS.md` |
| **Reddit** | Post image | `1200×630` | 1.91:1 | PNG/JPG | 20 MB | Optional attachment to technical post |
| **YouTube** | Thumbnail | `1280×720` | 16:9 | JPG/PNG | 2 MB | Explainer video thumbnail |
| **YouTube** | Banner | `2560×1440` | 16:9 | JPG/PNG | 6 MB | Channel banner — see `brand/BANNER-PROMPTS.md` |

---

## 2. Asset Inventory — What to Create

### A. Logo Variations (from `brand/LOGO-PROMPTS.md`)

| File | Dimensions | Used On |
|---|---|---|
| `warden-shield-primary-1024.png` | 1024×1024 | Source for all avatars |
| `warden-shield-primary-512.png` | 512×512 | GitHub avatar, docs |
| `warden-shield-primary-32.png` | 32×32 | Favicon |
| `warden-lockup-horizontal-2400x600.png` | 2400×600 | Banners, slides, README |
| `warden-avatar-512.png` | 512×512 (circle) | X, Instagram, Facebook, LinkedIn profiles |
| `warden-shield-outline-1024.png` | 1024×1024 | Light backgrounds, print |
| `warden-stamp-orange-1024.png` | 1024×1024 (rounded square) | Stickers, stamps |
| `warden-shield.svg` | Vector | Source file |

### B. Banner Images (from `brand/BANNER-PROMPTS.md`)

| File | Dimensions | Used On |
|---|---|---|
| `warden-banner-x-1500x500.png` | 1500×500 | X header |
| `warden-banner-linkedin-1128x191.png` | 1128×191 | LinkedIn cover |
| `warden-banner-facebook-820x312.png` | 820×312 | Facebook cover |
| `warden-banner-youtube-2560x1440.png` | 2560×1440 | YouTube channel |

### C. Post / Carousel Images

| File | Dimensions | Used On |
|---|---|---|
| `warden-carousel-problem-01-1080x1080.png` … `06` | 1080×1080 ×6 | Instagram carousel "The Problem" |
| `warden-carousel-how-01-1080x1080.png` … `06` | 1080×1080 ×6 | Instagram carousel "How It Works" |
| `warden-post-x-launch-1200x675.png` | 1200×675 | X launch thread image |
| `warden-post-linkedin-launch-1200x627.png` | 1200×627 | LinkedIn founder post |
| `warden-post-facebook-launch-1200x630.png` | 1200×630 | Facebook launch post |

### D. Screenshots (Capture from Terminal)

| File | Dimensions | How to Capture | Used On |
|---|---|---|---|
| `warden-terminal-run-1200x675.png` | 1200×675 | Terminal: `warden run --policy policy.yaml -- node server.js` | X, LinkedIn, Facebook |
| `warden-terminal-doctor-1200x675.png` | 1200×675 | Terminal: `warden doctor` | X, docs |
| `warden-terminal-blocked-1200x675.png` | 1200×675 | Terminal: blocked access attempt (red) | Instagram, LinkedIn |
| `warden-policy-example-800x600.png` | 800×600 | Code card: `policy.yaml` | All platforms |

**Screenshot tips:**
- Use dark terminal (e.g., Warp, iTerm with `#0a0a0a` bg)
- Font: JetBrains Mono, 14px
- Window chrome: rounded corners, no title bar
- Capture with `shot` or macOS `Cmd+Shift+4`

### E. PH Gallery Reuse

| Gallery HTML | Screenshot For | Post Image |
|---|---|---|
| `01-hero.html` | Hero visual | X launch, LinkedIn hero |
| `02-problem.html` | Problem diagram | Reddit, HN (link), Instagram slide 3 |
| `03-policy.html` | Policy code card | X tweet 2, Instagram slide 2 |
| `04-proof.html` | Proof table (verified claims) | LinkedIn technical, HN comment |
| `05-backends.html` | Backend matrix | X tweet 3, Instagram slide 3 |
| `06-install.html` | Install terminal | X tweet 9, all CTA posts |

Regenerate after orange rebrand:
```bash
cd warden-landing/product-hunt-gallery
# Update styles.css: --accent from green to #f97316, --bg to #0a0a0a
node render.cjs
ls output/*.png
```

---

## 3. File Naming Convention

```
warden-{category}-{platform}-{type}-{width}x{height}.png

Examples:
warden-shield-primary-1024.png
warden-banner-x-1500x500.png
warden-carousel-problem-01-1080x1080.png
warden-post-linkedin-launch-1200x627.png
warden-terminal-run-1200x675.png
```

Store generated images in `marketing/assets/images/` (create when needed) or directly in platform posts.

---

## 4. Export Checklist

- [ ] All PNGs exported at 2× for retina (Instagram/Facebook auto-downscale)
- [ ] Dark blacks are `#0a0a0a`, not `#000000` (true black looks cheap)
- [ ] Orange is exactly `#f97316` (check with eyedropper)
- [ ] Text is Inter / JetBrains Mono, not AI-generated text
- [ ] Safe areas checked on each platform (mobile + desktop preview)
- [ ] File sizes under platform limits (see table above)
- [ ] Alt text written for each image (see platform files)
