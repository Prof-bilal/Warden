import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { arch, cpus, platform, release } from "node:os";
import { fileURLToPath } from "node:url";

const repeat = Number(process.argv[2] ?? 20);
assert(Number.isInteger(repeat) && repeat > 0 && repeat <= 100, "repeat must be an integer from 1 to 100");
const result = spawnSync("go", ["test", "-json", "-count=" + repeat, "-timeout=20m", "-run",
  "^TestAutopilotNativeReadOnly$", "./cmd/warden"], {
  cwd: fileURLToPath(new URL("../../", import.meta.url)),
  env: { ...process.env, WARDEN_REQUIRE_AGENT_E2E: "1" },
  encoding: "utf8",
  maxBuffer: 16 * 1024 * 1024,
});
if (result.status !== 0) {
  process.stderr.write(result.stdout ?? "");
  process.stderr.write(result.stderr ?? "");
  throw result.error ?? Error("Autopilot fixture failed");
}
const events = result.stdout.trim().split("\n").map(line => JSON.parse(line));
const name = "TestAutopilotNativeReadOnly";
assert.equal(events.filter(event => event.Action === "pass" && event.Test === name).length, repeat);
const samples = events.flatMap(event => {
  const match = event.Test === name && event.Output?.match(/AUTOPILOT_TIMING setup_ms=([\d.]+) launch_ms=([\d.]+)/);
  return match ? [{ setupMs: Number(match[1]), launchMs: Number(match[2]) }] : [];
});
assert.equal(samples.length, repeat);

function timing(values) {
  const sorted = [...values].sort((left, right) => left - right);
  const middle = Math.floor(sorted.length / 2);
  return {
    median: sorted.length % 2 ? sorted[middle] : (sorted[middle - 1] + sorted[middle]) / 2,
    p95: sorted[Math.ceil(0.95 * sorted.length) - 1],
  };
}

process.stdout.write(JSON.stringify({
  platform: platform(), arch: arch(), release: release(), cpu: cpus()[0]?.model,
  runCount: repeat, unit: "milliseconds",
  setup: timing(samples.map(sample => sample.setupMs)),
  launch: timing(samples.map(sample => sample.launchMs)),
  samples,
}, null, 2) + "\n");
