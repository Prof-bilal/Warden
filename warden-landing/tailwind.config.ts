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
      },
      fontFamily: {
        sans: ["var(--font-geist-sans)", "system-ui", "sans-serif"],
        mono: ["var(--font-geist-mono)", "ui-monospace", "monospace"],
        display: ["var(--font-display)", "Newsreader", "GT Sectra", "Tiempos Text", "serif"],
      },
      maxWidth: {
        content: "72rem",
        display: "80rem", // 1280px — InvisibleTech page max-width
      },
      letterSpacing: {
        tighter: "-0.03em", // display 64px
        tightHero: "-0.024em", // 52px
        tightLg: "-0.018em", // 48px
        eyebrow: "0.12em",
        eyebrowLg: "0.14em",
      },
    },
  },
  plugins: [],
};

export default config;
