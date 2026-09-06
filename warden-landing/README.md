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

## Platform Support

| Platform | Backend | Status |
|----------|---------|--------|
| **Linux** | bubblewrap (unprivileged namespaces) | ✅ Ready |
| **macOS** | sandbox-exec, with Docker fallback | ✅ Ready |
| **Windows** | AppContainer + WFP + Job Objects + ETW audit | ✅ Ready |
| **Overall** | — | ✅ Ready |

## Compatibility Matrix

The Warden MCP compatibility matrix (18 servers) has been validated:

- **✅ Pass (14)**: filesystem, github, slack, postgres, sqlite, brave-search, gdrive, git, memory, time, sequential-thinking, notion, linear, tavily
- **⚠️ Conditional (2)**: fetch (was a Warden bug — ambiguous file access, now fixed via `policy.Normalize`), kubernetes (inherent — needs per-deployment cluster API host + kubeconfig)
- **❌ Fail (2)**: docker (inherent — requires Docker daemon socket, cannot be sandboxed), playwright (schema-gap — missing wildcard hosts & unix-socket grants)

See [docs/compatibility.md](docs/compatibility.md) for the full matrix and [testdata/compat/](testdata/compat/) for regression fixtures.
- Reduced-motion is respected both globally (`app/globals.css`) and inside
  the visualizer specifically (`useReducedMotion` from Framer Motion).
