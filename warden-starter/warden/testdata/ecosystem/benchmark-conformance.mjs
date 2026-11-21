import assert from "node:assert/strict";
import { execFile } from "node:child_process";
import { once } from "node:events";
import { readdir, readFile } from "node:fs/promises";
import { createServer, request } from "node:http";
import { join } from "node:path";
import { promisify } from "node:util";

const run = promisify(execFile);

export async function testMCPConformance(endpoint, token, directory) {
  // The official runner targets a loopback route. Relay only this fixture's
  // MCP path, supplying its disposable token without changing Warden auth.
  const server = createServer((incoming, outgoing) => {
    if (incoming.url !== "/mcp") { outgoing.writeHead(404).end(); return; }
    const upstream = request(endpoint, { method: incoming.method,
      headers: { ...incoming.headers, host: endpoint.host, authorization: `Bearer ${token}` } }, response => {
      outgoing.writeHead(response.statusCode, response.headers);
      response.pipe(outgoing);
    });
    upstream.on("error", () => outgoing.destroy());
    outgoing.on("close", () => upstream.destroy());
    incoming.pipe(upstream);
  });
  server.listen(0, "127.0.0.1");
  await once(server, "listening");
  try {
    const url = `http://127.0.0.1:${server.address().port}/mcp`;
    const findings = [];
    for (const scenario of ["server-initialize", "tools-list"]) {
      const results = join(directory, scenario);
      const args = [process.env.WARDEN_CONFORMANCE_JS, "server", "--url", url,
        "--spec-version", "2025-11-25", "--scenario", scenario, "--output-dir", results];
      try {
        await run(process.execPath, args, { timeout: 40000 });
      } catch (error) {
        throw Error(`Official MCP ${scenario}: ${error.stdout || error.stderr || error.message}`);
      }
      const paths = (await readdir(results, { recursive: true })).filter(path => path.endsWith("checks.json"));
      assert.equal(paths.length, 1, `no unique official check result for ${scenario}`);
      const checks = JSON.parse(await readFile(join(results, paths[0]), "utf8"));
      assert(Array.isArray(checks) && checks.length && checks.every(check => check.status === "SUCCESS"), `official protocol checks for ${scenario}: ${JSON.stringify(checks)}`);
      findings.push(...checks.map(check => `official MCP 2025-11-25 ${scenario}: ${check.id}=${check.status}`));
    }
    return findings;
  } finally {
    server.closeAllConnections();
    await new Promise(resolve => server.close(resolve));
  }
}
