import type { Config } from "tailwindcss";

const color = (name: string) =>
  `color-mix(in srgb, var(--${name}) calc(<alpha-value> * 100%), transparent)`;

const config: Config = {
  content: ["./app/**/*.{ts,tsx}", "./components/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        ink: {
          950: color("canvas"),
          900: color("surface"),
          800: color("surface-raised"),
          700: color("rule"),
          600: color("graphite"),
          500: color("graphite"),
        },
        paper: color("charcoal"),
        muted: color("graphite"),
        grant: { DEFAULT: color("charcoal"), subtle: color("surface") },
        deny: { DEFAULT: color("ink"), subtle: color("surface-raised") },
        progress: { DEFAULT: color("charcoal"), subtle: color("surface") },
        blueprint: color("link"),
        night: {
          950: color("canvas"),
          900: color("surface"),
          800: color("surface-raised"),
          700: color("surface-strong"),
          600: color("rule"),
        },
        mint: { DEFAULT: color("charcoal"), bright: color("ink"), subtle: color("surface") },
        acid: { DEFAULT: color("link"), bright: color("link"), deep: color("link"), subtle: color("surface") },
        void: color("canvas"),
        cell: color("canvas"),
        hair: color("rule"),
        "hair-soft": color("rule-soft"),
        offwhite: color("ink"),
      },
      fontFamily: {
        sans: ["var(--site-mono)"],
        mono: ["var(--site-mono)"],
        display: ["var(--site-mono)"],
        hero: ["var(--site-mono)"],
        wordmark: ["var(--site-mono)"],
      },
      borderRadius: {
        DEFAULT: "0px",
        sm: "0px",
        md: "0px",
        lg: "0px",
        xl: "0px",
        "2xl": "0px",
        "3xl": "0px",
      },
      maxWidth: {
        content: "80rem",
        display: "80rem", // 1280pxInvisibleTech page max-width
      },
      letterSpacing: {
        tighter: "-0.03em", // display 64px
        tightHero: "-0.024em", // 52px
        tightLg: "-0.018em", // 48px
        eyebrow: "0.12em",
        eyebrowLg: "0.14em",
        // Small interface labels
        micro: "0.16em",
      },
      backgroundImage: {
        // 1px hairline graph paper, 64px cells
        "bp-grid":
          "linear-gradient(to right, rgba(232,235,239,0.045) 1px, transparent 1px), linear-gradient(to bottom, rgba(232,235,239,0.045) 1px, transparent 1px)",
      },
      backgroundSize: {
        "bp-grid": "64px 64px",
      },
      keyframes: {
        "bp-sweep": {
          "0%": { transform: "translateX(-120%)" },
          "100%": { transform: "translateX(220%)" },
        },
        "bp-node": {
          "0%, 100%": { opacity: "0.35" },
          "50%": { opacity: "1" },
        },
        "bp-log-in": {
          "0%": { opacity: "0", transform: "translateY(-4px)" },
          "100%": { opacity: "1", transform: "none" },
        },
        "bp-dash": {
          "0%": { strokeDashoffset: "60" },
          "100%": { strokeDashoffset: "0" },
        },
      },
      animation: {
        "bp-sweep": "bp-sweep 6s linear infinite",
        "bp-node": "bp-node 2.4s ease-in-out infinite",
        "bp-log-in": "bp-log-in 0.4s ease-out both",
        "bp-dash": "bp-dash 1.6s ease-out both",
      },
    },
  },
  plugins: [],
};

export default config;
