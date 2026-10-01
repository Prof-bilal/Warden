import React from "react";
import { Composition } from "remotion";
import "../public/fonts/fonts.css";
import "./styles/globals.css";
import { WardenMarketingVideo } from "./compositions/WardenMarketingVideo";
import { ECOSYSTEM_LAUNCH_DURATION, ECOSYSTEM_LAUNCH_FPS, WardenEcosystemLaunch } from "./compositions/EcosystemLaunch";
import { FPS, TOTAL_DURATION } from "./data/timeline";

/**
 * Remotion root — registers every composition.
 * `WardenMarketingVideo` is the main deliverable.
 */
export const RemotionRoot: React.FC = () => {
  return (
    <>
      <Composition
        id="WardenMarketingVideo"
        component={WardenMarketingVideo}
        durationInFrames={TOTAL_DURATION}
        fps={FPS}
        width={1920}
        height={1080}
      />
      <Composition
        id="WardenEcosystemLaunch"
        component={WardenEcosystemLaunch}
        durationInFrames={ECOSYSTEM_LAUNCH_DURATION}
        fps={ECOSYSTEM_LAUNCH_FPS}
        width={1920}
        height={1080}
      />
    </>
  );
};
