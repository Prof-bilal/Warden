import React from "react";
import { AbsoluteFill, interpolate, useCurrentFrame, useVideoConfig } from "remotion";
import { C, FONTS, Scene, riseIn, springIn } from "../components/ui";
import { PermissionRow } from "../components/PermissionRow";
import { POLICY_SECTIONS, POLICY_ROWS } from "../data/product";

/*
 * Scene 05 — Policy (2:40–3:10)
 *
 * Left: REAL policy.yaml syntax (from warden-starter/warden/examples,
 * github-mcp-server.yaml, trimmed for legibility).
 * Right: permission rows mirroring the real `warden run` pre-launch summary.
 */
export const Policy: React.FC = () => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();

  const titleS = riseIn(frame, fps, 6);
  const yamlS = springIn(frame, fps, Math.round(1.0 * fps), { damping: 200, stiffness: 70 });
  const rowsS = springIn(frame, fps, Math.round(4.0 * fps), { damping: 200, stiffness: 70 });

  return (
    <Scene fadeOutAfter={0.8}>
      <AbsoluteFill
        style={{
          backgroundImage:
            "radial-gradient(rgba(255,255,255,0.04) 1px, transparent 1px)",
          backgroundSize: "36px 36px",
        }}
      />

      <div style={{ position: "absolute", top: 80, left: 0, right: 0, textAlign: "center", ...titleS }}>
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
          The boundary, in writing
        </div>
        <div style={{ fontSize: 60, fontWeight: 700, letterSpacing: "-0.02em" }}>
          One policy file. Explicit permissions.
        </div>
      </div>

      <div
        style={{
          position: "absolute",
          top: 260,
          left: 0,
          right: 0,
          display: "flex",
          justifyContent: "center",
          gap: 70,
          alignItems: "flex-start",
        }}
      >
        {/* policy.yaml */}
        <div
          style={{
            width: 700,
            background: C.panel,
            border: `1px solid ${C.border}`,
            borderRadius: 16,
            padding: "28px 34px",
            opacity: yamlS,
            transform: `translateY(${(1 - yamlS) * 20}px)`,
            boxShadow: "0 20px 60px rgba(0,0,0,0.4)",
          }}
        >
          <div style={{ fontFamily: FONTS.mono, fontSize: 19, color: C.faint, marginBottom: 18 }}>
            policy.yaml
          </div>
          <YamlCode frame={frame} fps={fps} />
        </div>

        {/* permission rows */}
        <div
          style={{
            width: 760,
            display: "flex",
            flexDirection: "column",
            gap: 14,
            opacity: rowsS,
          }}
        >
          {POLICY_SECTIONS.map((section, si) => (
            <div key={section}>
              <div
                style={{
                  fontFamily: FONTS.mono,
                  fontSize: 20,
                  letterSpacing: "0.22em",
                  textTransform: "uppercase",
                  color: C.faint,
                  margin: si === 0 ? "0 0 10px 4px" : "18px 0 10px 4px",
                }}
              >
                {section}
              </div>
              {POLICY_ROWS[si].map((row, ri) => {
                const delay = Math.round((4.6 + si * 2.2 + ri * 0.7) * fps);
                return (
                  <div key={row.resource} style={{ marginBottom: 10 }}>
                    <PermissionRow
                      resource={row.resource}
                      permission={row.permission}
                      status={row.status}
                      delay={delay}
                      width={760}
                    />
                  </div>
                );
              })}
            </div>
          ))}
        </div>
      </div>

      {/* footer line */}
      <div
        style={{
          position: "absolute",
          bottom: 66,
          left: 0,
          right: 0,
          textAlign: "center",
          fontSize: 28,
          color: C.muted,
          opacity: interpolate(frame, [24 * fps, 24 * fps + 14], [0, 1], {
            extrapolateLeft: "clamp",
            extrapolateRight: "clamp",
          }),
        }}
      >
        Blocked paths don't return “permission denied” — they return “not found”.
      </div>
    </Scene>
  );
};

/* ────────────────────────────  yaml code block  ──────────────────────────── */

const YAML_LINES: { text: string; tone: "key" | "str" | "num" | "comment" | "plain" }[] = [
  { text: 'command: ["/usr/bin/node", "server.js"]', tone: "plain" },
  { text: "filesystem:", tone: "key" },
  { text: '  read:  ["./data/cache"]', tone: "str" },
  { text: '  write: ["./data/output"]', tone: "str" },
  { text: "network:", tone: "key" },
  { text: '  allow: ["api.github.com"]', tone: "str" },
  { text: "env:", tone: "key" },
  { text: '  allow: ["GITHUB_TOKEN"]', tone: "str" },
  { text: "limits:", tone: "key" },
  { text: "  memory_mb: 256", tone: "num" },
  { text: "  timeout_s: 300", tone: "num" },
];

const TONE_COLOR: Record<string, string> = {
  key: C.cyan,
  str: C.greenSoft,
  num: C.amber,
  comment: C.faint,
  plain: C.text,
};

const YamlCode: React.FC<{ frame: number; fps: number }> = ({ frame, fps }) => (
  <div>
    {YAML_LINES.map((l, i) => {
      const o = interpolate(frame, [1.6 * fps + i * 5, 1.6 * fps + i * 5 + 8], [0, 1], {
        extrapolateLeft: "clamp",
        extrapolateRight: "clamp",
      });
      return (
        <div
          key={i}
          style={{
            fontFamily: FONTS.mono,
            fontSize: 26,
            lineHeight: 1.65,
            color: TONE_COLOR[l.tone],
            whiteSpace: "pre",
            opacity: o,
          }}
        >
          {l.text}
        </div>
      );
    })}
  </div>
);
