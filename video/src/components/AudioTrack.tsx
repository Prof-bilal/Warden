import React from "react";
import { Audio, interpolate, Sequence, useVideoConfig } from "remotion";
import { AUDIO_EVENTS, MUSIC_BED, NARRATION, SFX } from "../data/audio";
import { AUDIO_URLS, NARRATION_URLS } from "../data/audio-urls";
import { SCENE_START, SCENES, secondsToFrames, type SceneKey } from "../data/timeline";

/**
 * Renders every sound in the video from the centralized schedule
 * (data/audio.ts). Mounted once inside the main composition.
 *
 * Layers:
 *  1. Narration — per-scene mp3 starting SCENE_START + leadIn.
 *  2. Music bed — full length, ducked to musicDuck while narration plays,
 *     faded out over the last 4 seconds.
 *  3. SFX — scheduled per scene in data/audio.ts.
 */
export const AudioTrack: React.FC = () => {
  const { durationInFrames } = useVideoConfig();

  const sceneKeys = Object.keys(SCENE_START) as SceneKey[];
  const narrationSpans = sceneKeys.map((key) => ({
    key,
    from: SCENE_START[key] + Math.round(NARRATION.leadIn * 30),
    to:
      SCENE_START[key] +
      secondsToFrames(SCENES[key]) -
      5, // small tail so ducking releases between scenes
  }));

  /** Is narration audible at global frame f? */
  const narrationAt = (f: number) =>
    narrationSpans.some((s) => f >= s.from && f <= s.to);

  const bedVolume = (f: number) => {
    const fade = interpolate(
      f,
      [durationInFrames - 120, durationInFrames - 1],
      [MUSIC_BED.volume, 0],
      { extrapolateLeft: "clamp", extrapolateRight: "clamp" }
    );
    const duck = narrationAt(f)
      ? MUSIC_BED.volume * NARRATION.musicDuck
      : MUSIC_BED.volume;
    return Math.min(fade, duck);
  };

  return (
    <>
      {/* Narration: one track per scene */}
      {narrationSpans.map(({ key, from }) => (
        <Sequence key={`nar-${key}`} from={from}>
          <Audio src={NARRATION_URLS[key]} volume={NARRATION.volume} />
        </Sequence>
      ))}

      {/* Music bed */}
      <Audio src={AUDIO_URLS["music-bed"]} volume={bedVolume} />

      {/* SFX */}
      {AUDIO_EVENTS.map(({ frame, sfx }) => (
        <Sequence key={`${frame}-${sfx}`} from={frame}>
          <Audio src={AUDIO_URLS[sfx]} volume={SFX[sfx].volume * 0.7} />
        </Sequence>
      ))}
    </>
  );
};
