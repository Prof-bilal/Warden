"use client";

import { useEffect, useRef, useState } from "react";
import { Zap } from "lucide-react";

/**
 * BoundaryDemo — the full Warden workflow as a looping animation:
 * an AI agent requests a resource → the request travels → Warden's
 * boundary verdicts ALLOW (through to the resource) or DENY (stopped,
 * logged) — flip the switch to see what happens with no sandbox.
 */

const SCENE_MS = 3800;
const VERDICT_AT = 1.05; // seconds into the scene when the boundary verdicts

type Scene = {
  op: string;
  target: string;
  res: string;
  allow: boolean;
};

const SCENES: Scene[] = [
  { op: "fs.read", target: "./workspace/data.json", res: "./workspace", allow: true },
  { op: "fs.read", target: "~/.ssh/id_rsa", res: "~/.ssh", allow: false },
  { op: "net.connect", target: "api.github.com:443", res: "api.github.com", allow: true },
  { op: "env.get", target: "AWS_SECRET_ACCESS_KEY", res: "$AWS_SECRET", allow: false },
  { op: "net.connect", target: "evil.example.io", res: "evil.com", allow: false },
];

const RESOURCES = [
  { label: "./workspace", y: 11.5, sensitive: false },
  { label: "~/.ssh", y: 20, sensitive: true },
  { label: "api.github.com", y: 28.5, sensitive: false },
  { label: "$AWS_SECRET", y: 37, sensitive: true },
  { label: "evil.com", y: 45.5, sensitive: true },
];

function resY(label: string) {
  return RESOURCES.find((r) => r.label === label)?.y ?? 28.5;
}

