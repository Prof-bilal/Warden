// Zero-dependency tests for the stats contract: lib/stats.ts pure helpers
// plus source-level checks on the /api/stats route. Run with:
//   node --test stats.test.js   (from warden-landing/)
// Follows the npm-wrapper telemetry.test.js convention (plain node:test,
// no framework) because the landing has no JS test runner configured.
"use strict";

const assert = require("node:assert");
const fs = require("node:fs");
const path = require("node:path");
const { describe, it } = require("node:test");

const DIR = __dirname;

// The helpers are pure TS with no imports; load them by stripping types
// with a small transpile (good enough for these three functions — if the
// lib grows real imports, switch to a TS runner instead).
function loadStatsLib() {
  const src = fs.readFileSync(path.join(DIR, "lib", "stats.ts"), "utf8");
  const js = src
    .split("\n")
    .filter((line) => !line.trim().startsWith("//"))
    .join("\n")
    .replace(
      /export function formatCount\(n: unknown\): string \| null \{/,
      "function formatCount(n) {"
    )
    .replace(
      /export function parseGitHubStars\(data: unknown\): number \| null \{/,
      "function parseGitHubStars(data) {"
    )
    .replace(
      /export function parseNpmDownloads\(data: unknown\): number \| null \{/,
      "function parseNpmDownloads(data) {"
    )
    .replace(
      /const units: Array<\[number, string\]> = \[/,
      "const units = ["
    )
    .replace(
      /const v = \(data as Record<string, unknown>\)\.(stargazers_count|downloads);/g,
      "const v = data.$1;"
    );
  const fn = new Function(
    `${js}; return { formatCount, parseGitHubStars, parseNpmDownloads };`
  );
  return fn();
}

const { formatCount, parseGitHubStars, parseNpmDownloads } = loadStatsLib();

describe("formatCount", () => {
  it("passes through small numbers", () => {
    assert.equal(formatCount(0), "0");
    assert.equal(formatCount(4), "4"); // current real star count
    assert.equal(formatCount(999), "999");
  });
  it("compacts thousands/millions/billions", () => {
    assert.equal(formatCount(1219), "1.2k"); // current real downloads
    assert.equal(formatCount(1500), "1.5k");
    assert.equal(formatCount(23000), "23k");
    assert.equal(formatCount(2500000), "2.5M");
    assert.equal(formatCount(1200000000), "1.2B");
  });
  it("rejects non-numbers", () => {
    for (const bad of [null, undefined, "12", NaN, Infinity, -1, {}, []])
      assert.equal(formatCount(bad), null, JSON.stringify(bad));
  });
});

describe("parseGitHubStars", () => {
  it("reads stargazers_count", () => {
    assert.equal(parseGitHubStars({ stargazers_count: 4 }), 4);
  });
  it("rejects malformed bodies", () => {
    for (const bad of [
      null,
      [],
      {},
      { stargazers_count: "4" },
      { stargazers_count: -1 },
      { stars: 4 },
    ])
      assert.equal(parseGitHubStars(bad), null, JSON.stringify(bad));
  });
});

describe("parseNpmDownloads", () => {
  it("reads downloads", () => {
    assert.equal(parseNpmDownloads({ downloads: 1219 }), 1219);
  });
  it("rejects malformed bodies", () => {
    for (const bad of [
      null,
      [],
      {},
      { downloads: "1219" },
      { downloads: -5 },
      { count: 1219 },
    ])
      assert.equal(parseNpmDownloads(bad), null, JSON.stringify(bad));
  });
});

describe("route.ts contract", () => {
  const src = fs.readFileSync(
    path.join(DIR, "app", "api", "stats", "route.ts"),
    "utf8"
  );
  it("caches hourly to respect the 60 req/h GitHub limit", () => {
    assert.ok(src.includes("revalidate"));
    assert.ok(src.includes("3600"));
  });
  it("times out upstream fetches and degrades to null", () => {
    assert.ok(src.includes("AbortController"));
    assert.ok(src.includes("5000"));
    assert.ok(src.includes("return null"));
  });
  it("sends a User-Agent (required by the GitHub API)", () => {
    assert.ok(src.includes("User-Agent"));
  });
  it("endpoints overridable via env", () => {
    assert.ok(src.includes("WARDEN_STATS_GITHUB_URL"));
    assert.ok(src.includes("WARDEN_STATS_NPM_URL"));
  });
  it("response shape is exactly { stars, downloads }", () => {
    assert.ok(src.includes("stars:"));
    assert.ok(src.includes("downloads:"));
  });
});

describe("SocialProof.tsx contract", () => {
  const src = fs.readFileSync(
    path.join(DIR, "components", "SocialProof.tsx"),
    "utf8"
  );
  it("links npm by name and GitHub repo", () => {
    assert.ok(src.includes("https://www.npmjs.com/package/warden-sandbox-cli"));
    assert.ok(src.includes(">npm<"));
    assert.ok(src.includes("https://github.com/Prof-bilal/Warden"));
  });
  it("renders static labels first, upgrades on success only", () => {
    assert.ok(src.includes('"Star"'), "static GitHub fallback");
    assert.ok(src.includes("useState"), "no count until fetch resolves");
    assert.ok(src.includes("cancelled"), "no setState after unmount");
  });
  it("never breaks the hero on API failure", () => {
    assert.ok(src.includes(".catch("), "fetch failure swallowed");
    assert.ok(src.includes("!res.ok"), "non-2xx ignored");
  });
  it("is used by Hero and old badges are gone", () => {
    const hero = fs.readFileSync(
      path.join(DIR, "components", "Hero.tsx"),
      "utf8"
    );
    assert.ok(hero.includes("SocialProof"));
    assert.ok(!hero.includes("Featured on"), "badge moved to SocialProof");
  });
});
