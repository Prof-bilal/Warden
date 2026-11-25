// Check the npm-installed launcher against a locally staged native release asset.
import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { createHash } from "node:crypto";
import { cpSync, mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { createRequire } from "node:module";
import { tmpdir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

assert(process.env.WARDEN_BIN, "WARDEN_BIN must name the stamped native candidate");
const binary = resolve(process.env.WARDEN_BIN);
const wrapper = fileURLToPath(new URL("../../build/npm-wrapper", import.meta.url));
const require = createRequire(join(wrapper, "package.json"));
const { name, version } = require("./package.json");
const { platformAsset } = require("./lib/runtime.js");
const asset = platformAsset();
const windows = process.platform === "win32";
const root = mkdtempSync(join(tmpdir(), "warden native package "));
const home = join(root, "home");
const prefix = join(root, "npm");
const localAppData = join(home, "AppData", "Local");
const env = {
  ...process.env, HOME: home, USERPROFILE: home, LOCALAPPDATA: localAppData,
  npm_config_cache: join(root, "npm-cache"), npm_config_userconfig: join(root, "npmrc"),
  WARDEN_NO_UPDATE_CHECK: "1", WARDEN_NO_FIRST_RUN: "1", WARDEN_PIN_VERSION: "1",
};
function run(command, args, cwd = root) {
  const result = spawnSync(command, args, { cwd, env, encoding: "utf8", timeout: 60000 });
  assert.equal(result.status, 0, result.error?.message ?? result.stderr);
  return result.stdout.trim();
}
function npm(args, cwd) {
  if (!windows) return run("npm", args, cwd);
  return run(process.execPath, [join(dirname(process.execPath), "node_modules", "npm", "bin", "npm-cli.js"), ...args], cwd);
}

try {
  mkdirSync(home);
  writeFileSync(env.npm_config_userconfig, "");
  npm(["pack", "--pack-destination", root, "--offline"], wrapper);
  const tarball = join(root, `${name}-${version}.tgz`);
  npm(["install", "--global", "--prefix", prefix, "--ignore-scripts", "--offline", tarball]);

  const releaseDirectory = join(windows ? localAppData : join(home, ".cache"), "warden", version);
  mkdirSync(releaseDirectory, { recursive: true });
  const bytes = readFileSync(binary);
  const sha256 = createHash("sha256").update(bytes).digest("hex");
  cpSync(binary, join(releaseDirectory, asset));
  writeFileSync(join(releaseDirectory, "SHA256SUMS"), `${sha256}  ${asset}\n`);
  const direct = run(binary, ["--version"]);
  assert(direct.includes(version), `native binary did not report ${version}: ${direct}`);

  const installed = windows ? join(prefix, "node_modules", name, "bin", "warden") : join(prefix, "bin", "warden");
  if (windows) readFileSync(join(prefix, "warden.cmd"));
  assert.equal(run(process.execPath, [installed, "--version"]), direct);
  console.log(JSON.stringify({ asset, version, source: "local release binary and checksum", result: "pass" }));
} finally {
  rmSync(root, { recursive: true, force: true });
}