export default function BoundaryDemo() {
  const [wardenOn, setWardenOn] = useState(true);
  const [idx, setIdx] = useState(0);
  const [log, setLog] = useState<{ text: string; deny: boolean; dim: boolean }[]>([]);
  const logRef = useRef<HTMLDivElement>(null);

  const scene = SCENES[idx % SCENES.length];

  // Scene loop + verdict log timing
  useEffect(() => {
    const t: ReturnType<typeof setTimeout>[] = [];
    t.push(
      setTimeout(() => {
        setLog((prev) => {
          const deny = scene.allow ? false : wardenOn;
          const text = wardenOn
            ? deny
              ? `DENY   ${scene.op}  ${scene.target}  · logged`
              : `ALLOW  ${scene.op}  ${scene.target}`
            : scene.allow
              ? `ALLOW  ${scene.op}  ${scene.target}`
              : `ALLOW  ${scene.op}  ${scene.target}  ⚠ unsandboxed`;
          return [{ text, deny, dim: false }, ...prev].slice(0, 4);
        });
      }, VERDICT_AT * 1000),
    );
    const iv = setInterval(() => setIdx((i) => i + 1), SCENE_MS);
    return () => {
      clearInterval(iv);
      t.forEach(clearTimeout);
    };
  }, [idx, wardenOn, scene]);

  useEffect(() => {
    logRef.current?.scrollTo({ top: 0 });
  }, [log]);

  const ry = resY(scene.res);
  const sceneKey = `${idx}-${wardenOn}`;
  const denied = !scene.allow && wardenOn;

  return (
    <section className="border-t border-ink-800">
      <div className="mx-auto grid max-w-content items-center gap-10 px-6 py-20 md:grid-cols-[22rem_1fr] md:gap-16">
        {/* Left copy */}
        <div>
          <h2 className="text-[1.75rem] font-medium leading-[1.15] tracking-[-0.01em] text-paper">
            MCP has no concept of a boundary.
          </h2>
          <p className="mt-4 text-[1rem] leading-[1.65] text-muted">
            An agent asks for a file, a host, a secret — and the protocol
            happily complies. Flip the switch and watch Warden verdict every
            request at the boundary: granted calls pass, everything else is
            stopped and written down.
          </p>

          {/* Toggle */}
          <button
            onClick={() => setWardenOn((v) => !v)}
            aria-pressed={wardenOn}
            className={
              "mt-7 inline-flex items-center gap-3 rounded-full border px-4 py-2.5 text-[0.875rem] font-medium transition-colors " +
              (wardenOn
                ? "border-grant/50 bg-grant-subtle text-paper hover:border-grant"
                : "border-ink-600 bg-ink-900 text-muted hover:border-ink-500")
            }
          >
            <span
              className={
                "relative h-5 w-9 rounded-full transition-colors " +
                (wardenOn ? "bg-grant" : "bg-ink-600")
              }
            >
              <span
                className={
                  "absolute top-0.5 h-4 w-4 rounded-full bg-white transition-all " +
                  (wardenOn ? "left-[1.125rem]" : "left-0.5")
                }
              />
            </span>
            {wardenOn ? "Warden: ON" : "Warden: OFF"}
            <Zap size={14} className={wardenOn ? "text-grant" : "text-muted"} />
          </button>
        </div>

        {/* Right workflow visualization */}
        <div className="overflow-hidden rounded-[12px] border border-ink-700 bg-ink-900">
          <div className="relative mx-auto aspect-[16/9] w-full">
            {/* Status pill */}
            <div className="absolute left-4 top-4 z-10 flex items-center gap-2 rounded-full border border-white/10 bg-black/60 px-3 py-1 font-mono text-[0.6875rem] backdrop-blur-sm">
              <span className={"h-1.5 w-1.5 rounded-full " + (wardenOn ? "bg-grant animate-pulse" : "bg-deny")} />
              <span className={wardenOn ? "text-grant" : "text-deny"}>
                {wardenOn ? "ENFORCING" : "UNSANDBOXED"}
              </span>
            </div>

            <svg viewBox="0 0 100 57" className="h-full w-full">
              {/* ── Static stage ── */}

              {/* Agent */}
              <rect x="4" y="23" width="16" height="10" rx="1.2" fill="#1E242E" stroke="#6E93E8" strokeWidth="0.45" />
              <text x="12" y="27.2" textAnchor="middle" fontSize="2.3" fill="#E8EBEF" fontFamily="var(--font-geist-mono)">
                AI agent
              </text>
              <text x="12" y="30.6" textAnchor="middle" fontSize="1.7" fill="#8D95A5" fontFamily="var(--font-geist-mono)">
                mcp client
              </text>

              {/* Boundary zone */}
              {wardenOn ? (
                <g>
                  <rect x="36" y="8" width="26" height="41" rx="3" fill="rgba(65,255,84,0.035)" stroke="#3FB27E" strokeWidth="0.45" strokeDasharray="1.7 1.3">
                    <animate attributeName="stroke-dashoffset" from="0" to="6" dur="2.4s" repeatCount="indefinite" />
                  </rect>
                  <text x="49" y="5.6" textAnchor="middle" fontSize="2" fill="#3FB27E" fontFamily="var(--font-geist-mono)" letterSpacing="0.35">
                    WARDEN
                  </text>
                </g>
              ) : (
                <g opacity="0.3">
                  <rect x="36" y="8" width="26" height="41" rx="3" fill="none" stroke="#E2604F" strokeWidth="0.35" strokeDasharray="1 1.4" />
                  <text x="49" y="5.6" textAnchor="middle" fontSize="2" fill="#E2604F" fontFamily="var(--font-geist-mono)" letterSpacing="0.35">
                    NO SANDBOX
                  </text>
                </g>
              )}

              {/* Server inside the zone */}
              <rect x="43" y="24.5" width="12" height="7" rx="1" fill="#1E242E" stroke={wardenOn ? "#3FB27E" : "#4A5568"} strokeWidth="0.45" />
              <text x="49" y="28.7" textAnchor="middle" fontSize="2.3" fill="#E8EBEF" fontFamily="var(--font-geist-mono)">
                server
              </text>
              {wardenOn && (
                <text x="49" y="32.6" textAnchor="middle" fontSize="1.6" fill="#3FB27E" fontFamily="var(--font-geist-mono)">
                  sandboxed
                </text>
              )}

              {/* Resources column */}
              {RESOURCES.map((r) => (
                <g key={r.label}>
                  <circle cx="79" cy={r.y - 0.7} r="0.9" fill={r.sensitive ? "#E2604F" : "#3FB27E"} opacity="0.85" />
                  <text x="82" y={r.y} fontSize="2.1" fill={r.sensitive ? "#C9705F" : "#3FB27E"} fontFamily="var(--font-geist-mono)">
                    {r.label}
                  </text>
                </g>
              ))}
              <text x="82" y="50.5" fontSize="1.6" fill="#8D95A5" fontFamily="var(--font-geist-mono)">
                host resources
              </text>

              {/* ── Per-scene animation (remounted each scene) ── */}
              <g key={sceneKey}>
                {/* lane: agent → boundary, always visible (ON) */}
                {wardenOn && (
                  <line x1="20" y1="28" x2="35.5" y2="28" stroke={denied ? "#E2604F" : "#6E93E8"} strokeWidth="0.35" opacity="0.4" strokeDasharray="0.9 1.1">
                    <animate attributeName="opacity" values="0;0.4;0.4;0.25" keyTimes="0;0.1;0.85;1" dur={`${SCENE_MS / 1000}s`} fill="freeze" />
                  </line>
                )}
                {/* lane: server → resource, always visible (ON) */}
                {wardenOn && (
                  <line x1="55.5" y1="28" x2="76.5" y2={ry} stroke={scene.allow ? "#3FB27E" : "#4A5568"} strokeWidth="0.35" opacity={scene.allow ? "0.4" : "0.22"} strokeDasharray={scene.allow ? "0.9 1.1" : "0.6 1.4"}>
                    <animate attributeName="opacity" values={scene.allow ? "0;0.4;0.4;0.25" : "0;0.22;0.22;0.12"} keyTimes="0;0.1;0.85;1" dur={`${SCENE_MS / 1000}s`} fill="freeze" />
                  </line>
                )}
                {/* lane: agent → resource straight through (OFF) */}
                {!wardenOn && (
                  <line x1="20" y1="28" x2="76.5" y2={ry} stroke={scene.allow ? "#3FB27E" : "#E2604F"} strokeWidth="0.35" opacity="0.4" strokeDasharray="0.9 1.1">
                    <animate attributeName="opacity" values="0;0.4;0.4;0.25" keyTimes="0;0.1;0.85;1" dur={`${SCENE_MS / 1000}s`} fill="freeze" />
                  </line>
                )}

                {/* request label above agent */}
                <text x="12" y="19.5" textAnchor="middle" fontSize="1.8" fill="#8D95A5" fontFamily="var(--font-geist-mono)">
                  {scene.op}
                  <animate attributeName="opacity" values="0;1;1;0.85" keyTimes="0;0.08;0.9;1" dur={`${SCENE_MS / 1000}s`} fill="freeze" />
                </text>
                <text x="12" y="36.5" textAnchor="middle" fontSize="1.8" fill={denied ? "#E2604F" : scene.allow ? "#3FB27E" : "#E2604F"} fontFamily="var(--font-geist-mono)">
                  {scene.target.length > 18 ? scene.target.slice(0, 17) + "…" : scene.target}
                  <animate attributeName="opacity" values="0;1;1;0.85" keyTimes="0;0.08;0.9;1" dur={`${SCENE_MS / 1000}s`} fill="freeze" />
                </text>

                {/* request packet: agent → boundary (ON) */}
                {wardenOn && (
                  <circle r="1.3" fill={denied ? "#E2604F" : "#6E93E8"} opacity="0">
                    <animateMotion path="M20,28 L35.5,28" begin="0.15s" dur="0.9s" fill="freeze" />
                    <animate attributeName="opacity" values="0;1;1;1" keyTimes="0;0.05;0.93;1" dur={`${SCENE_MS / 1000}s`} fill="freeze" />
                    {denied && (
                      <animate attributeName="fill" begin="1.05s" dur="0.01s" fill="freeze" from="#6E93E8" to="#E2604F" />
                    )}
                    {/* fade packet right before scene ends */}
                    <animate attributeName="opacity" begin={`${SCENE_MS / 1000 - 0.4}s`} dur="0.35s" fill="freeze" from="1" to="0" />
                  </circle>
                )}

                {/* request packet: agent → resource, straight through (OFF) */}
                {!wardenOn && (
                  <circle r="1.3" fill={scene.allow ? "#3FB27E" : "#E2604F"} opacity="0">
                    <animateMotion path={`M20,28 L76.5,${ry}`} begin="0.15s" dur="1.7s" fill="freeze" />
                    <animate attributeName="opacity" values="0;1;1;1" keyTimes="0;0.05;0.93;1" dur={`${SCENE_MS / 1000}s`} fill="freeze" />
                    <animate attributeName="opacity" begin={`${SCENE_MS / 1000 - 0.4}s`} dur="0.35s" fill="freeze" from="1" to="0" />
                  </circle>
                )}

                {/* verdict at the boundary (ON) */}
                {wardenOn && denied && (
                  <g>
                    {/* impact ripple */}
                    <circle cx="36" cy="28" r="1" fill="none" stroke="#E2604F" strokeWidth="0.4">
                      <animate attributeName="r" begin="1.05s" dur="0.55s" fill="freeze" from="1" to="5.5" />
                      <animate attributeName="opacity" begin="1.05s" dur="0.55s" fill="freeze" from="0.9" to="0" />
                    </circle>
                    {/* X burst */}
                    <g stroke="#E2604F" strokeWidth="0.55" opacity="0">
                      <line x1="34.6" y1="26.6" x2="37.4" y2="29.4" />
                      <line x1="34.6" y1="29.4" x2="37.4" y2="26.6" />
                      <animate attributeName="opacity" begin="1.05s" dur="0.18s" fill="freeze" from="0" to="1" />
                    </g>
                    {/* logged tag */}
                    <text x="36" y="36.5" textAnchor="middle" fontSize="1.7" fill="#E2604F" fontFamily="var(--font-geist-mono)" opacity="0">
                      blocked · logged
                      <animate attributeName="opacity" begin="1.3s" dur="0.25s" fill="freeze" from="0" to="1" />
                    </text>
                  </g>
                )}
                {wardenOn && !denied && (
                  <g>
                    {/* boundary check stamp */}
                    <text x="36" y="26.2" textAnchor="middle" fontSize="2.6" fill="#3FB27E" fontFamily="var(--font-geist-mono)" opacity="0">
                      ✓
                      <animate attributeName="opacity" begin="1.05s" dur="0.2s" fill="freeze" from="0" to="1" />
                    </text>
                    {/* continue packet: server → resource */}
                    <circle r="1.2" fill="#3FB27E" opacity="0">
                      <animateMotion path={`M56,28 L76.5,${ry}`} begin="1.25s" dur="0.85s" fill="freeze" />
                      <animate attributeName="opacity" begin="1.25s" dur="0.1s" fill="freeze" from="0" to="1" />
                      <animate attributeName="opacity" begin={`${SCENE_MS / 1000 - 0.4}s`} dur="0.35s" fill="freeze" from="1" to="0" />
                    </circle>
                    {/* arrival ping */}
                    <circle cx="77.5" cy={ry - 0.7} r="1" fill="none" stroke="#3FB27E" strokeWidth="0.4" opacity="0">
                      <animate attributeName="r" begin="2.1s" dur="0.5s" fill="freeze" from="1" to="4" />
                      <animate attributeName="opacity" begin="2.1s" dur="0.5s" fill="freeze" from="0.9" to="0" />
                    </circle>
                  </g>
                )}

                {/* OFF-mode arrival: dangerous resource gets ⚠ */}
                {!wardenOn && !scene.allow && (
                  <g opacity="0">
                    <text x="77.5" y={ry - 3} fontSize="1.9" fill="#E2604F" fontFamily="var(--font-geist-mono)">
                      full access!
                      <animate attributeName="opacity" begin="1.95s" dur="0.25s" fill="freeze" from="0" to="1" />
                    </text>
                    <circle cx="77.5" cy={ry - 0.7} r="1" fill="none" stroke="#E2604F" strokeWidth="0.4">
                      <animate attributeName="r" begin="1.9s" dur="0.5s" fill="freeze" from="1" to="4" />
                      <animate attributeName="opacity" begin="1.9s" dur="0.5s" fill="freeze" from="0.9" to="0" />
                    </circle>
                    <animate attributeName="opacity" begin="1.95s" dur="0.2s" fill="freeze" from="0" to="1" />
                  </g>
                )}
              </g>
            </svg>
          </div>

          {/* Live audit log strip */}
          <div
            ref={logRef}
            className="h-[7.5rem] overflow-hidden border-t border-ink-700 bg-ink-950 px-5 py-3 font-mono text-[0.75rem] leading-[1.7]"
          >
            {log.length === 0 && <p className="text-muted">agent connecting…</p>}
            {log.map((line, i) => (
              <p
                key={`${line.text}-${i}`}
                className={
                  line.dim
                    ? "text-muted"
                    : line.text.startsWith("DENY")
                      ? "text-deny"
                      : line.text.includes("⚠")
                        ? "text-progress"
                        : "text-grant"
                }
                style={{ opacity: 1 - i * 0.18 }}
              >
                {line.text}
              </p>
            ))}
          </div>
        </div>
      </div>
    </section>
  );
}
