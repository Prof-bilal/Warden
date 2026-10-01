import React from "react";
import { AbsoluteFill, interpolate, useCurrentFrame, useVideoConfig } from "remotion";
import { C, FONTS, Scene, riseIn, springIn } from "../components/ui";
import { Logo, LogoMark } from "../components/Logo";
import { TAGLINE, INTRO_FLOW } from "../data/product";

/*
 * Scene 04 — Introduce Warden (2:00–2:40)
 *
 * Brand reveal, then the core diagram:
 *
 *   MCP SERVER
 *        │
 *        ▼
 *   ┌─────────────┐
 *   │   WARDEN    │
 *   │ POLICY +    │
 *   │ SANDBOX     │
 *   └──────┬──────┘
 *          ▼
 *    HOST SYSTEM
 *
 * Sensitive resources stay visually outside the boundary. The MCP "enters"
 * the boundary as a moving node — this is the conceptual heart of the video.
 */
export const WardenIntro: React.FC = () => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();

  /* ── timing (scene-local seconds) ──
   * 0.0–5.0   logo reveal + tagline
   * 5.0       diagram starts
   * 5.5       host system node
   * 7.0       boundary box draws
   * 9.0       POLICY + SANDBOX labels
   * 11.0      MCP server node appears, travels into the boundary
   * 16.0      sensitive resources appear outside (right side)
   * 22.0      sub-line: controlled environment + explicit permissions
   * 30.0      hold
   */
  const t = {
    logo: 0.3,
    tagline: 2.8,
    diagram: 4.6,
    host: 5.1,
    boundary: 6.6,
    labels: 8.6,
    travel: 10.6,
    outside: 15.6,
    subline: 21.6,
  };

  const logoS = springIn(frame, fps, Math.round(t.logo * fps), { damping: 200, stiffness: 70 });
  const taglineS = riseIn(frame, fps, Math.round(t.tagline * fps));
  const hostS = springIn(frame, fps, Math.round(t.host * fps), { damping: 200, stiffness: 70 });
  const labelsS = springIn(frame, fps, Math.round(t.labels * fps), { damping: 200, stiffness: 90 });
  const sublineS = riseIn(frame, fps, Math.round(t.subline * fps));

  // boundary draw-on
  const bDraw = interpolate(frame, [t.boundary * fps, t.boundary * fps + 24], [0, 1], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });

  // MCP node travel: from above the boundary to inside it
  const travel = interpolate(frame, [t.travel * fps, t.travel * fps + 30], [0, 1], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });

  const outsideS = springIn(frame, fps, Math.round(t.outside * fps), { damping: 200, stiffness: 70 });

  return (
    <Scene fadeOutAfter={0.8}>
      <AbsoluteFill
        style={{
          backgroundImage:
            "linear-gradient(to right, rgba(255,255,255,0.02) 1px, transparent 1px), linear-gradient(to bottom, rgba(255,255,255,0.02) 1px, transparent 1px)",
          backgroundSize: "56px 56px",
        }}
      />

      {/* Brand reveal */}
      <div
        style={{
          position: "absolute",
          top: 130,
          left: 0,
          right: 0,
          display: "flex",
          flexDirection: "column",
          alignItems: "center",
          opacity: interpolate(frame, [0, 20, 120, 138], [0, 1, 1, 0], {
            extrapolateLeft: "clamp",
            extrapolateRight: "clamp",
          }),
          transform: `scale(${interpolate(frame, [0, 138], [1, 1.05], { extrapolateLeft: "clamp", extrapolateRight: "clamp" })}) translateY(${(1 - logoS) * 16}px)`,
        }}
      >
        <Logo size={110} fontSize={96} glow />
        <div
          style={{
            marginTop: 30,
            fontSize: 38,
            color: C.muted,
            ...taglineS,
          }}
        >
          {TAGLINE}
        </div>
      </div>

      {/* Diagram */}
      {frame >= t.diagram * fps && (
        <AbsoluteFill style={{ justifyContent: "center" }}>
          {/* Host system (bottom) */}
          <div
            style={{
              position: "absolute",
              bottom: 120,
              left: "50%",
              transform: `translateX(-50%) translateY(${(1 - hostS) * 24}px)`,
              opacity: hostS,
            }}
          >
            <BigNode label="HOST SYSTEM" sub="the only machine you actually care about" />
          </div>

          {/* Boundary box */}
          <div
            style={{
              position: "absolute",
              left: "50%",
              top: 300,
              transform: `translateX(-50%) scaleY(${bDraw}) scaleX(${bDraw})`,
              opacity: bDraw,
              width: 720,
              height: 420,
              border: `2px solid ${C.greenSoft}`,
              borderRadius: 18,
              background: "rgba(22,163,74,0.05)",
              boxShadow: "0 0 90px rgba(22,163,74,0.12)",
            }}
          >
            {/* corner ticks */}
            {[
              { top: -6, left: -6 },
              { top: -6, right: -6 },
              { bottom: -6, left: -6 },
              { bottom: -6, right: -6 },
            ].map((pos, i) => (
              <div
                key={i}
                style={{
                  position: "absolute",
                  width: 12,
                  height: 12,
                  background: C.greenSoft,
                  borderRadius: 2,
                  opacity: interpolate(bDraw, [0.9, 1], [0, 1], { extrapolateLeft: "clamp", extrapolateRight: "clamp" }),
                  ...pos,
                }}
              />
            ))}

            {/* POLICY + SANDBOX labels */}
            <div
              style={{
                position: "absolute",
                top: 185,
                left: 0,
                right: 0,
                textAlign: "center",
                opacity: labelsS,
                transform: `translateY(${(1 - labelsS) * 14}px)`,
              }}
            >
              <div style={{ fontFamily: FONTS.mono, fontSize: 24, letterSpacing: "0.3em", color: C.muted }}>
                {INTRO_FLOW.boundaryItems[0]}
              </div>
              <div style={{ fontFamily: FONTS.mono, fontSize: 30, color: C.faint, margin: "6px 0" }}>+</div>
              <div style={{ fontFamily: FONTS.mono, fontSize: 24, letterSpacing: "0.3em", color: C.muted }}>
                {INTRO_FLOW.boundaryItems[2]}
              </div>
            </div>
          </div>

          {/* MCP server node above, travelling in */}
          <div
            style={{
              position: "absolute",
              left: "50%",
              top: 300 - 150 + travel * 195,
              transform: `translateX(-50%)`,
              opacity: interpolate(frame, [t.travel * fps, t.travel * fps + 8], [0, 1], {
                extrapolateLeft: "clamp",
                extrapolateRight: "clamp",
              }),
            }}
          >
            <NodeSmall label={INTRO_FLOW.server} />
            {/* arrow between server and boundary */}
            <div
              style={{
                position: "absolute",
                left: "50%",
                top: 78,
                transform: "translateX(-50%)",
                fontSize: 30,
                color: C.faint,
              }}
            >
              ↓
            </div>
          </div>

          {/* Sensitive resources outside the boundary (right side) */}
          {frame >= t.outside * fps && (
            <div
              style={{
                position: "absolute",
                right: 150,
                top: 330,
                display: "flex",
                flexDirection: "column",
                gap: 16,
                opacity: outsideS,
                transform: `translateX(${(1 - outsideS) * 30}px)`,
              }}
            >
              <div style={{ fontFamily: FONTS.mono, fontSize: 20, letterSpacing: "0.25em", color: C.faint, marginBottom: 6, textAlign: "right" }}>
                OUTSIDE THE BOUNDARY
                </div>
              {INTRO_FLOW.outside.map((o, i) => (
                <div
                  key={o}
                  style={{
                    alignSelf: "flex-end",
                    fontFamily: FONTS.mono,
                    fontSize: 24,
                    color: C.amber,
                    border: `1px solid rgba(245,158,11,0.3)`,
                    background: "rgba(245,158,11,0.06)",
                    borderRadius: 10,
                    padding: "10px 18px",
                    opacity: interpolate(frame, [t.outside * fps + i * 7, t.outside * fps + i * 7 + 10], [0, 1], {
                      extrapolateLeft: "clamp",
                      extrapolateRight: "clamp",
                    }),
                  }}
                >
                  {o}
                </div>
              ))}
            </div>
          )}

          {/* Sub-line */}
          <div
            style={{
              position: "absolute",
              bottom: 60,
              left: 0,
              right: 0,
              textAlign: "center",
              fontSize: 30,
              color: C.muted,
              ...sublineS,
            }}
          >
            A controlled environment and explicit permissions — nothing more.
          </div>
        </AbsoluteFill>
      )}
    </Scene>
  );
};

/* ────────────────────────────  small nodes  ──────────────────────────── */

const BigNode: React.FC<{ label: string; sub?: string }> = ({ label, sub }) => (
  <div
    style={{
      border: `1px solid ${C.border}`,
      background: C.panel,
      borderRadius: 14,
      padding: "24px 60px",
      textAlign: "center",
    }}
  >
    <div style={{ fontSize: 34, fontWeight: 700 }}>{label}</div>
    {sub && <div style={{ fontFamily: FONTS.mono, fontSize: 18, color: C.faint, marginTop: 6 }}>{sub}</div>}
  </div>
);

const NodeSmall: React.FC<{ label: string }> = ({ label }) => (
  <div
    style={{
      border: `1px solid ${C.border}`,
      background: C.panel2,
      borderRadius: 12,
      padding: "18px 40px",
      fontFamily: FONTS.mono,
      fontSize: 28,
      fontWeight: 600,
      whiteSpace: "nowrap",
    }}
  >
    {label}
  </div>
);
