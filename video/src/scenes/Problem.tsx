import React from "react";
import {
  AbsoluteFill,
  interpolate,
  useCurrentFrame,
  useVideoConfig,
} from "remotion";
import { C, FONTS, Scene, riseIn, springIn } from "../components/ui";

/*
 * Scene 02 — The MCP Problem (0:30–1:15)
 *
 * Animated system diagram:
 *   AI → MCP → Your Computer
 * The computer expands into the resources a host exposes, then the MCP
 * capabilities tree appears, and the scene closes on the visual thesis:
 *   Useful access ≠ Unrestricted access
 */
export const Problem: React.FC = () => {
  const frame = useCurrentFrame();
  const { fps, durationInFrames } = useVideoConfig();

  /* ── timing (scene-local seconds) ──
   * 0.0   title in
   * 1.2   AI node
   * 2.4   MCP node
   * 3.8   Your Computer node
   * 5.0   connection lines animate
   * 8.0   computer expands into resource chips
   * 14.0  MCP capability list (filesystem/network/environment/tools)
   * 21.5  "Useful access ≠ Unrestricted access"
   */
  const t = {
    title: 0.2,
    ai: 1.2,
    mcp: 2.6,
    computer: 4.2,
    lines: 5.4,
    expand: 8.0,
    caps: 14.0,
    neq: 21.5,
  };

  const title = riseIn(frame, fps, Math.round(t.title * fps));

  const nodeIn = (delayS: number) => springIn(frame, fps, Math.round(delayS * fps), { damping: 200, stiffness: 70 });

  // line draw progress
  const lineP = interpolate(frame, [t.lines * fps, t.lines * fps + 20], [0, 1], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });

  const expandS = springIn(frame, fps, Math.round(t.expand * fps), { damping: 200, stiffness: 60 });
  const capsS = springIn(frame, fps, Math.round(t.caps * fps), { damping: 200, stiffness: 70 });
  const neqS = springIn(frame, fps, Math.round(t.neq * fps), { damping: 200, stiffness: 60 });

  /* resource chips (left of the expanded computer) */
  const RESOURCES = [
    "Project files",
    "Environment variables",
    "SSH keys",
    "Credentials",
    "Private files",
    "Network",
  ];

  /* MCP capabilities (right column) */
  const CAPS = ["filesystem", "network", "environment", "tools"];

  const capsAppear = frame >= t.caps * fps;

  return (
    <Scene fadeOutAfter={0.8}>
      <AbsoluteFill
        style={{
          backgroundImage:
            "linear-gradient(to right, rgba(255,255,255,0.025) 1px, transparent 1px), linear-gradient(to bottom, rgba(255,255,255,0.025) 1px, transparent 1px)",
          backgroundSize: "56px 56px",
        }}
      />

      {/* Title */}
      <div
        style={{
          position: "absolute",
          top: 90,
          left: 0,
          right: 0,
          textAlign: "center",
          ...title,
        }}
      >
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
          The problem
        </div>
        <div style={{ fontSize: 64, fontWeight: 700, letterSpacing: "-0.02em" }}>
          MCP servers run with your permissions
        </div>
      </div>

      {/* diagram area */}
      <div
        style={{
          position: "absolute",
          top: 240,
          left: 0,
          right: 0,
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          gap: 0,
        }}
      >
          {/* AI node */}
          <Node
            label="AI"
            sub="client / agent"
            progress={nodeIn(t.ai)}
          />
          {/* connector */}
          <Connector progress={lineP} delayF={Math.round(t.lines * fps)} />
          {/* MCP node */}
          <Node
            label="MCP"
            sub="model context protocol"
            progress={nodeIn(t.mcp)}
          />
          <Connector progress={lineP} delayF={Math.round(t.lines * fps)} />
          {/* Your Computer node */}
          <Node
            label="Your Computer"
            sub="full user permissions"
            progress={nodeIn(t.computer)}
            accent
          />
      </div>

        {/* expanded resources (under computer node) */}
        {frame >= t.expand * fps && (
          <div
            style={{
              position: "absolute",
              top: 430,
              left: 0,
              right: 0,
              display: "flex",
              justifyContent: "center",
              gap: 18,
              flexWrap: "nowrap",
              opacity: expandS,
              transform: `translateY(${(1 - expandS) * 24}px)`,
            }}
        >
            {RESOURCES.map((r, i) => (
              <div
                key={r}
                style={{
                  fontFamily: FONTS.mono,
                  fontSize: 24,
                  color: C.muted,
                  border: `1px solid ${C.border}`,
                  background: C.panel,
                  borderRadius: 10,
                  padding: "12px 22px",
                  whiteSpace: "nowrap",
                  opacity: interpolate(frame, [t.expand * fps + i * 6, t.expand * fps + i * 6 + 10], [0, 1], {
                    extrapolateLeft: "clamp",
                    extrapolateRight: "clamp",
                  }),
                  transform: `translateY(${interpolate(
                    frame,
                    [t.expand * fps + i * 6, t.expand * fps + i * 6 + 10],
                    [14, 0],
                    { extrapolateLeft: "clamp", extrapolateRight: "clamp" }
                  )}px)`,
                }}
              >
                {r}
  </div>
            ))}
          </div>
        )}

        {/* MCP capability tree */}
        {capsAppear && (
          <div
            style={{
              position: "absolute",
              top: 590,
              left: 0,
              right: 0,
              display: "flex",
              justifyContent: "center",
              opacity: capsS,
              transform: `translateY(${(1 - capsS) * 18}px)`,
            }}
          >
            <div
              style={{
                border: `1px solid ${C.border}`,
                background: C.panel,
                borderRadius: 14,
                padding: "26px 40px",
                boxShadow: "0 20px 50px rgba(0,0,0,0.35)",
              }}
            >
            <div
              style={{ fontFamily: FONTS.mono, fontSize: 26, color: C.text, marginBottom: 12 }}
            >
              MCP
            </div>
            {CAPS.map((c, i) => (
              <div
                key={c}
                style={{
                  fontFamily: FONTS.mono,
                  fontSize: 24,
                  color: C.muted,
                  lineHeight: 1.7,
                  opacity: interpolate(
                    frame,
                    [t.caps * fps + i * 8, t.caps * fps + i * 8 + 10],
                    [0, 1],
                    { extrapolateLeft: "clamp", extrapolateRight: "clamp" }
                  ),
                }}
              >
                {(i === CAPS.length - 1 ? "  └─ " : "  ├─ ") + c}
              </div>
            ))}
            </div>
          </div>
        )}

        {/* Useful ≠ Unrestricted */}
        {frame >= t.neq * fps && (
          <AbsoluteFill
            style={{
              background: "rgba(9,9,11,0.96)",
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
            }}
          >
            <div
              style={{
                textAlign: "center",
                opacity: neqS,
                transform: `translateY(${(1 - neqS) * 18}px)`,
              }}
            >
              <div
                style={{
                  fontFamily: FONTS.mono,
                  fontSize: 30,
                  color: C.muted,
                  marginBottom: 26,
                  letterSpacing: "0.12em",
                }}
              >
                MCP
              </div>
              <div style={{ display: "flex", alignItems: "center", gap: 40 }}>
                <div style={{ fontSize: 72, fontWeight: 700, color: C.text }}>
                  Useful access
                </div>
                <div
                  style={{
                    fontFamily: FONTS.mono,
                    fontSize: 96,
                    color: C.amber,
                    fontWeight: 700,
                  }}
                >
                  ≠
                </div>
                <div style={{ fontSize: 72, fontWeight: 700, color: C.redSoft }}>
                  Unrestricted access
                </div>
              </div>
            </div>
          </AbsoluteFill>
        )}
    </Scene>
  );
};

