// A real pinned upstream task through a simulated stdio host. This does not
// certify a GUI client's lifecycle or any other profile/platform combination.
import { spawn, spawnSync } from "node:child_process";
import { once } from "node:events";
import { mkdtemp, mkdir, writeFile, readFile } from "node:fs/promises";
import { homedir, tmpdir } from "node:os";
import { join, resolve } from "node:path";
import { createInterface } from "node:readline";
import assert from "node:assert/strict";

const backend = { linux: "linux", darwin: "seatbelt", win32: "windows" }[process.platform];
assert(backend, "requires a native Warden backend");
const warden = resolve(process.env.WARDEN_BIN || (process.platform === "win32" ? "./warden.exe" : "./warden"));
const zeroConfig = process.env.WARDEN_SMOKE_ZERO_CONFIG === "1";
assert(!zeroConfig || backend === "linux", "the no-flags smoke scenario requires Linux");
const artifact = "@modelcontextprotocol/server-filesystem";
const version = "2.3.0";
const work = await mkdtemp(join(backend === "linux" ? tmpdir() : homedir(), "warden-ecosystem-smoke-"));
const prepare = process.env.WARDEN_SMOKE_PREPARE;
assert(!prepare || prepare === "fixture" || prepare === "sandbox", "unknown fixture preparation mode");
const runtime = process.env.WARDEN_SMOKE_RUNTIME
  ? resolve(process.env.WARDEN_SMOKE_RUNTIME)
  : join(work, "runtime");
