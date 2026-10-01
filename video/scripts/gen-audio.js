/**
 * Generates all audio for the Warden marketing video, programmatically.
 * Output: public/audio/*.wav (44.1kHz, 16-bit stereo)
 *
 *   node scripts/gen-audio.js
 *
 * Design notes:
 * - Music bed: slow ambient pad in D minor, evolving in intensity across
 *   ~5.5 minutes, quiet enough for narration to sit on top later.
 * - SFX: keystroke ticks (2 variants), allow chime, deny buzz, whoosh,
 *   finale swell. All synthesized deterministically — no external samples.
 */

const fs = require("fs");
const path = require("path");

const SR = 44100;
const OUT = path.join(__dirname, "..", "public", "audio");

fs.mkdirSync(OUT, { recursive: true });

/* ────────────────────────────  wav helpers  ──────────────────────────── */

function writeWav(name, samplesL, samplesR) {
  const n = samplesL.length;
  const stereo = samplesR !== undefined;
  const channels = stereo ? 2 : 1;
  const dataSize = n * channels * 2;
  const buf = Buffer.alloc(44 + dataSize);

  buf.write("RIFF", 0);
  buf.writeUInt32LE(36 + dataSize, 4);
  buf.write("WAVE", 8);
  buf.write("fmt ", 12);
  buf.writeUInt32LE(16, 16);
  buf.writeUInt16LE(1, 20); // PCM
  buf.writeUInt16LE(channels, 22);
  buf.writeUInt32LE(SR, 24);
  buf.writeUInt32LE(SR * channels * 2, 28);
  buf.writeUInt16LE(channels * 2, 32);
  buf.writeUInt16LE(16, 34);
  buf.write("data", 36);
  buf.writeUInt32LE(dataSize, 40);

  for (let i = 0; i < n; i++) {
    const l = Math.max(-1, Math.min(1, samplesL[i]));
    buf.writeInt16LE(Math.round(l * 32767), 44 + i * channels * 2);
    if (stereo) {
      const r = Math.max(-1, Math.min(1, samplesR[i]));
      buf.writeInt16LE(Math.round(r * 32767), 44 + i * channels * 2 + 2);
    }
  }
  fs.writeFileSync(path.join(OUT, name), buf);
  console.log(`wrote ${name} (${(n / SR).toFixed(2)}s)`);
}

const sec = (s) => Math.round(s * SR);

/* ────────────────────────────  dsp helpers  ──────────────────────────── */

const clamp = (x, a, b) => Math.max(a, Math.min(b, x));

/** Deterministic pseudo-noise (no Math.random — renders stay reproducible). */
let noiseState = 22222;
function noise() {
  noiseState = (noiseState * 1103515245 + 12345) & 0x7fffffff;
  return (noiseState / 0x3fffffff) - 1;
}

function envAD(t, attack, decay) {
  if (t < 0) return 0;
  if (t < attack) return t / attack;
  return Math.exp(-(t - attack) / decay);
}

/** Simple one-pole lowpass. */
function lowpass(buf, cutoffHz) {
  const dt = 1 / SR;
  const rc = 1 / (2 * Math.PI * cutoffHz);
  const a = dt / (rc + dt);
  let y = 0;
  for (let i = 0; i < buf.length; i++) {
    y += a * (buf[i] - y);
    buf[i] = y;
  }
  return buf;
}

/* ══════════════════════════  MUSIC BED  ══════════════════════════ */

function genMusicBed() {
  const durS = 326; // full video + a little tail
  const n = sec(durS);
  const L = new Float64Array(n);
  const R = new Float64Array(n);

  // D minor-ish drone: D2, D3, F3, A3, C4 — slow evolving pad
  const freqs = [73.42, 146.83, 174.61, 220.0, 261.63];
  const baseAmp = [0.30, 0.20, 0.16, 0.14, 0.10];

  // Intensity arc across the video (chapter-based):
  // hook subtle → problem grows → demo focused → finale lift → fade
  function intensity(t) {
    const m = t / 60; // minutes
    if (m < 0.5) return 0.55; // hook
    if (m < 2.0) return 0.75; // problem/scenario
    if (m < 2.67) return 0.85; // intro/policy
    if (m < 4.0) return 0.72; // demo/architecture
    if (m < 5.08) return 0.9; // platforms/why
    return 1.0; // cta
  }

  // slow LFO phase per voice for movement
  const lfoRate = [0.05, 0.073, 0.091, 0.061, 0.083];

  for (let i = 0; i < n; i++) {
    const t = i / SR;
    const I = intensity(t);

    // global fade in (2s) and fade out (last 6s)
    let g = 1;
    if (t < 2) g = t / 2;
    if (t > durS - 6) g = Math.max(0, (durS - t) / 6);
    g *= I;

    let l = 0;
    let r = 0;
    for (let v = 0; v < freqs.length; v++) {
      const lfo = 0.85 + 0.15 * Math.sin(2 * Math.PI * lfoRate[v] * t + v * 1.7);
      const detune = 1 + 0.0015 * Math.sin(2 * Math.PI * 0.021 * t + v * 2.3);
      const s = Math.sin(2 * Math.PI * freqs[v] * detune * t) * baseAmp[v] * lfo;
      // slight stereo spread per voice
      const pan = 0.5 + 0.35 * Math.sin(2 * Math.PI * 0.017 * t + v * 4.1);
      l += s * (1 - pan * 0.5);
      r += s * (0.5 + pan * 0.5);
    }

    // sub pulse every 2s for gentle momentum (very quiet)
    const pulse = Math.exp(-((t % 2) / 0.4)) * 0.05 * I;
    const sub = Math.sin(2 * Math.PI * 36.71 * t) * pulse;

    L[i] = (l + sub) * g * 0.4;
    R[i] = (r + sub) * g * 0.4;
  }

  lowpass(L, 1800);
  lowpass(R, 1800);
  writeWav("music-bed.wav", L, R);
}

