import React from "react";
import { AbsoluteFill, interpolate, useCurrentFrame, useVideoConfig } from "remotion";
import { C, FONTS, Scene, riseIn, springIn } from "../components/ui";

/*
 * Scene 07 — Architecture (4:00–4:30)
 *
 * The data-flow story, told with moving pulses:
 *   MCP SERVER → WARDEN (POLICY + SANDBOX) → ALLOWED → permitted host resources
 *                                          → DENIED  → blocked request
 *
 * Green pulse travels through the boundary when allowed; a request stops
 * dead at the boundary with a red pulse when denied.
 */
export const Architecture: React.FC = () => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();

  /* ── timing (scene-local seconds) ──
   * 0.2   title
   * 1.0   nodes appear
   * 3.5   pulse 1: allowed (green, travels through)
   * 9.5   pulse 2: denied (red, stops at boundary)
   * 15.5  pulse 3: allowed again
   * 21.0  internal component labels
   * 25.5  closing line
   */
  const t = {
    title: 0.2,
    nodes: 1.0,
    p1: 3.5,
    p2: 9.5,
    p3: 15.5,
    internals: 21.0,
    closing: 25.5,
  };

  const titleS = riseIn(frame, fps, Math.round(t.title * fps));
  const nodesS = springIn(frame, fps, Math.round(t.nodes * fps), { damping: 200, stiffness: 70 });

  const internalsS = riseIn(frame, fps, Math.round(t.internals * fps));
  const closingS = riseIn(frame, fps, Math.round(t.closing * fps));

  return (
    <Scene fadeOutAfter={0.8}>
      <AbsoluteFill
        style={{
          backgroundImage:
            "linear-gradient(to right, rgba(255,255,255,0.02) 1px, transparent 1px), linear-gradient(to bottom, rgba(255,255,255,0.02) 1px, transparent 1px)",
          backgroundSize: "56px 56px",
        }}
      />

      {/* Title */}
      <div style={{ position: "absolute", top: 84, left: 0, right: 0, textAlign: "center", ...titleS }}>
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
          Architecture
        </div>
        <div style={{ fontSize: 60, fontWeight: 700, letterSpacing: "-0.02em" }}>
          Every request crosses the boundary — or doesn't.
        </div>
      </div>

      {/* Diagram canvas */}
      <AbsoluteFill style={{ alignItems: "center", justifyContent: "center" }}>
        <div style={{ position: "relative", width: 1560, height: 560, opacity: nodesS }}>
          {/* MCP server node (left) */}
          <DiagramNode
            x={40}
            y={230}
            w={300}
            label="MCP SERVER"
            sub="asks for resources"
          />

          {/* Warden boundary (center) */}
          <div
            style={{
              position: "absolute",
              left: 520,
              top: 110,
              width: 420,
              height: 340,
              border: `2px solid ${C.greenSoft}`,
              borderRadius: 18,
              background: "rgba(22,163,74,0.05)",
              boxShadow: "0 0 90px rgba(22,163,74,0.10)",
              display: "flex",
              flexDirection: "column",
              alignItems: "center",
              justifyContent: "center",
              gap: 18,
            }}
          >
            <div style={{ fontFamily: FONTS.mono, fontSize: 26, letterSpacing: "0.28em", color: C.greenSoft, fontWeight: 700 }}>
              WARDEN
            </div>
            <BoundaryPart label="POLICY ENGINE" />
            <BoundaryPart label="SANDBOX BACKEND" />
            <BoundaryPart label="EGRESS PROXY" />
            <BoundaryPart label="AUDIT LOGGER" />
          </div>

          {/* allowed output (top right) */}
          <div style={{ position: "absolute", left: 1180, top: 140 }}>
            <div
              style={{
                border: "1px solid rgba(34,197,94,0.4)",
                background: C.greenDim,
                borderRadius: 14,
                padding: "24px 34px",
                textAlign: "center",
              }}
            >
              <div style={{ fontFamily: FONTS.mono, fontSize: 26, fontWeight: 700, color: C.greenSoft }}>
                ALLOWED
              </div>
              <div style={{ fontFamily: FONTS.mono, fontSize: 19, color: C.muted, marginTop: 8 }}>
                permitted host resources
              </div>
            </div>
          </div>

          {/* denied output (bottom right) */}
          <div style={{ position: "absolute", left: 1180, top: 330 }}>
            <div
              style={{
                border: "1px solid rgba(239,68,68,0.4)",
                background: C.redDim,
                borderRadius: 14,
                padding: "24px 34px",
                textAlign: "center",
              }}
            >
              <div style={{ fontFamily: FONTS.mono, fontSize: 26, fontWeight: 700, color: C.redSoft }}>
                DENIED
              </div>
              <div style={{ fontFamily: FONTS.mono, fontSize: 19, color: C.muted, marginTop: 8 }}>
                blocked request
              </div>
            </div>
          </div>

          {/* connectors */}
          <Wire x1={340} y1={260} x2={520} y2={260} color={C.border} />
          <Wire x1={940} y1={200} x2={1180} y2={190} color="rgba(34,197,94,0.35)" />
          <Wire x1={940} y1={320} x2={1180} y2={370} color="rgba(239,68,68,0.35)" />

          {/* pulses */}
          <Pulse
            frame={frame}
            fps={fps}
            startF={t.p1 * fps}
            path={[
              { x: 350, y: 260 },
              { x: 520, y: 260 },
              { x: 940, y: 200 },
              { x: 1170, y: 192 },
            ]}
            color={C.greenSoft}
          />
          <Pulse
            frame={frame}
            fps={fps}
            startF={t.p2 * fps}
            path={[
              { x: 350, y: 260 },
              { x: 520, y: 260 },
              { x: 700, y: 260 },
            ]}
            color={C.redSoft}
            stopAtEnd
          />
          <Pulse
            frame={frame}
            fps={fps}
            startF={t.p3 * fps}
            path={[
              { x: 350, y: 260 },
              { x: 520, y: 260 },
              { x: 940, y: 200 },
              { x: 1170, y: 192 },
            ]}
            color={C.greenSoft}
          />
        </div>
      </AbsoluteFill>

      {/* internals caption */}
      <div
        style={{
          position: "absolute",
          bottom: 96,
          left: 0,
          right: 0,
          textAlign: "center",
          fontFamily: FONTS.mono,
          fontSize: 22,
          color: C.muted,
          ...internalsS,
        }}
      >
        Policy engine → sandbox backend → egress proxy → audit log
      </div>

      {/* closing line */}
      <div
        style={{
          position: "absolute",
          bottom: 44,
          left: 0,
          right: 0,
          textAlign: "center",
          fontSize: 27,
          color: C.faint,
          ...closingS,
        }}
      >
        Fail-closed: if the sandbox can't be applied, Warden refuses to run.
      </div>
    </Scene>
  );
};

