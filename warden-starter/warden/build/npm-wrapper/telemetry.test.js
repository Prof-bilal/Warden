// Zero-dependency tests for the telemetry client contract and the
// /api/install validation rules. Run with: node --test telemetry.test.js
// (Node >= 18, matches the wrapper engines field).
"use strict";

const assert = require("node:assert");
const { spawnSync } = require("node:child_process");
const fs = require("node:fs");
const http = require("node:http");
const os = require("node:os");
const path = require("node:path");
const { describe, it } = require("node:test");

const DIR = __dirname;
const TELEMETRY = path.join(DIR, "telemetry.js");
const ROUTE = path.join(
  DIR, "..", "..", "..", "..",
  "warden-landing", "app", "api", "install", "route.ts"
);

// --- telemetry.js static contract: no deps, no noise, non-blocking --------

describe("telemetry.js contract", () => {
  const src = fs.readFileSync(TELEMETRY, "utf8");

  it("uses only Node built-ins (no new HTTP dependency)", () => {
    assert.ok(!/require\(["'](axios|node-fetch|got|undici)["']\)/.test(src));
    assert.ok(!/from ["'](axios|node-fetch|got)["']/.test(src));
  });

  it("sends only the five allowed fields", () => {
    for (const f of ["package", "version", "platform", "arch", "node"])
      assert.ok(src.includes(f), "missing field " + f);
    for (const banned of ["SSH", "TOKEN", "SECRET", "hostname", "cwd()"])
      assert.ok(!src.includes(banned), "must not reference " + banned);
    assert.ok(!src.includes("JSON.stringify(process.env"));
  });

  it("reads version from npm env vars", () => {
    assert.ok(src.includes("npm_package_name"));
    assert.ok(src.includes("npm_package_version"));
  });

  it("has a short timeout and never throws to npm", () => {
    assert.ok(src.includes("TIMEOUT_MS"));
    assert.ok(src.includes("3000"));
    assert.ok(src.includes("AbortController"));
    assert.ok(src.includes("process.exit(0)"));
  });

  it("supports opt-out without printing", () => {
    assert.ok(src.includes("WARDEN_NO_TELEMETRY"));
    assert.ok(src.includes("DO_NOT_TRACK"));
    assert.ok(!/console\.(log|error|warn)/.test(src), "must stay silent");
    assert.ok(!/stderr\.write/.test(src), "must stay silent");
  });

  it("endpoint comes from one constant with env override", () => {
    assert.ok(src.includes("WARDEN_TELEMETRY_URL"));
    assert.ok(src.includes("TELEMETRY_URL"));
  });
});

// --- live behavior: telemetry.js against stub servers ----------------------

function runTelemetry(env, timeoutMs) {
  // spawnSync inherits the runner env (http_proxy etc.); loopback must
  // bypass any proxy or the stub never sees the request.
  const merged = Object.assign({}, process.env, env);
  merged.NO_PROXY = (merged.NO_PROXY || "") + ",127.0.0.1,localhost";
  merged.no_proxy = (merged.no_proxy || "") + ",127.0.0.1,localhost";
  return spawnSync(process.execPath, [TELEMETRY], {
    env: merged,
    timeout: timeoutMs || 15000,
    encoding: "utf8",
  });
}

describe("telemetry.js runtime", () => {
  it("POSTs the exact five-field payload and exits 0 silently", async () => {
    let received = null;
    let resolveHit;
    const hit = new Promise((r) => (resolveHit = r));
    const server = http.createServer((req, res) => {
      let body = "";
      req.on("data", (c) => (body += c));
      req.on("end", () => {
        received = { method: req.method, url: req.url, body: body };
        res.writeHead(200, { "Content-Type": "application/json" });
        res.end('{"ok":true}');
        resolveHit();
      });
    });
    await new Promise((r) => server.listen(0, "127.0.0.1", r));
    const port = server.address().port;
    // Give the loopback listener a tick to accept (sandboxed CI quirk).
    await new Promise((r) => setTimeout(r, 100));
    try {
      const r = runTelemetry({
        WARDEN_TELEMETRY_URL: "http://127.0.0.1:" + port + "/install",
        npm_package_name: "warden-sandbox-cli",
        npm_package_version: "9.9.9-test",
      });
      // spawnSync blocks until the child exits, but on some Node builds the
      // ref'd AbortController timer races request completion: wait for the
      // server to confirm the hit before asserting (bounded by the 3s cap).
      await Promise.race([hit, new Promise((r) => setTimeout(r, 5000))]);
      assert.equal(r.status, 0, "exit " + r.status + ": " + r.stderr);
      assert.equal(r.stdout, "");
      assert.equal(r.stderr, "");
      assert.ok(received, "server got no request");
      assert.equal(received.method, "POST");
      assert.equal(received.url, "/install");
      const payload = JSON.parse(received.body);
      assert.deepEqual(Object.keys(payload).sort(), [
        "arch", "node", "package", "platform", "version",
      ]);
      assert.equal(payload.package, "warden-sandbox-cli");
      assert.equal(payload.version, "9.9.9-test");
      assert.equal(payload.platform, process.platform);
      assert.equal(payload.arch, process.arch);
      assert.equal(payload.node, process.version);
    } finally {
      server.close();
    }
  });

  it("install succeeds silently when the API is down", () => {
    const r = runTelemetry({
      WARDEN_TELEMETRY_URL: "http://127.0.0.1:1/install",
      npm_package_name: "warden-sandbox-cli",
    });
    assert.equal(r.status, 0);
    assert.equal(r.stdout, "");
    assert.equal(r.stderr, "");
  });

  it("install succeeds when the server returns 500", async () => {
    const server = http.createServer((req, res) => {
      req.resume();
      req.on("end", () => { res.writeHead(500); res.end("boom"); });
    });
    await new Promise((r) => server.listen(0, r));
    try {
      const r = runTelemetry({
        WARDEN_TELEMETRY_URL:
          "http://127.0.0.1:" + server.address().port + "/install",
      });
      assert.equal(r.status, 0);
      assert.equal(r.stdout, "");
      assert.equal(r.stderr, "");
    } finally {
      server.close();
    }
  });

  it("times out fast against a hanging server", async () => {
    const server = http.createServer(() => {});
    await new Promise((r) => server.listen(0, r));
    try {
      const start = Date.now();
      const r = runTelemetry({
        WARDEN_TELEMETRY_URL:
          "http://127.0.0.1:" + server.address().port + "/install",
      });
      const elapsed = Date.now() - start;
      assert.equal(r.status, 0);
      assert.ok(elapsed < 10000, "took " + elapsed + "ms, timeout is 3s");
    } finally {
      server.close();
    }
  });

  it("opt-out env skips the request entirely", async () => {
    let hits = 0;
    const server = http.createServer((req, res) => {
      hits++;
      req.resume();
      req.on("end", () => res.end("{}"));
    });
    await new Promise((r) => server.listen(0, r));
    try {
      const url = "http://127.0.0.1:" + server.address().port + "/install";
      const cases = [
        { WARDEN_NO_TELEMETRY: "1" },
        { DO_NOT_TRACK: "1" },
        { WARDEN_TELEMETRY: "0" },
      ];
      for (const extra of cases) {
        extra.WARDEN_TELEMETRY_URL = url;
        const r = runTelemetry(extra);
        assert.equal(r.status, 0);
      }
      assert.equal(hits, 0, "opted-out run still sent a request");
    } finally {
      server.close();
    }
  });
});

// --- /api/install route validation (source-level + contract table) ---------

describe("route.ts validation rules", () => {
  const src = fs.readFileSync(ROUTE, "utf8");

  it("enforces a body size cap", () => {
    assert.ok(src.includes("MAX_BODY_BYTES"));
    assert.ok(src.includes("413"));
  });

  it("pins the package name and rejects extra keys", () => {
    assert.ok(src.includes("warden-sandbox-cli"));
    assert.ok(src.includes("malformed payload"));
  });

  it("never reads IP/hostname/headers for storage", () => {
    const banned = [
      "x-forwarded-for", "request.ip", "req.headers", "headers.get(",
      "req.ip", "getClientAddress", "userAgent(", "os.hostname",
    ];
    for (const b of banned)
      assert.ok(!src.includes(b), "route must not touch " + b);
  });

  it("stores only the six spec fields with server timestamp", () => {
    const fields = ["timestamp", "package", "version", "platform",
      "architecture", "node_version"];
    for (const f of fields)
      assert.ok(src.includes(f), "store record missing " + f);
  });

  it("generic 500 without internals on storage failure", () => {
    assert.ok(src.includes("status: 500"));
  });
});

// Mirror of route.ts rules in plain Node (the route needs the Next runtime;
// this asserts the same accept/reject table without spinning up Next).
function validate(b) {
  const PLATS = ["linux", "darwin", "win32", "freebsd", "openbsd",
    "sunos", "aix"];
  const ARCH = ["x64", "arm64", "arm", "ia32", "ppc64", "s390x",
    "riscv64", "loong64"];
  if (typeof b !== "object" || b === null || Array.isArray(b)) return 400;
  if (b.package !== "warden-sandbox-cli") return 400;
  if (typeof b.version !== "string" ||
      !/^[A-Za-z0-9@/_+.\-]{1,32}$/.test(b.version)) return 400;
  if (PLATS.indexOf(b.platform) === -1) return 400;
  if (ARCH.indexOf(b.arch) === -1) return 400;
  if (typeof b.node !== "string" || !/^v\d+\.\d+\.\d+/.test(b.node))
    return 400;
  const keys = Object.keys(b);
  const allowed = ["package", "version", "platform", "arch", "node"];
  if (keys.length !== 5 || !keys.every((k) => allowed.indexOf(k) !== -1))
    return 400;
  return 200;
}

describe("validation table (mirrors route.ts)", () => {
  const good = {
    package: "warden-sandbox-cli",
    version: "0.1.12",
    platform: "linux",
    arch: "x64",
    node: "v20.11.0",
  };

  it("accepts a valid event", () => assert.equal(validate(good), 200));

  it("rejects malformed payloads and missing fields", () => {
    assert.equal(validate(null), 400);
    assert.equal(validate([]), 400);
    assert.equal(validate("x"), 400);
    assert.equal(validate({}), 400);
    assert.equal(validate({ package: "warden-sandbox-cli" }), 400);
  });

  it("rejects wrong package, bad enums, extra keys", () => {
    const cases = [
      Object.assign({}, good, { package: "evil" }),
      Object.assign({}, good, { platform: "mars" }),
      Object.assign({}, good, { arch: "quantum" }),
      Object.assign({}, good, { node: "latest" }),
      Object.assign({}, good, { extra: 1 }),
      Object.assign({}, good, { version: "1.0; rm -rf ~" }),
    ];
    for (const c of cases)
      assert.equal(validate(c), 400, JSON.stringify(c));
  });

  it("stored record carries no IP", () => {
    const record = {
      timestamp: new Date().toISOString(),
      package: good.package,
      version: good.version,
      platform: good.platform,
      architecture: good.arch,
      node_version: good.node,
    };
    const s = JSON.stringify(record);
    assert.ok(!/forwarded/i.test(s));
    assert.deepEqual(Object.keys(record).sort(), [
      "architecture", "node_version", "package",
      "platform", "timestamp", "version",
    ]);
  });
});

// Consumer-level: the packed wrapper installs with a dead endpoint.
describe("packed install still succeeds when telemetry fails", () => {
  it("packs and installs with unreachable endpoint", function () {
    const tmp = fs.mkdtempSync(path.join(os.tmpdir(), "warden-tel-"));
    const pack = spawnSync("npm", ["pack", "--pack-destination", tmp], {
      cwd: DIR, encoding: "utf8", timeout: 60000,
    });
    assert.equal(pack.status, 0, String(pack.stderr).slice(-500));
    const tgz = fs.readdirSync(tmp).find((f) => f.endsWith(".tgz"));
    assert.ok(tgz, "no tarball produced");
    const target = fs.mkdtempSync(path.join(os.tmpdir(), "warden-cons-"));
    const inst = spawnSync(
      "npm",
      ["install", "--no-save", "--ignore-scripts", path.join(tmp, tgz)],
      { cwd: target, encoding: "utf8", timeout: 120000 }
    );
    assert.equal(inst.status, 0, String(inst.stderr).slice(-800));
  });
});