/* ══════════════════════════  SFX  ══════════════════════════ */

function genKeyTick(variant) {
  const durS = 0.05;
  const n = sec(durS);
  const L = new Float64Array(n);
  for (let i = 0; i < n; i++) {
    const t = i / SR;
    const e = envAD(t, 0.001, 0.006);
    const click = noise() * 0.7;
    const tone = Math.sin(2 * Math.PI * (variant === 0 ? 2400 : 1900) * t) * 0.25;
    L[i] = (click + tone) * e * 0.5;
  }
  lowpass(L, 6000);
  writeWav(variant === 0 ? "key-tick-1.wav" : "key-tick-2.wav", L, L);
}

function genEnter() {
  const durS = 0.12;
  const n = sec(durS);
  const L = new Float64Array(n);
  for (let i = 0; i < n; i++) {
    const t = i / SR;
    const e = envAD(t, 0.001, 0.03);
    const s =
      Math.sin(2 * Math.PI * 880 * t) * 0.3 +
      Math.sin(2 * Math.PI * 1320 * t) * 0.15 +
      noise() * 0.2;
    L[i] = s * e * 0.5;
  }
  writeWav("key-enter.wav", L, L);
}

function genAllow() {
  const durS = 0.5;
  const n = sec(durS);
  const L = new Float64Array(n);
  const R = new Float64Array(n);
  // two-note upward chime: E5 → A5
  const notes = [
    { f: 659.26, t0: 0.0 },
    { f: 880.0, t0: 0.09 },
  ];
  for (let i = 0; i < n; i++) {
    const t = i / SR;
    let s = 0;
    for (const note of notes) {
      const e = envAD(t - note.t0, 0.004, 0.16);
      s += Math.sin(2 * Math.PI * note.f * t) * e * 0.4;
      s += Math.sin(2 * Math.PI * note.f * 2 * t) * e * 0.08;
    }
    L[i] = s;
    R[i] = s * 0.95;
  }
  writeWav("allow.wav", L, R);
}

function genDeny() {
  const durS = 0.42;
  const n = sec(durS);
  const L = new Float64Array(n);
  const R = new Float64Array(n);
  // low buzz: 110Hz square-ish + 220Hz, short
  for (let i = 0; i < n; i++) {
    const t = i / SR;
    const e = envAD(t, 0.002, 0.12);
    const sq = Math.sign(Math.sin(2 * Math.PI * 110 * t)) * 0.22;
    const s = sq + Math.sin(2 * Math.PI * 220 * t) * 0.12;
    L[i] = s * e;
    R[i] = s * e * 0.9;
  }
  lowpass(L, 2400);
  lowpass(R, 2400);
  writeWav("deny.wav", L, R);
}

function genWhoosh() {
  const durS = 0.9;
  const n = sec(durS);
  const L = new Float64Array(n);
  const R = new Float64Array(n);
  // filtered noise swell with pitch-down sweep
  for (let i = 0; i < n; i++) {
    const t = i / SR;
    const p = t / durS;
    const e = Math.sin(Math.PI * p) ** 1.5;
    const sweep = Math.sin(2 * Math.PI * (600 - 350 * p) * t) * 0.15;
    L[i] = (noise() * 0.35 + sweep) * e;
    R[i] = (noise() * 0.35 + sweep) * e;
  }
  lowpass(L, 1400);
  lowpass(R, 1400);
  writeWav("whoosh.wav", L, R);
}

function genFinale() {
  const durS = 4.5;
  const n = sec(durS);
  const L = new Float64Array(n);
  const R = new Float64Array(n);
  // rising D-major resolve: D4 F#4 A4 D5, soft swell
  const notes = [293.66, 369.99, 440.0, 587.33];
  for (let i = 0; i < n; i++) {
    const t = i / SR;
    const g = Math.min(1, t / 1.8) * (1 - Math.max(0, (t - 3.2) / 1.3));
    let s = 0;
    notes.forEach((f, v) => {
      const onset = Math.max(0, Math.min(1, (t - v * 0.35) / 0.4));
      s += Math.sin(2 * Math.PI * f * t) * onset * (0.16 - v * 0.02);
    });
    L[i] = s * g;
    R[i] = s * g * 0.97;
  }
  lowpass(L, 3200);
  lowpass(R, 3200);
  writeWav("finale.wav", L, R);
}

/* ══════════════════════════  main  ══════════════════════════ */

genMusicBed();
genKeyTick(0);
genKeyTick(1);
genEnter();
genAllow();
genDeny();
genWhoosh();
genFinale();
console.log("done.");
