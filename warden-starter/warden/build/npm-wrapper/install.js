#!/usr/bin/env node
"use strict";

const https = require("https");
const fs = require("fs");
const path = require("path");
const os = require("os");

const VERSION = process.env.npm_package_version || "0.1.0";

const platform = os.platform();
const arch = os.arch();

const platformMap = {
  linux: "linux",
  darwin: "darwin",
  win32: "windows",
};

const archMap = {
  x64: "amd64",
  arm64: "arm64",
};

const osName = platformMap[platform];
const archName = archMap[arch];

// ── Terminal capability detection (mirrors Go ui) ──
function colorEnabled() {
  if (process.env.NO_COLOR) return false;
  if (process.env.FORCE_COLOR) return true;
  if (process.env.TERM === "dumb") return false;
  if (process.env.WARDEN_NO_COLOR) return false;
  return Boolean(process.stderr.isTTY);
}
function isCI() {
  if (process.env.CI && process.env.CI !== "0" && process.env.CI.toLowerCase() !== "false") return true;
  return Boolean(process.env.GITHUB_ACTIONS || process.env.GITLAB_CI || process.env.CIRCLECI || process.env.BUILDKITE);
}
function supportsUnicode() {
  if (process.env.WARDEN_NO_UNICODE || process.env.NO_UNICODE) return false;
  if (process.env.TERM === "dumb") return false;
  const lang = (process.env.LANG || process.env.LC_ALL || "").toLowerCase();
  if (lang.includes("utf-8") || lang.includes("utf8")) return true;
  return true; // assume modern terminal
}
function isTTY() {
  return Boolean(process.stderr.isTTY) && !isCI();
}
const useColor = colorEnabled();
const useUnicode = supportsUnicode();
const tty = isTTY();

function c(code, s) { return useColor ? `\x1b[${code}m${s}\x1b[0m` : s; }
const green = (s) => c(32, s);
const red = (s) => c(31, s);
const cyan = (s) => c(36, s);
const dim = (s) => c(2, s);
const bold = (s) => c(1, s);
const check = useUnicode ? "✓" : "OK";
const cross = useUnicode ? "✗" : "x";
const frames = useUnicode ? ["⠋","⠙","⠹","⠸","⠼","⠴","⠦","⠧","⠇","⠏"] : ["-","\\","|","/"];

// ── Banner ──
const banner = [
  "██     ██  █████  ██████  ██████  ███████ ███    ██",
  "██     ██ ██   ██ ██   ██ ██   ██ ██      ████   ██",
  "██  █  ██ ███████ ██████  ██   ██ █████   ██ ██  ██",
  "██ ███ ██ ██   ██ ██   ██ ██   ██ ██      ██  ██ ██",
  " ███ ███  ██   ██ ██   ██ ██████  ███████ ██   ████",
];
function printBanner() {
  if (tty) {
    for (const line of banner) {
      process.stderr.write((useColor ? cyan(line) : line) + "\n");
    }
    const subtitle = "MCP SERVER SANDBOX";
    const bw = banner[0].length;
    const pad = " ".repeat(Math.max(0, Math.floor((bw - subtitle.length)/2)));
    process.stderr.write(pad + (useColor ? dim(subtitle) : subtitle) + "\n");
    process.stderr.write("\n");
    const tagline = "Secure execution for MCP servers";
    const pad2 = " ".repeat(Math.max(0, Math.floor((bw - tagline.length)/2)));
    process.stderr.write(pad2 + (useColor ? dim(tagline) : tagline) + "\n");
    process.stderr.write("\n");
  } else {
    // CI / non-TTY: compact header, no large art to keep logs clean
    process.stderr.write((useColor ? bold(cyan("WARDEN")) : "WARDEN") + " — " + (useColor ? dim("MCP Server Sandbox") : "MCP Server Sandbox") + "  v" + VERSION + "\n");
  }
}

// ── Progress helpers ──
function printStaticStep(current, total, msg, ok) {
  const status = ok ? (useColor ? green("OK") : "OK") : (useColor ? red("FAIL") : "FAIL");
  process.stderr.write(`[${current}/${total}] ${msg}... ${status}\n`);
}
function printCheck(msg) {
  process.stderr.write(`${useColor ? green(check) : check} ${msg}\n`);
}
function printCross(msg) {
  process.stderr.write(`${useColor ? red(cross) : cross} ${msg}\n`);
}

let spinnerTimer = null;
let spinnerIdx = 0;
function startSpinner(msg) {
  if (!tty) return () => {};
  spinnerIdx = 0;
  process.stderr.write(`\r${useColor ? cyan(frames[0]) : frames[0]} ${msg}   `);
  spinnerTimer = setInterval(() => {
    spinnerIdx = (spinnerIdx + 1) % frames.length;
    const f = useColor ? cyan(frames[spinnerIdx]) : frames[spinnerIdx];
    process.stderr.write(`\r${f} ${msg}   `);
  }, 80);
  return (ok, finalMsg) => {
    clearInterval(spinnerTimer);
    spinnerTimer = null;
    // clear line
    process.stderr.write("\r" + " ".repeat(60) + "\r");
    if (ok) printCheck(finalMsg || msg);
    else printCross(finalMsg || msg);
  };
}

// ── Main ──
if (!osName) {
  process.stderr.write(`${useColor ? red(cross) : cross} ${useColor ? red("SANDBOX UNAVAILABLE") : "SANDBOX UNAVAILABLE"}\n\n`);
  process.stderr.write(`warden: unsupported platform "${platform}"\n`);
  process.stderr.write(dim("Warden fails closed when sandboxing is unavailable.\n"));
  process.exit(1);
}

