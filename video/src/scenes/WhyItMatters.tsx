import React from "react";
import { AbsoluteFill, interpolate, useCurrentFrame, useVideoConfig } from "remotion";
import { C, FONTS, Scene, riseIn, springIn } from "../components/ui";

/*
 * Scene 09 — Why it matters (4:50–5:05)
 *
 * Side-by-side comparison, then the mantra holds:
 *   "Give tools the access they need. Not the access they don't."
 */
export const WhyItMatters: React.FC = () => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();

  const t = {
    title: 0.1,
    cards: 0.8,
    mantra: 9.0,
  };

  const titleS = riseIn(frame, fps, Math.round(t.title * fps));
  const leftS = springIn(frame, fps, Math.round(t.cards * fps), { damping: 200, stiffness: 70 });
  const rightS = springIn(frame, fps, Math.round(t.cards * fps + 6), { damping: 200, stiffness: 70 });
  const mantraS = springIn(frame, fps, Math.round(t.mantra * fps), { damping: 200, stiffness: 55 });
  const showMantra = frame >= t.mantra * fps;

  return (
    <Scene fadeOutAfter={0.7}>
      {/* comparison */}
      <AbsoluteFill
        style={{
          alignItems: "center",
          justifyContent: "center",
          opacity: showMantra ? interpolate(frame, [t.mantra * fps, t.mantra * fps + 12], [1, 0], { extrapolateLeft: "clamp", extrapolateRight: "clamp" }) : 1,
        }}
      >
        <div style={{ position: "absolute", top: 120, left: 0, right: 0, textAlign: "center", ...titleS }}>
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
            Why it matters
          </div>
          <div style={{ fontSize: 58, fontWeight: 700, letterSpacing: "-0.02em" }}>
            Same MCP server. Two very different runtimes.
          </div>
        </div>

        <div style={{ display: "flex", gap: 60, marginTop: 120 }}>
          {/* without */}
          <div
            style={{
              width: 620,
              border: `1px solid ${C.border}`,
              background: C.panel,
              borderRadius: 18,
              padding: "36px 44px",
              opacity: leftS,
              transform: `translateY(${(1 - leftS) * 24}px)`,
            }}
          >
            <div style={{ fontFamily: FONTS.mono, fontSize: 22, letterSpacing: "0.2em", color: C.redSoft, marginBottom: 26 }}>
              WITHOUT A BOUNDARY
            </div>
            {["MCP", "Host", "Broad permissions"].map((s, i) => (
              <Step key={s} label={s} delay={i * 6} tone={i === 2 ? "red" : "plain"} frame={frame} fps={fps} />
            ))}
          </div>

          {/* with warden */}
          <div
            style={{
              width: 620,
              border: "1px solid rgba(34,197,94,0.35)",
              background: "rgba(22,163,74,0.05)",
              borderRadius: 18,
              padding: "36px 44px",
              opacity: rightS,
              transform: `translateY(${(1 - rightS) * 24}px)`,
              boxShadow: "0 0 80px rgba(22,163,74,0.08)",
            }}
          >
            <div style={{ fontFamily: FONTS.mono, fontSize: 22, letterSpacing: "0.2em", color: C.greenSoft, marginBottom: 26 }}>
              WITH WARDEN
            </div>
            {["MCP", "Warden", "Policy", "Sandbox", "Permitted resources"].map((s, i) => (
              <Step key={s} label={s} delay={8 + i * 6} tone={i === 4 ? "green" : i === 0 ? "plain" : "accent"} frame={frame} fps={fps} />
            ))}
          </div>
        </div>
      </AbsoluteFill>

      {/* mantra overlay */}
      {showMantra && (
        <AbsoluteFill
          style={{
            alignItems: "center",
            justifyContent: "center",
            background: C.bg,
            opacity: mantraS,
          }}
        >
          <div style={{ textAlign: "center" }}>
            <div style={{ fontSize: 84, fontWeight: 800, letterSpacing: "-0.02em", lineHeight: 1.25 }}>
              Give tools the access they need.
            </div>
            <div style={{ fontSize: 84, fontWeight: 800, letterSpacing: "-0.02em", lineHeight: 1.25, color: C.greenSoft }}>
              Not the access they don't.
            </div>
          </div>
        </AbsoluteFill>
      )}
    </Scene>
  );
};

const Step: React.FC<{ label: string; delay: number; tone: "plain" | "red" | "green" | "accent"; frame: number; fps: number }> = ({
  label,
  delay,
  tone,
  frame,
  fps,
}) => {
  const o = interpolate(frame, [delay, delay + 10], [0, 1], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });
  const color =
    tone === "red" ? C.redSoft : tone === "green" ? C.greenSoft : tone === "accent" ? C.text : C.muted;
  return (
    <div style={{ marginBottom: 18, opacity: o }}>
      <div
        style={{
          border: `1px solid ${C.border}`,
          background: C.panel2,
          borderRadius: 10,
          padding: "14px 22px",
          fontFamily: FONTS.mono,
          fontSize: 25,
          color,
          textAlign: "center",
        }}
      >
        {label}
      </div>
    </div>
  );
};
