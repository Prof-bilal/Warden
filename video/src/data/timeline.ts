/**
 * Centralized timeline for the Warden marketing video.
 *
 * Durations are in SECONDS at 30 fps. Every scene reads its timing from
 * here — do not scatter raw frame numbers across components.
 *
 * Total: 322s = 5:22 (target was 5:00–5:20; CTA holds to 5:22 with fade).
 */
export const FPS = 30;

export const SCENES = {
  hook: 30, // 0:00–0:30
  problem: 45, // 0:30–1:15
  accessScenario: 45, // 1:15–2:00
  wardenIntro: 40, // 2:00–2:40
  policy: 30, // 2:40–3:10
  demo: 50, // 3:10–4:00
  architecture: 30, // 4:00–4:30
  platform: 20, // 4:30–4:50
  whyItMatters: 15, // 4:50–5:05
  cta: 17, // 5:05–5:22
} as const;

export type SceneKey = keyof typeof SCENES;

export const secondsToFrames = (s: number) => Math.round(s * FPS);

/** Frame offsets for each scene, derived automatically from SCENES. */
export const SCENE_START: Record<SceneKey, number> = (() => {
  const out = {} as Record<SceneKey, number>;
  let acc = 0;
  for (const [key, dur] of Object.entries(SCENES)) {
    out[key as SceneKey] = acc;
    acc += secondsToFrames(dur);
  }
  return out;
})();

export const TOTAL_DURATION = Object.values(SCENES).reduce(
  (acc, s) => acc + secondsToFrames(s),
  0
);

/** Main composition display name for humans (e.g. chapter markers). */
export const SCENE_LABELS: Record<SceneKey, string> = {
  hook: "Hook",
  problem: "The MCP problem",
  accessScenario: "Access scenario",
  wardenIntro: "Introducing Warden",
  policy: "Policy",
  demo: "Real demo",
  architecture: "Architecture",
  platform: "Platforms & sandboxes",
  whyItMatters: "Why it matters",
  cta: "Open source",
};
