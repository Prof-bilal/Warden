// Run from any directory. The suite never writes to a user's client config.
import assert from "node:assert/strict";
import { spawn, spawnSync } from "node:child_process";
import { createHash } from "node:crypto";
import { existsSync, mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { arch, cpus, homedir, release, tmpdir } from "node:os";
import { join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { parseArgs } from "node:util";

const directory = fileURLToPath(new URL(".", import.meta.url));
const moduleDir = resolve(directory, "../..");
const manifestBytes = readFileSync(join(directory, "benchmark-scenarios.json"));
const manifest = JSON.parse(manifestBytes);
const { values: options } = parseArgs({ options: {
  suite: { type: "string", default: "pr" }, output: { type: "string", default: "dist/benchmark" },
  repeat: { type: "string", default: "1" }, binary: { type: "string" }, baseline: { type: "string" },
  scenario: { type: "string", multiple: true }, help: { type: "boolean" },
} });
if (options.help) {
  console.log("node benchmark.mjs --suite pr|full|docker|registry|performance [--repeat 2] [--binary FILE] [--baseline FILE] [--scenario ID] --output DIR");
  process.exit(0);
}
const repeat = Number(options.repeat), suite = options.suite;
assert(["pr", "full", "docker", "registry", "performance"].includes(suite), "invalid suite");
assert(Number.isInteger(repeat) && repeat > 0 && repeat <= 5, "repeat must be an integer from 1 to 5");
const ids = new Set(manifest.scenarios.map(row => row.id));
assert.equal(ids.size, manifest.scenarios.length, "scenario IDs must be unique");
const cli = readFileSync(join(moduleDir, "cmd/warden/main.go"), "utf8");
const dispatch = cli.slice(cli.indexOf("switch os.Args[1]"));
const covered = new Set(manifest.scenarios.flatMap(row => row.commands.map(name => name.split(" ")[0])));
const publicCommands = [...dispatch.slice(0, dispatch.indexOf("\n\tdefault:")).matchAll(/^\tcase (.+):$/gm)]
  .flatMap(([, clause]) => Array.from(clause.matchAll(/"([a-z][a-z-]*)"/g), match => match[1]));
assert(publicCommands.length >= 20, "could not inventory the public CLI");
assert.deepEqual(publicCommands.filter(name => !covered.has(name)), [], "public CLI commands missing benchmark coverage");
for (const id of options.scenario ?? []) assert(ids.has(id), `unknown scenario: ${id}`);
const requested = manifest.scenarios.filter(row => options.scenario
  ? options.scenario.includes(row.id)
  : row.tier === suite || (suite === "full" && row.tier === "pr"));
assert(suite === "performance" ? !options.scenario : requested.some(row => row.platforms.includes(process.platform)), "no applicable scenarios selected");

const output = resolve(options.output);
mkdirSync(output, { recursive: true });
const runDirectory = mkdtempSync(join(output, "run-"));
const root = mkdtempSync(join(process.platform !== "linux" || requested.some(row => row.tier === "docker") ? homedir() : tmpdir(), ".warden-benchmark-"));
const nativeBackend = { linux: "linux", darwin: "seatbelt", win32: "windows" }[process.platform];
const digest = bytes => createHash("sha256").update(bytes).digest("hex");
const sys = ["PATH", "PATHEXT", "SystemRoot", "WINDIR", "SystemDrive", "ComSpec", "ProgramFiles",
  "ProgramFiles(x86)", "ProgramData", "PROCESSOR_ARCHITECTURE", "GOROOT", "GOPATH", "GOMODCACHE",
  "GOCACHE", "GOMAXPROCS", "CI", "WARDEN_DOCKER_IMAGE", "WARDEN_CODEX_BIN", "WARDEN_REVIEW_PYTHON", "WARDEN_CONFORMANCE_JS"];
const cleanEnv = Object.fromEntries(sys.filter(key => process.env[key]).map(key => [key, process.env[key]]));
const nativeSuffix = process.platform === "win32" ? ".exe" : "";
const binary = options.binary ? resolve(options.binary) : join(root, "warden" + nativeSuffix);
function caseEnv(directory) {
  const home = join(directory, "home"), state = join(directory, "state"), temp = join(directory, "tmp");
  mkdirSync(home, { recursive: true }); mkdirSync(temp, { recursive: true });
  return { ...cleanEnv, HOME: home, USERPROFILE: home, APPDATA: join(home, "appdata"), LOCALAPPDATA: join(home, "local"),
    TMPDIR: temp, TMP: temp, TEMP: temp, XDG_CONFIG_HOME: join(home, "config"), XDG_DATA_HOME: join(home, "data"),
    XDG_CACHE_HOME: join(home, "cache"), XDG_STATE_HOME: state, WARDEN_SETUP_STATE_DIR: join(home, "setup"),
    npm_config_cache: join(home, "npm-cache"), npm_config_userconfig: join(home, ".npmrc"), PIP_CONFIG_FILE: process.platform === "win32" ? "NUL" : "/dev/null",
    UV_CACHE_DIR: join(home, "uv-cache"), WARDEN_NO_UPDATE_CHECK: "1", WARDEN_NO_FIRST_RUN: "1",
    WARDEN_REQUIRE_AGENT_E2E: "1", NO_COLOR: "1" };
}
const report = {
  version: 1, suite, repeat, scope: options.scenario ?? "suite", started: new Date().toISOString(),
  environment: { platform: process.platform, architecture: arch(), release: release(), cpu: cpus()[0]?.model, logicalCPUCount: cpus().length,
    dockerImage: cleanEnv.WARDEN_DOCKER_IMAGE, node: process.version },
  fixtureManifestSHA256: digest(manifestBytes), fixtureInputs: manifest.inputs, sourceFixture: manifest.sourceFixture,
  dependencyLocks: {
    mcpSDK: digest(readFileSync(join(moduleDir, "mcp-test/package-lock.json"))),
    go: digest(readFileSync(join(moduleDir, "go.sum"))),
  },
  scenarios: [],
};
function capture(command, args, { env, cwd = moduleDir, timeout = 180000 } = {}) {
  return new Promise(resolveResult => {
    const posix = process.platform !== "win32";
    const child = spawn(command, args, { cwd, env, detached: posix, stdio: ["ignore", "pipe", "pipe"] });
    const stdout = [], stderr = [];
    let size = 0, condition;
    const stop = () => {
      if (posix && child.pid) {
        try { process.kill(-child.pid, "SIGKILL"); } catch (error) { if (error.code !== "ESRCH") child.kill(); }
      } else if (child.pid) {
        spawnSync("taskkill", ["/PID", String(child.pid), "/T", "/F"], { stdio: "ignore", timeout: 5000 });
        child.kill();
      }
    };
    const collect = chunks => chunk => {
      size += chunk.length;
      if (size <= 4 * 1024 * 1024) chunks.push(chunk);
      else if (!condition) { condition = "output limit exceeded"; stop(); }
    };
    child.stdout.on("data", collect(stdout)); child.stderr.on("data", collect(stderr));
    child.on("error", error => { condition ??= error.message; });
    const timer = setTimeout(() => { condition = "timeout"; stop(); }, timeout);
    child.on("close", (code, signal) => {
      clearTimeout(timer);
      if (posix) stop();
      resolveResult({ code, signal, condition, stdout: Buffer.concat(stdout).toString(), stderr: Buffer.concat(stderr).toString() });
    });
  });
}
function passed(result, context) {
  assert(!result.condition && result.code === 0, `${context}: ${result.condition ?? result.code}\n${result.stderr.slice(-2000)}`);
  return result.stdout;
}
function goEvidence(source, names) {
  const actions = source.trim().split("\n").filter(Boolean).map(line => JSON.parse(line));
  for (const name of names) assert(actions.some(event => event.Test === name && event.Action === "pass"), `required Go test did not pass: ${name}`);
  assert(!actions.some(event => event.Action === "skip"), "required Go test skipped");
  return names;
}
function record() {
  writeFileSync(join(runDirectory, "report.json"), JSON.stringify(report, null, 2) + "\n");
  const rows = report.scenarios.map(row => `| ${row.attempt} | ${row.id} | ${row.status} | ${row.evidence ?? "—"} |`);
  const timing = Object.entries(report.performance?.summary ?? {}).map(([key, values]) =>
    `| ${key} | ${values.count} | ${values.median.toFixed(2)} | ${values.p95.toFixed(2)} | ${values.min.toFixed(2)}–${values.max.toFixed(2)} |`);
  writeFileSync(join(runDirectory, "report.md"), [
    "# Warden benchmark", "", `Suite ${suite} on ${report.environment.platform}/${report.environment.architecture}`,
    `Commit ${report.commit ?? "unknown"}`, "", "| Run | Scenario | Verdict | Evidence |", "|---|---|---|---|", ...rows, "",
    ...(report.error ? [`Setup error — ${report.error}`] : []),
    ...(report.performance ? ["", "## Performance", "", `Failed samples ${report.performance.failedSamples}`, "",
      "| Metric and variant | Samples | Median | p95 | Range |", "|---|---:|---:|---:|---:|", ...timing,
      "", "See performance.json for raw samples, definitions, and same-host baseline differences."] : []),
  ].join("\n") + "\n");
}

try {
  const goPaths = passed(await capture("go", ["env", "-json", "GOPATH", "GOMODCACHE", "GOCACHE"], { env: process.env }), "Go cache discovery");
  Object.assign(cleanEnv, JSON.parse(goPaths));
  const build = caseEnv(join(root, "build"));
  // Keep tool caches when explicitly supplied and record their versions.
  const git = args => capture("git", args, { env: process.env });
  report.commit = passed(await git(["rev-parse", "HEAD"]), "Git revision").trim();
  report.trackedTreeDirty = !!passed(await git(["status", "--porcelain", "--untracked-files=no"]), "Git status").trim();
  report.environment.go = passed(await capture("go", ["version"], { env: build }), "Go prerequisite").trim();
  if (process.platform === "linux") {
    report.environment.npm = passed(await capture("npm", ["--version"], { env: build }), "npm prerequisite").trim();
  }
  const wrapper = JSON.parse(readFileSync(join(moduleDir, "build/npm-wrapper/package.json")));
  if (!options.binary) {
    const stamp = requested.some(row => row.id === "packed-launcher") ? wrapper.version : report.commit.slice(0, 12);
    passed(await capture("go", ["build", "-ldflags", `-X github.com/warden-sandbox/warden/internal/version.Version=${stamp}`,
      "-o", binary, "./cmd/warden"], { env: build, timeout: 300000 }), "candidate build");
  }
  report.binary = { sha256: digest(readFileSync(binary)), version: passed(await capture(binary, ["--version"], { env: build }), "candidate version").trim() };
  const source = join(root, "source");
  if (suite === "performance" || requested.some(row => ["policy", "trace", "workspace"].includes(row.driver))) {
    mkdirSync(source);
    // git show fetches known, trusted source bytes without fetching or checking out a branch.
    const representative = "warden-starter/warden/internal/policy/policy.go";
    const file = passed(await git(["show", `${manifest.sourceFixture}:${representative}`]), "pinned Warden source fixture");
    writeFileSync(join(source, "policy.go"), file);
    report.sourceSHA256 = digest(file);
  }

  if (suite === "performance") {
    const { benchmarkPerformance } = await import("./benchmark-performance.mjs");
    const baseline = options.baseline ? resolve(options.baseline) : undefined;
    report.performance = await benchmarkPerformance({ binary, baseline, source, root, caseEnv, capture, passed });
    writeFileSync(join(runDirectory, "performance.json"), JSON.stringify(report.performance, null, 2) + "\n");
    if (report.performance.failedSamples) process.exitCode = 1;
  }
  const workflow = fileURLToPath(new URL("./benchmark-workflows.mjs", import.meta.url));
  for (let attempt = 1; attempt <= repeat; attempt++) {
    let runtime, scenarioInputs;
    for (const scenario of requested) {
      const row = { id: scenario.id, attempt, expected: scenario.expect, commands: scenario.commands };
      report.scenarios.push(row);
      if (!scenario.platforms.includes(process.platform)) { row.status = "unsupported"; row.reason = `Only ${scenario.platforms.join(", ")}`; continue; }
      const working = join(root, `${attempt}-${scenario.id}`), env = {
        ...caseEnv(working), WARDEN_BIN: binary, WARDEN_BENCH_SOURCE: source, WARDEN_BENCH_BACKEND: nativeBackend, ...scenario.env,
      };
      const started = performance.now();
      let result;
      try {
        if (scenario.tier === "docker") assert.match(env.WARDEN_DOCKER_IMAGE ?? "", /@sha256:[0-9a-f]{64}$/, "digest-pinned WARDEN_DOCKER_IMAGE required");
        if (scenario.id === "official-mcp") assert(env.WARDEN_CONFORMANCE_JS && existsSync(env.WARDEN_CONFORMANCE_JS), "pinned WARDEN_CONFORMANCE_JS required");
        else delete env.WARDEN_CONFORMANCE_JS;
        if (scenario.id === "packed-launcher") {
          runtime = join(working, "runtime");
          passed(await capture(binary, ["packs", "prepare", "filesystem", "--output", runtime, "--backend", "linux"], { env }), "Filesystem package preparation");
          env.WARDEN_SMOKE_RUNTIME = runtime;
        }
        if (scenario.id === "codex-host") {
          assert(scenarioInputs && existsSync(scenarioInputs), "run packed-launcher before codex-host");
          env.WARDEN_SCENARIO_INPUTS = scenarioInputs;
        }
        const invoking = scenario.package
          ? ["go", ["test", "-json", "-p", "1", "-count=1", "-timeout=3m", "-run", `^(${scenario.tests.join("|")})$`, scenario.package]]
          : [process.execPath, scenario.script ? [join(directory, scenario.script), ...(scenario.args ?? [])] : [workflow, scenario.driver]];
        result = await capture(...invoking, { env, timeout: 240000 });
        row.evidence = `${attempt}-${scenario.id}.log`;
        // The child environment is allowlisted; redact ephemeral paths and fixture credentials for shared artifacts.
        const redact = text => text.replaceAll(root, "<scratch>").replaceAll(JSON.stringify(root).slice(1, -1), "<scratch>")
          .replace(/(?:fixture-token-|native-test-|disposable-test-token-)[x]{20,}/g, "<fixture-token>");
        writeFileSync(join(runDirectory, row.evidence), redact(result.stdout + result.stderr));
        const stdout = passed(result, scenario.id);
        if (scenario.package) row.checks = goEvidence(stdout, scenario.tests);
        else {
          const outcome = JSON.parse(stdout);
          if (outcome.status === "unsupported") {
            row.status = "unsupported";
            row.reason = outcome.reason ?? "backend unavailable";
            row.checks = [];
            row.durationMs = Math.round(performance.now() - started);
            continue;
          }
          assert.equal(outcome.status, "pass", "no explicit end-to-end verdict");
          row.checks = outcome.checks ?? [`${outcome.pack}@${outcome.version}: assertions passed`];
          row.upstream = outcome.upstream;
          row.host = outcome.host;
          row.backend = outcome.backend;
          if (scenario.id === "packed-launcher") scenarioInputs = join(outcome.evidenceDirectory, "private", "scenario-inputs.json");
        }
        row.status = "pass";
      } catch (error) {
        row.status = result?.condition === "timeout" ? "timeout" : result?.code === null || !result ? "setup-error" : "fail";
        row.reason = String(error.message).replaceAll(root, "<scratch>").slice(0, 2500);
        process.exitCode = 1;
      }
      row.durationMs = Math.round(performance.now() - started);
      console.log(`${attempt}/${repeat} ${scenario.id}: ${row.status}${row.reason ? " — " + row.reason.split("\n")[0] : ""}`);
    }
  }
} catch (error) {
  report.error = error.message.replaceAll(root, "<scratch>");
  console.error(report.error);
  process.exitCode = 1;
} finally {
  report.finished = new Date().toISOString();
  try {
    rmSync(root, { force: true, recursive: true, maxRetries: 5, retryDelay: 100 });
  } catch (error) {
    report.cleanupError = error.message.replaceAll(root, "<scratch>");
    process.exitCode = 1;
  }
  report.result = process.exitCode ? "fail" : "pass";
  record();
  console.log(`Evidence: ${runDirectory}`);
}
