import React from "react";
import { AbsoluteFill, interpolate, useCurrentFrame, useVideoConfig } from "remotion";
import { C, FONTS, Scene, riseIn, springIn, StatusBadge } from "../components/ui";

/*
 * Scene 03 — Access Scenario (1:15–2:00)
 *
 * Two tool calls from the same MCP server:
 *   filesystem.read  project/src/app.ts   → necessary, allowed
 *   filesystem.read  ~/.ssh/id_ed25519    → sensitive, highlighted
 *
 * The point: what matters is where the security boundary sits and who
 * controls it. Warden is not introduced yet — this scene stays generic.
 */
export const AccessScenario: React.FC = () => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();

  /* ── timing (scene-local seconds) ──
   * 0.2  title
   * 1.2  request 1 card
   * 3.8  ALLOWED badge 1
   * 7.5  request 2 card
   * 10.2 sensitive highlight
   * 15.0 mapping visual (useful / sensitive)
   * 22.0 explainer line
   * 27.5 the question (major moment)
   */
  const t = {
    title: 0.2,
    req1: 1.2,
    badge1: 3.8,
    req2: 7.5,
    highlight: 10.2,
    mapping: 15.0,
    explain: 22.0,
    question: 27.5,
  };

  const title = riseIn(frame, fps, Math.round(t.title * fps));
  const qS = springIn(frame, fps, Math.round(t.question * fps), { damping: 200, stiffness: 60 });

  const explainOpacity = interpolate(frame, [t.explain * fps, t.explain * fps + 15], [0, 1], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });

  return (
    <Scene fadeOutAfter={0.8}>
      <AbsoluteFill
        style={{
          backgroundImage:
            "radial-gradient(rgba(255,255,255,0.04) 1px, transparent 1px)",
          backgroundSize: "36px 36px",
        }}
      />

      {/* Title */}
      <div style={{ position: "absolute", top: 90, left: 0, right: 0, textAlign: "center", ...title }}>
        <div
          style={{
            fontFamily: FONTS.mono,
            fontSize: 22,
            letterSpacing: "0.35em",
            textTransform: "uppercase",
            color: C.faint,
            marginBottom: 14,
          }}
        >
          Access scenario
        </div>
        <div style={{ fontSize: 62, fontWeight: 700, letterSpacing: "-0.02em" }}>
          Same server. Two very different requests.
        </div>
      </div>

      {/* Request cards */}
      <div
        style={{
          position: "absolute",
          top: 260,
          left: 0,
          right: 0,
          display: "flex",
          justifyContent: "center",
          gap: 44,
        }}
      >
        <RequestCard
          op="filesystem.read"
          target="project/src/app.ts"
          kind="out"
          appearAt={t.req1}
          badgeAt={t.badge1}
          frame={frame}
          fps={fps}
        />
        <RequestCard
          op="filesystem.read"
          target="~/.ssh/id_ed25519"
          kind="sensitive"
          appearAt={t.req2}
          badgeAt={t.highlight}
          frame={frame}
          fps={fps}
        />
      </div>

      {/* Mapping visual */}
      {frame >= t.mapping * fps && (
        <div
          style={{
            position: "absolute",
            top: 640,
            left: 0,
            right: 0,
            display: "flex",
            justifyContent: "center",
            gap: 180,
          }}
        >
          {[
            { top: "PROJECT FILE", bottom: "useful", color: C.greenSoft, d: 0 },
            { top: "SSH PRIVATE KEY", bottom: "sensitive", color: C.redSoft, d: 8 },
          ].map((m) => {
            const o = interpolate(frame, [t.mapping * fps + m.d * 2, t.mapping * fps + m.d * 2 + 10], [0, 1], {
              extrapolateLeft: "clamp",
              extrapolateRight: "clamp",
            });
            return (
              <div key={m.top} style={{ textAlign: "center", opacity: o }}>
                <div style={{ fontFamily: FONTS.mono, fontSize: 28, fontWeight: 600, color: C.text }}>
                  {m.top}
                </div>
                <div style={{ fontSize: 34, color: C.faint, lineHeight: 1.2 }}>↓</div>
                <div style={{ fontSize: 40, fontWeight: 700, color: m.color }}>{m.bottom}</div>
              </div>
            );
          })}
        </div>
      )}

      {/* Explainer */}
      <div
        style={{
          position: "absolute",
          top: 880,
          left: 0,
          right: 0,
          textAlign: "center",
          fontSize: 30,
          color: C.muted,
          opacity: explainOpacity,
        }}
      >
        Reading a project file may be necessary. Reading a private key is a different story —
        the boundary decides.
      </div>

      {/* The question — major moment */}
      {frame >= t.question * fps && (
        <AbsoluteFill
          style={{
            background: "rgba(9,9,11,0.97)",
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
          }}
        >
          <div
            style={{
              textAlign: "center",
              opacity: qS,
              transform: `translateY(${(1 - qS) * 18}px)`,
            }}
          >
            <div
              style={{
                fontFamily: FONTS.mono,
                fontSize: 26,
                letterSpacing: "0.3em",
                textTransform: "uppercase",
                color: C.faint,
                marginBottom: 28,
              }}
            >
              The question is
            </div>
            <div style={{ fontSize: 92, fontWeight: 800, letterSpacing: "-0.02em" }}>
              Who controls that boundary?
            </div>
          </div>
        </AbsoluteFill>
      )}
    </Scene>
  );
};

