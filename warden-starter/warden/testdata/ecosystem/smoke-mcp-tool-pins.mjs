// Run with Node and WARDEN_BIN set to a freshly built CLI on Linux.
import assert from "node:assert/strict";
import { spawn, spawnSync } from "node:child_process";
import { once } from "node:events";
import { copyFile, mkdir, mkdtemp, readFile, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join, resolve } from "node:path";
import { createInterface } from "node:readline";
import { terminal } from "./benchmark-terminal.mjs";

assert(process.env.WARDEN_BIN, "WARDEN_BIN is required");
const bin = resolve(process.env.WARDEN_BIN);
const root = await mkdtemp(join(tmpdir(), "warden-pin-smoke-"));
try {
  const data = join(root, "data");
  const state = join(root, "state");
  await Promise.all([mkdir(data), mkdir(state)]);
  const entry = join(data, "bridge-server.mjs");
  const allowed = join(data, "note.txt");
  const policy = join(state, "policy.yaml");
  const rules = join(state, "rules.json");
  const journal = join(state, "journal.jsonl");
  await copyFile(new URL("bridge-server.mjs", import.meta.url), entry);
  await writeFile(allowed, "fixture note");
  await writeFile(policy, `filesystem: { read: ${JSON.stringify([data])} }\nlimits: { timeout_s: 30 }\n`);
  await writeFile(rules, JSON.stringify({ version: 1, tools: { read: {} } }));
  const env = { PATH: process.env.PATH, HOME: state, XDG_CONFIG_HOME: state, XDG_STATE_HOME: state };

  async function session(description, check) {
    const args = ["connect", "--backend", "linux", "--rules", rules, "--policy", policy,
      "--journal", journal, "--pin-server", "fixture", "--", process.execPath, entry, description];
    const proc = spawn(bin, args, { env, stdio: ["pipe", "pipe", "pipe"] });
    const closed = once(proc, "close");
    const failSafe = setTimeout(() => proc.kill("SIGKILL"), 20_000);
    const lines = createInterface({ input: proc.stdout })[Symbol.asyncIterator]();
    let stderr = "";
    proc.stderr.on("data", (chunk) => { stderr += chunk; });
    let id = 0;
    const send = (message) => proc.stdin.write(JSON.stringify({ jsonrpc: "2.0", ...message }) + "\n");
    async function request(method, params) {
      send({ id: ++id, method, params });
      const line = await lines.next();
      assert(!line.done, `bridge closed during ${method}: ${stderr}`);
      const reply = JSON.parse(line.value);
      assert.equal(reply.id, id);
      return reply;
    }
    try {
      const init = await request("initialize", {
        protocolVersion: "2025-11-25", capabilities: {}, clientInfo: { name: "fixture", version: "1" },
      });
      assert.equal(init.error, undefined, `${JSON.stringify(init)}\n${stderr}`);
      send({ method: "notifications/initialized" });
      await check(request);
    } finally {
      proc.stdin.end();
      const [exit] = await closed;
      clearTimeout(failSafe);
      assert.equal(exit, 0, stderr);
    }
  }

  const call = { name: "read", arguments: { path: allowed } };
  const evidence = {};
  await session("Read a fixture note.", async (request) => {
    const list = await request("tools/list");
    assert.deepEqual(list.result.tools.map(({ name, description }) => [name, description]), [["read", "Read a fixture note."]]);
    assert.equal((await request("tools/call", call)).result.content[0].text, "fixture note");
    evidence.first = { tools: list.result.tools.map(({ name }) => name) };
  });
  await session("Read a note and return its length.", async (request) => {
    const list = await request("tools/list");
    assert.deepEqual(list.result.tools, []);
    const denied = await request("tools/call", call);
    assert.equal(denied.error.code, -32001);
    assert.match(denied.error.message, /definition changed/);
    evidence.second = { tools: list.result.tools, denial: denied.error };
  });
  assert.match(await readFile(journal, "utf8"), /tool definition differs from pinned hash/);
  const diff = spawnSync(bin, ["mcp", "pin", "--server", "fixture", "--diff", "read"], { env, encoding: "utf8" });
  assert.equal(diff.status, 0, diff.stderr);
  assert.match(diff.stdout, /Accepted SHA-256: [a-f0-9]{64}/);
  assert.match(diff.stdout, /Observed SHA-256: [a-f0-9]{64}/);
  await session("Read a note and return its length.", async (request) => {
    assert.deepEqual((await request("tools/list")).result.tools, []);
  });
  const accept = ["mcp", "pin", "--server", "fixture", "--accept", "read"];
  const prompt = "Type yes to approve [default: deny]: ";
  await terminal(bin, accept, [[prompt, "yes"]], { env });
  await session("Read a note and return its length.", async (request) => {
    const list = await request("tools/list");
    assert.deepEqual(list.result.tools.map(({ name }) => name), ["read"]);
    assert.equal((await request("tools/call", call)).result.content[0].text, "fixture note");
  });
  console.log(JSON.stringify({ status: "pass", checks: ["initial pin", "changed tool hidden and blocked",
    "change audited, diff is read-only", "explicit terminal acceptance restores the tool"],
    ...evidence, audit: "tool definition differs from pinned hash", diff: diff.stdout.trim() }, null, 2));
} finally {
  await rm(root, { recursive: true, force: true });
}
