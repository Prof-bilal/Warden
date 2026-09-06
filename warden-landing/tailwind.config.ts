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
      },
      maxWidth: {
        content: "72rem",
      },
    },
  },
  plugins: [],
};

export default config;
