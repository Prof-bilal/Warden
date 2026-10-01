// An actual installed Codex app-server, isolated local configuration, no model
// turn or production credentials. Uses mcpServer/tool/call per official docs:
// https://learn.chatgpt.com/docs/app-server
import { spawn, spawnSync } from "node:child_process";
import { mkdtemp, mkdir, writeFile, readFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { dirname, join } from "node:path";
import readline from "node:readline";
import assert from "node:assert/strict";

if (!process.env.WARDEN_SCENARIO_INPUTS) throw Error("WARDEN_SCENARIO_INPUTS required from smoke-npm-release output");
const input = JSON.parse(await readFile(process.env.WARDEN_SCENARIO_INPUTS, "utf8"));
const root = await mkdtemp(join(tmpdir(), "warden-codex-host-")), home = join(root, "isolated-codex-home");
await mkdir(home, { mode: 0o700 });
const config = join(home, "config.toml");
const args = ["connect", "--rules", input.rules, "--policy", input.policy, "--journal", join(input.privateDirectory, "codex-journal.jsonl"), "--backend", "linux", "--", process.execPath, input.upstream, input.project];
const quote = JSON.stringify;
await writeFile(config, `model = "local-test"\nmodel_provider = "offline_fixture"\napproval_policy = "never"\n\n[model_providers.offline_fixture]\nname = "No inference in this MCP test"\nbase_url = "http://127.0.0.1:1/v1"\nwire_api = "responses"\nrequires_openai_auth = false\n\n[mcp_servers.filesystem]\ncommand = ${quote(input.installed)}\nargs = ${quote(args)}\nstartup_timeout_sec = 30\ntool_timeout_sec = 30\n\n[mcp_servers.filesystem.env]\nHOME = ${quote(input.env.HOME)}\nXDG_CONFIG_HOME = ${quote(input.env.XDG_CONFIG_HOME)}\nXDG_STATE_HOME = ${quote(input.env.XDG_STATE_HOME)}\nWARDEN_NO_UPDATE_CHECK = "1"\n`, { mode: 0o600 });
const codex = process.env.WARDEN_CODEX_BIN || "codex";
const env = { PATH: `${dirname(process.execPath)}:${process.env.PATH || ""}`, HOME: root, CODEX_HOME: home, XDG_CONFIG_HOME: home, XDG_STATE_HOME: home, TERM: "dumb", WARDEN_NO_UPDATE_CHECK: "1" };
const version = spawnSync(codex, ["--version"], { encoding: "utf8", env }); assert.equal(version.status, 0, version.stderr);
const server = spawn(codex, ["app-server", "--stdio"], { cwd: root, env, stdio: ["pipe", "pipe", "pipe"] });
let stderr = "", next = 0; server.stderr.on("data", b => stderr = (stderr + b).slice(-16000));
const pending = new Map();
readline.createInterface({ input: server.stdout }).on("line", line => {
  let msg; try { msg = JSON.parse(line); } catch { return; }
  if (msg.id !== undefined && pending.has(msg.id)) { pending.get(msg.id)(msg); pending.delete(msg.id); }
});
const request = (method, params) => new Promise((done, reject) => {
  const id = ++next; const timer = setTimeout(() => { pending.delete(id); reject(Error(`Codex ${method} timeout: ${stderr}`)); }, 45000);
  pending.set(id, msg => { clearTimeout(timer); done(msg); });
  server.stdin.write(JSON.stringify({ id, method, params }) + "\n");
});
const success = async (method, params) => { const r = await request(method, params); assert(!r.error, JSON.stringify(r.error)); return r.result; };
try {
  await success("initialize", { clientInfo: { name: "warden_real_host_test", version: "1" }, capabilities: { experimentalApi: true } });
  server.stdin.write(JSON.stringify({ method: "initialized", params: {} }) + "\n");
  const started = await success("thread/start", { cwd: input.project, ephemeral: true, model: "local-test", modelProvider: "offline_fixture", approvalPolicy: "never", sandbox: "read-only" });
  const threadId = started.thread.id;
  let status;
  for (let i = 0; i < 30; i++) {
    const result = await success("mcpServerStatus/list", { threadId, serverName: "filesystem", detail: "toolsAndAuthOnly" });
    status = result.data.find(s => s.name === "filesystem");
    if (status && Object.keys(status.tools || {}).length) break;
    await new Promise(r => setTimeout(r, 100));
  }
  assert(status, "Codex did not discover configured server");
  const tools = Object.values(status.tools || {}).map(t => t.name);
  assert(tools.includes("read_text_file") && tools.includes("write_file"), `Codex discovery failed: ${JSON.stringify(status)}\n${stderr}`);
  assert.deepEqual(tools.sort(), ["read_text_file", "write_file"], "Codex discovered tools outside the reviewed rules");
  assert(!tools.includes("list_directory"), "upstream directory-listing tool became visible in real host");
  const call = (tool, args) => request("mcpServer/tool/call", { threadId, server: "filesystem", tool, arguments: args });
  const allowed = await call("read_text_file", { path: input.selected }); assert(!allowed.error, JSON.stringify(allowed)); assert(!allowed.result.isError); assert.match(JSON.stringify(allowed.result), /release-test-sentinel/);
  const write = await call("write_file", { path: input.selected, content: "overwrite" }); assert(write.error || write.result.isError);
  const outside = await call("read_text_file", { path: input.outside }); assert(outside.error || outside.result.isError);
  assert.match(await readFile(input.selected, "utf8"), /release-test-sentinel/);
  console.log(JSON.stringify({ status: "pass", host: version.stdout.trim(), backend: "linux", upstream: "@modelcontextprotocol/server-filesystem@2026.8.31", transport: "npm launcher + stdio", checks: ["actual app-server MCP startup/discovery", "allowed read", "OS-denied write", "OS-denied outside read", "discovery filtering", "no model turn or production credentials"], evidenceDirectory: root }));
} finally {
  server.stdin.end();
  if (server.exitCode === null) server.kill("SIGTERM");
}
