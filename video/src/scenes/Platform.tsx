import React from "react";
import { AbsoluteFill, useCurrentFrame, useVideoConfig } from "remotion";
import { C, FONTS, Scene, riseIn, springIn } from "../components/ui";
import { PLATFORMS } from "../data/product";

/*
 * Scene 08 — Platforms & sandboxes (4:30–4:50)
 *
 * Simple platform cards using the verified backend matrix from the README.
 * Short by design — this proves Warden operates at the sandbox/runtime layer.
 */
export const Platform: React.FC = () => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();

  const titleS = riseIn(frame, fps, 4);
  const noteS = riseIn(frame, fps, Math.round(14.5 * fps));

  return (
    <Scene fadeOutAfter={0.7}>
      <AbsoluteFill
        style={{
          backgroundImage:
            "radial-gradient(rgba(255,255,255,0.04) 1px, transparent 1px)",
          backgroundSize: "36px 36px",
        }}
      />

      {/* Title */}
      <div style={{ position: "absolute", top: 100, left: 0, right: 0, textAlign: "center", ...titleS }}>
        <div
          style={{
            fontFamily: FONTS.mono,
            fontSize: 22,
            letterSpacing: "0.35em",
            textTransform: "uppercase",
            color: C.faint,
            marginBottom: 12,
          }}
        >
          Under the hood
        </div>
        <div style={{ fontSize: 58, fontWeight: 700, letterSpacing: "-0.02em" }}>
          OS-native sandboxes. No VM required.
        </div>
      </div>

      {/* Cards */}
      <div
        style={{
          position: "absolute",
          top: 330,
          left: 0,
          right: 0,
          display: "flex",
          justifyContent: "center",
          gap: 30,
        }}
      >
        {PLATFORMS.map((p, i) => (
          <PlatformCard
            key={p.os}
            os={p.os}
            backend={p.backend}
            detail={p.detail}
            delay={Math.round((1.0 + i * 0.5) * fps)}
            frame={frame}
            fps={fps}
          />
        ))}
      </div>

      {/* fail-closed note */}
      <div
        style={{
          position: "absolute",
          bottom: 90,
          left: 0,
          right: 0,
          textAlign: "center",
          fontSize: 28,
          color: C.muted,
          ...noteS,
        }}
      >
        If the sandbox primitives can't be applied, Warden{" "}
        <span style={{ color: C.redSoft }}>refuses to run</span>.
      </div>
    </Scene>
  );
};

/* ────────────────────────────  card  ──────────────────────────── */

const PlatformCard: React.FC<{
  os: string;
  backend: string;
  detail: string;
  delay: number;
  frame: number;
  fps: number;
}> = ({ os, backend, detail, delay, frame, fps }) => {
  const s = springIn(frame, fps, delay, { damping: 200, stiffness: 80 });

  return (
    <div
      style={{
        width: 420,
        border: `1px solid ${C.border}`,
        background: C.panel,
        borderRadius: 16,
        padding: "30px 30px",
        opacity: s,
        transform: `translateY(${(1 - s) * 24}px)`,
      }}
    >
      <div style={{ fontFamily: FONTS.mono, fontSize: 22, color: C.faint, letterSpacing: "0.16em" }}>
        {os}
      </div>
      <div style={{ fontSize: 40, fontWeight: 700, margin: "10px 0 8px" }}>{backend}</div>
      <div style={{ fontFamily: FONTS.mono, fontSize: 19, color: C.muted, lineHeight: 1.5 }}>
        {detail}
      </div>
    </div>
  );
};
