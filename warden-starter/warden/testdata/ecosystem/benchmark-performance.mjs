import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { copyFileSync, mkdirSync, readFileSync, readdirSync, realpathSync, writeFileSync } from "node:fs";
import { createRequire } from "node:module";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const load = createRequire(new URL("../../mcp-test/package.json", import.meta.url));
const { Client } = load("@modelcontextprotocol/sdk/client/index.js");
const { StdioClientTransport } = load("@modelcontextprotocol/sdk/client/stdio.js");
const digest = bytes => createHash("sha256").update(bytes).digest("hex");
const workflow = fileURLToPath(new URL("./benchmark-workflows.mjs", import.meta.url));

function summarize(raw) {
  const values = [...raw].sort((a, b) => a - b);
  const middle = Math.floor(values.length / 2);
  const median = values.length % 2 ? values[middle] : (values[middle - 1] + values[middle]) / 2;
  return { count: values.length, median, p95: values[Math.ceil(values.length * 0.95) - 1],
    min: values[0], max: values.at(-1) };
}

function processTreeRSS(root) {
  if (!root || process.platform !== "linux") return undefined;
  const rows = [];
  for (const entry of readdirSync("/proc")) {
    if (!/^\d+$/.test(entry)) continue;
    try {
      const status = readFileSync(`/proc/${entry}/status`, "utf8");
      rows.push({ pid: Number(entry), parent: Number(status.match(/^PPid:\s+(\d+)/m)?.[1]),
        rss: Number(status.match(/^VmRSS:\s+(\d+)/m)?.[1] ?? 0) });
    } catch (error) {
      if (!["ENOENT", "ESRCH", "EACCES"].includes(error.code)) throw error;
    }
  }
  const queue = [root];
  let total = 0;
  while (queue.length) {
    const pid = queue.pop();
    total += rows.find(row => row.pid === pid)?.rss ?? 0;
    queue.push(...rows.filter(row => row.parent === pid && row.pid !== pid).map(row => row.pid));
  }
  return total || undefined;
}

