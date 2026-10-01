import React from "react";
import { useCurrentFrame, useVideoConfig, interpolate } from "remotion";
import { springIn } from "./ui";
import { C, FONTS } from "./ui";

/**
 * Reusable permission row used in the Policy scene.
 *
 *   <PermissionRow resource="./data" permission="filesystem" status="allowed" />
 *
 * Rows mirror the real `warden run` summary format (docs/cli.md):
 *   ✓ ./data (read) / ✗ everything else
 */
export const PermissionRow: React.FC<{
  resource: string;
  permission: string;
  status: "allowed" | "denied";
  delay?: number;
  width?: number;
  detail?: string;
  highlightPulse?: number; // frame at which the row pulses (access moment)
}> = ({
  resource,
  permission,
  status,
  delay = 0,
  width = 720,
  detail,
  highlightPulse,
}) => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();
  const s = springIn(frame, fps, delay, { damping: 200, stiffness: 110, mass: 0.7 });
  const allowed = status === "allowed";

  // Pulse when highlighted (used in Architecture scene)
  let pulse = 0;
  if (highlightPulse !== undefined && frame >= highlightPulse) {
    pulse = interpolate(frame, [highlightPulse, highlightPulse + 14], [0.55, 0], {
      extrapolateLeft: "clamp",
      extrapolateRight: "clamp",
    });
  }

  const accent = allowed ? C.greenSoft : C.redSoft;
  const dim = allowed ? C.greenDim : C.redDim;

  return (
    <div
      style={{
        width,
        display: "flex",
        alignItems: "center",
        gap: 18,
        padding: "16px 22px",
        borderRadius: 12,
        background: C.panel,
        border: `1px solid ${allowed ? "rgba(34,197,94,0.25)" : "rgba(239,68,68,0.25)"}`,
        opacity: s,
        transform: `translateX(${(1 - s) * 26}px)`,
        position: "relative",
        overflow: "hidden",
        fontFamily: FONTS.mono,
      }}
    >
      {pulse > 0 && (
        <div
          style={{
            position: "absolute",
            inset: 0,
            background: dim,
            opacity: pulse,
            pointerEvents: "none",
          }}
        />
      )}
      {/* status glyph */}
      <div
        style={{
          width: 34,
          height: 34,
          borderRadius: 8,
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          background: dim,
          color: accent,
          fontSize: 19,
          fontWeight: 700,
          flexShrink: 0,
        }}
      >
        {allowed ? "✓" : "✗"}
      </div>
      {/* resource */}
      <div style={{ fontSize: 24, color: C.text, fontWeight: 500, whiteSpace: "nowrap" }}>
        {resource}
      </div>
      {detail && (
        <div style={{ fontSize: 19, color: C.faint, whiteSpace: "nowrap" }}>{detail}</div>
      )}
      <div style={{ flex: 1 }} />
      {/* permission kind */}
      <div style={{ fontSize: 19, color: C.muted, letterSpacing: "0.06em" }}>{permission}</div>
    </div>
  );
};
