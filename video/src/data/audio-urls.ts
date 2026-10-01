/**
 * Audio asset URLs, resolved at bundle time via webpack imports.
 * This avoids any dependency on publicDir/staticFile resolution during
 * rendering — the bundler emits the files and hands us final URLs.
 */
import musicBed from "../../public/audio/music-bed.wav";
import keyTick1 from "../../public/audio/key-tick-1.wav";
import keyTick2 from "../../public/audio/key-tick-2.wav";
import keyEnter from "../../public/audio/key-enter.wav";
import allowSfx from "../../public/audio/allow.wav";
import denySfx from "../../public/audio/deny.wav";
import whoosh from "../../public/audio/whoosh.wav";
import finale from "../../public/audio/finale.wav";

// Per-scene narration (edge-tts neural voice), one file per scene.
import nar01 from "../../public/audio/narration/01-hook.mp3";
import nar02 from "../../public/audio/narration/02-problem.mp3";
import nar03 from "../../public/audio/narration/03-access.mp3";
import nar04 from "../../public/audio/narration/04-intro.mp3";
import nar05 from "../../public/audio/narration/05-policy.mp3";
import nar06 from "../../public/audio/narration/06-demo.mp3";
import nar07 from "../../public/audio/narration/07-architecture.mp3";
import nar08 from "../../public/audio/narration/08-platforms.mp3";
import nar09 from "../../public/audio/narration/09-why.mp3";
import nar10 from "../../public/audio/narration/10-cta.mp3";

export const AUDIO_URLS = {
  "music-bed": musicBed,
  key1: keyTick1,
  key2: keyTick2,
  enter: keyEnter,
  allow: allowSfx,
  deny: denySfx,
  whoosh,
  finale,
} as const;

/** Narration URL per scene key (imports resolved at bundle time). */
export const NARRATION_URLS = {
  hook: nar01,
  problem: nar02,
  accessScenario: nar03,
  wardenIntro: nar04,
  policy: nar05,
  demo: nar06,
  architecture: nar07,
  platform: nar08,
  whyItMatters: nar09,
  cta: nar10,
} as const;
