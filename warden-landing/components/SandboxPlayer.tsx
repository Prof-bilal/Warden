"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { Pause, Play, RotateCcw, Film } from "lucide-react";

/**
 * SandboxPlayer — a Remotion-style "video" rendered entirely in React.
 * Every frame is derived from a timeline value `t` (seconds), so the
 * animation is seekable, loops seamlessly, and ships zero video bytes.
 */

const DURATION = 24;

type Event = { t: number; op: string; target: string; allow: boolean };

const EVENTS: Event[] = [
  { t: 8.4, op: "fs.read", target: "./workspace/data.json", allow: true },
  { t: 9.8, op: "fs.read", target: "~/.ssh/id_rsa", allow: false },
  { t: 11.2, op: "net.connect", target: "api.github.com:443", allow: true },
  { t: 12.6, op: "net.connect", target: "evil.example.io", allow: false },
  { t: 14.0, op: "env.get", target: "AWS_SECRET_ACCESS_KEY", allow: false },
  { t: 15.4, op: "fs.write", target: "./workspace/output.csv", allow: true },
  { t: 16.8, op: "net.dns", target: "malware.test", allow: false },
  { t: 18.2, op: "fs.write", target: "/etc/hosts", allow: false },
];

const BOOT_LINES = [
  { t: 5.0, text: "[sandbox] backend: bubblewrap (unprivileged namespace)", tone: "dim" },
  { t: 5.7, text: "[grant]   fs.read  ./workspace", tone: "ok" },
  { t: 6.3, text: "[grant]   fs.write ./workspace/output", tone: "ok" },
  { t: 6.9, text: "[grant]   net      api.github.com:443", tone: "ok" },
  { t: 7.5, text: "[deny]    everything else → invisible", tone: "bad" },
] as const;

const POLICY_LINES = [
  "filesystem:",
  "  read:  [./workspace]",
  "  write: [./workspace/output]",
  "network:",
  "  allow: [api.github.com:443]",
  "env:",
  "  allow: [GITHUB_TOKEN]",
] as const;

const clamp01 = (x: number) => Math.min(1, Math.max(0, x));
const easeOut = (x: number) => 1 - Math.pow(1 - x, 3);

function fmt(t: number) {
  const s = Math.floor(t);
  const d = Math.floor((t - s) * 10);
  return `0:${String(s).padStart(2, "0")}.${d}`;
}