if (!zeroConfig && !prepare && !process.env.WARDEN_SMOKE_RUNTIME) {
  throw new Error(`Set WARDEN_SMOKE_RUNTIME to a prepared installation of ${artifact}@${version}`);
}
const upstream = join(runtime, "node_modules", artifact, "dist", "index.js");
const dataHome = join(work, "xdg");
const data = zeroConfig ? join(dataHome, "warden", "data", "filesystem") : join(work, "data");
await mkdir(data, { recursive: true });
await writeFile(join(data, "allowed.txt"), "warden-allowed-fixture\n");
await writeFile(join(data, ".env"), "warden-protected-fixture\n");
await writeFile(join(work, "outside.txt"), "warden-outside-fixture\n");
const policy = join(work, "filesystem.yaml");
const config = join(work, "mcp.json");
const original = JSON.stringify({ mcpServers: { filesystem: { command: process.execPath, args: [upstream, data], env: {}, alwaysAllow: [] } }, unrelated: "preserved" }, null, 2);
await writeFile(config, original, { mode: 0o600 });
const env = { ...process.env, HOME: work, USERPROFILE: work, XDG_CONFIG_HOME: join(work, "config"), XDG_DATA_HOME: dataHome, XDG_STATE_HOME: join(work, "state"), WARDEN_SETUP_STATE_DIR: join(work,"private-setup"), WARDEN_NO_FIRST_RUN: "1" };
function cli(args) {
  const result = spawnSync(warden, args, { env, encoding: "utf8", timeout: args[1] === "prepare" ? 180000 : 30000 });
  if (result.status !== 0) throw new Error(`CLI failed (${result.status}): ${result.stderr}`);
  return result;
}
const base = ["--client", "cursor", "--config", config, "--server", "filesystem", "--policy", policy, "--backend", backend, "--use-policy-command"];
let launcher = { command: warden, args: ["run", "npx", artifact] };
if (!zeroConfig) {
  if (prepare === "sandbox") {
    cli(["packs", "prepare", "filesystem", "--output", runtime, "--backend", backend]);
  } else if (prepare === "fixture") {
    // This stages a test dependency. The protocol session itself uses Warden.
    await mkdir(runtime, { recursive: true });
    const npm = process.platform === "win32" ? "npm.cmd" : "npm";
    const result = spawnSync(npm, ["install", "--ignore-scripts", "--no-audit", "--no-fund",
      "--save-exact", `${artifact}@${version}`], {
      cwd: runtime, stdio: ["ignore", "ignore", "inherit"], timeout: 180000,
      shell: process.platform === "win32",
    });
    assert.equal(result.status, 0, result.error?.message ?? "could not stage pinned Filesystem fixture");
  }
  const pkg = JSON.parse(await readFile(join(runtime, "node_modules", artifact, "package.json"), "utf8"));
  assert.equal(pkg.version, version, "smoke test requires the recorded upstream version");
  cli(["packs", "generate", "filesystem", "--runtime", runtime, "--path", data, "--output", policy]);
  cli(["wrap", ...base, "--dry-run"]);
  assert.equal(await readFile(config, "utf8"), original, "preview must not write");
  cli(["wrap", ...base, "--yes"]);
  const wrapped = await readFile(config, "utf8");
  cli(["wrap", ...base, "--yes"]);
  assert.equal(await readFile(config, "utf8"), wrapped, "second wrap must be idempotent");
  launcher = JSON.parse(wrapped).mcpServers.filesystem;
}
const child = spawn(launcher.command, launcher.args, { cwd: work, env, stdio: ["pipe", "pipe", "pipe"] });
const exited = once(child, "exit");
let stderr = "";
child.stderr.on("data", (b) => { stderr = (stderr + b).slice(-8000); });
const pending = new Map();
const otherOutput = [];
createInterface({ input: child.stdout }).on("line", (line) => {
  let message;
  try { message = JSON.parse(line); } catch { otherOutput.push(line); return; }
  const request = pending.get(message.id);
  if (request) { pending.delete(message.id); request.resolve(message); }
});
let nextId = 1;
function request(method, params) {
  const id = nextId++;
  return new Promise((resolveRequest, reject) => {
    const timer = setTimeout(() => { pending.delete(id); reject(new Error(`MCP request timed out: ${method}\n${stderr}`)); }, method === "initialize" && zeroConfig ? 90000 : 15000);
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
  assert.ok(write.result?.isError || write.error, "read-only policy must block a write within the server's allowed directory");
  await assert.rejects(readFile(join(data, "forbidden-write.txt")), { code: "ENOENT" });
  const outside = await request("tools/call", { name: "read_text_file", arguments: { path: join(work, "outside.txt") } });
  assert.ok(outside.result?.isError || outside.error, "out-of-profile path must be denied");
  const secret = await request("tools/call", { name: "read_text_file", arguments: { path: join(data, ".env") } });
  assert.ok(secret.result?.isError || secret.error, "protected file within the readable directory must be denied");
  child.stdin.end();
  await Promise.race([exited, new Promise((_, reject) => setTimeout(() => reject(new Error("server did not exit after stdin closed")), 10000))]);
  assert.deepEqual(otherOutput, [], "stdout must carry only MCP messages");
  if (zeroConfig) {
    const installed = join(dataHome, "warden", "runtimes", "filesystem", version, "node_modules", artifact, "package.json");
    assert.equal(JSON.parse(await readFile(installed, "utf8")).version, version);
  } else {
    cli(["unwrap", "--client", "cursor", "--config", config, "--server", "filesystem", "--yes"]);
    assert.equal(await readFile(config, "utf8"), original, "undo must restore original configuration exactly");
  }
  const checks = ["initialize", "tools/list", "allowed read", "denied write", "denied outside path", "denied protected file", "clean stdout"];
  checks.push(...(zeroConfig ? ["fresh install", "catalog version"] : ["preview", "apply", "second wrap", "undo"]));
  if (prepare === "sandbox") checks.push("sandboxed registry preparation");
  if (prepare === "fixture") checks.push("host-staged pinned fixture");
  console.log(JSON.stringify({ status: "pass", upstream: `${artifact}@${version}`, backend, checks, evidenceDirectory: work }));
} finally {
  child.stdin.end();
  if (child.exitCode === null) child.kill("SIGTERM");
}
