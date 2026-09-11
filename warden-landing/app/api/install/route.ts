import { NextResponse } from "next/server";
import { promises as fs } from "fs";
import path from "path";

export const runtime = "nodejs";
export const dynamic = "force-dynamic";

// Accepts anonymous npm install pings. Stores one JSONL line per event in
// data/installs.jsonl (gitignored). No IP, hostname, username, or any
// request header is ever persisted — only the five validated body fields
// plus a server-side timestamp.
const MAX_BODY_BYTES = 4096;

const ALLOWED_PLATFORMS = new Set([
  "linux",
  "darwin",
  "win32",
  "freebsd",
  "openbsd",
  "sunos",
  "aix",
]);

const ALLOWED_ARCH = new Set([
  "x64",
  "arm64",
  "arm",
  "ia32",
  "ppc64",
  "s390x",
  "riscv64",
  "loong64",
]);

function isSafeToken(s: unknown, maxLen: number): s is string {
  return (
    typeof s === "string" &&
    s.length > 0 &&
    s.length <= maxLen &&
    /^[A-Za-z0-9@/_+.\-]+$/.test(s)
  );
}

function isNodeVersion(s: unknown): s is string {
  return (
    typeof s === "string" &&
    s.length >= 2 &&
    s.length <= 16 &&
    /^v\d+\.\d+\.\d+/.test(s)
  );
}

function storePath(): string {
  // Outside public/ so events are never web-servable.
  return path.join(process.cwd(), "data", "installs.jsonl");
}

export async function POST(req: Request) {
  let raw = "";
  try {
    // Read as text with a hard size cap instead of req.json(): a huge or
    // slow body can never exhaust memory here.
    const reader = req.body?.getReader();
    if (!reader) return NextResponse.json({ ok: true });
    const chunks: Uint8Array[] = [];
    let size = 0;
    for (;;) {
      const { done, value } = await reader.read();
      if (done) break;
      size += value.byteLength;
      if (size > MAX_BODY_BYTES) {
        try {
          await reader.cancel();
        } catch {
          /* ignore */
        }
        return NextResponse.json(
          { ok: false, error: "payload too large" },
          { status: 413 }
        );
      }
      chunks.push(value);
    }
    raw = Buffer.concat(chunks).toString("utf8");
  } catch {
    return NextResponse.json({ ok: true });
  }
  if (!raw) return NextResponse.json({ ok: true });

  let body: unknown;
  try {
    body = JSON.parse(raw);
  } catch {
    return NextResponse.json(
      { ok: false, error: "invalid JSON" },
      { status: 400 }
    );
  }
  if (typeof body !== "object" || body === null || Array.isArray(body)) {
    return NextResponse.json(
      { ok: false, error: "malformed payload" },
      { status: 400 }
    );
  }
  const b = body as Record<string, unknown>;
  const pkg = b.package;
  const version = b.version;
  const platform = b.platform;
  const arch = b.arch;
  const node = b.node;

  // Validate each field; reject anything unexpected. The package field is
  // pinned to this project's wrapper name to stop the endpoint being used
  // as a generic logging sink.
  if (pkg !== "warden-sandbox-cli") {
    return NextResponse.json(
      { ok: false, error: "malformed payload" },
      { status: 400 }
    );
  }
  if (
    !isSafeToken(version, 32) ||
    typeof platform !== "string" ||
    !ALLOWED_PLATFORMS.has(platform) ||
    typeof arch !== "string" ||
    !ALLOWED_ARCH.has(arch) ||
    !isNodeVersion(node)
  ) {
    return NextResponse.json(
      { ok: false, error: "malformed payload" },
      { status: 400 }
    );
  }
  // Reject payloads carrying extra keys: the client contract is exactly
  // these five fields, and extras are a sign of misuse or probing.
  const keys = Object.keys(b);
  if (
    keys.length !== 5 ||
    !keys.every((k) =>
      ["package", "version", "platform", "arch", "node"].includes(k)
    )
  ) {
    return NextResponse.json(
      { ok: false, error: "malformed payload" },
      { status: 400 }
    );
  }

  // Persist: one JSONL line. Field names match the telemetry table spec
  // (timestamp, package, version, platform, architecture, node_version).
  // Deliberately no IP/headers/hostname/user-agent here — the Request object
  // exposes them, but we never read them.
  const record = {
    timestamp: new Date().toISOString(),
    package: pkg,
    version,
    platform,
    architecture: arch,
    node_version: node,
  };
  try {
    const file = storePath();
    await fs.mkdir(path.dirname(file), { recursive: true });
    await fs.appendFile(file, JSON.stringify(record) + "\n", "utf8");
  } catch {
    // Storage failure must not leak internals: generic 500, no detail.
    return NextResponse.json({ ok: false }, { status: 500 });
  }

  return NextResponse.json({ ok: true });
}
