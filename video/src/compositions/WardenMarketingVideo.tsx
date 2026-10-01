import React from "react";
import { AbsoluteFill, Sequence } from "remotion";
import {
  SCENES,
  SCENE_START,
  secondsToFrames,
} from "../data/timeline";
import { Hook } from "../scenes/Hook";
import { Problem } from "../scenes/Problem";
import { AccessScenario } from "../scenes/AccessScenario";
import { WardenIntro } from "../scenes/WardenIntro";
import { Policy } from "../scenes/Policy";
import { WardenDemo } from "../scenes/WardenDemo";
import { Architecture } from "../scenes/Architecture";
import { Platform } from "../scenes/Platform";
import { WhyItMatters } from "../scenes/WhyItMatters";
import { CTA } from "../scenes/CTA";
import { AudioTrack } from "../components/AudioTrack";

/**
 * WardenMarketingVideo — the 5-minute marketing composition.
 *
 * Scenes are laid out on a single absolute timeline (data/timeline.ts).
 * Each scene is mounted in a <Sequence> at its computed offset; scenes read
 * scene-local time via useCurrentFrame().
 */
export const WardenMarketingVideo: React.FC = () => {
  return (
    <AbsoluteFill style={{ backgroundColor: "#09090b" }}>
      <AudioTrack />
      <Sequence from={SCENE_START.hook} durationInFrames={secondsToFrames(SCENES.hook)}>
        <Hook />
      </Sequence>
      <Sequence from={SCENE_START.problem} durationInFrames={secondsToFrames(SCENES.problem)}>
        <Problem />
      </Sequence>
      <Sequence from={SCENE_START.accessScenario} durationInFrames={secondsToFrames(SCENES.accessScenario)}>
        <AccessScenario />
      </Sequence>
      <Sequence from={SCENE_START.wardenIntro} durationInFrames={secondsToFrames(SCENES.wardenIntro)}>
        <WardenIntro />
      </Sequence>
      <Sequence from={SCENE_START.policy} durationInFrames={secondsToFrames(SCENES.policy) - 1}>
        <Policy />
      </Sequence>
      <Sequence from={SCENE_START.demo} durationInFrames={secondsToFrames(SCENES.demo) - 1}>
        <WardenDemo />
      </Sequence>
      <Sequence from={SCENE_START.architecture} durationInFrames={secondsToFrames(SCENES.architecture) - 1}>
        <Architecture />
      </Sequence>
      <Sequence from={SCENE_START.platform} durationInFrames={secondsToFrames(SCENES.platform) - 1}>
        <Platform />
      </Sequence>
      <Sequence from={SCENE_START.whyItMatters} durationInFrames={secondsToFrames(SCENES.whyItMatters) - 1}>
        <WhyItMatters />
      </Sequence>
      <Sequence from={SCENE_START.cta} durationInFrames={secondsToFrames(SCENES.cta)}>
        <CTA />
      </Sequence>
    </AbsoluteFill>
  );
};
