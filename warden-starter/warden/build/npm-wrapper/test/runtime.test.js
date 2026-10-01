"use strict";
const { test } = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const { createHash } = require("node:crypto");
const { parseVersion, compareVersions, platformAsset, resolveVersion, checkedURL, verify, ensureBinary } = require("../lib/runtime");
const asset = "warden-linux-amd64", version = "0.2.0-beta.1";
const binary = Buffer.from("trusted fixture bytes");
const sums = Buffer.from(`${createHash("sha256").update(binary).digest("hex")}  ${asset}\n`);
function directory(t) { const p = fs.mkdtempSync(path.join(os.tmpdir(), "warden-wrapper-test-")); t.after(() => fs.rmSync(p, { recursive: true, force: true })); return p; }

test("release versions and channels never confuse previews with stable", t => {
  for (const value of ["../release", "0.2.0/x", "0.2.0-beta.01", "v0.2.0", "0.02.0"]) assert.throws(() => parseVersion(value));
  assert.equal(compareVersions("0.2.0-beta.2", "0.2.0-beta.10"), -1);
  assert.equal(compareVersions("0.2.0-beta.1", "0.2.0"), -1);
  assert.equal(compareVersions("0.2.0-alpha.1", "0.2.0-beta.1"), -1);
  const root = directory(t);
  for (const v of ["0.1.17", "0.1.18", "9.0.0-beta.1"]) { fs.mkdirSync(path.join(root, v)); fs.writeFileSync(path.join(root, v, asset), binary); }
  assert.equal(resolveVersion(root, "0.1.17", asset), "0.1.18");
  assert.equal(resolveVersion(root, version, asset), version);
  assert.equal(resolveVersion(root, "0.1.17", asset, true), "0.1.17");
  assert.throws(() => platformAsset("win32", "arm64"));
});

test("downloads require trusted HTTPS endpoints without credential redirects", () => {
  for (const url of ["http://github.com/file", "https://evil.example/file", "https://token@github.com/file", "https://github.com:8443/file", "https://github.com/file#fragment"]) assert.throws(() => checkedURL(url));
  assert.equal(checkedURL("https://release-assets.githubusercontent.com/file?signature=fixture").protocol, "https:");
});

test("fresh download verifies before writing and works offline afterward", async t => {
  const root = directory(t), requests = [];
  const fetch = async url => { requests.push(url); return url.endsWith("SHA256SUMS") ? sums : binary; };
  const file = await ensureBinary(root, version, asset, fetch, () => {});
  assert.equal(requests.length, 2);
  assert.deepEqual(fs.readFileSync(file), binary);
  await ensureBinary(root, version, asset, async () => { throw Error("offline"); }, () => {});
  fs.writeFileSync(file, "tampered cached binary");
  await assert.rejects(ensureBinary(root, version, asset, fetch, () => {}), /mismatch/);
  assert.equal(requests.length, 2, "tamper cannot silently fall back or redownload");
});

test("bad checksum, missing entry and duplicate entries fail closed", async t => {
  assert.throws(() => verify(binary, Buffer.concat([sums, sums]), asset), /exactly one/);
  assert.throws(() => verify(binary, Buffer.from("invalid\n"), asset), /exactly one/);
  const root = directory(t);
  await assert.rejects(ensureBinary(root, version, asset, async url => url.endsWith("SHA256SUMS") ? sums : Buffer.from("bad"), () => {}), /mismatch/);
  assert.deepEqual(fs.readdirSync(path.join(root, version)), []);
});

test("interrupted downloads leave no executable or temporary files", async t => {
  const root = directory(t);
  await assert.rejects(ensureBinary(root, version, asset, async url => { if (url.endsWith("SHA256SUMS")) return sums; throw Error("interrupted"); }, () => {}), /interrupted/);
  assert.deepEqual(fs.readdirSync(path.join(root, version)), []);
});

test("legacy cache is authenticated before use and never trusted just for existing", async t => {
  const root = directory(t), dir = path.join(root, version);
  fs.mkdirSync(dir); fs.writeFileSync(path.join(dir, asset), binary);
  const requests = [];
  await ensureBinary(root, version, asset, async url => { requests.push(url); return sums; }, () => {});
  assert.equal(requests.length, 1); assert.match(requests[0], /SHA256SUMS$/);
});

test("failed cache writes leave no temporary files or executable", async t => {
  const root = directory(t), originalSync = fs.fsyncSync;
  fs.fsyncSync = () => { throw Error("disk write failed"); };
  try {
    await assert.rejects(ensureBinary(root, version, asset, async url => url.endsWith("SHA256SUMS") ? sums : binary, () => {}), /disk write failed/);
    assert.deepEqual(fs.readdirSync(path.join(root, version)), []);
  } finally { fs.fsyncSync = originalSync; }
});

test("symlinked cache executable and version directory are refused", { skip: process.platform === "win32" }, async t => {
  const root = directory(t), dir = path.join(root, version), target = path.join(root, "target");
  fs.mkdirSync(dir); fs.writeFileSync(target, binary); fs.symlinkSync(target, path.join(dir, asset));
  await assert.rejects(ensureBinary(root, version, asset, async () => sums, () => {}), /invalid cached/);
  fs.rmSync(dir, { recursive: true }); fs.mkdirSync(path.join(root, "other")); fs.symlinkSync(path.join(root, "other"), dir);
  await assert.rejects(ensureBinary(root, version, asset, async () => sums, () => {}), /symlink/);
});

test("concurrent first launches leave one complete verified cache", async t => {
  const root = directory(t), fetch = async url => { await new Promise(r => setTimeout(r, 5)); return url.endsWith("SHA256SUMS") ? sums : binary; };
  const files = await Promise.all(Array.from({ length: 4 }, () => ensureBinary(root, version, asset, fetch, () => {})));
  assert.equal(new Set(files).size, 1);
  assert.deepEqual(fs.readdirSync(path.join(root, version)).sort(), ["SHA256SUMS", asset].sort());
});
