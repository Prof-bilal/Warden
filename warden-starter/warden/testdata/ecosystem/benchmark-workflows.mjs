import assert from "node:assert/strict";
import { execFile, spawnSync } from "node:child_process";
import { createHash } from "node:crypto";
import { existsSync, mkdirSync, readFileSync, realpathSync, writeFileSync } from "node:fs";
import { once } from "node:events";
import { createServer } from "node:http";
import { homedir } from "node:os";
import { join } from "node:path";
import { promisify } from "node:util";
import { terminal } from "./benchmark-terminal.mjs";

const bin = process.env.WARDEN_BIN;
const native = ["--backend", process.env.WARDEN_BENCH_BACKEND ?? "auto"];
const workspace = join(homedir(), "project");
mkdirSync(workspace, { recursive: true });
process.chdir(workspace);
const digest = bytes => createHash("sha256").update(bytes).digest("hex");
const file = (name, bytes) => {
  const path = join(workspace, name);
  writeFileSync(path, bytes, { mode: 0o600, flag: "wx" });
  return path;
};
function run(command, args, code = 0, options = {}) {
  const result = spawnSync(command, args, { encoding: "utf8", timeout: 90000, windowsHide: true, ...options });
  assert.ifError(result.error);
  assert.equal(result.status, code, `${command} ${args[0]}: ${result.stderr}\n${result.stdout}`);
  return result;
}
const warden = (args, code, options) => run(bin, args, code, options);
const node = (args, code) => run(process.execPath, args, code);
function fixtureSource() {
  return file("policy.go", readFileSync(join(process.env.WARDEN_BENCH_SOURCE, "policy.go")));
}

