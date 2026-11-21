import assert from "node:assert/strict";
import { once } from "node:events";
import { mkdir, writeFile } from "node:fs/promises";
import { createServer } from "node:http";
import { createRequire } from "node:module";
import { join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const site = fileURLToPath(new URL("../../../../warden-landing/", import.meta.url));
const load = createRequire(join(site, "package.json"));
const next = load("next");
const app = next({ dir: site, dev: false, hostname: "127.0.0.1" });
const handler = app.getRequestHandler();
const evidence = { scenario: "website-install-and-setup", checks: [] };
const output = resolve(process.argv[2] ?? "dist/benchmark-website");
await mkdir(output, { recursive: true });
const server = createServer((req, res) => handler(req, res));
try {
  await app.prepare();
  server.listen(0, "127.0.0.1");
  await once(server, "listening");
  const base = `http://127.0.0.1:${server.address().port}`;
  async function page(path, pattern) {
    const response = await fetch(base + path, { signal: AbortSignal.timeout(15000) });
    assert.equal(response.status, 200, path);
    assert.match(await response.text(), pattern);
    evidence.checks.push(path);
  }
  await page("/docs/install", /warden-sandbox-cli@beta/);
  await page("/setup/codex", /Warden setup for (?:OpenAI )?Codex/i);
  await page("/setup/claude-code", /Warden setup for Claude Code/i);
  await page("/v2", /warden-v2-intro\.mp4/);
  const bad = await fetch(base + "/api/install", { method: "POST", body: "invalid fixture JSON", signal: AbortSignal.timeout(15000) });
  assert.equal(bad.status, 400);
  assert.deepEqual(await bad.json(), { ok: false, error: "invalid JSON" });
  evidence.checks.push("install endpoint rejects malformed event");
  evidence.status = "pass";
} catch (error) {
  evidence.status = "fail";
  evidence.reason = error.message;
  process.exitCode = 1;
} finally {
  if (server.listening) {
    server.closeAllConnections();
    await new Promise(resolveClose => server.close(resolveClose));
  }
  await app.close?.();
  await writeFile(join(output, "website.json"), JSON.stringify(evidence, null, 2) + "\n");
  console.log(JSON.stringify(evidence));
}
