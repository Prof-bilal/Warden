# warden-landing

Landing page for Warden, built with Next.js 14 (App Router), TypeScript,
Tailwind CSS, and Framer Motion for the one animated moment on the page —
the interactive sandbox demo in the hero.

## Setup

```bash
npm install
npm run dev
```

Then open http://localhost:3000.

> This was written without running `npm install` — it was built in an
> offline sandbox with no network access, so the code hasn't been through
> an actual build. It should work as-is with the versions pinned in
> package.json, but if you hit a version mismatch on first install, it's
> most likely Next/Tailwind needing a minor bump — run `npm install`
> and check the terminal output first.

## Before you deploy

1. **Replace every `yourname` placeholder** in `components/Nav.tsx`,
   `components/Hero.tsx`, `components/Cta.tsx`, and `components/Footer.tsx`
   with your real GitHub org/repo path and Go module path (should match
   whatever you set in the main repo's `go.mod`).
2. **`components/Backends.tsx` and `components/Compatibility.tsx`
   hardcode status values** ("Ready", "In progress", "Verified",
   "Testing") — keep these in sync with `ROADMAP.md` as milestones land.
   Don't let the landing page claim something the repo hasn't shipped yet.
3. Add a real OG image / favicon — currently there's just text metadata
   in `app/layout.tsx`.

## Design notes

- **Colors and type scale** live in `tailwind.config.ts` — the palette is
  built around the product's actual grant/deny mechanic (green = granted,
  red = denied, amber = in progress) used structurally, not as decoration.
- **Geist Sans/Mono** are loaded via the official `geist` npm package
  through `next/font` — no external font requests at runtime.
- **The only entrance/scroll animation on the page is the hero's
  `SandboxVisualizer`** (`components/SandboxVisualizer.tsx`), which cycles
  through simulated access attempts against a toggleable policy. Everything
  else is static by design — see `/mnt/skills/public/frontend-design` if
  you're extending this, specifically the guidance against scattering
  fade-in effects across every section.
- Reduced-motion is respected both globally (`app/globals.css`) and inside
  the visualizer specifically (`useReducedMotion` from Framer Motion).
