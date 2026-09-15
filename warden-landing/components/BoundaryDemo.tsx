"use client";

import { useEffect, useRef, useState } from "react";
import {
  Bot,
  Check,
  FileCode2,
  Globe,
  KeyRound,
  Pause,
  Play,
  RotateCcw,
  ShieldCheck,
  ShieldX,
  X,
} from "lucide-react";

/**
 * BoundaryDemoredesigned interactive Warden boundary console.
 * Pattern (from security-demo research: PipeLab/Curate-Me style):
 * scenario picker → readable 3-node pipeline (agent → gate → resource)
 * → big verdict card → audit log + counters. Same JetBrains Mono
 * hero font, Warden ink/grant/deny palette.
 */

const SCENE_MS = 4200;
const VERDICT_AT = 1150;

type Scene = {
  op: string;
  target: string;
  short: string;
  resource: string;
  rule: string;
  allow: boolean;
  sensitive: boolean;
};

const SCENES: Scene[] = [
  {
    op: "fs.read",
    target: "./workspace/data.json",
    short: "data.json",
    resource: "./workspace",
    rule: "fs.allow ./workspace/**",
    allow: true,
    sensitive: false,
  },
  {
    op: "fs.read",
    target: "~/.ssh/id_rsa",
    short: "id_rsa",
    resource: "~/.ssh",
    rule: "no matching allow rule",
    allow: false,
    sensitive: true,
  },
  {
    op: "net.connect",
    target: "api.github.com:443",
    short: "github:443",
    resource: "api.github.com",
    rule: "net.allow api.github.com:443",
    allow: true,
    sensitive: false,
  },
  {
    op: "env.get",
    target: "AWS_SECRET_ACCESS_KEY",
    short: "AWS_SECRET",
    resource: "$AWS_SECRET",
    rule: "no matching allow rule",
    allow: false,
    sensitive: true,
  },
  {
    op: "net.connect",
    target: "evil.example.io",
    short: "evil.example.io",
    resource: "evil.com",
    rule: "net.deny * (default)",
    allow: false,
    sensitive: true,
  },
];

type LogLine = {
  id: number;
  kind: "allow" | "deny" | "warn";
  text: string;
};

function ResourceIcon({ scene }: { scene: Scene }) {
  const cls = "text-muted";
  if (scene.op.startsWith("fs")) return <FileCode2 size={16} className={cls} />;
  if (scene.op.startsWith("env")) return <KeyRound size={16} className={cls} />;
  return <Globe size={16} className={cls} />;
}