/* ────────────────────────────  request card  ──────────────────────────── */

const RequestCard: React.FC<{
  op: string;
  target: string;
  kind: "out" | "sensitive";
  appearAt: number;
  badgeAt: number;
  frame: number;
  fps: number;
}> = ({ op, target, kind, appearAt, badgeAt, frame, fps }) => {
  const s = springIn(frame, fps, Math.round(appearAt * fps), { damping: 200, stiffness: 80 });
  const showBadge = frame >= badgeAt * fps;
  const sensitive = kind === "sensitive";

  const hl = interpolate(
    frame,
    [badgeAt * fps, badgeAt * fps + 12],
    [0, 1],
    { extrapolateLeft: "clamp", extrapolateRight: "clamp" }
  );

  return (
    <div
      style={{
        width: 620,
        borderRadius: 16,
        border: `1px solid ${sensitive && hl > 0 ? `rgba(245,158,11,${0.25 + hl * 0.5})` : C.border}`,
        background: C.panel,
        padding: "30px 34px",
        opacity: s,
        transform: `translateY(${(1 - s) * 22}px)`,
        boxShadow:
          sensitive && hl > 0
            ? `0 0 ${hl * 70}px rgba(245,158,11,${hl * 0.12})`
            : "0 20px 50px rgba(0,0,0,0.35)",
      }}
    >
      <div style={{ fontFamily: FONTS.mono, fontSize: 20, color: C.faint, marginBottom: 14 }}>
        tool call
      </div>
      <div style={{ fontFamily: FONTS.mono, fontSize: 32, color: C.text, fontWeight: 600 }}>
        {op}
      </div>
      <div
        style={{
          fontFamily: FONTS.mono,
          fontSize: 28,
          color: sensitive ? C.amber : C.muted,
          marginTop: 10,
        }}
      >
        target: {target}
      </div>
      <div style={{ marginTop: 26 }}>
        {showBadge ? (
          sensitive ? (
            <span
              style={{
                display: "inline-flex",
                alignItems: "center",
                gap: 12,
                fontFamily: FONTS.mono,
                fontSize: 20,
                fontWeight: 600,
                letterSpacing: "0.08em",
                color: C.amber,
                background: "rgba(245,158,11,0.12)",
                border: "1px solid rgba(245,158,11,0.35)",
                borderRadius: 999,
                padding: "8px 20px",
              }}
            >
              ⚠ SENSITIVE
            </span>
          ) : (
            <StatusBadge status="allowed" />
          )
        ) : (
          <div style={{ height: 44 }} />
        )}
      </div>
    </div>
  );
};
