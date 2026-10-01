// OS-native gateway lifecycle regression. No npm server install or model call;
// uses the audited disposable Node fixture. Private files live outside /tmp
// because Seatbelt's existing runtime exposes host temporary directories.
import { spawn, spawnSync } from "node:child_process";
import { mkdtemp, mkdir, copyFile, writeFile, readFile, rm } from "node:fs/promises";
import { homedir } from "node:os";
import { join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import readline from "node:readline";
import assert from "node:assert/strict";
if (!process.env.WARDEN_BIN) throw Error("WARDEN_BIN required");
const bin = resolve(process.env.WARDEN_BIN), root = await mkdtemp(join(homedir(), ".warden-native-test-"));
const runtime = join(root, "runtime"), data = join(root, "data"), priv = join(root, "private");
for (const dir of [runtime, data, priv]) await mkdir(dir, { mode: 0o700 });
const fixture = join(runtime, "server.mjs"); await copyFile(fileURLToPath(new URL("bridge-server.mjs", import.meta.url)), fixture);
const allowed = join(data, "allowed.txt"), outside = join(root, "outside.txt"); await writeFile(allowed, "native-allowed-sentinel"); await writeFile(outside, "native-outside-sentinel");
const policy = join(priv, "policy.yaml"), rules = join(priv, "rules.json"), journal = join(priv, "journal.jsonl"), control = join(priv, "control.json");
await writeFile(policy, `filesystem:\n  read: ${JSON.stringify([runtime, data])}\nnetwork: {allow: []}\nenv: {allow: []}\nlimits: {memory_mb: 512, timeout_s: 60}\n`);
await writeFile(rules, JSON.stringify({ version: 1, tools: { read: {}, write: {} }, resources: [], prompts: [], timeout_seconds: 15 }));
const env = { ...process.env, HOME: priv, XDG_CONFIG_HOME: priv, XDG_STATE_HOME: priv, APPDATA: priv, LOCALAPPDATA: priv, WARDEN_NO_UPDATE_CHECK: "1", WARDEN_GATEWAY_TOKEN: "native-test-" + "x".repeat(48) };
const args = ["--policy", policy, "--rules", rules, "--journal", journal, "--", process.execPath, fixture];
const init = { protocolVersion: "2025-11-25", clientInfo: { name: "native-test", version: "1" }, capabilities: {} };
const children = [];
try {
  const stdio = spawn(bin, ["connect", ...args], { env, stdio: ["pipe", "pipe", "pipe"] }); children.push(stdio); let errors = "", next = 0; stdio.stderr.on("data", b => errors += b);
  const pending = new Map(); readline.createInterface({ input: stdio.stdout }).on("line", line => { const m = JSON.parse(line); pending.get(m.id)?.(m); pending.delete(m.id); });
  const rpc = (method, params) => new Promise((done, reject) => { const id = ++next, timer = setTimeout(() => reject(Error(`native RPC timeout: ${errors}`)), 30000); pending.set(id, m => { clearTimeout(timer); done(m); }); stdio.stdin.write(JSON.stringify({ jsonrpc: "2.0", id, method, params }) + "\n"); });
  assert((await rpc("initialize", init)).result); stdio.stdin.write(JSON.stringify({ jsonrpc: "2.0", method: "notifications/initialized" }) + "\n");
  const read = await rpc("tools/call", { name: "read", arguments: { path: allowed } }); assert.match(JSON.stringify(read.result), /native-allowed-sentinel/);
  assert.equal((await rpc("tools/call", { name: "write", arguments: { path: allowed, content: "overwrite" } })).result.isError, true);
  assert.equal((await rpc("tools/call", { name: "read", arguments: { path: outside } })).result.isError, true);
  assert.equal((await rpc("tools/call", { name: "erase", arguments: { path: allowed } })).error.code, -32001);
  const end = new Promise(r => stdio.once("exit", r)); stdio.stdin.end(); await Promise.race([end, new Promise((_, reject) => setTimeout(() => reject(Error("native stdio cleanup timeout")), 10000).unref())]);
  const http = spawn(bin, ["serve", "--listen", "127.0.0.1:0", "--token-env", "WARDEN_GATEWAY_TOKEN", "--control-file", control, ...args], { env, stdio: ["ignore", "ignore", "pipe"] }); children.push(http); errors = ""; http.stderr.on("data", b => errors += b);
  let endpoint;
  for (let i = 0; i < 400; i++) { try { endpoint = new URL("/mcp", JSON.parse(await readFile(control, "utf8")).url); break; } catch { await new Promise(r => setTimeout(r, 25)); } }
  assert(endpoint, errors);
  const headers = { Authorization: `Bearer ${env.WARDEN_GATEWAY_TOKEN}`, Accept: "application/json, text/event-stream", "Content-Type": "application/json", "MCP-Protocol-Version": "2025-11-25" };
  const response = await fetch(endpoint, { method: "POST", headers, body: JSON.stringify({ jsonrpc: "2.0", id: 1, method: "initialize", params: init }) }); assert.equal(response.status, 200, await response.clone().text()); assert((await response.json()).result);
  const session = response.headers.get("mcp-session-id");
  assert.equal((await fetch(endpoint, { method: "DELETE", headers: { ...headers, "mcp-session-id": session } })).status, 204);
  const exited = new Promise(r => http.once("exit", r)); const stop = spawnSync(bin, ["stop", "--control-file", control], { env, encoding: "utf8", timeout: 15000 }); assert.equal(stop.status, 0, stop.stderr);
  await Promise.race([exited, new Promise((_, reject) => setTimeout(() => reject(Error("native HTTP cleanup timeout")), 10000).unref())]);
  assert.equal(await readFile(allowed, "utf8"), "native-allowed-sentinel");
  console.log(JSON.stringify({ status: "pass", platform: process.platform, arch: process.arch, checks: ["native sandboxed stdio startup", "allowed read", "OS-denied write/outside read", "MCP-denied tool", "stdin EOF cleanup", "HTTP native session creation/deletion", "authenticated stop cleanup"], fixture: "disposable trusted Node fixture; not a GUI-host or real npm-server claim" }));
} finally {
  for (const child of children) if (child.exitCode === null) child.kill("SIGTERM");
  await rm(root, { recursive: true, force: true });
}
