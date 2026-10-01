"use strict";
const fs = require("node:fs");
const path = require("node:path");
const os = require("node:os");
const https = require("node:https");
const { createHash, randomBytes } = require("node:crypto");
const releaseBase = "https://github.com/Prof-bilal/Warden/releases/download/";
const maxBinary = 64 * 1024 * 1024, maxSums = 1024 * 1024;
const semver = /^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?$/;
function parseVersion(value) {
  const match = semver.exec(value);
  if (!match || (match[4] || "").split(".").some(v => /^\d+$/.test(v) && v.length > 1 && v[0] === "0")) throw Error("invalid release version");
  return { core: match.slice(1, 4).map(BigInt), pre: match[4] ? match[4].split(".") : [] };
}
function compareVersions(a, b) {
  const av = parseVersion(a), bv = parseVersion(b);
  for (let i = 0; i < 3; i++) if (av.core[i] !== bv.core[i]) return av.core[i] < bv.core[i] ? -1 : 1;
  if (!av.pre.length || !bv.pre.length) return av.pre.length === bv.pre.length ? 0 : av.pre.length ? -1 : 1;
  for (let i = 0; i < Math.max(av.pre.length, bv.pre.length); i++) {
    const x = av.pre[i], y = bv.pre[i];
    if (x === undefined || y === undefined) return x === undefined ? -1 : 1;
    if (x === y) continue;
    const xn = /^\d+$/.test(x), yn = /^\d+$/.test(y);
    if (xn && yn) return BigInt(x) < BigInt(y) ? -1 : 1;
    if (xn !== yn) return xn ? -1 : 1;
    return x < y ? -1 : 1;
  }
  return 0;
}
function platformAsset(platform = os.platform(), arch = os.arch()) {
  const names = { "linux:x64": "linux-amd64", "linux:arm64": "linux-arm64", "darwin:x64": "darwin-amd64", "darwin:arm64": "darwin-arm64", "win32:x64": "windows-amd64.exe" };
  const name = names[`${platform}:${arch}`];
  if (!name) throw Error(`unsupported platform ${platform}/${arch}`);
  return `warden-${name}`;
}
function cacheRoot() {
  return os.platform() === "win32" ? path.join(process.env.LOCALAPPDATA || path.join(os.homedir(), "AppData", "Local"), "warden") : path.join(os.homedir(), ".cache", "warden");
}
function resolveVersion(root, pinned, asset, exact = false) {
  // Preview installs are exact. Stable users never inherit a cached preview.
  if (exact || parseVersion(pinned).pre.length) return pinned;
  let selected = pinned;
  try {
    for (const entry of fs.readdirSync(root, { withFileTypes: true })) {
      if (!entry.isDirectory() || !semver.test(entry.name)) continue;
      try {
        if (parseVersion(entry.name).pre.length || compareVersions(entry.name, selected) < 0) continue;
        const info = fs.lstatSync(path.join(root, entry.name, asset));
        if (info.isFile() && !info.isSymbolicLink()) selected = entry.name;
      } catch { /* unrelated or incomplete cache entry */ }
    }
  } catch (error) { if (error.code !== "ENOENT") throw error; }
  return selected;
}
function checkedURL(value) {
  const url = new URL(value);
  const hosts = ["github.com", "release-assets.githubusercontent.com", "objects.githubusercontent.com", "github-releases.githubusercontent.com"];
  if (url.protocol !== "https:" || !hosts.includes(url.hostname) || url.username || url.password || (url.port && url.port !== "443") || url.hash) throw Error("release download must use a trusted HTTPS host");
  return url;
}
function download(value, limit, redirects = 0, deadline = Date.now() + 120000) {
  const url = checkedURL(value);
  if (redirects > 5 || Date.now() >= deadline) return Promise.reject(Error("release download redirect/time limit"));
  return new Promise((resolve, reject) => {
    const remaining = deadline - Date.now();
    const request = https.get(url, { timeout: Math.min(15000, remaining), headers: { "User-Agent": "warden-npm-wrapper", Accept: "application/octet-stream" } }, response => {
      if ([301, 302, 303, 307, 308].includes(response.statusCode) && response.headers.location) {
        response.resume();
        let next;
        try { next = checkedURL(new URL(response.headers.location, url).href); } catch (error) { reject(error); return; }
        download(next.href, limit, redirects + 1, deadline).then(resolve, reject);
        return;
      }
      if (response.statusCode !== 200) { response.resume(); reject(Error(`HTTP ${response.statusCode} fetching release asset`)); return; }
      if (Number(response.headers["content-length"]) > limit) { response.destroy(); reject(Error("release asset too large")); return; }
      let size = 0;
      const chunks = [];
      response.on("data", chunk => {
        size += chunk.length;
        if (size > limit) { response.destroy(Error("release asset too large")); return; }
        chunks.push(chunk);
      });
      response.on("error", reject);
      response.on("aborted", () => reject(Error("release download interrupted")));
      response.on("end", () => resolve(Buffer.concat(chunks)));
    });
    const timer = setTimeout(() => request.destroy(Error("release download timed out")), remaining);
    request.on("close", () => clearTimeout(timer));
    request.on("timeout", () => request.destroy(Error("release download stalled")));
    request.on("error", reject);
  });
}
function checksum(sums, asset) {
  if (sums.length > maxSums) throw Error("checksum file too large");
  const matches = [];
  for (const line of sums.toString("utf8").split(/\r?\n/)) {
    const match = /^([a-fA-F0-9]{64}) [ *]([^\s]+)$/.exec(line);
    if (match && match[2] === asset) matches.push(match[1].toLowerCase());
  }
  if (matches.length !== 1) throw Error("checksum file must contain exactly one matching asset");
  return matches[0];
}
function verify(binary, sums, asset) {
  if (!binary.length || binary.length > maxBinary) throw Error("invalid release binary size");
  if (createHash("sha256").update(binary).digest("hex") !== checksum(sums, asset)) throw Error("binary checksum mismatch; refusing to execute");
}
function readRegular(file, limit) {
  const info = fs.lstatSync(file);
  if (!info.isFile() || info.isSymbolicLink() || info.size > limit) throw Error("invalid cached release file");
  const fd = fs.openSync(file, fs.constants.O_RDONLY | (fs.constants.O_NOFOLLOW || 0));
  try {
    if (!fs.fstatSync(fd).isFile()) throw Error("invalid cached release file");
    const bytes = fs.readFileSync(fd);
    if (bytes.length > limit) throw Error("cached release file too large");
    return bytes;
  } finally { fs.closeSync(fd); }
}
function privateDirectory(dir) {
  fs.mkdirSync(dir, { recursive: true, mode: 0o700 });
  const info = fs.lstatSync(dir);
  if (!info.isDirectory() || info.isSymbolicLink()) throw Error("release cache directory symlink refused");
  if (os.platform() !== "win32") fs.chmodSync(dir, 0o700);
}
function atomicWrite(file, data, mode) {
  const temporary = `${file}.tmp-${randomBytes(12).toString("hex")}`;
  const fd = fs.openSync(temporary, "wx", mode);
  try {
    try { fs.writeFileSync(fd, data); fs.fsyncSync(fd); } finally { fs.closeSync(fd); }
    fs.renameSync(temporary, file);
  } finally { try { fs.unlinkSync(temporary); } catch (error) { if (error.code !== "ENOENT") throw error; } }
}
async function ensureBinary(root, version, asset, fetch = download, progress = message => console.error(message)) {
  parseVersion(version);
  if (!/^warden-(linux|darwin|windows)-(amd64|arm64)(?:\.exe)?$/.test(asset)) throw Error("invalid release asset");
  privateDirectory(root);
  const dir = path.join(root, version);
  privateDirectory(dir);
  const file = path.join(dir, asset), sumsFile = path.join(dir, "SHA256SUMS");
  const base = `${releaseBase}v${version}/`;
  let binary, sums;
  try { binary = readRegular(file, maxBinary); } catch (error) { if (error.code !== "ENOENT") throw error; }
  try { sums = readRegular(sumsFile, maxSums); } catch (error) { if (error.code !== "ENOENT") throw error; }
  if (!binary || !sums) {
    progress(`Verifying Warden v${version} release download…`);
    if (!sums) sums = await fetch(`${base}SHA256SUMS`, maxSums);
    if (!binary) binary = await fetch(`${base}${asset}`, maxBinary);
    verify(binary, sums, asset);
    atomicWrite(sumsFile, sums, 0o600);
    if (!fs.existsSync(file)) atomicWrite(file, binary, 0o700);
  }
  verify(readRegular(file, maxBinary), readRegular(sumsFile, maxSums), asset);
  if (os.platform() !== "win32") fs.chmodSync(file, 0o700);
  return file;
}
module.exports = { parseVersion, compareVersions, platformAsset, cacheRoot, resolveVersion, checkedURL, download, checksum, verify, ensureBinary };
