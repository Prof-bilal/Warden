/**
 * Shared UI primitives and hooks for the Warden video.
 * Everything is frame-driven: no independent clocks, no randomness.
 */
import React from "react";
import { AbsoluteFill, interpolate, spring, useCurrentFrame, useVideoConfig } from "remotion";

/* ────────────────────────────  palette  ──────────────────────────── */

export const C = {
  bg: "#09090b",
  panel: "#111113",
  panel2: "#16161a",
  border: "#27272a",
  borderSoft: "#1e1e22",
  text: "#f4f4f5",
  muted: "#a1a1aa",
  faint: "#71717a",
  green: "#16a34a",
  greenSoft: "#22c55e",
  greenDim: "rgba(22,163,74,0.16)",
  red: "#dc2626",
  redSoft: "#ef4444",
  redDim: "rgba(220,38,38,0.14)",
  amber: "#f59e0b",
  cyan: "#22d3ee",
  violet: "#8b5cf6",
};

export const FONTS = {
  sans: '"Inter", system-ui, sans-serif',
  mono: '"JetBrains Mono", ui-monospace, "SF Mono", monospace',
};

/* ────────────────────────────  timing constants  ──────────────────────────── */

/** Seconds a scene holds at the end before cut/fade. */
export const TAIL = 0.6;

/** Duration (frames) of the shared cross-fade at scene boundaries. */
export const FADE = 10;

/* ────────────────────────────  entrance helpers  ──────────────────────────── */

export const springIn = (
  frame: number,
  fps: number,
  delay = 0,
  config?: Partial<{ damping: number; stiffness: number; mass: number }>
) =>
  spring({
    frame: frame - delay,
    fps,
    config: { damping: 200, stiffness: 90, mass: 0.9, ...config },
  });

export const fadeIn = (frame: number, delay = 0, dur = 12) =>
  interpolate(frame, [delay, delay + dur], [0, 1], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });

export const riseIn = (frame: number, fps: number, delay = 0) => {
  const s = springIn(frame, fps, delay, { damping: 200, stiffness: 80 });
  return {
    opacity: s,
    transform: `translateY(${(1 - s) * 18}px)`,
  };
};

/* ────────────────────────────  primitives  ──────────────────────────── */

export const Scene: React.FC<{ children: React.ReactNode; fadeOutAfter?: number }> = ({
  children,
  fadeOutAfter,
}) => {
  const frame = useCurrentFrame();
  const { durationInFrames } = useVideoConfig();
  const tailFrames = (fadeOutAfter ?? TAIL) * 30;
  const opacity =
    tailFrames > 0
      ? interpolate(
          frame,
          [durationInFrames - tailFrames, durationInFrames - 1],
          [1, 0],
          { extrapolateLeft: "clamp", extrapolateRight: "clamp" }
        )
      : 1;
  return <AbsoluteFill style={{ backgroundColor: C.bg }}>{children}</AbsoluteFill>;
};

export const SectionTitle: React.FC<{
  kicker: string;
  title: string;
  delay?: number;
  align?: "left" | "center";
  style?: React.CSSProperties;
}> = ({ kicker, title, delay = 0, align = "left", style }) => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();
  const left = riseIn(frame, fps, delay);
  const right = riseIn(frame, fps, delay + 5);
  return (
    <div style={{ textAlign: align, ...style }}>
      <div
        style={{
          fontFamily: FONTS.mono,
          fontSize: 22,
          letterSpacing: "0.35em",
          textTransform: "uppercase",
          color: C.faint,
          marginBottom: 14,
          ...left,
        }}
      >
        {kicker}
      </div>
      <div
        style={{
          fontSize: 68,
          fontWeight: 700,
          letterSpacing: "-0.02em",
          lineHeight: 1.05,
          ...right,
          opacity: frame < delay + 5 ? 0 : right.opacity,
        }}
      >
        {title}
  </div>
    </div>
  );
};

export const StatusBadge: React.FC<{
  status: "allowed" | "denied";
  label?: string;
  size?: number;
  style?: React.CSSProperties;
}> = ({ status, label, size = 30, style }) => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();
  const s = springIn(frame, fps, 0, { damping: 12, stiffness: 160 });
  const allowed = status === "allowed";
  return (
    <span
      style={{
        display: "inline-flex",
        alignItems: "center",
        gap: Math.round(size * 0.3),
        fontFamily: FONTS.mono,
        fontSize: size * 0.55,
        fontWeight: 600,
        letterSpacing: "0.08em",
        color: allowed ? C.greenSoft : C.redSoft,
        background: allowed ? C.greenDim : C.redDim,
        border: `1px solid ${allowed ? "rgba(34,197,94,0.35)" : "rgba(239,68,68,0.35)"}`,
        borderRadius: 999,
        padding: `${Math.round(size * 0.18)}px ${Math.round(size * 0.62)}px`,
        transform: `scale(${s})`,
        ...style,
      }}
    >
      {allowed ? "✓" : "✗"} {label ?? (allowed ? "ALLOWED" : "DENIED")}
    </span>
  );
};

/** A small square glyph used across diagrams: node boxes, file chips, etc. */
export const Chip: React.FC<{
  label: string;
  color?: string;
  bg?: string;
  border?: string;
  fontSize?: number;
  padding?: string;
  style?: React.CSSProperties;
}> = ({ label, color = C.muted, bg = C.panel, border = C.border, fontSize = 22, padding = "10px 18px", style }) => (
  <div
    style={{
      fontFamily: FONTS.mono,
      fontSize,
      color,
      background: bg,
      border: `1px solid ${border}`,
      borderRadius: 10,
      padding,
      whiteSpace: "nowrap",
      ...style,
    }}
  >
    {label}
  </div>
);

/** Full-bleed scene background with subtle grid + vignette. */
export const SceneBg: React.FC<{ variant?: "grid" | "dots" | "plain" }> = ({ variant = "grid" }) => {
  const frame = useCurrentFrame();
  const { durationInFrames } = useVideoConfig();
  const drift = frame * 0.15;
  return (
    <AbsoluteFill
      style={{
        backgroundImage:
          variant === "grid"
            ? "linear-gradient(to right, rgba(255,255,255,0.03) 1px, transparent 1px), linear-gradient(to bottom, rgba(255,255,255,0.03) 1px, transparent 1px)"
            : variant === "dots"
              ? "radial-gradient(rgba(255,255,255,0.05) 1px, transparent 1px)"
              : undefined,
        backgroundSize: variant === "grid" ? "56px 56px" : "36px 36px",
        transform: `translateY(${-drift}px)`,
        opacity: interpolate(frame, [0, 20, durationInFrames - 20, durationInFrames - 1], [0.55, 0.9, 0.9, 0.55], {
          extrapolateLeft: "clamp",
          extrapolateRight: "clamp",
        }),
      }}
    />
  );
};

export const Kicker: React.FC<{ children: React.ReactNode; delay?: number; style?: React.CSSProperties }> = ({
  children,
  delay = 0,
  style,
}) => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();
  return (
    <div
      style={{
        fontFamily: FONTS.mono,
        fontSize: 22,
        letterSpacing: "0.35em",
        textTransform: "uppercase",
        color: C.faint,
        ...riseIn(frame, fps, delay),
        ...style,
      }}
    >
      {children}
    </div>
  );
};