const workflows = {
  proof() {
    let output;
    try {
      const hostEnv = { ...process.env };
      if (process.env.WARDEN_BENCH_HOST_HOME) {
        hostEnv.HOME = process.env.WARDEN_BENCH_HOST_HOME;
        hostEnv.USERPROFILE = process.env.WARDEN_BENCH_HOST_HOME;
      }
      output = warden(["doctor", "--prove"], undefined, { env: hostEnv }).stdout;
    } catch (error) {
      if (/sandbox backend unavailable|sandbox-exec is not installed/i.test(error.message)) {
        return { status: "unsupported", reason: error.message.split("\n")[0] };
      }
      throw error;
    }
    const checks = ["secret-file read", "write outside grant", "blocked network host", "environment sentinel"];
    for (const check of checks) assert(output.split(/\r?\n/).includes(`PASS ${check}`), output);
    return checks;
  },
  policy() {
    const source = fixtureSource();
    const wardenYAML = file("warden.yaml", "version: 2\nfilesystem: {read: ['./policy.go']}\n");
    const explicit = file("explicit.yaml", "version: 2\nfilesystem: {read: ['./policy.go']}\nlimits: {timeout_s: 30}\n");
    const read = 'process.stdout.write(require("node:fs").readFileSync(process.argv[1]))';
    const target = [process.execPath, "-e", read, source];
    let launch;
    try {
      launch = warden(["run", ...native, "--", ...target]);
    } catch (error) {
      if (/sandbox backend unavailable|sandbox-exec is not installed/i.test(error.message)) {
        return { status: "unsupported", reason: error.message.split("\n")[0] };
      }
      throw error;
    }
    assert.equal(digest(launch.stdout), digest(readFileSync(source)));
    const selected = warden(["run", "--dry-run", "--policy", explicit, "--", ...target]).stdout;
    assert.match(selected, new RegExp(digest(readFileSync(explicit))));
    const local = warden(["run", "--dry-run", "--", ...target]).stdout;
    assert.match(local, new RegExp(digest(readFileSync(wardenYAML))));
    const invalid = warden(["run", "--policy", explicit, "--policy-sha256", "0".repeat(64), "--", ...target], 2);
    assert.match(invalid.stderr, /policy changed/);
    assert.equal(invalid.stdout, "");
    warden(["validate", wardenYAML, explicit]);
    const wider = file("wider.yaml", "version: 2\nfilesystem: {read: ['./policy.go']}\nnetwork: {allow: ['docs.example.org']}\n");
    const diff = warden(["policy-diff", "--before", wardenYAML, "--after", wider, "--fail-on", "widen"], 3);
    assert.equal(JSON.parse(diff.stdout).review_required, true);
    assert.match(warden(["--version"]).stdout, /warden version/);
    assert.match(warden(["help", "run"]).stderr, /--dry-run/);
    return ["pinned source bytes read through native backend", "cwd and explicit digests", "changed digest refused", "permission expansion detected"];
  },
  client() {
    const conf = file("client.json", JSON.stringify({ mcpServers: { fixture: { command: process.execPath, args: ["--version"] } }, unrelated: { keep: true } }, null, 2) + "\n");
    const original = readFileSync(conf);
    const policy = file("client-policy.yaml", "version: 2\nfilesystem: {}\n");
    const common = ["--client", "generic", "--config", conf, "--server", "fixture"];
    warden(["wrap", ...common, ...native, "--policy", policy, "--dry-run"]);
    assert.deepEqual(readFileSync(conf), original);
    warden(["wrap", ...common, ...native, "--policy", policy, "--yes"]);
    const wrapped = readFileSync(conf);
    const decoded = JSON.parse(wrapped);
    assert.deepEqual(decoded.unrelated, { keep: true });
    assert(decoded.mcpServers.fixture.args.includes(digest(readFileSync(policy))));
    warden(["wrap", ...common, ...native, "--policy", policy, "--yes"]);
    assert.deepEqual(readFileSync(conf), wrapped);
    const inventory = warden(["inventory", "--client", "generic", "--config", conf]).stdout;
    assert.match(inventory, /fixture/);
    warden(["unwrap", ...common, "--yes"]);
    assert.deepEqual(readFileSync(conf), original);
    assert(JSON.parse(warden(["clients", "--json"]).stdout).some(client => client.ID === "generic"));
    const policies = ["--config", conf, "--policies", join(workspace, "policies")];
    warden(["gateway", "init", ...policies]);
    assert.match(warden(["gateway", "list", ...policies]).stdout, /fixture\s+stdio.*\[policy ok\]/);
    assert.match(warden(["gateway", "run", ...policies, ...native, "--server", "fixture"]).stdout, /^v\d/);
    const own = file("own.yaml", `version: 2\ncommand: [${JSON.stringify(process.execPath)}, "--version"]\nfilesystem: {}\n`);
    const entry = ["--client", "generic", "--config", conf, "--server", "own"];
    warden(["add", "own", "to", "generic", "--config", conf, ...native, "--policy", own, "--yes"]);
    assert(JSON.parse(readFileSync(conf)).mcpServers.own.args.includes(digest(readFileSync(own))));
    warden(["unwrap", ...entry, "--yes"]);
    assert.deepEqual(readFileSync(conf), original);
    return ["dry-run preserves settings bytes", "pinned idempotent wrap and exact undo", "inventory and adapter discovery", "registered gateway launcher starts", "explicit-policy add and exact undo"];
  },
  async trace() {
    const source = fixtureSource();
    const target = ["/usr/bin/cat", source];
    assert.equal(digest(warden(["trace", "--", ...target]).stdout), digest(readFileSync(source)));
    const starter = join(workspace, "starter.yaml");
    warden(["init", "--output", starter, "--", ...target]);
    assert.match(readFileSync(starter, "utf8"), /policy\.go/);
    warden(["validate", starter]);
    const learned = join(workspace, "learned.yaml");
    await terminal(bin, ["learn", "--output", learned, "--", ...target], [
      ["Approve one grant? [y/N] ", "y"], ["Type yes to approve [default: deny]: ", "yes"],
    ]);
    const report = JSON.parse(readFileSync(learned + ".report.json"));
    assert.deepEqual(report.approved_reads, [realpathSync(source)]);
    assert.deepEqual(report.requested_hosts, []);
    assert.equal(report.exit_code, 0);
    assert.equal(report.policy_sha256, digest(readFileSync(learned)));
    assert.equal(report.policy_source, "learned");
    assert.equal(digest(warden(["run", "--policy", learned, "--policy-sha256", report.policy_sha256]).stdout), digest(readFileSync(source)));
    const ungranted = file("grants.yaml", "version: 2\nfilesystem: {}\n");
    const grant = ["grant", "path", source, "--policy", ungranted, "--for", "30m"];
    const denied = spawnSync(bin, ["run", "--policy", ungranted, "--", ...target], { encoding: "utf8", timeout: 30000 });
    assert.equal(denied.status, 1, denied.stderr);
    assert.equal(denied.stdout, "");
    const grantPrompt = "Approve one grant? [y/N] ";
    const beforeGrant = readFileSync(ungranted);
    await terminal(bin, grant, [[grantPrompt, "n"]], { expectedExit: 1 });
    assert.deepEqual(readFileSync(ungranted), beforeGrant);
    await terminal(bin, grant, [[grantPrompt, "y"]]);
    assert.equal(digest(warden(["run", "--policy", ungranted, "--", ...target]).stdout), digest(readFileSync(source)));
    const failed = warden(["run", "unknown-benchmark-launcher"], 2);
    assert.match(failed.stderr, /policy/);
    const wizard = ["/usr/bin/printf", "%s", "WARDEN_WIZARD_RESULT"];
    const wizardPrompt = "Type yes to approve [default: deny]: ";
    await terminal(bin, ["run", "--", ...wizard], [[wizardPrompt, "no"]], { expectedExit: 2 });
    const approved = await terminal(bin, ["run", "--", ...wizard], [[wizardPrompt, "yes"]]);
    assert.match(approved, /WARDEN_WIZARD_RESULT/);
    const events = warden(["logs", "--tail", "1000", "--explain"]).stdout;
    assert.match(events, /policy\.go/);
    return ["trusted trace and starter policy", "TTY review and pinned learned policy", "timed grant declined then approved with real access", "unknown launcher refused", "wizard explicitly declined then approved"];
  },
  outputs() {
    const policy = file("manifest-policy.yaml", `version: 2\ncommand: [${JSON.stringify(bin)}, "version"]\nfilesystem: {}\n`);
    const manifest = warden(["k8s", "render", "--policy", policy, "--image", "warden-test:fixture"]).stdout;
    assert.match(manifest, /kind: Deployment/);
    assert.match(manifest, /warden-test:fixture/);
    const docker = warden(["k8s", "docker", "--policy", policy, "--image", "warden-test:fixture"]).stdout;
    assert.match(docker, /docker run/);
    const log = join(workspace, "pilot.jsonl");
    warden(["report", "record", "--file", log, "--client", "codex", "--event", "first-protected-task", "--seconds", "22"]);
    assert.equal(JSON.parse(warden(["report", "summary", "--file", log]).stdout).records, 1);
    return ["Kubernetes deployment and Docker command", "one end-to-end local report record"];
  },
  install() {
    const npm = join(workspace, "npm"), pip = join(workspace, "pip");
    warden(["guard", "npm", "install", "is-number@7.0.0", "--output", npm]);
    const pkg = join(npm, "node_modules", "is-number");
    assert.equal(JSON.parse(readFileSync(join(pkg, "package.json"))).version, "7.0.0");
    assert.equal(node(["-e", "const isNumber = require(process.argv[1]); if (!isNumber('42') || isNumber('no')) process.exit(1)", pkg]).status, 0);
    warden(["guard", "pip", "install", "idna==3.10", "--output", pip]);
    const python = process.platform === "win32" ? "python" : "python3";
    run(python, ["-I", "-c", "import sys; sys.path.insert(0, sys.argv[1]); import idna; assert idna.__version__ == '3.10'; assert idna.encode('bücher.example') == b'xn--bcher-kva.example'", pip]);
    assert(existsSync(join(pip, "idna-3.10.dist-info")));
    return ["is-number@7.0.0 installed and called", "idna==3.10 wheel installed and called"];
  },
  async workspace() {
    const source = fixtureSource();
    const secret = "WARDEN_BENCHMARK_SYNTHETIC";
    const envFile = file(".env", `PROTECTED=${secret}\n`);
    run("git", ["init", "--quiet", workspace]);
    const parent = realpathSync(homedir());
    const policy = join(parent, "agent.yaml");
    writeFileSync(policy, `filesystem: {write: [./project]}\nagent:\n  workspace: ./project\n  redaction:\n    apply: review\n    files: [{path: .env, format: dotenv, fields: [PROTECTED]}]\n`);
    const comment = "\n// Warden benchmark edit\n";
    const edit = "const fs = require('node:fs'); if (fs.readFileSync('.env','utf8').includes(process.argv[1])) process.exit(9); fs.appendFileSync('policy.go', process.argv[2])";
    const before = readFileSync(source, "utf8");
    const args = ["agent", "--policy", policy, "--", process.execPath, "-e", edit, secret, comment];
    const prompt = "Type apply (anything else discards).";
    await terminal(bin, args, [[prompt, "no"]]);
    assert.equal(readFileSync(source, "utf8"), before);
    await terminal(bin, args, [[prompt, "apply"]]);
    assert.equal(readFileSync(source, "utf8"), before + comment);
    assert.equal(readFileSync(envFile, "utf8"), `PROTECTED=${secret}\n`);
    return ["real Git workspace discarded and approved edit", "protected value redacted in projection and preserved on host"];
  },
  async docker() {
    const shared = join(workspace, "shared"), output = join(workspace, "output");
    mkdirSync(shared); mkdirSync(output);
    const allowed = join(shared, "allowed.txt");
    writeFileSync(allowed, "read control");
    const decoy = file("private.txt", "unchanged");
    const policy = file("docker.yaml", `filesystem: {read: [${JSON.stringify(shared)}], write: [${JSON.stringify(output)}]}\nenv: {allow: [WARDEN_BENCH_ALLOWED]}\n`);
    let receipts = 0;
    const listener = createServer((req, res) => { receipts++; res.end("positive control"); });
    listener.listen(0, "127.0.0.1");
    await once(listener, "listening");
    const url = `http://127.0.0.1:${listener.address().port}/`;
    const target = `
const assert = require('node:assert/strict'), fs = require('node:fs');
const [allowed, decoy, output, url] = process.argv.slice(1);
assert.equal(fs.readFileSync(allowed, 'utf8'), 'read control');
fs.writeFileSync(output, 'write control');
assert.equal(process.env.WARDEN_BENCH_ALLOWED, 'env control');
assert.equal(process.env.WARDEN_BENCH_DENIED, undefined);
assert.throws(() => fs.readFileSync(decoy), { code: 'ENOENT' });
assert.throws(() => fs.writeFileSync(decoy, 'changed'), error => ['ENOENT', 'EROFS', 'EACCES'].includes(error.code));
(async () => {
  await assert.rejects(fetch(url, { signal: AbortSignal.timeout(2000) }));
  console.log('DOCKER_PROBE_COMPLETED');
})().catch(err => { console.error(err); process.exitCode = 1; });`;
    try {
      assert.equal(await (await fetch(url)).text(), "positive control");
      const saved = join(output, "written.txt");
      const { stdout } = await promisify(execFile)(bin, ["run", "--backend", "docker", "--policy", policy,
        "--", process.execPath, "-e", target, allowed, decoy, saved, url], {
        env: { ...process.env, WARDEN_BENCH_ALLOWED: "env control", WARDEN_BENCH_DENIED: "synthetic" },
        timeout: 45000,
      });
      assert.match(stdout, /DOCKER_PROBE_COMPLETED/);
      assert.equal(readFileSync(saved, "utf8"), "write control");
      assert.equal(readFileSync(decoy, "utf8"), "unchanged");
      assert.equal(receipts, 1, "only the observer's positive HTTP control may reach the listener");
      return ["allowed file and env controls", "denied file read and write", "denied env canary", "zero sandbox traffic at the observer"];
    } finally {
      listener.closeAllConnections();
      await new Promise(resolveClose => listener.close(resolveClose));
    }
  },
};

const name = process.argv[2];
assert(Object.hasOwn(workflows, name), `unknown driver ${name}`);
const result = await workflows[name]();
console.log(JSON.stringify(result?.status === "unsupported"
  ? result
  : { status: "pass", checks: result }));
