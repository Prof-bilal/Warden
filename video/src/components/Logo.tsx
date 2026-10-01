import React from "react";
import { C, FONTS } from "./ui";

/**
 * Warden logo mark — a faithful inline-SVG reproduction of the repository's
 * official icon (warden-landing/public/icon.svg): green shield, dark visor.
 * Inlined so renders never depend on image loading timing.
 */
export const LogoMark: React.FC<{ size?: number; glow?: boolean }> = ({ size = 96, glow = false }) => (
  <svg
    width={size}
    height={size}
    viewBox="0 0 512 512"
    fill="none"
    style={
      glow
        ? { filter: `drop-shadow(0 0 ${size / 5}px rgba(22,163,74,0.45))` }
        : undefined
    }
  >
    <path
      d="M256 48L100 120v136c0 112 66 184 156 216 90-32 156-104 156-216V120L256 48z"
      fill="#16a34a"
      stroke="#15803d"
      strokeWidth={8}
    />
    <path d="M216 208h-24v80h24v-80zm104 0h-24v80h24v-80z" fill="#0a0f14" />
    <path d="M216 208h104v24H216v-24zm0 32h104v24H216v-24z" fill="#0a0f14" />
  </svg>
);

/** Full wordmark lockup: mark + WARDEN text. */
export const Logo: React.FC<{
  size?: number;
  fontSize?: number;
  glow?: boolean;
  color?: string;
}> = ({ size = 96, fontSize = 84, glow = false, color = C.text }) => (
  <div style={{ display: "flex", alignItems: "center", gap: fontSize * 0.42 }}>
    <LogoMark size={size} glow={glow} />
    <div
      style={{
        fontFamily: FONTS.sans,
        fontSize,
        fontWeight: 800,
        letterSpacing: "0.16em",
        color,
        lineHeight: 1,
      }}
    >
      WARDEN
    </div>
  </div>
);
