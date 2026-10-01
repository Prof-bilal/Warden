// A real pinned upstream task through a simulated stdio host. This does not
// certify a GUI client's lifecycle or any other profile/platform combination.
import { spawn, spawnSync } from "node:child_process";
import { once } from "node:events";
import { mkdtemp, mkdir, writeFile, readFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join, resolve } from "node:path";
import { createInterface } from "node:readline";
import assert from "node:assert/strict";

const warden = resolve(process.env.WARDEN_BIN || "./warden");
const runtime = resolve(process.env.WARDEN_SMOKE_RUNTIME || "");
if (!process.env.WARDEN_SMOKE_RUNTIME) throw new Error("Set WARDEN_SMOKE_RUNTIME to a prepared installation of @modelcontextprotocol/server-filesystem@2026.8.31");
const upstream = join(runtime, "node_modules/@modelcontextprotocol/server-filesystem/dist/index.js");
const pkg = JSON.parse(await readFile(join(runtime, "node_modules/@modelcontextprotocol/server-filesystem/package.json"), "utf8"));
assert.equal(pkg.version, "2026.8.31", "smoke test requires the recorded upstream version");
const work = await mkdtemp(join(tmpdir(), "warden-ecosystem-smoke-"));
const data = join(work, "data");
await mkdir(data);
await writeFile(join(data, "allowed.txt"), "warden-allowed-fixture\n");
await writeFile(join(work, "outside.txt"), "warden-outside-fixture\n");
const policy = join(work, "filesystem.yaml");
const config = join(work, "mcp.json");
const original = JSON.stringify({ mcpServers: { filesystem: { command: process.execPath, args: [upstream, data], env: {}, alwaysAllow: [] } }, unrelated: "preserved" }, null, 2);
await writeFile(config, original, { mode: 0o600 });
const env = { ...process.env, XDG_STATE_HOME: join(work, "state"), WARDEN_SETUP_STATE_DIR: join(work,"private-setup"), WARDEN_NO_FIRST_RUN: "1" };
function cli(args) {
  const result = spawnSync(warden, args, { env, encoding: "utf8", timeout: 30000 });
  if (result.status !== 0) throw new Error(`CLI failed (${result.status}): ${result.stderr}`);
  return result;
}
cli(["packs", "generate", "filesystem", "--runtime", runtime, "--path", data, "--output", policy]);
const base = ["--client", "cursor", "--config", config, "--server", "filesystem", "--policy", policy, "--backend", "linux", "--use-policy-command"];
cli(["wrap", ...base, "--dry-run"]);
assert.equal(await readFile(config, "utf8"), original, "preview must not write");
cli(["wrap", ...base, "--yes"]);
const wrapped = await readFile(config, "utf8");
cli(["wrap", ...base, "--yes"]);
assert.equal(await readFile(config, "utf8"), wrapped, "second wrap must be idempotent");
const launcher = JSON.parse(wrapped).mcpServers.filesystem;
const child = spawn(launcher.command, launcher.args, { env, stdio: ["pipe", "pipe", "pipe"] });
const exited = once(child, "exit");
let stderr = "";
child.stderr.on("data", (b) => { stderr = (stderr + b).slice(-8000); });
const pending = new Map();
createInterface({ input: child.stdout }).on("line", (line) => {
  let message;
  try { message = JSON.parse(line); } catch { return; }
  const request = pending.get(message.id);
  if (request) { pending.delete(message.id); request.resolve(message); }
});
let nextId = 1;
function request(method, params) {
  const id = nextId++;
  return new Promise((resolveRequest, reject) => {
    const timer = setTimeout(() => { pending.delete(id); reject(new Error(`MCP request timed out: ${method}\n${stderr}`)); }, 15000);
    pending.set(id, { resolve: message => { clearTimeout(timer); resolveRequest(message); } });
    child.stdin.write(JSON.stringify({ jsonrpc: "2.0", id, method, params }) + "\n");
  });
}
try {
  const init = await request("initialize", { protocolVersion: "2025-11-25", capabilities: {}, clientInfo: { name: "warden-stdio-smoke", version: "1" } });
  assert.ok(init.result, "initialize must complete");
  child.stdin.write(JSON.stringify({ jsonrpc: "2.0", method: "notifications/initialized" }) + "\n");
  const discovery = await request("tools/list", {});
  const names = discovery.result.tools.map(t => t.name);
  assert.ok(names.includes("read_text_file") && names.includes("write_file"), "expected filesystem tools must be discoverable");
  const read = await request("tools/call", { name: "read_text_file", arguments: { path: join(data, "allowed.txt") } });
  assert.ok(!read.result.isError && JSON.stringify(read.result).includes("warden-allowed-fixture"), "allowed read must succeed");
  const write = await request("tools/call", { name: "write_file", arguments: { path: join(data, "forbidden-write.txt"), content: "blocked" } });
  assert.ok(write.result?.isError || write.error, "read-only mount must block a write within the server's allowed directory");
  const outside = await request("tools/call", { name: "read_text_file", arguments: { path: join(work, "outside.txt") } });
  assert.ok(outside.result?.isError || outside.error, "out-of-profile path must be denied");
  child.stdin.end();
  await Promise.race([exited, new Promise((_, reject) => setTimeout(() => reject(new Error("server did not exit after stdin closed")), 10000))]);
  cli(["unwrap", "--client", "cursor", "--config", config, "--server", "filesystem", "--yes"]);
  assert.equal(await readFile(config, "utf8"), original, "undo must restore original configuration exactly");
  console.log(JSON.stringify({ status: "pass", upstream: `${pkg.name}@${pkg.version}`, backend: "linux", host: "simulated stdio host (Cursor config shape)", checks: ["initialize", "tools/list", "allowed read", "denied write", "denied outside path", "preview", "apply", "second wrap", "undo"], evidenceDirectory: work }));
} finally {
  child.stdin.end();
  if (child.exitCode === null) child.kill("SIGTERM");
}
