/**
 * Real Warden facts, verified from the repository (README, ARCHITECTURE.md,
 * docs/cli.md, docs/quickstart.md, examples/*.yaml).
 *
 * Nothing in this file may be invented. If a fact is not in the repo, it
 * does not appear in the video.
 */

export const REPO_URL = "https://github.com/Prof-bilal/Warden";
export const REPO_SHORT = "github.com/Prof-bilal/Warden";
export const NPM_PACKAGE = "warden-sandbox-cli";
export const DOCS_URL = "prof-bilal.github.io/Warden";
export const LICENSE = "MIT";

export const TAGLINE = "The sandbox runtime for MCP servers";
export const MANTRA = "Run MCPs. Don't blindly trust them.";

/** Scene 04 — the boundary flow shown on screen. */
export const INTRO_FLOW = {
  server: "MCP SERVER",
  boundary: "WARDEN",
  boundaryItems: ["POLICY", "+", "SANDBOX"],
  host: "HOST SYSTEM",
  outside: ["~/.ssh/", ".env", "credentials/", "~/.aws/"],
} as const;

/**
 * Scene 05 — permission summary mirroring the exact "warden run" pre-launch
 * summary sections printed by the real CLI (docs/cli.md). Green = granted
 * in policy.yaml, red = "everything else" (deny by default).
 */
export type PermissionRow = {
  resource: string;
  permission: string;
  status: "allowed" | "denied";
};

export const POLICY_ROWS: PermissionRow[][] = [
  [
    { resource: "./data", permission: "filesystem", status: "allowed" },
    { resource: "./output", permission: "filesystem", status: "allowed" },
    { resource: "everything else", permission: "filesystem", status: "denied" },
  ],
  [
    { resource: "api.github.com", permission: "network", status: "allowed" },
    { resource: "everything else", permission: "network", status: "denied" },
  ],
  [
    { resource: "GITHUB_TOKEN", permission: "environment", status: "allowed" },
    { resource: "all unspecified variables", permission: "environment", status: "denied" },
  ],
];

export const POLICY_SECTIONS = [
  "Filesystem",
  "Network",
  "Environment",
] as const;

/**
 * Scene 06 — demo commands and outputs. Each mirrors the real CLI:
 * - `warden run` pre-launch summary (docs/cli.md, TTY stderr block)
 * - `warden doctor` report (docs/cli.md)
 * - `warden logs` audit formatting "✓ ALLOWED / ✗ BLOCKED" (docs/cli.md)
 * - audit JSONL fields (ARCHITECTURE.md: timestamp, type, action, resource, allowed, reason)
 */
export const DEMO_POLICY_YAML = `command: ["/usr/bin/node", "./server/dist/index.js"]
filesystem:
  read: ["./data"]
  write: ["./output"]
network:
  allow: ["api.github.com"]
env:
  allow: ["GITHUB_TOKEN"]
limits:
  memory_mb: 512
  timeout_s: 300`;

export const DEMO_RUN_SUMMARY = `WARDEN
──────────────────────────────────────
Policy     policy.yaml
Backend    linux
Command    /usr/bin/node server.js

Filesystem
  ✓ ./data (read)
  ✓ ./output (write)
  ✗ everything else

Network
  ✓ api.github.com
  ✗ everything else

Environment
  ✓ GITHUB_TOKEN
  ✗ all unspecified variables

──────────────────────────────────────
✓ Sandbox active`;

export const DEMO_ACCESS_EVENTS = [
  { op: "filesystem.read", target: "./data/policy-notes.md", allowed: true },
  { op: "network.connect", target: "api.github.com:443", allowed: true },
  { op: "filesystem.read", target: "~/.ssh/id_ed25519", allowed: false },
  { op: "filesystem.read", target: ".env", allowed: false },
] as const;

export const DEMO_LOGS = `✓ ALLOWED  file  read      ./data/policy-notes.md
✓ ALLOWED  net   connect   api.github.com
✗ BLOCKED  file  read      ~/.ssh/id_ed25519
✗ BLOCKED  file  read      .env`;

export const DEMO_AUDIT_JSONL = `{"type":"file","action":"read","resource":"./data/policy-notes.md","allowed":true}
{"type":"file","resource":"~/.ssh/id_ed25519","allowed":false}`;

/**
 * Scene 06 — REAL captured output from running warden v0.1.16 on Linux
 * (captured September 2026 via `npx warden-sandbox-cli`).
 * - audit lines: verbatim from ~/.local/state/warden/audit.jsonl
 * - doctor report: verbatim `warden doctor` (v0.1.16, linux/amd64)
 */
export const REAL_AUDIT_LINES = [
  `{"timestamp":"2026-09-30T14:16:38.539408023Z","type":"network","action":"connect","resource":"api.github.com:443","allowed":true,"reason":"host allowed"}`,
  `{"timestamp":"2026-09-30T14:16:40.363068544Z","type":"network","action":"connect","resource":"example.com:443","allowed":true,"reason":"host allowed"}`,
  `{"timestamp":"2026-09-30T14:16:42.103154347Z","type":"network","action":"CONNECT","resource":"google.com:443","allowed":false,"reason":"host is not in network.allow"}`,
] as const;

export const REAL_DOCTOR_ENV = [
  "• Operating system        linux/amd64",
  "• Warden version          v0.1.16",
  "✓ Sandbox backend         linux",
  "✓ Namespace support       available (bwrap)",
  "✓ Network proxy           available (egress allowlist)",
  "✓ Policy engine           ready (YAML + validation)",
  "✓ Audit (strace)          available",
  "✓ Fail-closed             enabled (never runs unsandboxed)",
] as const;

export const REAL_DOCTOR_POSTURE = [
  "✓ Filesystem isolation    explicit paths only",
  "✓ Environment filtering   explicit vars only",
  "✓ Network policy          explicit hosts only",
  "✓ Fail-closed behavior    enforced",
] as const;

/** Scene 08 — platform enforcement matrix, verbatim facts from the README table. */
export const PLATFORMS = [
  {
    os: "Linux",
    backend: "bubblewrap",
    detail: "user / network / pid / ipc namespaces",
  },
  {
    os: "macOS",
    backend: "Seatbelt",
    detail: "sandbox-exec profiles generated from policy",
  },
  {
    os: "Windows",
    backend: "AppContainer",
    detail: "restricted token + WFP filters + Job Objects",
  },
  {
    os: "Fallback",
    backend: "Docker",
    detail: "--network none, read-only root, tmpfs",
  },
] as const;

/** Scene 10 — install commands (all real). */
export const INSTALL_COMMANDS = [
  "npm install -g warden-sandbox-cli",
  "warden trace -- node server.js",
  "warden init",
  "warden run --policy policy.yaml",
] as const;
