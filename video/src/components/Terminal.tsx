import React, { useMemo } from "react";
import { useCurrentFrame, useVideoConfig, interpolate } from "remotion";
import type { TermLine, TermSegment, Tone } from "../data/terminal";
import { C, FONTS } from "./ui";

/* ────────────────────────────  tones  ──────────────────────────── */

const TONE: Record<Tone, string> = {
  text: C.text,
  muted: C.muted,
  faint: C.faint,
  green: C.greenSoft,
  red: C.redSoft,
  cyan: C.cyan,
  amber: C.amber,
  heading: C.text,
};

export const toneColor = (t?: Tone) => TONE[t ?? "text"];

/** Single rendered output line. */
const OutputLine: React.FC<{ text: string; tone?: Tone }> = ({ text, tone }) => (
  <div
    style={{
      fontFamily: FONTS.mono,
      fontSize: 23,
      lineHeight: 1.55,
      color: toneColor(tone),
      whiteSpace: "pre",
    }}
  >
    {text || "\u00A0"}
  </div>
);

/** Highlighted inline segment (sensitive-file callout). */
const BoxedSegment: React.FC<{ text: string; tone?: Tone }> = ({ text, tone }) => (
  <span
    style={{
      color: toneColor(tone ?? "text"),
      border: `1px solid ${C.redSoft}`,
      background: C.redDim,
      borderRadius: 6,
      padding: "1px 6px",
      margin: "0 2px",
      whiteSpace: "pre",
    }}
  >
    {text}
  </span>
);

/**
 * Highlight box for output rows that mention sensitive files.
 * Positioned in normal flow (as a left-anchored absolute overlay per row),
 * so it tracks the row it belongs to regardless of scroll position.
 */
const RowHighlight: React.FC<{ children: React.ReactNode }> = ({ children }) => (
  <span
    style={{
      color: C.amber,
      border: `1.5px solid ${C.amber}`,
      background: "rgba(245,158,11,0.10)",
      borderRadius: 8,
      padding: "0 8px",
      whiteSpace: "pre",
    }}
  >
    {children}
  </span>
);

