import type { Config } from "tailwindcss";

const config: Config = {
  content: ["./app/**/*.{ts,tsx}", "./components/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        ink: {
          950: "#10141A",
          900: "#171C24",
          800: "#1E242E",
          700: "#2A303C",
          600: "#3A4150",
        },
        paper: "#E8EBEF",
        muted: "#8D95A5",
        grant: {
          DEFAULT: "#3FB27E",
          subtle: "rgba(63, 178, 126, 0.12)",
        },
        deny: {
          DEFAULT: "#E2604F",
          subtle: "rgba(226, 96, 79, 0.12)",
        },
        progress: {
          DEFAULT: "#D6A24A",
          subtle: "rgba(214, 162, 74, 0.12)",
        },
        blueprint: "#6E93E8",
        night: {
          950: "#050907",
          900: "#080D0A",
          800: "#0C1310",
          700: "#142019",
          600: "#1D2E24",
        },
        mint: {
          DEFAULT: "#3BF0A0",
          bright: "#6BFFC0",
          subtle: "rgba(59, 240, 160, 0.10)",
        },
        acid: {
          DEFAULT: "#5EF08A",
          bright: "#8CFFB0",
          deep: "#39D353",
          subtle: "rgba(94, 240, 138, 0.10)",
        },
        void: "#0B0D0C",
        cell: "#0E100F",
        hair: "#23262A",
        "hair-soft": "rgba(232, 235, 239, 0.06)",
        offwhite: "#E8E8E3",
      },
      fontFamily: {
        sans: ["var(--font-geist-sans)", "system-ui", "sans-serif"],
        mono: ["var(--font-geist-mono)", "ui-monospace", "monospace"],
        display: ["var(--font-display)", "Newsreader", "GT Sectra", "Tiempos Text", "serif"],
        hero: ["var(--font-hero-mono)", "JetBrains Mono", "Fira Code", "var(--font-geist-mono)", "ui-monospace", "monospace"],
        wordmark: ["var(--font-wordmark)", "Space Grotesk", "system-ui", "sans-serif"],
      },
      maxWidth: {
        content: "72rem",
        display: "80rem", // 1280pxInvisibleTech page max-width
      },
      letterSpacing: {
        tighter: "-0.03em", // display 64px
        tightHero: "-0.024em", // 52px
        tightLg: "-0.018em", // 48px
        eyebrow: "0.12em",
        eyebrowLg: "0.14em",
        // blueprint micro-labels
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
