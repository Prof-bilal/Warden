import React from "react";
import {
  AbsoluteFill,
  interpolate,
  useCurrentFrame,
  useVideoConfig,
} from "remotion";
import { Terminal } from "../components/Terminal";
import type { TermLine } from "../data/terminal";
import { C, Scene, riseIn } from "../components/ui";

/*
 * Scene 01 — Hook (0:00–0:30)
 *
 * A terminal comes up in the dark, a generic MCP server starts, the user
 * lists their home directory — and the sensitive files light up.
 * The MCP commands shown are intentionally generic (not attributed to
 * Warden); Warden itself is introduced in scene 04.
 */
export const Hook: React.FC = () => {
  const frame = useCurrentFrame();
  const { fps, durationInFrames } = useVideoConfig();

  /* ── script ── */
  const LINES: TermLine[] = [
    {
      kind: "cmd",
      segments: [
        { text: "mcp-server ", tone: "text" as const },
        { text: "start", tone: "cyan" as const },
      ],
    },
    { kind: "out", text: "✓ MCP server connected", tone: "green" },
    { kind: "out", text: "✓ Tools available", tone: "green" },
    { kind: "out", text: "✓ Ready", tone: "green" },
    { kind: "cmd", segments: [{ text: "ls", tone: "text" as const }, { text: " ~", tone: "cyan" as const }] },
    { kind: "out", text: "projects/" },
    { kind: "out", text: "documents/" },
    { kind: "out", text: ".env", tone: "amber", highlight: true },
    { kind: "out", text: ".ssh/", tone: "amber", highlight: true },
    { kind: "out", text: "credentials/", tone: "amber", highlight: true },
  ];

  /* ── timeline (scene-local seconds) ──
   * 0.8–2.2  type `mcp-server start`
   * 2.6      checkmarks cascade
   * 5.0–6.2  type `ls ~`
   * 6.4      listings cascade
   * 12.0     sensitive rows highlight, screen dim starts
   * 14.0     statement 1
   * 20.5     statement 2
   */
  const plan = [
    { line: LINES[0], startS: 0.8, endS: 2.2 },
    { line: LINES[1], startS: 2.6, endS: 2.6 },
    { line: LINES[2], startS: 2.9, endS: 2.9 },
    { line: LINES[3], startS: 3.2, endS: 3.2 },
    { line: LINES[4], startS: 5.0, endS: 6.2 },
    { line: LINES[5], startS: 6.5, endS: 6.5 },
    { line: LINES[6], startS: 6.8, endS: 6.8 },
    { line: LINES[7], startS: 7.1, endS: 7.1 },
    { line: LINES[8], startS: 7.4, endS: 7.4 },
    { line: LINES[9], startS: 7.7, endS: 7.7 },
  ];

  const tFreeze = 11.5;
  const t1 = 14.0;
  const t2 = 20.5;

  // Terminal zooms slightly toward the end
  const zoom = interpolate(frame, [tFreeze * fps, durationInFrames], [1, 1.04], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });

  // Highlight boxes only appear after freeze
  const planView = plan.map((p) =>
    p.line.kind === "out" && p.line.highlight && frame < tFreeze * fps
      ? { ...p, line: { ...p.line, highlight: false } }
      : p
  );

  const s1 = riseIn(frame, fps, Math.round(t1 * fps));
  const s2 = riseIn(frame, fps, Math.round(t2 * fps));

  return (
    <Scene fadeOutAfter={0.9}>
      <AbsoluteFill style={{ backgroundColor: "#000" }} />
      <AbsoluteFill
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          transform: `scale(${zoom})`,
        }}
      >
        {/* blinking cursor before the terminal appears */}
        {frame < 20 && (
          <div
            style={{
              position: "absolute",
              top: 120,
              left: 160,
              width: 18,
              height: 34,
              background: C.text,
              opacity: Math.floor(frame / 12) % 2 === 0 ? 1 : 0,
            }}
          />
        )}
        <Terminal
          title="zsh — mcp-demo"
          plan={planView}
          frame={frame}
          width={1460}
          height={580}
          fontSize={26}
        />
      </AbsoluteFill>

      {/* progressive dim overlay beneath the text */}
      <AbsoluteFill
        style={{
          background: "#000",
          opacity: interpolate(frame, [tFreeze * fps, t1 * fps], [0, 0.42], {
            extrapolateLeft: "clamp",
            extrapolateRight: "clamp",
          }),
          pointerEvents: "none",
        }}
      />
      {/* keep statements above the dim */}
      {(frame >= t1 * fps || frame >= t2 * fps) && (
        <AbsoluteFill style={{ pointerEvents: "none" }}>
          {frame >= t1 * fps && (
            <div
              style={{
                position: "absolute",
                top: 110,
                left: 60,
                right: 60,
                textAlign: "center",
                fontSize: 62,
                fontWeight: 700,
                letterSpacing: "-0.02em",
                color: C.text,
                textShadow: "0 4px 40px rgba(0,0,0,0.9)",
                ...s1,
              }}
            >
              Your MCP doesn't need access to your entire computer.
            </div>
          )}
          {frame >= t2 * fps && (
            <div
              style={{
                position: "absolute",
                bottom: 90,
                left: 0,
                right: 0,
                textAlign: "center",
                fontSize: 72,
                fontWeight: 800,
                color: C.greenSoft,
                textShadow: "0 4px 40px rgba(0,0,0,0.95)",
                ...s2,
              }}
            >
              So why give it that access?
            </div>
          )}
        </AbsoluteFill>
      )}
    </Scene>
  );
};
