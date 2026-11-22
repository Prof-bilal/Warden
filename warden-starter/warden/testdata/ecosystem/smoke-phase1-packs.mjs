// Live Linux checks for the Memory and Git profiles using disposable data.
import assert from "node:assert/strict";
import { spawn, spawnSync } from "node:child_process";
import { once } from "node:events";
import { mkdir, mkdtemp, readFile, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join, resolve } from "node:path";
import { createInterface } from "node:readline";

const warden = resolve(process.env.WARDEN_BIN ?? "./warden");
const work = await mkdtemp(join(tmpdir(), "warden-pack-review-"));
const env = { PATH: process.env.PATH, HOME: work, XDG_CACHE_HOME: join(work, "cache"), XDG_STATE_HOME: join(work, "state"), WARDEN_NO_FIRST_RUN: "1" };
function run(command, args) {
  const result = spawnSync(command, args, { env, cwd: work, encoding: "utf8", timeout: 120000 });
  assert.equal(result.status, 0, result.error?.message || result.stderr);
  return result.stdout;
}

async function session(policy, argv, extraEnv, check) {
  const child = spawn(warden, ["run", "--backend", "linux", "--policy", policy, "--", ...argv], {
    cwd: work, env: { ...env, ...extraEnv }, stdio: ["pipe", "pipe", "pipe"],
  });
  const closed = once(child, "close");
  let stderr = "";
  child.stderr.on("data", bytes => { stderr = (stderr + bytes).slice(-8000); });
  const lines = createInterface({ input: child.stdout })[Symbol.asyncIterator]();
  let id = 0;
  async function request(method, params) {
    child.stdin.write(JSON.stringify({ jsonrpc: "2.0", id: ++id, method, params }) + "\n");
    let timer;
    try {
      const line = await Promise.race([lines.next(), new Promise((_, reject) => {
        timer = setTimeout(() => reject(Error(method + " timed out: " + stderr)), 15000);
      })]);
      assert(!line.done, stderr);
      const msg = JSON.parse(line.value);
      assert.equal(msg.id, id);
      return msg;
    } finally {
      clearTimeout(timer);
    }
  }
  try {
    const init = await request("initialize", { protocolVersion: "2025-11-25", capabilities: {}, clientInfo: { name: "warden-pack-review", version: "1" } });
    assert(init.result, JSON.stringify(init));
    child.stdin.write('{"jsonrpc":"2.0","method":"notifications/initialized"}\n');
    await check((name, args) => request("tools/call", { name, arguments: args }));
    child.stdin.end();
    const [code] = await closed;
    assert.equal(code, 0, stderr);
  } finally {
    child.kill();
  }
}

const id = process.argv[2];
const policy = join(work, id + ".yaml");
const catalog = JSON.parse(await readFile(new URL("../../internal/packs/catalog.json", import.meta.url), "utf8"));
const pack = catalog.find(entry => entry.id === id);
const runtime = join(work, "runtime");
const data = join(work, "data");
const outside = join(work, "outside");
await Promise.all([mkdir(data), mkdir(outside)]);

if (id === "memory") {
  run(warden, ["packs", "prepare", id, "--output", runtime, "--backend", "linux"]);
  run(warden, ["packs", "generate", id, "--runtime", runtime, "--path", data, "--output", policy]);
  const entity = { name: "warden-fixture", entityType: "test", observations: ["scoped storage"] };
  await session(policy, [], { MEMORY_FILE_PATH: join(data, "memory.jsonl") }, async call => {
    const result = await call("create_entities", { entities: [entity] });
    assert(result.result && !result.result.isError, JSON.stringify(result));
  });
  assert.match(await readFile(join(data, "memory.jsonl"), "utf8"), /warden-fixture/);
  await session(policy, [], { MEMORY_FILE_PATH: join(outside, "memory.jsonl") }, async call => {
    const result = await call("create_entities", { entities: [entity] });
    assert(result.error || result.result?.isError, JSON.stringify(result));
  });
  await assert.rejects(readFile(join(outside, "memory.jsonl")), { code: "ENOENT" });
} else if (id === "git") {
  const python = process.env.WARDEN_REVIEW_PYTHON ?? "/usr/bin/python3";
  run("uv", ["pip", "install", "--python", python, "--target", runtime, "--only-binary", ":all:", `${pack.artifact}==${pack.version}`]);
  for (const repo of [data, outside]) run("git", ["init", "--quiet", repo]);
  await writeFile(join(data, "status-fixture.txt"), "untracked fixture\n");
  run(warden, ["packs", "generate", id, "--runtime", runtime, "--path", data, "--output", policy]);
  const launcher = [python, "-c", "import runpy,sys; sys.path.insert(0,sys.argv.pop(1)); runpy.run_module('mcp_server_git',run_name='__main__')", runtime, "--repository", data];
  await session(policy, launcher, {}, async call => {
    const status = await call("git_status", { repo_path: data });
    assert(status.result && !status.result.isError && JSON.stringify(status).includes("status-fixture.txt"), JSON.stringify(status));
    const write = await call("git_add", { repo_path: data, files: ["status-fixture.txt"] });
    assert(write.error || write.result?.isError, JSON.stringify(write));
    const readElsewhere = await call("git_status", { repo_path: outside });
    assert(readElsewhere.error || readElsewhere.result?.isError, JSON.stringify(readElsewhere));
  });
  assert.equal(run("git", ["-C", data, "diff", "--cached", "--name-only"]), "");
} else {
  throw Error("Expected git or memory profile");
}
console.log(JSON.stringify({ status: "pass", pack: id, version: pack.version, evidenceDirectory: work }));