/* ────────────────────────────  diagram parts  ──────────────────────────── */

const DiagramNode: React.FC<{ x: number; y: number; w: number; label: string; sub?: string }> = ({
  x,
  y,
  w,
  label,
  sub,
}) => (
  <div
    style={{
      position: "absolute",
      left: x,
      top: y,
      width: w,
      border: `1px solid ${C.border}`,
      background: C.panel,
      borderRadius: 14,
      padding: "22px 26px",
      textAlign: "center",
    }}
  >
    <div style={{ fontFamily: FONTS.mono, fontSize: 25, fontWeight: 600 }}>{label}</div>
    {sub && <div style={{ fontSize: 18, color: C.faint, marginTop: 6 }}>{sub}</div>}
  </div>
);

const BoundaryPart: React.FC<{ label: string }> = ({ label }) => (
  <div
    style={{
      fontFamily: FONTS.mono,
      fontSize: 20,
      color: C.text,
      border: `1px solid ${C.border}`,
      background: C.panel,
      borderRadius: 10,
      padding: "10px 22px",
      letterSpacing: "0.08em",
    }}
  >
    {label}
  </div>
);

const Wire: React.FC<{ x1: number; y1: number; x2: number; y2: number; color: string }> = ({
  x1,
  y1,
  x2,
  y2,
  color,
}) => {
  const dx = x2 - x1;
  const dy = y2 - y1;
  const len = Math.sqrt(dx * dx + dy * dy);
  const angle = Math.atan2(dy, dx);
  return (
    <div
      style={{
        position: "absolute",
        left: x1,
        top: y1,
        width: len,
        height: 2,
        background: color,
        transform: `rotate(${angle}rad)`,
        transformOrigin: "0 50%",
      }}
    />
  );
};

/** A traveling dot that follows a polyline path. */
const Pulse: React.FC<{
  frame: number;
  fps: number;
  startF: number;
  path: { x: number; y: number }[];
  color: string;
  stopAtEnd?: boolean;
}> = ({ frame, fps, startF, path, color, stopAtEnd }) => {
  const dur = 45; // frames to traverse
  const p = interpolate(frame, [startF, startF + dur], [0, stopAtEnd ? 0.62 : 1], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });

  if (frame < startF) return null;

  // total path length
  let total = 0;
  const segs: number[] = [];
  for (let i = 0; i < path.length - 1; i++) {
    const dx = path[i + 1].x - path[i].x;
    const dy = path[i + 1].y - path[i].y;
    const l = Math.sqrt(dx * dx + dy * dy);
    segs.push(l);
    total += l;
  }

  let target = p * total;
  let x = path[0].x;
  let y = path[0].y;
  for (let i = 0; i < segs.length; i++) {
    if (target <= segs[i]) {
      const f = target / segs[i];
      x = path[i].x + (path[i + 1].x - path[i].x) * f;
      y = path[i].y + (path[i + 1].y - path[i].y) * f;
      break;
    }
    target -= segs[i];
  }

  const fadeOut = stopAtEnd
    ? interpolate(frame, [startF + dur, startF + dur + 18], [1, 0], {
        extrapolateLeft: "clamp",
        extrapolateRight: "clamp",
      })
    : interpolate(frame, [startF + dur - 6, startF + dur + 10], [1, 0], {
        extrapolateLeft: "clamp",
        extrapolateRight: "clamp",
      });

  return (
    <div
      style={{
        position: "absolute",
        left: x - 7,
        top: y - 7,
        width: 14,
        height: 14,
        borderRadius: 999,
        background: color,
        boxShadow: `0 0 18px ${color}`,
        opacity: fadeOut,
      }}
    />
  );
};