export const Terminal: React.FC<{
  title?: string;
  plan: { line: TermLine; startS: number; endS: number }[];
  frame: number;
  width?: number | string;
  height?: number | string;
  fontSize?: number;
  style?: React.CSSProperties;
}> = ({
  title = "warden-demo",
  plan,
  frame,
  width = 1460,
  height = 860,
  fontSize = 23,
  style,
}) => {
  const { fps } = useVideoConfig();

  // Blinking cursor: half-period of 16 frames (~0.53s), like a real terminal.
  const cursorOn = Math.floor(frame / 16) % 2 === 0;
  const timeNow = frame / fps;

  const visible = useMemo(() => {
    const rows: React.ReactNode[] = [];
    plan.forEach((p, n) => {
      const line = p.line;
      if (line.kind === "cmd") {
        // Command lines appear (with cursor) only once typing starts.
        if (timeNow < p.startS) return;
        const chars = fullText(line).length;
        const typedChars = Math.floor(
          interpolate(timeNow, [p.startS, p.endS], [0, chars], {
            extrapolateLeft: "clamp",
            extrapolateRight: "clamp",
          })
        );
        const typingDone = timeNow >= p.endS;
        rows.push(
          <div
            key={`c${n}`}
            style={{
              fontFamily: FONTS.mono,
              fontSize,
              lineHeight: 1.55,
              whiteSpace: "pre",
            }}
          >
            <span style={{ color: C.greenSoft }}>$ </span>
            {renderSegments(line.segments, typedChars)}
            {!typingDone && cursorOn && (
              <span
                style={{
                  display: "inline-block",
                  width: 12,
                  height: fontSize,
                  background: C.text,
                  verticalAlign: "text-bottom",
                  marginLeft: 1,
                }}
              />
            )}
          </div>
        );
        return;
      }

      const gap = line.gapBefore ?? 0;
      const appearAt = p.startS + gap;
      if (timeNow < appearAt) return;

      const op = interpolate(timeNow, [appearAt, appearAt + 4], [0, 1], {
        extrapolateLeft: "clamp",
        extrapolateRight: "clamp",
      });

      if (line.kind === "out") {
        rows.push(
          <div key={`o${n}`} style={{ opacity: op }}>
            {line.highlight ? (
              <RowHighlight>{line.text}</RowHighlight>
            ) : (
              <OutputLine text={line.text} tone={line.tone} />
            )}
          </div>
        );
      } else {
        // out-block: cascade lines in, 1.5 frames (0.05s) apart
        rows.push(
          <div key={`b${n}`}>
            {line.lines.map((l, i) => {
              const t0 = appearAt + i * 0.05;
              const o = interpolate(timeNow, [t0, t0 + 4], [0, 1], {
                extrapolateLeft: "clamp",
                extrapolateRight: "clamp",
              });
              return (
                <div key={i} style={{ opacity: o }}>
                  <OutputLine text={l} tone={line.tone} />
                </div>
              );
            })}
          </div>
        );
      }
    });
    return rows;
  }, [plan, timeNow, fontSize, cursorOn]);

  // gentle auto-scroll: keep the newest content anchored to the bottom
  const contentRows = plan.reduce((acc, p) => {
    const line = p.line;
    if (line.kind === "out-block") return acc + line.lines.length;
    return acc + 1;
  }, 0);
  const lineH = fontSize * 1.55;
  const contentHeight = typeof height === "number" ? height - 56 - 52 : 760;
  const scrollTop = Math.max(0, contentRows * lineH - contentHeight);

  return (
    <div
      style={{
        width,
        height,
        background: C.panel,
        border: `1px solid ${C.border}`,
        borderRadius: 14,
        overflow: "hidden",
        display: "flex",
        flexDirection: "column",
        boxShadow: "0 30px 80px rgba(0,0,0,0.55)",
        ...style,
      }}
    >
      {/* Title bar */}
      <div
        style={{
          height: 56,
          flexShrink: 0,
          display: "flex",
          alignItems: "center",
          gap: 10,
          padding: "0 20px",
          borderBottom: `1px solid ${C.borderSoft}`,
          background: C.panel2,
        }}
      >
        {["#ff5f57", "#febc2e", "#28c840"].map((c) => (
          <div key={c} style={{ width: 14, height: 14, borderRadius: 999, background: c }} />
        ))}
        <div style={{ marginLeft: 12, fontFamily: FONTS.mono, fontSize: 19, color: C.faint }}>
          {title}
        </div>
        <div style={{ flex: 1 }} />
      </div>

      {/* Body */}
      <div style={{ height: 52, flexShrink: 0 }} />
      <div
        style={{
          flex: 1,
          padding: "0 30px 26px 30px",
          overflow: "hidden",
          display: "flex",
          flexDirection: "column",
          justifyContent: "flex-end",
        }}
      >
        <div style={{ transform: `translateY(${-scrollTop}px)`, marginBottom: -scrollTop }}>
          {visible}
        </div>
      </div>
    </div>
  );
};

/* ────────────────────────────  helpers  ──────────────────────────── */

function fullText(line: Extract<TermLine, { kind: "cmd" }>) {
  return line.segments.map((s) => s.text).join("");
}

function renderSegments(segments: TermSegment[], typedChars: number) {
  let remaining = typedChars;
  const out: React.ReactNode[] = [];
  segments.forEach((seg, i) => {
    const take = Math.max(0, Math.min(seg.text.length, remaining));
    if (take > 0) {
      out.push(
        seg.box && take === seg.text.length ? (
          <BoxedSegment key={i} text={seg.text.slice(0, take)} tone={seg.tone} />
        ) : (
          <span key={i} style={{ color: toneColor(seg.tone), whiteSpace: "pre" }}>
            {seg.text.slice(0, take)}
          </span>
        )
      );
      remaining -= take;
    }
  });
  return out;
}
