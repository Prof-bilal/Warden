"use client";

import { useEffect, useRef, useState } from "react";
import Link from "next/link";
import { ArrowRight, Check, Copy, Github, Pause, Play, Volume2, VolumeX } from "lucide-react";
import StatRow from "@/components/bp/StatRow";

const installCommand = "npm i -g warden-sandbox-cli";

const stats: [string, string][] = [
  ["18", "MCP servers tested"],
  ["8/8", "Linux proof checks passed"],
  ["7/7", "attack simulations contained"],
  ["0", "permissions granted by default"],
];

const labels = ["Isolate", "Control", "Observe", "Safe by default"];

// Marks that the hero video has already played once in this browser session,
// so the landing page does not replay it every time the visitor navigates back.
const PLAYED_KEY = "warden-hero-played";

export default function Hero() {
  const [copied, setCopied] = useState(false);
  const [muted, setMuted] = useState(true);
  const [playing, setPlaying] = useState(true);
  const videoRef = useRef<HTMLVideoElement | null>(null);

  useEffect(() => {
    const video = videoRef.current;
    if (!video) return;

    let alreadyPlayed = false;
    try {
      alreadyPlayed = window.sessionStorage.getItem(PLAYED_KEY) === "1";
    } catch {
      alreadyPlayed = false;
    }

    // Already seen this session, leave it on the poster frame, no autoplay.
    if (alreadyPlayed) {
      setPlaying(false);
      return;
    }

    const markPlayed = () => {
      try {
        window.sessionStorage.setItem(PLAYED_KEY, "1");
      } catch {
        /* storage unavailable, fall back to normal autoplay behavior */
      }
    };

    // Try to start with sound; browsers may block unmuted autoplay, so fall
    // back to muted playback and let the viewer enable sound explicitly.
    video.muted = false;
    const attempt = video.play();
    if (!attempt) return;
    attempt.then(
      () => {
        setMuted(false);
        markPlayed();
      },
      () => {
        video.muted = true;
        setMuted(true);
        video.play().then(markPlayed).catch(() => {});
      },
    );
  }, []);

  function toggleSound() {
    const video = videoRef.current;
    if (!video) return;
    video.muted = !video.muted;
    setMuted(video.muted);
    if (!video.muted) video.play().catch(() => {});
  }

  function togglePlay() {
    const video = videoRef.current;
    if (!video) return;
    if (video.paused) {
      video.play().catch(() => {});
    } else {
      video.pause();
    }
  }

  function copyInstall() {
    navigator.clipboard.writeText(installCommand).catch(() => {});
    setCopied(true);
    setTimeout(() => setCopied(false), 1800);
  }

  return (
    <section className="relative overflow-hidden border-b border-hair">
      <div className="grid lg:grid-cols-[0.2fr_0.6fr_0.2fr]">
        {/* left, pipeline labels */}
        <div className="relative flex flex-col justify-center border-b border-hair p-6 lg:border-b-0 lg:border-r">
          <ul className="font-mono text-[10px] uppercase tracking-micro">
            {labels.map((label, i) => (
              <li
                key={label}
                className={`flex items-center gap-2.5 py-2.5 text-offwhite/60 ${
                  i === 0 ? "border-y border-hair" : "border-b border-hair"
                }`}
              >
                <span className="h-1 w-1 rounded-full bg-acid" aria-hidden />
                {label}
              </li>
            ))}
          </ul>
          <p className="mt-6 font-mono text-[9.5px] uppercase leading-[1.9] tracking-micro text-offwhite/30">
            For a more
            <br />
            open AI ecosystem
          </p>
        </div>

        {/* center, hero video */}
        <div className="relative flex items-center justify-center overflow-hidden border-b border-hair lg:border-b-0 lg:border-r">
          <video
            ref={videoRef}
            className="h-auto w-full object-cover"
            src="/videos/warden-hero.mp4"
            poster="/videos/warden-hero-poster.jpg"
            muted
            playsInline
            preload="metadata"
            onPlay={() => setPlaying(true)}
            onPause={() => setPlaying(false)}
            onEnded={() => setPlaying(false)}
          />
          <div className="absolute bottom-3 right-3 flex items-center overflow-hidden rounded-full border border-hair bg-void/80 shadow-[0_6px_20px_rgba(0,0,0,0.5)] backdrop-blur-sm">
            <button
              type="button"
              onClick={togglePlay}
              aria-label={playing ? "Pause video" : "Play video"}
              aria-pressed={playing}
              className="flex h-9 w-9 items-center justify-center text-offwhite/70 transition-colors hover:bg-offwhite/[0.07] hover:text-acid"
            >
              {playing ? <Pause size={14} /> : <Play size={14} />}
            </button>
            <span className="h-4 w-px bg-hair" aria-hidden />
            <button
              type="button"
              onClick={toggleSound}
              aria-label={muted ? "Enable sound" : "Mute"}
              aria-pressed={!muted}
              className="flex h-9 w-9 items-center justify-center text-offwhite/70 transition-colors hover:bg-offwhite/[0.07] hover:text-acid"
            >
              {muted ? <VolumeX size={14} /> : <Volume2 size={14} />}
            </button>
          </div>
        </div>

        {/* right, statement */}
        <div className="flex flex-col justify-center p-6">
          <h1 className="font-mono text-[1.1rem] font-semibold uppercase leading-[1.4] tracking-[-0.01em] text-offwhite lg:text-[1.22rem]">
            Give AI tools
            <br />
            <span className="text-acid">permissions</span>,
            <br />
            not your
            <br />
            <span className="text-acid">machine</span>.
          </h1>
          <p className="mt-5 font-sans text-[12.5px] leading-[1.7] text-offwhite/45">
            An open-source permission layer and sandbox for AI tools and MCP
            servers.
          </p>
        </div>
      </div>

      {/* action row */}
      <div className="flex flex-wrap items-center gap-3 border-b border-hair px-6 py-4">
        <p className="inline-flex items-center gap-2 border border-hair px-3 py-1 font-mono text-[9px] uppercase tracking-micro text-offwhite/55">
          <span className="h-1.5 w-1.5 rounded-full bg-acid" aria-hidden />
          Open source · MIT licensed · Fail closed
        </p>
        <Link
          href="/docs/install"
          className="inline-flex items-center gap-2 bg-acid px-4 py-2 font-mono text-[12px] font-semibold uppercase tracking-[0.04em] text-void transition-colors hover:bg-acid-bright"
        >
          Get started
          <ArrowRight size={13} aria-hidden />
        </Link>
        <a
          href="https://github.com/Prof-bilal/Warden"
          target="_blank"
          rel="noopener noreferrer"
          className="inline-flex items-center gap-2 border border-hair bg-cell px-4 py-2 font-mono text-[12px] font-semibold uppercase tracking-[0.04em] text-offwhite/85 transition-colors hover:border-offwhite/20"
        >
          <Github size={13} aria-hidden />
          View on GitHub
        </a>
        <div className="flex items-center gap-2 border border-hair bg-cell py-2 pl-3 pr-1.5">
          <code className="font-mono text-[11px] text-offwhite/50">
            <span className="text-acid">&gt;</span> {installCommand}
          </code>
          <button
            type="button"
            onClick={copyInstall}
            aria-label="Copy install command"
            className="flex h-6 w-6 items-center justify-center rounded border border-hair text-offwhite/40 transition-colors hover:border-acid/40 hover:text-acid"
          >
            {copied ? <Check size={12} className="text-acid" /> : <Copy size={12} />}
          </button>
        </div>
      </div>

      <StatRow stats={stats} className="border-b border-hair" />
    </section>
  );
}
