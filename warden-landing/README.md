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

## Release status

The site is live and in production use. When shipping a new Warden release,
update `softwareVersion` in `app/page.tsx` so the structured-data metadata
matches the published npm package (`warden-sandbox-cli`, whose version lives
in `warden-starter/warden/build/npm-wrapper/package.json`).

## Keeping claims honest

- `components/Backends.tsx` and `components/Compatibility.tsx` hardcode
  status values ("Ready", "Verified", …)keep them in sync with
  `ROADMAP.md` and `warden-starter/warden/TESTING-PLATFORMS.md` as
  milestones land. Don't let the landing page claim something the repo
  hasn't shipped yet.
- Windows requires an elevated (Administrator) shell to attach the WFP
  filters and ETW audit session; without elevation Warden fails closed
  with a clear message rather than running unaudited. The authoritative
  cross-machine CI verification stateand the remaining cross-platform
  test-debt itemsare tracked in
  [`REMAINING_WORK.md`](https://github.com/Prof-bilal/Warden/blob/main/warden-starter/warden/REMAINING_WORK.md).

## Compatibility Matrix

The Warden MCP compatibility matrix (18 servers) has been validated:

- **✅ Pass (14)**: filesystem, github, slack, postgres, sqlite, brave-search, gdrive, git, memory, time, sequential-thinking, notion, linear, tavily
- **⚠️ Conditional (2)**: fetch (was a Warden bugambiguous file access, now fixed via `policy.Normalize`), kubernetes (inherentneeds per-deployment cluster API host + kubeconfig)
- **❌ Fail (2)**: docker (inherentrequires Docker daemon socket, cannot be sandboxed), playwright (schema-gapno fully arbitrary hosts & unix-socket grants)

See [docs/compatibility.md](../warden-starter/warden/docs/compatibility.md) for the full matrix and
[testdata/compat/](../warden-starter/warden/testdata/compat/) for regression fixtures.

Reduced-motion is respected both globally (`app/globals.css`) and inside
the visualizer specifically (`useReducedMotion` from Framer Motion).
