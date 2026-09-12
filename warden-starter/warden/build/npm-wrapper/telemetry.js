#!/usr/bin/env node
"use strict";

// Anonymous installation telemetry for warden-sandbox-cli.
//
// Sends a single minimal POST on npm install so maintainers can estimate
// real installation activity beyond the npm download count.
//
// Design contract (do not weaken):
// - NEVER fails the install: all errors/timeouts are swallowed silently.
// - NEVER prints anything: npm output stays clean.
// - NEVER sends anything identifying: only package name, version, platform,
//   arch, and Node version. No paths, no env vars, no source, no MCP data.
// - Runs with a hard timeout so a dead/slow endpoint cannot stall installs.
// - No dependencies: uses only Node built-ins (Node >= 18 per engines).
// - Opt-outs (any one disables): WARDEN_NO_TELEMETRY=1, DO_NOT_TRACK=1,
//   WARDEN_TELEMETRY=0, or npm --ignore-scripts (skips postinstall entirely).
// - Endpoint is one constant, overridable via WARDEN_TELEMETRY_URL.

const TELEMETRY_URL =
  process.env.WARDEN_TELEMETRY_URL ||
  "https://warden-six-rouge.vercel.app/api/install";

const TIMEOUT_MS = 3000; // hard cap: never stall an install on telemetry

function optedOut() {
  const env = process.env;
  if (env.WARDEN_NO_TELEMETRY === "1") return true;
  if (env.DO_NOT_TRACK === "1") return true;
  if (env.WARDEN_TELEMETRY === "0") return true;
  return false;
}

function payload() {
  return {
    package: process.env.npm_package_name || "warden-sandbox-cli",
    version: process.env.npm_package_version || "unknown",
    platform: process.platform,
    arch: process.arch,
    node: process.version,
  };
}

// Fire-and-forget: exit is never blocked. The parent (postinstall hook)
// does not await this script's network result — the script itself exits 0
// immediately after scheduling the request.
async function main() {
  if (optedOut()) return;
  let url;
  try {
    url = new URL(TELEMETRY_URL); // validate scheme/host once
    if (url.protocol !== "https:") return;
  } catch {
    return;
  }
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), TIMEOUT_MS);
  // NOTE: no unref here. In Node >= 22 a pending unref'd timer lets the
  // process exit before the fetch resolves, so the ping would be dropped.
  // The 3s cap bounds the worst case; telemetry must be *sent*, then exit.
  try {
    await fetch(url.toString(), {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload()),
      signal: controller.signal,
    });
  } catch {
    // Swallowed by design: telemetry must never fail an install.
  } finally {
    clearTimeout(timer);
  }
}

main().then(
  () => process.exit(0),
  () => process.exit(0) // even an unexpected throw exits 0
);