/* ────────────────────────────  diagram parts  ──────────────────────────── */

const Node: React.FC<{
  label: string;
  sub?: string;
  progress: number;
  accent?: boolean;
}> = ({ label, sub, progress, accent }) => (
  <div
    style={{
      width: 340,
      padding: "30px 24px",
      borderRadius: 16,
      border: `1px solid ${accent ? "rgba(34,197,94,0.45)" : C.border}`,
      background: accent ? "rgba(22,163,74,0.08)" : C.panel,
      textAlign: "center",
      opacity: progress,
      transform: `translateY(${(1 - progress) * 20}px) scale(${0.97 + progress * 0.03})`,
      boxShadow: accent ? "0 0 60px rgba(22,163,74,0.15)" : "0 20px 50px rgba(0,0,0,0.35)",
    }}
  >
    <div style={{ fontSize: 40, fontWeight: 700, letterSpacing: "-0.01em" }}>{label}</div>
    {sub && (
      <div style={{ fontFamily: FONTS.mono, fontSize: 19, color: C.muted, marginTop: 8 }}>
        {sub}
      </div>
    )}
  </div>
);

const Connector: React.FC<{ progress: number; delayF?: number }> = ({ progress, delayF }) => (
  <div style={{ width: 120, height: 2, position: "relative", margin: "0 6px" }}>
    <div
      style={{
        position: "absolute",
        inset: 0,
        background: C.border,
      }}
    />
    <div
      style={{
        position: "absolute",
        top: 0,
        bottom: 0,
        left: 0,
        width: `${progress * 100}%`,
        background: C.greenSoft,
        boxShadow: "0 0 12px rgba(34,197,94,0.6)",
      }}
    />
    {/* arrowhead */}
    <div
      style={{
        position: "absolute",
        right: -2,
        top: -5,
        width: 0,
        height: 0,
        borderTop: "6px solid transparent",
        borderBottom: "6px solid transparent",
        borderLeft: `10px solid ${C.border}`,
        opacity: progress > 0.95 ? 1 : 0,
      }}
    />
  </div>
);