export default function BoundaryDemo() {
  const [wardenOn, setWardenOn] = useState(true);
  const [idx, setIdx] = useState(0);
  const [playing, setPlaying] = useState(true);
  const [runId, setRunId] = useState(0);
  const [showVerdict, setShowVerdict] = useState(false);
  const [log, setLog] = useState<LogLine[]>([]);
  const [stats, setStats] = useState({ allow: 0, deny: 0, exposed: 0 });
  const idRef = useRef(0);

  const scene = SCENES[idx % SCENES.length];
  const denied = !scene.allow && wardenOn;
  const exposed = !scene.allow && !wardenOn; // sensitive request passes with no sandbox
  const runKey = `${idx}-${wardenOn}-${runId}`;

  // auto-advance scenes
  useEffect(() => {
    if (!playing) return;
    const iv = setInterval(() => setIdx((i) => (i + 1) % SCENES.length), SCENE_MS);
    return () => clearInterval(iv);
  }, [playing, runId]);

  // verdict timing per scene: reveal card + append audit log + counters
  useEffect(() => {
    setShowVerdict(false);
    const t = setTimeout(() => {
      setShowVerdict(true);
      idRef.current += 1;
      const id = idRef.current;
      const kind: LogLine["kind"] = denied ? "deny" : exposed ? "warn" : "allow";
      const text =
        denied || exposed
          ? `${denied ? "DENY" : "ALLOW"}  ${scene.op}  ${scene.target}${exposed ? "  ⚠ unsandboxed" : "  · logged"}`
          : `ALLOW  ${scene.op}  ${scene.target}`;
      setLog((prev) => [{ id, kind, text }, ...prev].slice(0, 5));
      setStats((prev) => ({
        allow: prev.allow + (denied ? 0 : 1),
        deny: prev.deny + (denied ? 1 : 0),
        exposed: prev.exposed + (exposed ? 1 : 0),
      }));
    }, VERDICT_AT);
    return () => clearTimeout(t);
  }, [runKey]); // eslint-disable-line react-hooks/exhaustive-deps

  function pick(i: number) {
    setIdx(i);
    setRunId((r) => r + 1);
    setPlaying(true);
  }

  function toggleWarden() {
    setWardenOn((v) => !v);
    setRunId((r) => r + 1);
  }

  function replay() {
    setLog([]);
    setStats({ allow: 0, deny: 0, exposed: 0 });
    idRef.current = 0;
    setIdx(0);
    setRunId((r) => r + 1);
    setPlaying(true);
  }

  const packetAnim = !wardenOn
    ? "bd-packet-through"
    : denied
      ? "bd-packet-stop"
      : "bd-packet-pass";
  const packetColor = !wardenOn
    ? scene.allow
      ? "#3FB27E"
      : "#E2604F"
    : denied
      ? "#E2604F"
      : "#6E93E8";

  return (
    <section className="border-t border-ink-800 bg-ink-950">
      <div className="mx-auto max-w-content px-6 py-20 md:py-28">
        {/* header */}
        <div className="mx-auto max-w-[640px] text-center">
          <p className="mb-4 font-hero text-[12px] font-bold uppercase tracking-[0.12em] text-grant">
            Live boundary demo
          </p>
          <h2 className="font-hero text-[1.75rem] font-bold leading-[1.15] tracking-[-0.02em] text-paper md:text-[2.25rem]">
            MCP has no concept of a boundary.
          </h2>
          <p className="mt-4 font-hero text-[15px] leading-[1.7] text-muted md:text-[16px]">
            An agent asks for a file, a host, a secretand the protocol happily complies. Pick a
            request, flip the switch, and watch Warden verdict every call at the boundary.
          </p>
        </div>

        {/* console */}
        <div className="mx-auto mt-10 max-w-4xl overflow-hidden rounded-2xl border border-ink-700 bg-ink-900">
          {/* control bar */}
          <div className="flex flex-wrap items-center gap-3 border-b border-ink-700 bg-ink-950/60 px-4 py-3 md:px-5">
            <span
              className={
                "inline-flex items-center gap-2 rounded-full border px-3 py-1 font-hero text-[11px] font-bold tracking-[0.08em] " +
                (wardenOn
                  ? "border-grant/40 bg-grant-subtle text-grant"
                  : "border-deny/40 bg-deny-subtle text-deny")
              }
            >
              <span
                className={"h-1.5 w-1.5 rounded-full " + (wardenOn ? "bg-grant" : "bg-deny")}
              />
              {wardenOn ? "ENFORCING" : "UNSANDBOXED"}
            </span>

            <button
              onClick={toggleWarden}
              aria-pressed={wardenOn}
              className={
                "inline-flex items-center gap-2.5 rounded-full border px-3 py-1.5 font-hero text-[12px] font-semibold transition-colors " +
                (wardenOn
                  ? "border-grant/50 bg-grant-subtle text-paper hover:border-grant"
                  : "border-ink-600 bg-ink-900 text-muted hover:border-ink-500")
              }
            >
              <span
                className={
                  "relative h-4 w-8 rounded-full transition-colors " +
                  (wardenOn ? "bg-grant" : "bg-ink-600")
                }
              >
                <span
                  className={
                    "absolute top-0.5 h-3 w-3 rounded-full bg-white transition-all " +
                    (wardenOn ? "left-[1.0625rem]" : "left-0.5")
                  }
                />
              </span>
              {wardenOn ? "Warden: ON" : "Warden: OFF"}
            </button>

            <div className="ml-auto flex items-center gap-1.5">
              <button
                onClick={() => setPlaying((p) => !p)}
                aria-label={playing ? "Pause demo" : "Play demo"}
                className="flex h-8 w-8 items-center justify-center rounded-lg text-muted transition-colors hover:bg-ink-800 hover:text-paper"
              >
                {playing ? <Pause size={15} /> : <Play size={15} />}
              </button>
              <button
                onClick={replay}
                aria-label="Replay demo"
                className="flex h-8 w-8 items-center justify-center rounded-lg text-muted transition-colors hover:bg-ink-800 hover:text-paper"
              >
                <RotateCcw size={15} />
              </button>
            </div>
          </div>

          {/* scene progress */}
          <div className="h-[2px] bg-ink-800">
            <div
              key={runKey + (playing ? "-play" : "-pause")}
              className="h-full bg-grant/70"
              style={{
                animation: `bd-progress-fill ${SCENE_MS}ms linear forwards`,
                animationPlayState: playing ? "running" : "paused",
              }}
            />
          </div>

          {/* scenario picker */}
          <div className="flex gap-2 overflow-x-auto border-b border-ink-700 px-4 py-3 md:px-5">
            {SCENES.map((s, i) => {
              const active = i === idx % SCENES.length;
              return (
                <button
                  key={s.short}
                  onClick={() => pick(i)}
                  className={
                    "flex shrink-0 items-center gap-2 rounded-full border px-3 py-1.5 font-hero text-[12px] transition-colors " +
                    (active
                      ? "border-blueprint/60 bg-blueprint/10 text-paper"
                      : "border-ink-700 bg-ink-950 text-muted hover:border-ink-500 hover:text-paper")
                  }
                >
                  <span
                    className="h-1.5 w-1.5 rounded-full"
                    style={{ background: s.allow ? "#3FB27E" : "#E2604F" }}
                  />
                  <span className="text-muted">{s.op}</span>
                  <span>{s.short}</span>
                </button>
              );
            })}
          </div>

          <div className="px-4 py-6 md:px-8 md:py-8">
            {/* pipeline: agent → gate → resource */}
            <div className="grid grid-cols-[1fr_auto_1fr] items-stretch gap-2 md:gap-4">
              {/* agent */}
              <div className="min-w-0 rounded-xl border border-blueprint/40 bg-ink-950 p-3 text-center md:p-4">
                <Bot size={18} className="mx-auto text-blueprint" />
                <p className="mt-2 truncate font-hero text-[13px] font-bold text-paper md:text-[14px]">
                  AI agent
                </p>
                <p className="font-hero text-[11px] text-muted">mcp client</p>
                <p className="mt-2 truncate font-hero text-[11px] text-blueprint md:text-[12px]">
                  {scene.op}
                </p>
              </div>

              {/* gate */}
              <div className="flex flex-col items-center justify-center px-1">
                <div
                  className={
                    "flex h-16 w-16 items-center justify-center rounded-2xl border-2 border-dashed transition-colors md:h-20 md:w-20 " +
                    (wardenOn
                      ? "border-grant/70 bg-grant-subtle"
                      : "border-deny/40 bg-deny-subtle opacity-70") +
                    (wardenOn && showVerdict && denied ? " motion-safe:animate-[bd-deny-shake_0.4s_ease]" : "")
                  }
                  style={wardenOn ? { animation: "bd-gate-glow 2.4s ease-in-out infinite" } : undefined}
                >
                  {wardenOn ? (
                    showVerdict ? (
                      denied ? (
                        <X size={26} className="text-deny" />
                      ) : (
                        <Check size={26} className="text-grant" />
                      )
                    ) : (
                      <ShieldCheck size={26} className="text-grant" />
                    )
                  ) : (
                    <ShieldX size={26} className="text-deny/70" />
                  )}
                </div>
                <p
                  className={
                    "mt-2 font-hero text-[10px] font-bold tracking-[0.2em] " +
                    (wardenOn ? "text-grant" : "text-deny/70")
                  }
                >
                  {wardenOn ? "WARDEN" : "NO GATE"}
                </p>
              </div>

              {/* resource */}
              <div
                className={
                  "min-w-0 rounded-xl border bg-ink-950 p-3 text-center md:p-4 " +
                  (scene.sensitive ? "border-deny/40" : "border-grant/40")
                }
              >
                <div className="mx-auto flex justify-center">
                  <ResourceIcon scene={scene} />
                </div>
                <p className="mt-2 truncate font-hero text-[13px] font-bold text-paper md:text-[14px]">
                  {scene.resource}
                </p>
                <p
                  className={
                    "font-hero text-[11px] " + (scene.sensitive ? "text-deny" : "text-grant")
                  }
                >
                  {scene.sensitive ? "sensitive" : "granted"}
                </p>
                <p className="mt-2 truncate font-hero text-[11px] text-muted md:text-[12px]">
                  {scene.target}
                </p>
              </div>
            </div>

            {/* packet track */}
            <div className="relative mx-[11%] mt-1 h-6" aria-hidden>
              <div className="absolute left-0 right-0 top-1/2 h-px -translate-y-1/2 bg-ink-600" />
              <div
                key={"pkt-" + runKey}
                className="absolute top-1/2 h-2.5 w-2.5 -translate-x-1/2 -translate-y-1/2 rounded-full motion-reduce:animate-none"
                style={{
                  background: packetColor,
                  boxShadow: `0 0 12px 2px ${packetColor}66`,
                  animation: `${packetAnim} ${SCENE_MS}ms linear forwards`,
                  animationPlayState: playing ? "running" : "paused",
                }}
              />
            </div>

            {/* verdict card */}
            <div className="mt-2 min-h-[86px]">
              {showVerdict ? (
                <div
                  key={"v-" + runKey}
                  className={
                    "rounded-xl border px-4 py-3 motion-safe:animate-[bd-verdict-pop_0.3s_ease] " +
                    (denied
                      ? "border-deny/50 bg-deny-subtle"
                      : exposed
                        ? "border-progress/50 bg-progress-subtle"
                        : "border-grant/50 bg-grant-subtle")
                  }
                >
                  <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
                    <span
                      className={
                        "rounded px-1.5 py-0.5 font-hero text-[12px] font-bold tracking-[0.06em] " +
                        (denied
                          ? "bg-deny text-ink-950"
                          : exposed
                            ? "bg-progress text-ink-950"
                            : "bg-grant text-ink-950")
                      }
                    >
                      {denied ? "DENY" : exposed ? "LEAKED" : "ALLOW"}
                    </span>
                    <code className="truncate font-hero text-[13px] text-paper md:text-[14px]">
                      {scene.op} {scene.target}
                    </code>
                  </div>
                  <p className="mt-1.5 font-hero text-[12px] text-muted">
                    {denied && (
                      <>
                        blocked at boundary · {scene.rule} ·{" "}
                        <span className="text-deny">logged</span>
                      </>
                    )}
                    {exposed && (
                      <>
                        no sandboxsensitive {scene.op} passed ·{" "}
                        <span className="text-progress">would be denied by Warden</span>
                      </>
                    )}
                    {!denied && !exposed && (
                      <>
                        {scene.rule} · <span className="text-grant">1.2ms · logged</span>
                      </>
                    )}
                  </p>
                </div>
              ) : (
                <div className="rounded-xl border border-dashed border-ink-700 px-4 py-3">
                  <p className="font-hero text-[12px] text-muted">
                    <span className="motion-safe:animate-pulse">●</span> evaluating {scene.op} at
                    the boundary…
                  </p>
                </div>
              )}
            </div>

            {/* bottom split: audit log + counters */}
            <div className="mt-4 grid gap-4 md:grid-cols-[1fr_12rem]">
              <div className="min-h-[9.5rem] rounded-xl border border-ink-700 bg-ink-950 px-4 py-3 font-hero text-[12px] leading-[1.9]">
                <p className="mb-1 text-[10px] font-bold uppercase tracking-[0.14em] text-muted">
                  Audit log
                </p>
                {log.length === 0 && <p className="text-muted">agent connecting…</p>}
                {log.map((line) => (
                  <p
                    key={line.id}
                    className={
                      "truncate motion-safe:animate-[bd-log-in_0.25s_ease] " +
                      (line.kind === "deny"
                        ? "text-deny"
                        : line.kind === "warn"
                          ? "text-progress"
                          : "text-grant")
                    }
                  >
                    {line.text}
                  </p>
                ))}
              </div>
              <div className="flex flex-row gap-3 md:flex-col">
                {[
                  { label: "allowed", value: stats.allow, cls: "text-grant" },
                  { label: "denied", value: stats.deny, cls: "text-deny" },
                  { label: "exposed", value: stats.exposed, cls: "text-progress" },
                ].map((s) => (
                  <div
                    key={s.label}
                    className="flex-1 rounded-xl border border-ink-700 bg-ink-950 px-4 py-3 text-center"
                  >
                    <p className={"font-hero text-2xl font-bold " + s.cls}>{s.value}</p>
                    <p className="font-hero text-[10px] uppercase tracking-[0.14em] text-muted">
                      {s.label}
                    </p>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