export async function benchmarkPerformance({ binary, baseline, source, root, caseEnv, capture, passed }) {
  assert.equal(process.platform, "linux", "dedicated timing lane currently requires native Linux");
  const realRoot = realpathSync(root);
  const data = join(realRoot, "timing-data");
  mkdirSync(data);
  const document = join(data, "policy.go");
  copyFileSync(join(source, "policy.go"), document);
  const expected = digest(readFileSync(document));
  const variants = ["direct", ...(baseline ? ["beta"] : []), "candidate"];
  const bins = { beta: baseline, candidate: binary };
  const raw = [];
  async function sample(metric, variant, trial, operation) {
    const row = { metric, variant, trial };
    try {
      row.value = await operation();
      row.status = "pass";
    } catch (error) {
      row.status = "fail";
      row.error = String(error.message).replaceAll(root, "<scratch>").slice(0, 1000);
    }
    raw.push(row);
  }
  async function prepare(installRoot, env) {
    mkdirSync(installRoot, { recursive: true });
    const runtime = join(installRoot, "runtime");
    passed(await capture(binary, ["packs", "prepare", "filesystem", "--output", runtime, "--backend", "linux"], { env }), "pinned Filesystem setup");
    const packagePath = join(runtime, "node_modules", "@modelcontextprotocol", "server-filesystem");
    const pkg = JSON.parse(readFileSync(join(packagePath, "package.json")));
    const entry = typeof pkg.bin === "string" ? pkg.bin : Object.values(pkg.bin)[0];
    assert.equal(pkg.version, "2.3.0");
    return { runtime, entry: join(packagePath, entry) };
  }
  async function session(variant, prepared, env, requests = 0, measureRSS = false) {
    const p = join(dirname(prepared.runtime), "policy.yaml");
    writeFileSync(p, `filesystem: {read: ${JSON.stringify([prepared.runtime, data])}, protect_secrets: true}\n`);
    const argv = [process.execPath, prepared.entry, data];
    const transport = new StdioClientTransport({
      command: bins[variant] ?? argv[0],
      args: bins[variant] ? ["run", "--backend", "linux", "--policy", p, "--", ...argv] : argv.slice(1),
      env, stderr: "ignore",
    });
    const client = new Client({ name: "warden-benchmark", version: "1" }, { capabilities: {} });
    const read = async () => {
      const result = await client.callTool({ name: "read_text_file", arguments: { path: document } }, undefined, { timeout: 15000 });
      assert(!result.isError);
      assert.equal(digest(result.content[0].text), expected);
    };
    try {
      const start = performance.now();
      await client.connect(transport, { timeout: 15000 });
      await read();
      const startup = performance.now() - start;
      const calls = [];
      const rss = measureRSS ? [processTreeRSS(transport.pid)] : [];
      for (let n = 0; n < requests; n++) {
        const call = performance.now();
        await read();
        calls.push(performance.now() - call);
        if (measureRSS) rss.push(processTreeRSS(transport.pid));
      }
      return { startup, calls, ...(rss.some(Number.isFinite) ? { peakRSSKiB: Math.max(...rss.filter(Number.isFinite)) } : {}) };
    } finally { await client.close(); }
  }
  const common = join(realRoot, "timing-common");
  const commonEnv = caseEnv(join(common, "environment"));
  const prepared = await prepare(common, commonEnv);
  for (let trial = 1; trial <= 35; trial++) {
    const offset = (trial - 1) % variants.length;
    for (const variant of [...variants.slice(offset), ...variants.slice(0, offset)]) {
      const env = caseEnv(join(realRoot, `startup-${trial}-${variant}`));
      await sample(trial <= 5 ? "startup-warmup-ms" : "startup-ms", variant, trial, async () => (await session(variant, prepared, env)).startup);
    }
  }
  for (let trial = 1; trial <= 5; trial++) {
    for (const variant of variants) {
      const env = caseEnv(join(realRoot, `steady-${trial}-${variant}`));
      await sample("steady-calls-ms", variant, trial, async () => (await session(variant, prepared, env, 20)).calls);
      if (trial === 1) await sample("sampled-process-tree-rss-kib", variant, trial, async () => (await session(variant, prepared, env, 5, true)).peakRSSKiB);
    }
    const path = join(realRoot, `fresh-setup-${trial}`);
    await sample("fresh-setup-ms", "candidate", trial, async () => {
      const env = caseEnv(join(path, "environment"));
      const before = performance.now();
      const fresh = await prepare(path, env);
      await session("candidate", fresh, env);
      return performance.now() - before;
    });
    for (const variant of trial % 2 ? ["direct", "candidate"] : ["candidate", "direct"]) {
      const path = join(realRoot, `fresh-install-${trial}-${variant}`);
      const env = caseEnv(path);
      const target = join(path, "installed");
      await sample("fresh-npm-install-ms", variant, trial, async () => {
        const before = performance.now();
        const args = variant === "candidate" ? ["guard", "npm", "install", "is-number@7.0.0", "--output", target]
          : ["install", "--ignore-scripts", "--no-audit", "--no-fund", "--prefix", target, "is-number@7.0.0"];
        passed(await capture(variant === "candidate" ? binary : "npm", args, { env }), "benign npm install");
        const elapsed = performance.now() - before;
        const pkg = join(target, "node_modules", "is-number");
        assert.equal(JSON.parse(readFileSync(join(pkg, "package.json"))).version, "7.0.0");
        return elapsed;
      });
    }
    for (const variant of variants.filter(name => name !== "direct")) {
      const env = { ...caseEnv(join(realRoot, `workspace-${trial}-${variant}`)), WARDEN_BIN: bins[variant], WARDEN_BENCH_SOURCE: source };
      await sample("workspace-review-and-apply-ms", variant, trial, async () => {
        const before = performance.now();
        const outcome = JSON.parse(passed(await capture(process.execPath, [workflow, "workspace"], { env }), "workspace projection and apply"));
        assert.equal(outcome.status, "pass");
        return performance.now() - before;
      });
    }
  }
  const summary = {};
  for (const { metric, variant } of raw) {
    const key = `${metric}:${variant}`;
    if (summary[key] || metric === "startup-warmup-ms") continue;
    const values = raw.filter(row => row.metric === metric && row.variant === variant && row.status === "pass")
      .flatMap(row => row.value).filter(Number.isFinite);
    if (values.length) summary[key] = summarize(values);
  }
  const comparison = [];
  for (const [key, candidate] of Object.entries(summary)) {
    if (!key.endsWith(":candidate")) continue;
    const metric = key.split(":")[0];
    for (const variant of ["direct", "beta"]) {
      const reference = summary[`${metric}:${variant}`];
      if (reference) {
        const difference = candidate.median - reference.median;
        comparison.push({ metric, reference: variant, medianDifference: difference, percent: reference.median ? 100 * difference / reference.median : null });
      }
    }
  }
  return {
    experimental: true,
    baseline: baseline ? { sha256: digest(readFileSync(baseline)), version: passed(await capture(baseline, ["--version"], { env: commonEnv }), "beta version").trim() } : null,
    definitions: { startup: "process launch to first validated MCP read", fresh: "empty application cache and destination; OS cache uncontrolled",
      setup: "pinned pack preparation through first validated read; includes registry access", install: "real npm registry; guarded vs direct npm with lifecycle disabled",
      workspace: "one declined and one approved project edit, including process launch and scripted review",
      rss: "maximum observed sum of process-tree RSS across six snapshots; shared pages may be counted repeatedly" },
    raw, summary, comparison, failedSamples: raw.filter(row => row.status === "fail").length,
  };
}