export default function SandboxPlayer() {
  const [t, setT] = useState(0);
  const [playing, setPlaying] = useState(true);
  const rafRef = useRef<number | null>(null);
  const lastRef = useRef<number | null>(null);
  const trackRef = useRef<HTMLDivElement>(null);

  // rAF timeline loop
  useEffect(() => {
    if (!playing) {
      lastRef.current = null;
      return;
    }
    const step = (now: number) => {
      if (lastRef.current === null) lastRef.current = now;
      const dt = (now - lastRef.current) / 1000;
      lastRef.current = now;
      setT((prev) => (prev + dt) % DURATION);
      rafRef.current = requestAnimationFrame(step);
    };
    rafRef.current = requestAnimationFrame(step);
    return () => {
      if (rafRef.current !== null) cancelAnimationFrame(rafRef.current);
      lastRef.current = null;
    };
  }, [playing]);

  const seek = useCallback((clientX: number) => {
    const el = trackRef.current;
    if (!el) return;
    const rect = el.getBoundingClientRect();
    const frac = clamp01((clientX - rect.left) / rect.width);
    setT(frac * DURATION);
  }, []);

  const onTrackDown = useCallback(
    (e: React.PointerEvent) => {
      seek(e.clientX);
      const move = (ev: PointerEvent) => seek(ev.clientX);
      const up = () => {
        window.removeEventListener("pointermove", move);
        window.removeEventListener("pointerup", up);
      };
      window.addEventListener("pointermove", move);
      window.addEventListener("pointerup", up);
    },
    [seek],
  );

  // Restart loop cleanly
  function restart() {
    setT(0);
    setPlaying(true);
  }

  // ── Scene computation (pure functions of t) ────────────────────
  const typing = clamp01(t / 1.6);
  const cmd = "warden init".slice(0, Math.floor(typing * "warden init".length));
  const policyReveal = clamp01((t - 1.9) / 2.4);
  const policyCount = Math.floor(easeOut(policyReveal) * POLICY_LINES.length);
  const bootVisible = BOOT_LINES.filter((l) => t >= l.t);
  const eventsVisible = EVENTS.filter((e) => t >= e.t).slice(-6);
  const summaryIn = clamp01((t - 19.4) / 0.7);
  const summaryVisible = t >= 19.4;
  const sceneIdx = t < 4.6 ? 1 : t < 8 ? 2 : summaryVisible ? 4 : 3;

  // pulse ring scale from t (deterministic, loops)
  const pulse = 1 + 0.04 * Math.sin(t * 2.1);
  const sweep = (t % 3) / 3;

  return (
    <div className="overflow-hidden rounded-[12px] border border-ink-700 bg-ink-950">
      {/* Window chrome */}
      <div className="flex items-center gap-2 border-b border-ink-800 bg-ink-900 px-4 py-2.5">
        <span className="h-2.5 w-2.5 rounded-full bg-[#E2604F]/70" />
        <span className="h-2.5 w-2.5 rounded-full bg-[#F59E0B]/70" />
        <span className="h-2.5 w-2.5 rounded-full bg-[#3FB27E]/70" />
        <span className="ml-3 font-mono text-[0.6875rem] text-muted">
          warden-demo · rendered in React · {DURATION}s loop
        </span>
        <span className="ml-auto inline-flex items-center gap-1.5 rounded-full border border-ink-700 px-2 py-0.5 font-mono text-[0.625rem] uppercase tracking-[0.08em] text-muted">
          <Film size={10} />
          Scene {sceneIdx}/4
        </span>
      </div>

      {/* Stage */}
      <div
        className="relative aspect-[16/8.2] w-full cursor-default select-none bg-[#0A0E14]"
        onClick={() => setPlaying((v) => !v)}
        role="button"
        aria-label={playing ? "Pause animation" : "Play animation"}
      >
        {/* faint grid + sweep */}
        <div
          className="pointer-events-none absolute inset-0 opacity-[0.5]"
          style={{
            backgroundImage:
              "linear-gradient(to right, rgba(110,147,232,0.05) 1px, transparent 1px), linear-gradient(to bottom, rgba(110,147,232,0.05) 1px, transparent 1px)",
            backgroundSize: "28px 28px",
          }}
          aria-hidden
        />
        <div
          className="pointer-events-none absolute inset-y-0 w-24 bg-gradient-to-r from-transparent via-[#6E93E8]/[0.045] to-transparent"
          style={{ left: `${sweep * 110 - 10}%` }}
          aria-hidden
        />

        {/* Scene 1+2 — left: terminal */}
        <div className="absolute inset-0 flex flex-col p-5 font-mono text-[0.8125rem] leading-[1.9] sm:p-7 sm:text-[0.875rem]">
          <div className="min-h-0 flex-1 overflow-hidden">
            {/* typed command */}
            <p>
              <span className="text-[#3FB27E]">$ </span>
              <span className="text-paper">{cmd}</span>
              {t < 1.8 && <span className="text-[#3FB27E]">▌</span>}
            </p>

            {/* policy yaml streams in */}
            {policyCount > 0 && (
              <div className="mt-1 border-l-2 border-[#6E93E8]/40 pl-3">
                <p className="text-muted"># .warden/policy.yaml</p>
                {POLICY_LINES.slice(0, policyCount).map((l, i) => (
                  <p key={l} style={{ opacity: 0.55 + 0.45 * easeOut(clamp01(policyReveal * POLICY_LINES.length - i)) }}>
                    <span className="text-[#6E93E8]">{l.split(":")[0]}</span>
                    <span className="text-muted">:</span>
                    <span className="text-paper/85">{l.split(":").slice(1).join(":")}</span>
                  </p>
                ))}
              </div>
            )}

            {/* run command + boot */}
            {t >= 4.6 && (
              <p className="mt-1">
                <span className="text-[#3FB27E]">$ </span>
                <span className="text-paper">warden run --policy .warden/policy.yaml -- npx server</span>
              </p>
            )}
            {bootVisible.map((l) => (
              <p key={l.text} className={l.tone === "ok" ? "text-[#3FB27E]" : l.tone === "bad" ? "text-[#E2604F]" : "text-muted"}>
                {l.text}
              </p>
            ))}
          </div>

          {/* verdict stream — pinned bottom */}
          <div className="flex min-h-[9.5rem] flex-col justify-end gap-[2px] overflow-hidden pt-2">
            {eventsVisible.map((e) => (
              <p key={e.target} className="flex items-center gap-2 truncate">
                <span className="text-muted/70">{fmt(e.t)}</span>
                <span
                  className={
                    "inline-block w-12 rounded-sm px-1 text-center text-[0.625rem] font-semibold tracking-wider " +
                    (e.allow ? "bg-[#3FB27E]/15 text-[#3FB27E]" : "bg-[#E2604F]/15 text-[#E2604F]")
                  }
                >
                  {e.allow ? "ALLOW" : "DENY"}
                </span>
                <span className="text-[#6E93E8]">{e.op}</span>
                <span className="text-paper/85">{e.target}</span>
              </p>
            ))}
          </div>
        </div>

        {/* Boundary pulse — right side, scenes 2–4 */}
        {t >= 5 && (
          <div className="pointer-events-none absolute right-[6%] top-1/2 hidden -translate-y-1/2 md:block" aria-hidden>
            <div
              className="relative h-40 w-40 rounded-full border border-dashed border-[#3FB27E]/50 lg:h-48 lg:w-48"
              style={{ transform: `scale(${pulse})` }}
            >
              <span className="absolute -top-5 left-1/2 -translate-x-1/2 whitespace-nowrap font-mono text-[0.5625rem] tracking-[0.2em] text-[#3FB27E]/80">
                BOUNDARY
              </span>
              <span className="absolute left-1/2 top-1/2 h-10 w-10 -translate-x-1/2 -translate-y-1/2 rounded-md border border-[#6E93E8]/50 bg-ink-900" />
              {/* denied hit marker */}
              {EVENTS.some((e) => !e.allow && t - e.t < 0.9 && t >= e.t) && (
                <span className="absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 text-lg font-bold text-[#E2604F]" style={{ opacity: 0.9 }}>
                  ✕
                </span>
              )}
            </div>
          </div>
        )}

        {/* Summary card — scene 4 */}
        {summaryVisible && (
          <div
            className="absolute left-1/2 top-6 -translate-x-1/2 rounded-lg border border-[#3FB27E]/40 bg-black/70 px-5 py-2.5 text-center backdrop-blur-sm"
            style={{ opacity: easeOut(summaryIn), transform: `translate(-50%, ${(1 - easeOut(summaryIn)) * -12}px)` }}
          >
            <p className="font-mono text-[0.8125rem] text-[#3FB27E]">3 allowed · 5 denied · 0 escapes</p>
            <p className="mt-0.5 font-mono text-[0.625rem] tracking-wider text-muted">OVERHEAD 0.6MS · FAIL-CLOSED</p>
          </div>
        )}
      </div>

      {/* Player controls */}
      <div className="flex items-center gap-3 border-t border-ink-800 bg-ink-900 px-4 py-2.5">
        <button
          onClick={() => setPlaying((v) => !v)}
          className="flex h-8 w-8 items-center justify-center rounded-full border border-ink-600 text-paper transition-colors hover:border-blueprint hover:text-blueprint"
          aria-label={playing ? "Pause" : "Play"}
        >
          {playing ? <Pause size={13} /> : <Play size={13} className="ml-0.5" />}
        </button>
        <button
          onClick={restart}
          className="flex h-8 w-8 items-center justify-center rounded-full border border-ink-700 text-muted transition-colors hover:border-ink-500 hover:text-paper"
          aria-label="Restart"
        >
          <RotateCcw size={12} />
        </button>

        {/* scrubber */}
        <div
          ref={trackRef}
          onPointerDown={onTrackDown}
          className="group relative h-6 flex-1 cursor-pointer"
          role="slider"
          aria-label="Seek"
          aria-valuemin={0}
          aria-valuemax={DURATION}
          aria-valuenow={Math.round(t)}
        >
          <div className="absolute top-1/2 h-1 w-full -translate-y-1/2 rounded-full bg-ink-700">
            {/* scene markers */}
            {[4.6, 8, 19.4].map((m) => (
              <span key={m} className="absolute top-1/2 h-2 w-px -translate-y-1/2 bg-ink-500" style={{ left: `${(m / DURATION) * 100}%` }} />
            ))}
          </div>
          <div
            className="absolute top-1/2 h-1 -translate-y-1/2 rounded-full bg-blueprint"
            style={{ width: `${(t / DURATION) * 100}%` }}
          />
          <div
            className="absolute top-1/2 h-3 w-3 -translate-x-1/2 -translate-y-1/2 rounded-full bg-blueprint shadow-[0_0_8px_rgba(110,147,232,0.7)]"
            style={{ left: `${(t / DURATION) * 100}%` }}
          />
        </div>

        <span className="font-mono text-[0.6875rem] text-muted">
          {fmt(t)} <span className="text-muted/50">/ {fmt(DURATION)}</span>
        </span>
      </div>
    </div>
  );
}
