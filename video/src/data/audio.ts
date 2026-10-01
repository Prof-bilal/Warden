/**
 * Centralized audio event schedule for the whole video.
 *
 * Events are declared per scene in SCENE-LOCAL SECONDS, then flattened to
 * global frames using SCENE_START from data/timeline.ts — so re-timing a
 * scene automatically moves its sounds. Keep every new sound here; never
 * scatter raw <Audio> tags inside scenes.
 *
 * Volume notes: the music bed sits low so narration can be added on top;
 * SFX are mixed just above it.
 */
import { SCENE_START, type SceneKey } from "./timeline";

export const SFX = {
  key1: { src: "key-tick-1.wav", volume: 0.45 },
  key2: { src: "key-tick-2.wav", volume: 0.45 },
  enter: { src: "key-enter.wav", volume: 0.5 },
  allow: { src: "allow.wav", volume: 0.55 },
  deny: { src: "deny.wav", volume: 0.6 },
  whoosh: { src: "whoosh.wav", volume: 0.45 },
  finale: { src: "finale.wav", volume: 0.65 },
} as const;

export type SfxKey = keyof typeof SFX;

export const MUSIC_BED = { src: "music-bed.wav", volume: 0.3 };

/**
 * Narration voiceover: one mp3 per scene, dropped at each scene start with
 * a small lead-in so speech begins just after the scene cut. Volume sits
 * above the music bed; SFX tuck underneath narration where they collide.
 */
export const NARRATION = {
  volume: 1.0,
  /** Delay from scene start to narration start, in scene-local seconds. */
  leadIn: 0.6,
  /** Duck the music bed under narration. */
  musicDuck: 0.55,
} as const;

/** Per-scene SFX events: [scene-local second, sound]. */
const SCENE_EVENTS: Record<SceneKey, [number, SfxKey][]> = {
  hook: [
    // typing `mcp-server start`
    [0.9, "key1"], [1.1, "key2"], [1.3, "key1"], [1.5, "key2"],
    [1.7, "key1"], [1.9, "key2"], [2.1, "key1"], [2.2, "enter"],
    // checkmarks
    [2.6, "key1"], [2.9, "key2"], [3.2, "enter"],
    // typing `ls ~`
    [5.0, "key1"], [5.2, "key2"], [5.4, "key1"], [5.6, "key2"],
    [5.8, "key1"], [6.0, "key2"], [6.2, "enter"],
    // listings cascade
    [6.5, "key1"], [6.8, "key2"], [7.1, "key1"], [7.4, "key2"], [7.7, "enter"],
    // freeze + statements
    [11.5, "whoosh"],
    [14.0, "whoosh"],
    [20.5, "whoosh"],
  ],
  problem: [
    [1.2, "whoosh"], // AI node
    [2.6, "whoosh"], // MCP node
    [4.2, "whoosh"], // computer node
    [8.0, "whoosh"], // resources expand
    [14.0, "whoosh"], // capability tree
    [21.5, "finale"], // Useful ≠ Unrestricted
  ],
  accessScenario: [
    [1.2, "whoosh"], // request 1
    [3.8, "allow"], // ALLOWED badge
    [7.5, "whoosh"], // request 2
    [10.2, "deny"], // sensitive highlight
    [27.5, "finale"], // the question
  ],
  wardenIntro: [
    [0.3, "finale"], // brand reveal
    [7.0, "whoosh"], // boundary draws
    [11.0, "whoosh"], // MCP travels in
    [16.0, "deny"], // sensitive resources outside
  ],
  policy: [
    [1.0, "whoosh"], // policy card in
    [4.6, "allow"], [5.3, "allow"], [6.0, "deny"],
    [6.8, "allow"], [7.5, "deny"],
    [8.2, "allow"], [8.9, "deny"],
  ],
  demo: [
    [1.0, "whoosh"], // policy card
    [4.5, "whoosh"], // run terminal in
    [6.1, "key1"], [6.3, "key2"], [6.5, "key1"], [6.7, "key2"],
    [6.9, "key1"], [7.1, "key2"], [7.3, "enter"],
    [9.2, "allow"], // ✓ Sandbox active
    [22.0, "whoosh"], // logs terminal
    [24.1, "allow"], [25.6, "allow"], [27.1, "deny"], [28.6, "deny"],
    [36.5, "whoosh"], // doctor terminal
    [37.1, "key1"], [37.3, "key2"], [37.5, "enter"],
    [40.8, "allow"], // Status: READY
  ],
  architecture: [
    [3.5, "allow"], // allow pulse
    [9.5, "deny"], // deny pulse
    [15.5, "allow"], // allow pulse 2
  ],
  platform: [
    [1.0, "whoosh"], [1.5, "whoosh"], [2.0, "whoosh"], [2.5, "whoosh"],
  ],
  whyItMatters: [
    [0.8, "whoosh"], // left card
    [1.4, "whoosh"], // right card
    [9.0, "finale"], // mantra
  ],
  cta: [
    [1.0, "key1"], [2.4, "key2"], [3.8, "key1"], [5.2, "key2"], [5.4, "enter"],
    [9.0, "whoosh"], // clone command
    [12.5, "finale"], // final lockup
  ],
};

/** Flattened schedule: global frame + sound. */
export const AUDIO_EVENTS: { frame: number; sfx: SfxKey }[] = (
  Object.keys(SCENE_EVENTS) as SceneKey[]
).flatMap((scene) =>
  SCENE_EVENTS[scene].map(([localS, sfx]) => ({
    frame: SCENE_START[scene] + Math.round(localS * 30),
    sfx,
  }))
).sort((a, b) => a.frame - b.frame);
