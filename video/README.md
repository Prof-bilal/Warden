# Warden — Marketing Video (Remotion)

A 5:22 programmatic marketing video for **Warden**, the sandbox runtime for
MCP servers, built entirely with Remotion + React + TypeScript.
1920×1080 @ 30fps.

## Ecosystem launch video

`WardenEcosystemLaunch` is a 66-second, caption-led launch cut built for social
posting and product announcements. It introduces the new candidate policy packs,
reversible multi-client setup, standard MCP connections, creator evidence and the
open-source call to action. It is marketing-led rather than a CLI walkthrough.
It intentionally ships without narration or music so it is ready for silent
autoplay; add licensed music or a voice track in your target platform if desired.

```bash
npm run render:launch
```

This writes `out/warden-ecosystem-launch.mp4`.

> Everything shown on screen is verified against the Warden repository
> (README, ARCHITECTURE.md, docs/cli.md, docs/quickstart.md, examples/*.yaml).
> Policy syntax, CLI output formats, platform backends, and install commands
> are the real ones. Nothing is invented and no unsupported security claims
> are made.

## Commands

```bash
npm install                # install dependencies
npm run dev                # open Remotion Studio to preview/edit
npm run typecheck          # tsc --noEmit
npm run render             # render out/warden-marketing.mp4 (≈10 min)
```

Production render:

```bash
npx remotion render WardenMarketingVideo out/warden-marketing.mp4
```

On memory-constrained machines, render in 1200-frame chunks and concatenate
(see `remotion.config.ts`; chunks used: `--frames=0-1199`, `1200-2399`, …).

## Structure

```
src/
├── Root.tsx                     # registers the composition + fonts
├── index.ts                     # registerRoot entry
├── compositions/
│   └── WardenMarketingVideo.tsx # assembles all 10 scenes
├── scenes/                      # one file per scene (see below)
├── components/
│   ├── Terminal.tsx             # reusable terminal: typing, cursor, blocks
│   ├── PermissionRow.tsx        # reusable allow/deny row with pulse
│   ├── Logo.tsx                 # inline-SVG shield mark + wordmark
│   └── ui.tsx                   # palette, springs, Scene wrapper, badges
├── data/
│   ├── timeline.ts              # centralized scene durations (single source)
│   ├── product.ts               # verified Warden facts
│   └── terminal.ts              # terminal script types
└── styles/globals.css           # design tokens
```

## Timeline (data/timeline.ts)

| Scene | Time | Content |
|---|---|---|
| 01 Hook | 0:00–0:30 | Terminal opening, sensitive files highlight, hook question |
| 02 Problem | 0:30–1:15 | AI → MCP → Your Computer diagram, "Useful ≠ Unrestricted" |
| 03 Access Scenario | 1:15–2:00 | Two tool calls: necessary vs sensitive, the boundary question |
| 04 Warden Intro | 2:00–2:40 | Brand reveal, MCP enters the Warden boundary |
| 05 Policy | 2:40–3:10 | Real policy.yaml + permission rows (warden run summary format) |
| 06 Real Demo | 3:10–4:00 | warden run + warden logs with real output formats |
| 07 Architecture | 4:00–4:30 | Allow/deny pulses through the boundary, fail-closed note |
| 08 Platforms | 4:30–4:50 | bubblewrap / Seatbelt / AppContainer / Docker cards |
| 09 Why It Matters | 4:50–5:05 | Without vs with Warden, mantra hold |
| 10 CTA | 5:05–5:22 | Install path, git clone, final lockup, fade to black |

Adjust any scene duration in `data/timeline.ts` — scene offsets are computed
automatically.

## Audio

All sound is generated programmatically — no external samples:

- `scripts/gen-audio.js` synthesizes the music bed (ambient D-minor pad with
  an intensity arc per chapter) and 7 SFX (keystroke ticks, enter, allow
  chime, deny buzz, whoosh, finale swell) as WAV files into `public/audio/`.
  Re-run it after changing the timeline length: `node scripts/gen-audio.js`.
- `src/data/audio.ts` is the centralized SFX schedule — events are declared
  per scene in scene-local seconds and flattened to global frames, so
  re-timing a scene automatically moves its sounds.
- `src/components/AudioTrack.tsx` mounts everything once in the composition;
  the music bed fades out over the last 4 seconds.

The mix sits low (bed ≈ −12 dB, SFX above it) so a narration track can be
added on top without re-balancing. `voiceover.md` contains the full narration
script, matched scene by scene to the timeline (~5:20 runtime, calm/technical
tone).

## Fonts

Inter + JetBrains Mono are vendored in `public/fonts/` (woff2, loaded via
`Root.tsx`) so renders never depend on network font loading.
