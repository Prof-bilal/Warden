import React from "react";
import { AbsoluteFill, interpolate, useCurrentFrame, useVideoConfig } from "remotion";
import { C, FONTS, Scene, riseIn, springIn } from "../components/ui";
import { Logo } from "../components/Logo";
import { REPO_URL, REPO_SHORT, INSTALL_COMMANDS, MANTRA, DOCS_URL } from "../data/product";

/*
 * Scene 10 — Open source / CTA (5:05–5:22)
 *
 * Real install path, real clone URL. No stars, no statistics — the repo
 * doesn't publish any in its README and none are invented here.
 * Final lockup holds ~4.5s, then fades to black.
 */
export const CTA: React.FC = () => {
  const frame = useCurrentFrame();
  const { fps, durationInFrames } = useVideoConfig();

  /* ── timing (scene-local seconds) ──
   * 0.3   kicker
   * 1.0   install commands type in sequentially
   * 9.0   clone command
   * 12.5  transition to final lockup
   * 12.5–17  hold mantra + repo (≈4.5s), then fade to black
   */
  const t = {
    kicker: 0.3,
    cmds: 1.0,
    clone: 9.0,
    final: 12.5,
  };

  const kickerS = riseIn(frame, fps, Math.round(t.kicker * fps));
  const finalS = springIn(frame, fps, Math.round(t.final * fps), { damping: 200, stiffness: 55 });
  const showFinal = frame >= t.final * fps;

  // global fade to black at the very end
  const fadeOut = interpolate(frame, [durationInFrames - 30, durationInFrames - 1], [1, 0], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });

  return (
    <Scene fadeOutAfter={0}>
      <AbsoluteFill
        style={{
          backgroundImage:
            "linear-gradient(to right, rgba(255,255,255,0.02) 1px, transparent 1px), linear-gradient(to bottom, rgba(255,255,255,0.02) 1px, transparent 1px)",
          backgroundSize: "56px 56px",
          opacity: showFinal
            ? interpolate(frame, [t.final * fps, t.final * fps + 10], [1, 0], { extrapolateLeft: "clamp", extrapolateRight: "clamp" })
            : 1,
        }}
      >
        {/* kicker */}
        <div
          style={{
            position: "absolute",
            top: 120,
            left: 0,
            right: 0,
            textAlign: "center",
            ...kickerS,
          }}
        >
          <div
            style={{
              fontFamily: FONTS.mono,
              fontSize: 22,
              letterSpacing: "0.35em",
              textTransform: "uppercase",
              color: C.faint,
              marginBottom: 16,
            }}
          >
            Open source · MIT licensed
          </div>
          <div style={{ fontSize: 56, fontWeight: 700, letterSpacing: "-0.02em" }}>
            Try it on your own MCP servers
          </div>
        </div>

        {/* install commands */}
        <div
          style={{
            position: "absolute",
            top: 330,
            left: "50%",
            transform: "translateX(-50%)",
            width: 1100,
            background: C.panel,
            border: `1px solid ${C.border}`,
            borderRadius: 16,
            padding: "34px 44px",
            opacity: interpolate(frame, [t.cmds * fps, t.cmds * fps + 10], [0, 1], {
              extrapolateLeft: "clamp",
              extrapolateRight: "clamp",
            }),
          }}
        >
          {INSTALL_COMMANDS.map((cmd, i) => {
            const startF = t.cmds * fps + i * 42;
            const o = interpolate(frame, [startF, startF + 10], [0, 1], {
              extrapolateLeft: "clamp",
              extrapolateRight: "clamp",
            });
            return (
              <div
                key={cmd}
                style={{
                  fontFamily: FONTS.mono,
                  fontSize: 30,
                  lineHeight: 1.85,
                  whiteSpace: "pre",
                  opacity: o,
                }}
              >
                <span style={{ color: C.greenSoft }}>$ </span>
                <span style={{ color: cmd.startsWith("warden") ? C.cyan : C.text }}>{cmd}</span>
              </div>
            );
          })}
          <div
            style={{
              fontFamily: FONTS.mono,
              fontSize: 20,
              color: C.faint,
              marginTop: 18,
              opacity: interpolate(frame, [t.clone * fps - 20, t.clone * fps], [0, 1], {
                extrapolateLeft: "clamp",
                extrapolateRight: "clamp",
              }),
            }}
          >
            docs: {DOCS_URL}
          </div>
        </div>

        {/* clone command */}
        <div
          style={{
            position: "absolute",
            top: 720,
            left: 0,
            right: 0,
            textAlign: "center",
            fontFamily: FONTS.mono,
            fontSize: 34,
            opacity: interpolate(frame, [t.clone * fps, t.clone * fps + 12], [0, 1], {
              extrapolateLeft: "clamp",
              extrapolateRight: "clamp",
            }),
          }}
        >
          <span style={{ color: C.greenSoft }}>$ </span>
          git clone {REPO_URL}
        </div>
      </AbsoluteFill>

      {/* final lockup */}
      {showFinal && (
        <AbsoluteFill
          style={{
            alignItems: "center",
            justifyContent: "center",
            opacity: finalS * fadeOut,
          }}
        >
          <div style={{ textAlign: "center", transform: `translateY(${(1 - finalS) * 20}px)` }}>
            <div style={{ display: "flex", justifyContent: "center" }}>
              <Logo size={130} fontSize={110} glow />
            </div>
            <div
              style={{
                marginTop: 44,
                fontSize: 62,
                fontWeight: 700,
                letterSpacing: "-0.01em",
                lineHeight: 1.3,
              }}
            >
              {MANTRA.split(".")[0]}.
              <br />
              <span style={{ color: C.greenSoft }}>{MANTRA.split(".").slice(1).join(".").trim()}.</span>
            </div>
            <div
              style={{
                marginTop: 56,
                fontFamily: FONTS.mono,
                fontSize: 36,
                color: C.muted,
              }}
            >
              {REPO_SHORT}
            </div>
          </div>
        </AbsoluteFill>
      )}
    </Scene>
  );
};