if (!archName) {
  process.stderr.write(`${useColor ? red(cross) : cross} unsupported architecture "${arch}"\n`);
  process.exit(1);
}

const ext = platform === "win32" ? ".exe" : "";
const fileName = `warden-${osName}-${archName}${ext}`;
const url = `https://github.com/Prof-bilal/Warden/releases/download/v${VERSION}/${fileName}`;

const binDir = path.join(__dirname, "bin");
const binPath = path.join(binDir, platform === "win32" ? "warden.exe" : "warden");

function download(url, dest) {
  return new Promise((resolve, reject) => {
    const file = fs.createWriteStream(dest);
    https
      .get(url, { timeout: 30000 }, (res) => {
        if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
          download(res.headers.location, dest).then(resolve).catch(reject);
          return;
        }
        if (res.statusCode !== 200) {
          reject(new Error(`HTTP ${res.statusCode} downloading ${url}`));
          return;
        }
        res.pipe(file);
        file.on("finish", () => {
          file.close();
          resolve();
        });
      })
      .on("error", (err) => {
        fs.unlink(dest, () => {});
        reject(err);
      });
  });
}

(async () => {
  printBanner();
  if (tty) {
    process.stderr.write(dim("Installing Warden...") + "\n\n");
  }

  const stepsTotal = 4;
  let step = 0;

  // Step 1: Checking platform
  if (tty) {
    const stop = startSpinner("Checking platform");
    await new Promise(r => setTimeout(r, 120)); // brief, not slowing actual work
    const ok = Boolean(osName && archName);
    stop(ok, `Checking platform  ${dim(`(${osName}/${archName})`)}`);
    if (!ok) process.exit(1);
  } else {
    step++; printStaticStep(step, stepsTotal, "Checking platform", true);
  }

  // Step 2: Installing runtime / downloading
  if (!fs.existsSync(binDir)) fs.mkdirSync(binDir, { recursive: true });

  if (!fs.existsSync(binPath)) {
    if (tty) {
      const stop = startSpinner(`Installing runtime  ${dim(fileName)}`);
      try {
        await download(url, binPath);
        stop(true, `Installing runtime  ${dim(fileName)}`);
      } catch (err) {
        stop(false, `Installing runtime  ${dim(fileName)}`);
        process.stderr.write(`\n${useColor ? red(cross) : cross} ${useColor ? red("Installation failed") : "Installation failed"}: ${err.message}\n`);
        process.stderr.write(`\nYou can manually install from: https://github.com/Prof-bilal/Warden/releases\n`);
        process.stderr.write(dim("Warden fails closed when sandboxing is unavailable.") + "\n");
        process.exit(1);
      }
    } else {
      step++; process.stderr.write(`[${step}/${stepsTotal}] Installing runtime... `);
      try {
        await download(url, binPath);
        process.stderr.write(useColor ? green("OK") : "OK");
        process.stderr.write("\n");
      } catch (err) {
        process.stderr.write(useColor ? red("FAIL") : "FAIL");
        process.stderr.write(`\nFailed to download warden binary: ${err.message}\n`);
        process.stderr.write(`\nYou can manually install from: https://github.com/Prof-bilal/Warden/releases\n`);
        process.exit(1);
      }
    }
  } else {
    if (tty) printCheck(`Installing runtime  ${dim("(cached)")}`);
    else { step++; printStaticStep(step, stepsTotal, "Installing runtime (cached)", true); }
  }

  // Step 3: Installing CLI (chmod)
  if (platform !== "win32") {
    try { fs.chmodSync(binPath, 0o755); } catch {}
  }
  if (tty) printCheck("Installing CLI");
  else { step++; printStaticStep(step, stepsTotal, "Installing CLI", true); }

  // Step 4: Verifying installation
  if (tty) {
    const stop = startSpinner("Verifying installation");
    await new Promise(r => setTimeout(r, 80));
    let verified = fs.existsSync(binPath);
    if (verified) {
      try { fs.accessSync(binPath, fs.constants.X_OK); } catch { verified = platform === "win32"; }
    }
    stop(verified, "Verifying installation");
    if (!verified) {
      process.stderr.write(`${useColor ? red(cross) : cross} Verification failed\n`);
      process.exit(1);
    }
  } else {
    step++; const ok = fs.existsSync(binPath);
    printStaticStep(step, stepsTotal, "Verifying installation", ok);
    if (!ok) process.exit(1);
  }

  // Success
  process.stderr.write("\n");
  process.stderr.write(dim("─────────────────────────────────────") + "\n\n");
  process.stderr.write(`${useColor ? green(check) : check} ${useColor ? bold(green(`Warden v${VERSION} installed successfully.`)) : `Warden v${VERSION} installed successfully.`}\n`);
  process.stderr.write("\n");
  process.stderr.write(bold("Get started:") + "\n\n");
  process.stderr.write(`  ${useColor ? cyan("warden init") : "warden init"}\n`);
  process.stderr.write(`  ${useColor ? cyan("warden run --policy policy.yaml -- <server>") : "warden run --policy policy.yaml -- <server>"}\n`);
  process.stderr.write(`  ${useColor ? cyan("warden doctor") : "warden doctor"}${useColor ? dim("  — check sandbox readiness") : "  — check sandbox readiness"}\n`);
  process.stderr.write("\n");
  process.stderr.write(useColor ? dim("Security:") : "Security:");
  process.stderr.write("\n");
  process.stderr.write((useColor ? dim("  Warden fails closed when sandboxing") : "  Warden fails closed when sandboxing") + "\n");
  process.stderr.write((useColor ? dim("  is unavailable.") : "  is unavailable.") + "\n");
  process.stderr.write("\n");
})();
