import React from "react";
import { AbsoluteFill, interpolate, useCurrentFrame, useVideoConfig } from "remotion";
import { Terminal } from "../components/Terminal";
import type { TermLine } from "../data/terminal";
import { C, FONTS, Scene, riseIn } from "../components/ui";
import {
  DEMO_POLICY_YAML,
  REAL_AUDIT_LINES,
  REAL_DOCTOR_ENV,
  REAL_DOCTOR_POSTURE,
} from "../data/product";

/*
 * Scene 06 — Real Warden demo (3:10–4:00)
 *
 * The credibility anchor. Everything on screen mirrors the real CLI:
 * - policy.yaml fields from the actual schema (docs/schema.md, examples/)
 * - `warden run` pre-launch summary, verbatim format from docs/cli.md
 * - `warden logs` audit formatting (✓ ALLOWED / ✗ BLOCKED) from docs/cli.md
 * - one raw JSONL audit line, matching the audit format in ARCHITECTURE.md
 */
export const WardenDemo: React.FC = () => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();

  /* ── timing (scene-local seconds) ──
   * 0.3   title
   * 1.0   policy card in (centered)
   * 4.5   run terminal enters; policy card shrinks to the left
   * 5.5   banner + summary cascade
   * 9.2   ✓ Sandbox active
   * 22.0  switch to `warden logs`
   * 24.1  audit events cascade (ALLOWED / BLOCKED)
   * 36.5  switch to `warden doctor`
   * 40.8  Status: READY
   */
  const t = {
    title: 0.3,
    policy: 1.0,
    swap: 4.5,
    sandboxActive: 9.2,
    logs: 22.0,
    doctor: 36.5,
  };

  const titleS = riseIn(frame, fps, Math.round(t.title * fps));

  /* policy card geometry animation */
  const swapP = interpolate(frame, [t.swap * fps, t.swap * fps + 22], [0, 1], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });
  const policyLeft = interpolate(swapP, [0, 1], [530, 70]);
  const policyScale = interpolate(swapP, [0, 1], [1, 0.72]);
  const policyTop = interpolate(swapP, [0, 1], [250, 320]);

  /* run terminal visibility */
  const runIn = interpolate(frame, [t.swap * fps, t.swap * fps + 14], [0, 1], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });
  /* logs terminal replaces run terminal */
  const logsIn = interpolate(frame, [t.logs * fps, t.logs * fps + 12], [0, 1], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });
  const runOut = 1 - logsIn;

  /* captions */
  const cap1 = riseIn(frame, fps, Math.round(12.0 * fps));
  const cap2 = riseIn(frame, fps, Math.round(31.0 * fps));

  /* ── run terminal script (real output format, docs/cli.md) ── */
  const RUN_LINES: TermLine[] = [
    { kind: "cmd", segments: [{ text: "warden ", tone: "text" }, { text: "run", tone: "cyan" }, { text: " --policy policy.yaml", tone: "text" }] },
    { kind: "out-block", gapBefore: 0.3, lines: ["WARDEN", "──────────────────────────────────────"] },
    { kind: "out-block", gapBefore: 1.0, lines: ["Policy     policy.yaml", "Backend    linux", "Command    /usr/bin/node server.js"] },
    { kind: "out-block", gapBefore: 1.2, lines: ["Filesystem", "  ✓ ./data (read)", "  ✓ ./output (write)", "  ✗ everything else"] },
    { kind: "out-block", gapBefore: 0.8, lines: ["Network", "  ✓ api.github.com", "  ✗ everything else"] },
    { kind: "out-block", gapBefore: 0.8, lines: ["Environment", "  ✓ GITHUB_TOKEN", "  ✗ all unspecified variables"] },
    { kind: "out", gapBefore: 1.2, text: "──────────────────────────────────────" },
    { kind: "out", gapBefore: 0.2, text: "✓ Sandbox active", tone: "green" },
  ];
  const runPlan = planify(RUN_LINES, 5.5, [0.6, 1.0, 1.4, 1.8, 2.2, 2.6, 3.0, 3.4]);

  /* ── logs terminal script (✓ ALLOWED / ✗ BLOCKED format from docs/cli.md;
   *    hosts match the REAL audit events captured from a live run) ── */
  const LOG_LINES: TermLine[] = [
    { kind: "cmd", segments: [{ text: "warden ", tone: "text" }, { text: "logs", tone: "cyan" }, { text: " --tail 50", tone: "text" }] },
    { kind: "out", gapBefore: 0.4, text: "✓ ALLOWED  net  connect   api.github.com:443", tone: "green" },
    { kind: "out", gapBefore: 0.9, text: "✓ ALLOWED  net  connect   example.com:443", tone: "green" },
    { kind: "out", gapBefore: 1.4, text: "✗ BLOCKED  net  CONNECT   google.com:443", tone: "red" },
    { kind: "out", gapBefore: 1.9, text: "✗ BLOCKED  file  read     ~/.ssh/id_ed25519", tone: "red" },
    { kind: "out", gapBefore: 2.6, text: REAL_AUDIT_LINES[2], tone: "faint" },
  ];
  const logsPlan = planify(LOG_LINES, 0.5, [0.6, 2.1, 3.6, 5.1, 6.6, 8.1]);

  /* ── doctor terminal script — verbatim report from real `warden doctor`
   *    (warden v0.1.16, linux/amd64, captured from a live host) ── */
  const DOCTOR_LINES: TermLine[] = [
    { kind: "cmd", segments: [{ text: "warden ", tone: "text" }, { text: "doctor", tone: "cyan" }] },
    { kind: "out-block", gapBefore: 0.4, lines: ["WARDEN DOCTOR", ""] },
    { kind: "out-block", gapBefore: 0.3, lines: ["Environment", "────────────────────────────────"] },
    { kind: "out-block", gapBefore: 0.4, lines: [...REAL_DOCTOR_ENV.slice(0, 4)] },
    { kind: "out-block", gapBefore: 0.5, lines: [...REAL_DOCTOR_ENV.slice(4)] },
    { kind: "out-block", gapBefore: 0.6, lines: ["Security posture", "────────────────────────────────"] },
    { kind: "out-block", gapBefore: 0.4, lines: [...REAL_DOCTOR_POSTURE] },
    { kind: "out", gapBefore: 0.8, text: "Status: READY", tone: "green" },
  ];
  const doctorPlan = planify(DOCTOR_LINES, 0.5, [0.6, 1.6, 2.2, 2.9, 3.7, 4.5, 5.1, 5.9, 7.0]);

  const showLogs = frame >= t.logs * fps;
  const showDoctor = frame >= t.doctor * fps;
  const doctorIn = interpolate(frame, [t.doctor * fps, t.doctor * fps + 12], [0, 1], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });
  const logsOut = 1 - doctorIn;

  /* captions */
  const cap3 = riseIn(frame, fps, Math.round(41.5 * fps));

  return (
    <Scene fadeOutAfter={0.8}>
      <AbsoluteFill
        style={{
          backgroundImage:
            "linear-gradient(to right, rgba(255,255,255,0.02) 1px, transparent 1px), linear-gradient(to bottom, rgba(255,255,255,0.02) 1px, transparent 1px)",
          backgroundSize: "56px 56px",
        }}
      />

      {/* Title */}
      <div style={{ position: "absolute", top: 74, left: 0, right: 0, textAlign: "center", ...titleS }}>
        <div
          style={{
            fontFamily: FONTS.mono,
            fontSize: 22,
            letterSpacing: "0.35em",
            textTransform: "uppercase",
            color: C.faint,
            marginBottom: 10,
          }}
        >
          The actual CLI
        </div>
        <div style={{ fontSize: 56, fontWeight: 700, letterSpacing: "-0.02em" }}>
          Real commands. Real output.
        </div>
      </div>

      {/* policy card */}
      <div
        style={{
          position: "absolute",
          left: policyLeft,
          top: policyTop,
          transform: `scale(${policyScale})`,
          transformOrigin: "top left",
          opacity: interpolate(frame, [t.policy * fps, t.policy * fps + 12], [0, 1], {
            extrapolateLeft: "clamp",
            extrapolateRight: "clamp",
          }),
          width: 860,
          background: C.panel,
          border: `1px solid ${C.border}`,
          borderRadius: 16,
          padding: "26px 32px",
          boxShadow: "0 20px 60px rgba(0,0,0,0.4)",
        }}
      >
        <div style={{ fontFamily: FONTS.mono, fontSize: 19, color: C.faint, marginBottom: 16 }}>
          policy.yaml
        </div>
        {DEMO_POLICY_YAML.split("\n").map((l, i) => (
          <div
            key={i}
            style={{
              fontFamily: FONTS.mono,
              fontSize: 24,
              lineHeight: 1.6,
              color: l.startsWith("#") ? C.faint : yamlColor(l),
              whiteSpace: "pre",
              opacity: interpolate(frame, [t.policy * fps + 6 + i * 3, t.policy * fps + 12 + i * 3], [0, 1], {
                extrapolateLeft: "clamp",
                extrapolateRight: "clamp",
              }),
            }}
          >
            {l}
          </div>
        ))}
      </div>

      {/* run terminal */}
      {!showLogs && (
        <div style={{ position: "absolute", left: 740, top: 210, opacity: runIn }}>
          <Terminal
            title="warden run"
            plan={runPlan}
            frame={frame}
            width={1120}
            height={700}
            fontSize={21}
          />
        </div>
      )}

      {/* logs terminal */}
      {showLogs && !showDoctor && (
        <div style={{ position: "absolute", left: 740, top: 300, opacity: logsIn }}>
          <Terminal
            title="warden logs"
            plan={logsPlan}
            frame={frame}
            width={1120}
            height={520}
            fontSize={21}
          />
        </div>
      )}
      {/* keep logs terminal mounted but invisible during doctor transition */}
      {showDoctor && logsOut > 0.02 && (
        <div style={{ position: "absolute", left: 740, top: 300, opacity: logsOut * logsIn, pointerEvents: "none" }}>
          <Terminal
            title="warden logs"
            plan={logsPlan}
            frame={frame}
            width={1120}
            height={520}
            fontSize={21}
          />
        </div>
      )}

      {/* doctor terminal */}
      {showDoctor && (
        <div style={{ position: "absolute", left: 740, top: 230, opacity: doctorIn }}>
          <Terminal
            title="warden doctor"
            plan={doctorPlan}
            frame={frame}
            width={1120}
            height={640}
            fontSize={21}
          />
        </div>
      )}
      {/* keep run terminal mounted but invisible during fade-out transition */}
      {showLogs && runOut > 0.02 && (
        <div style={{ position: "absolute", left: 740, top: 210, opacity: runOut * runIn, pointerEvents: "none" }}>
          <Terminal
            title="warden run"
            plan={runPlan}
            frame={frame}
            width={1120}
            height={700}
            fontSize={21}
          />
        </div>
      )}

      {/* captions */}
      <div
        style={{
          position: "absolute",
          bottom: 64,
          left: 740,
          width: 1120,
          textAlign: "center",
          fontSize: 26,
          color: C.muted,
          ...cap1,
          opacity: frame < t.logs * fps ? cap1.opacity : 0,
        }}
      >
        The MCP client talks stdio straight through — sandboxing is invisible to the protocol.
      </div>
      <div
        style={{
          position: "absolute",
          bottom: 64,
          left: 740,
          width: 1120,
          textAlign: "center",
          fontSize: 26,
          color: C.muted,
          opacity: frame >= t.logs * fps && !showDoctor ? cap2.opacity : 0,
        }}
      >
        Every access attempt is recorded — allowed and blocked.
      </div>
      <div
        style={{
          position: "absolute",
          bottom: 64,
          left: 740,
          width: 1120,
          textAlign: "center",
          fontSize: 26,
          color: C.muted,
          opacity: showDoctor ? cap3.opacity : 0,
        }}
      >
        Doctor verifies the host can actually enforce the sandbox — before you run anything.
      </div>
    </Scene>
  );
};

/* ────────────────────────────  helpers  ──────────────────────────── */

/** Distribute output lines over time: each entry gets [startS, startS+dur]. */
function planify(
  lines: TermLine[],
  startS: number,
  offsetsS: number[]
): { line: TermLine; startS: number; endS: number }[] {
  const out: { line: TermLine; startS: number; endS: number }[] = [];
  let idx = 0;
  for (const line of lines) {
    if (line.kind === "cmd") {
      const start = startS + (offsetsS[idx] ?? 0);
      out.push({ line, startS: start, endS: start + 1.1 });
    } else {
      const start = startS + (offsetsS[idx] ?? 0);
      out.push({ line, startS: start, endS: start + 0.1 });
    }
    idx++;
  }
  return out;
}

function yamlColor(line: string): string {
  if (/^\s*(command|filesystem|network|env|limits|read|write|allow|memory_mb|timeout_s):/.test(line)) return C.cyan;
  if (line.includes("[") || line.includes('"')) return C.greenSoft;
  if (/:\s*\d+$/.test(line)) return C.amber;
  return C.text;
}
