// Real npm pack/install and real filesystem MCP workflows. Release downloads
// use a disposable HTTPS fixture because the candidate is not published yet.
// No production URL override is added to the shipped launcher.
import { spawn, spawnSync } from "node:child_process";
import { mkdtemp, mkdir, writeFile, readFile, rm, stat, cp, rename } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join, resolve, dirname } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";
import { createHash } from "node:crypto";
import https from "node:https";
import assert from "node:assert/strict";

const binaryPath = resolve(process.env.WARDEN_BIN || "");
const runtime = resolve(process.env.WARDEN_SMOKE_RUNTIME || "");
if (!process.env.WARDEN_BIN || !process.env.WARDEN_SMOKE_RUNTIME) throw Error("WARDEN_BIN (stamped candidate) and WARDEN_SMOKE_RUNTIME (prepared filesystem) required");
if (process.platform !== "linux") throw Error("Real filesystem release scenario currently requires native Linux");
const wrapper = resolve(dirname(fileURLToPath(import.meta.url)), "../../build/npm-wrapper");
const metadata = JSON.parse(await readFile(join(wrapper, "package.json"), "utf8"));
const root = await mkdtemp(join(tmpdir(), "warden-npm-scenarios-"));
const priv = join(root, "private"), home = join(priv, "home"), project = join(root, "project"), data = join(project, "docs"), prefix = join(priv, "npm-prefix");
for (const dir of [priv, home, project, data, prefix]) await mkdir(dir, { recursive: true, mode: 0o700 });
const env = { ...process.env, HOME: home, XDG_CONFIG_HOME: join(priv, "config"), XDG_STATE_HOME: join(priv, "state"), WARDEN_SETUP_STATE_DIR: join(priv, "setup"), WARDEN_NO_UPDATE_CHECK: "1", WARDEN_NO_FIRST_RUN: "1", npm_config_cache: join(priv, "npm-cache"), npm_config_userconfig: join(priv, "npmrc"), npm_config_audit: "false", npm_config_fund: "false" };
await writeFile(env.npm_config_userconfig, "");
const bytes = await readFile(binaryPath), asset = "warden-linux-amd64";
const hash = createHash("sha256").update(bytes).digest("hex");
const run = (cmd, args, extra = {}) => new Promise((done, reject) => {
  const child = spawn(cmd, args, { env, cwd: priv, ...extra }); let stdout = "", stderr = "";
  child.stdout?.on("data", b => stdout += b); child.stderr?.on("data", b => stderr += b);
  const timer = setTimeout(() => { child.kill("SIGKILL"); reject(Error(`timeout running ${cmd}`)); }, 90000);
  child.on("error", reject); child.on("exit", code => { clearTimeout(timer); done({ code, stdout, stderr }); });
});
const success = async (cmd, args, extra) => { const r = await run(cmd, args, extra); assert.equal(r.code, 0, r.stderr); return r; };
const checks = [];
const keyFile = join(priv, "tls.key"), certFile = join(priv, "tls.crt");
await success("openssl", ["req", "-x509", "-newkey", "rsa:2048", "-nodes", "-days", "1", "-subj", "/CN=localhost", "-addext", "subjectAltName=IP:127.0.0.1,DNS:localhost", "-keyout", keyFile, "-out", certFile]);
let mode = "good", requests = 0;
const server = https.createServer({ key: await readFile(keyFile), cert: await readFile(certFile) }, (req, res) => {
  requests++;
  if (mode === "offline") { res.writeHead(503); res.end(); return; }
  if (mode === "redirect") { res.writeHead(302, { Location: "https://untrusted.invalid/asset" }); res.end(); return; }
  const name = req.url.split("/").at(-1);
  if (name === "SHA256SUMS") { res.end(`${mode === "checksum" ? "0".repeat(64) : hash}  ${asset}\n`); return; }
  if (name !== asset) { res.writeHead(404); res.end(); return; }
  res.writeHead(200, { "Content-Length": bytes.length });
  if (mode === "truncated") { res.write(bytes.subarray(0, 256)); setTimeout(() => res.destroy(), 10); return; }
  res.end(bytes);
});
await new Promise(r => server.listen(0, "127.0.0.1", r));
const fixtureURL = `https://127.0.0.1:${server.address().port}`;
const preload = join(priv, "fixture-transport.cjs");
await writeFile(preload, `const https = require('node:https'); const get = https.get; https.get = function(url, options, callback) { const original = new URL(url); if (original.hostname === 'github.com' && original.pathname.startsWith('/Prof-bilal/Warden/releases/download/')) url = new URL(original.pathname + original.search, ${JSON.stringify(fixtureURL)}); return get.call(this, url, options, callback); };\n`);
env.NODE_OPTIONS = `--require=${preload}`; env.NODE_EXTRA_CA_CERTS = certFile;
const sdk = join(runtime, "node_modules/@modelcontextprotocol/sdk");
const { Client } = await import(pathToFileURL(join(sdk, "dist/esm/client/index.js")));
const { StdioClientTransport } = await import(pathToFileURL(join(sdk, "dist/esm/client/stdio.js")));
const { StreamableHTTPClientTransport } = await import(pathToFileURL(join(sdk, "dist/esm/client/streamableHttp.js")));
const installed = join(prefix, "bin/warden"), cache = join(home, ".cache/warden", metadata.version), cacheFile = join(cache, asset);
try {
  const pack = await success("npm", ["pack", wrapper, "--pack-destination", priv, "--json"]);
  const packed = JSON.parse(pack.stdout); assert(packed[0].files.some(f => f.path === "lib/runtime.js"));
  const tarball = join(priv, packed[0].filename);
  await success("npm", ["install", "--global", "--prefix", prefix, "--ignore-scripts", "--offline", tarball]);
  const version = await success(installed, ["--version"]); assert.match(version.stdout, new RegExp(metadata.version.replaceAll(".", "\\.")));
  assert(!version.stdout.includes("Verifying")); assert.match(version.stderr, /Verifying/);
  assert.equal(createHash("sha256").update(await readFile(cacheFile)).digest("hex"), hash);
  checks.push("real npm tarball, isolated global install, verified first download, clean stdout");
  mode = "offline"; const count = requests;
  await success(installed, ["--version"]); assert.equal(requests, count); checks.push("verified offline cached launch");
  // Stable cached binaries cannot override an explicitly installed preview.
  const stable = join(home, ".cache/warden/9.0.0"); await mkdir(stable); await writeFile(join(stable, asset), "untrusted higher stable cache");
  await success(installed, ["--version"]); checks.push("preview channel remains pinned despite higher cached stable");
  await writeFile(cacheFile + ".replacement", "tampered"); await rename(cacheFile + ".replacement", cacheFile);
  const tamper = await run(installed, ["--version"]);
  assert.notEqual(tamper.code, 0); assert.match(tamper.stderr, /checksum mismatch/); assert.equal(tamper.stdout, "");
  checks.push("tampered cached executable refused without fallback");
  await rm(cache, { recursive: true });
  for (const failure of ["checksum", "truncated", "redirect"]) {
    mode = failure; const result = await run(installed, ["--version"]); assert.notEqual(result.code, 0, failure); assert.equal(result.stdout, "");
    await assert.rejects(stat(cacheFile)); await rm(cache, { recursive: true, force: true });
  }
  checks.push("bad checksum, truncated HTTPS download and untrusted redirect refused"); mode = "good";
  // Upgrade a packed older wrapper without touching the user's npm prefix.
  const oldSource = join(priv, "old-package"); await cp(wrapper, oldSource, { recursive: true });
  const old = { ...metadata, version: "0.1.17" }; await writeFile(join(oldSource, "package.json"), JSON.stringify(old));
  const oldPack = JSON.parse((await success("npm", ["pack", oldSource, "--pack-destination", priv, "--json"])).stdout)[0];
  await success("npm", ["install", "--global", "--prefix", prefix, "--ignore-scripts", "--offline", join(priv, oldPack.filename)]);
  await success("npm", ["install", "--global", "--prefix", prefix, "--ignore-scripts", "--offline", tarball]);
  await success(installed, ["--version"]); checks.push("npm package upgrade from a packed legacy-version fixture to candidate");
  const selected = join(data, "README.txt"), outside = join(project, "outside.txt"); await writeFile(selected, "project documentation: release-test-sentinel"); await writeFile(outside, "outside-sentinel");
  const policy = join(priv, "filesystem.yaml"), rules = join(priv, "rules.json"), control = join(priv, "control.json"), journal = join(priv, "journal.jsonl");
  await success(installed, ["packs", "generate", "filesystem", "--runtime", runtime, "--path", data, "--output", policy]);
  const upstream = join(runtime, "node_modules/@modelcontextprotocol/server-filesystem/dist/index.js");
  // Let the upstream validate the parent project, so Warden (rather than only
  // the upstream's own narrower roots) must reject outside reads and writes.
  await writeFile(rules, JSON.stringify({ version: 1, tools: { read_text_file: {}, write_file: {} }, resources: [], prompts: [], timeout_seconds: 30 }));
  const args = ["--policy", policy, "--rules", rules, "--backend", "linux", "--journal", journal, "--", process.execPath, upstream, project];
  const workflow = async client => {
    const names = (await client.listTools()).tools.map(t => t.name); assert.deepEqual(names.sort(), ["read_text_file", "write_file"]);
    const read = await client.callTool({ name: "read_text_file", arguments: { path: selected } }); assert(!read.isError); assert.match(JSON.stringify(read), /release-test-sentinel/);
    const write = await client.callTool({ name: "write_file", arguments: { path: selected, content: "overwrite" } }); assert.equal(write.isError, true);
    const blocked = await client.callTool({ name: "read_text_file", arguments: { path: outside } }); assert.equal(blocked.isError, true);
    await assert.rejects(client.callTool({ name: "list_directory", arguments: { path: data } })); assert.match(await readFile(selected, "utf8"), /release-test-sentinel/);
  };
  // Force a genuine first launch with piped MCP traffic already waiting.
  await rm(cache, { recursive: true });
  const stdio = new Client({ name: "warden-npm-real-upstream", version: "1" }, { capabilities: {} });
  const transport = new StdioClientTransport({ command: installed, args: ["connect", ...args], env, stderr: "pipe" });
  try { await stdio.connect(transport); await workflow(stdio); checks.push("first-run npm MCP stdio with official filesystem server: read, OS denials, hidden tool"); } finally { await stdio.close(); }
  env.WARDEN_GATEWAY_TOKEN = "disposable-test-token-" + "x".repeat(48);
  const gateway = spawn(installed, ["serve", "--listen", "127.0.0.1:0", "--token-env", "WARDEN_GATEWAY_TOKEN", "--control-file", control, ...args], { env, stdio: ["ignore", "ignore", "pipe"] });
  let errors = ""; gateway.stderr.on("data", b => errors += b);
  try {
    let endpoint;
    for (let i = 0; i < 200; i++) { try { endpoint = new URL("/mcp", JSON.parse(await readFile(control, "utf8")).url); break; } catch { await new Promise(r => setTimeout(r, 25)); } }
    assert(endpoint, errors);
    const client = new Client({ name: "warden-npm-http-real-upstream", version: "1" }, { capabilities: {} });
    try { await client.connect(new StreamableHTTPClientTransport(endpoint, { requestInit: { headers: { Authorization: `Bearer ${env.WARDEN_GATEWAY_TOKEN}` } } })); await workflow(client); } finally { await client.close(); }
    assert.equal((await fetch(endpoint, { method: "POST", body: "{}" })).status, 401);
    const exited = new Promise(r => gateway.once("exit", r));
    await success(installed, ["stop", "--control-file", control]);
    await Promise.race([exited, new Promise((_, reject) => setTimeout(() => reject(Error("gateway stop timeout")), 10000).unref())]);
    checks.push("npm HTTP gateway: real server allowed/blocked tasks, unauthenticated refusal, clean stop");
  } finally { if (gateway.exitCode === null) gateway.kill("SIGTERM"); }
  const config = join(priv, "cursor.json"); const original = JSON.stringify({ mcpServers: { filesystem: { command: process.execPath, args: [upstream, data], env: {}, alwaysAllow: [] } }, unrelated: "preserved" }); await writeFile(config, original);
  const setup = ["--client", "cursor", "--config", config, "--server", "filesystem", "--policy", policy, "--backend", "linux", "--use-policy-command"];
  await success(installed, ["wrap", ...setup, "--dry-run"]); assert.equal(await readFile(config, "utf8"), original);
  await success(installed, ["wrap", ...setup, "--yes"]); const wrapped = await readFile(config, "utf8"); await success(installed, ["wrap", ...setup, "--yes"]); assert.equal(await readFile(config, "utf8"), wrapped);
  const before = await readFile(policy); await writeFile(policy, Buffer.concat([before, Buffer.from("\n# changed reviewed bytes\n")]));
  const launcher = JSON.parse(wrapped).mcpServers.filesystem; const drift = await run(launcher.command, launcher.args); assert.notEqual(drift.code, 0); assert.match(drift.stderr, /policy changed/); await writeFile(policy, before);
  await success(installed, ["unwrap", "--client", "cursor", "--config", config, "--server", "filesystem", "--yes"]); assert.equal(await readFile(config, "utf8"), original);
  checks.push("npm preview/apply/idempotence/policy-drift refusal/exact rollback with disposable host config");
  await writeFile(join(priv, "scenario-inputs.json"), JSON.stringify({ installed, policy, rules, upstream, project, data, selected, outside, runtime, privateDirectory: priv, env: { HOME: home, XDG_CONFIG_HOME: env.XDG_CONFIG_HOME, XDG_STATE_HOME: env.XDG_STATE_HOME } }, null, 2));
  console.log(JSON.stringify({ status: "pass", version: metadata.version, backend: "linux", upstream: "@modelcontextprotocol/server-filesystem@2026.8.31", releaseOrigin: "local HTTPS fixture, candidate not published", checks, evidenceDirectory: root }));
} finally { await new Promise(r => server.close(r)); }
