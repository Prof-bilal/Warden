"use client";

import { useState } from "react";
import { Check, Copy } from "lucide-react";
import Eyebrow from "@/components/Eyebrow";

const INSTALL_CMD =
  "curl -LO https://github.com/Prof-bilal/Warden/releases/latest/download/warden-linux-amd64";

const NPM_CMD = "npm install -g warden-sandbox-cli";

export default function Hero() {
  const [copied, setCopied] = useState(false);
  const [npmCopied, setNpmCopied] = useState(false);

  function handleCopy() {
    navigator.clipboard.writeText(INSTALL_CMD);
    setCopied(true);
    setTimeout(() => setCopied(false), 1800);
  }

  function handleNpmCopy() {
    navigator.clipboard.writeText(NPM_CMD);
    setNpmCopied(true);
    setTimeout(() => setNpmCopied(false), 1800);
  }

  return (
    // InvisibleTech editorial hero — layout + font only, colors preserved from warden (ink/paper/muted)
    <section className="relative overflow-hidden border-b border-ink-700 bg-ink-950">
      {/* graph-paper dot grid — very faint, masked to top, no color takeover */}
      <div className="pointer-events-none absolute inset-0 hero-dot-grid opacity-[0.18]" aria-hidden />
      {/* subtle top fade wash — uses existing ink-900 at low opacity, not a new palette */}
      <div
        className="pointer-events-none absolute inset-x-0 top-0 h-[420px] bg-gradient-to-b from-ink-900/30 to-transparent"
        aria-hidden
      />

      {/* 1280px max-width container — InvisibleTech page model */}
      <div className="relative mx-auto max-w-[1280px] px-4 pb-12 pt-8 sm:px-6 md:pb-24 md:pt-16 lg:pb-28 lg:pt-20">
        {/* ── Eyebrow: 6px dot + Apkpraktikal-style mono small caps, wide tracking ── */}
        <Eyebrow>SANDBOX RUNTIME — FOR MCP SERVERS</Eyebrow>

        {/* ── Two-column editorial rhythm: headline ~55% / intro ~45%, gap 24-64px ── */}
        <div className="mt-6 grid gap-8 lg:grid-cols-[1.08fr_0.92fr] lg:items-end lg:gap-12 xl:gap-16">
          {/* Left: Two-tone headline — Apk Galeria → Newsreader, negative tracking */}
          <div>
            <h1 className="hero-display max-w-[900px] text-[2rem] font-normal leading-[0.95] tracking-[-0.02em] sm:text-[2.6rem] md:text-[3.75rem] lg:text-[4.25rem] xl:text-[64px] xl:leading-[1] xl:tracking-[-0.030em]">
              {/* first phrase in paper (primary), second phrase in muted — the InvisibleTech call-and-response */}
              <span className="block text-paper">
                Your MCP servers
                <span className="font-light text-muted"> don&apos;t need</span>
              </span>
              <span className="block font-normal tracking-[-0.02em] text-muted xl:tracking-[-0.030em]">
                your whole filesystem.
              </span>
            </h1>

            {/* Pill meta — InvisibleTech badges are 9999px, hairline 1px, no shadow */}
            <div className="mt-6 flex flex-wrap gap-2">
              <span className="inline-flex items-center rounded-full border border-ink-700 bg-ink-900 px-3 py-1 font-mono text-[0.6875rem] uppercase tracking-[0.08em] text-muted">
                Grants, not promises
              </span>
              <span className="inline-flex items-center rounded-full border border-ink-700 bg-ink-900 px-3 py-1 font-mono text-[0.6875rem] uppercase tracking-[0.08em] text-muted">
                Fail-closed · Audited
              </span>
            </div>
          </div>

          {/* Right: supporting body + CTAs — right column at ~45% width on desktop */}
          <div className="flex max-w-[32rem] flex-col lg:ml-auto lg:max-w-[30rem]">
            <p className="hero-display text-[1.0625rem] font-normal leading-[1.65] text-muted md:text-[1.125rem] md:leading-[1.7]">
              Warden runs them in a sandbox that only sees what you grant — a folder, a hostname, nothing more.
              Everything else fails, and Warden writes down that it tried.
            </p>

            {/* Install pills — 9999px radius, hairline border, mono, no shadow, comfortable 24px element gap */}
            <div className="mt-8 flex flex-col gap-3">
              <button
                onClick={handleNpmCopy}
                className="group flex min-w-0 w-full items-center justify-between gap-4 overflow-hidden rounded-full border border-ink-600 bg-ink-900 px-4 py-[11px] text-left font-mono text-[0.8125rem] leading-none text-paper transition-colors hover:border-ink-500 hover:bg-ink-800 md:px-5 md:py-3 md:text-[0.875rem]"
                aria-label="Copy npm install command"
              >
                <span className="truncate">
                  <span className="text-muted">$ </span>
                  {NPM_CMD}
                </span>
                {npmCopied ? (
                  <Check size={15} className="shrink-0 text-grant" />
                ) : (
                  <Copy size={15} className="shrink-0 text-muted transition-colors group-hover:text-paper" />
                )}
              </button>

              <button
                onClick={handleCopy}
                className="group flex min-w-0 w-full items-center justify-between gap-4 overflow-hidden rounded-full border border-ink-700 bg-ink-900 px-4 py-[11px] text-left font-mono text-[0.8125rem] leading-none text-muted transition-colors hover:border-ink-600 hover:text-paper md:px-5 md:py-3 md:text-[0.875rem]"
                aria-label="Copy curl install command"
              >
                <span className="truncate">
                  <span className="text-muted">$ </span>
                  {INSTALL_CMD}
                </span>
                {copied ? (
                  <Check size={15} className="shrink-0 text-grant" />
                ) : (
                  <Copy size={15} className="shrink-0 text-muted transition-colors group-hover:text-paper" />
                )}
              </button>

              <p className="px-1 font-mono text-[0.6875rem] uppercase tracking-[0.06em] text-muted/80">
                Works with any MCP client · No daemon · Linux · macOS · Windows
              </p>
            </div>
          </div>
        </div>

        {/* Visualizer — 12px card radius (InvisibleTech cards), 80px section gap above */}
        <div className="mt-12 md:mt-16 lg:mt-20">
          {/* subtle hairline divider before demo — editorial rhythm, 80px gap */}
          <div className="mb-8 hidden h-px bg-ink-800 lg:block" aria-hidden />
          <div className="overflow-hidden rounded-[12px] border border-ink-700 bg-ink-900">
            <video
              className="aspect-video w-full"
              width={1920}
              height={1080}
              autoPlay
              muted
              loop
              controls
              playsInline
              preload="auto"
              aria-label="Warden demo: interactive sandbox visualizer showing policy grants and access attempts"
            >
              <source src="/videos/video-9ReotrYUC6t1GFvpAcsf.mp4" type="video/mp4" />
              Your browser does not support the video tag.
            </video>
          </div>
          <p className="mt-3 text-center font-mono text-[0.6875rem] uppercase tracking-[0.08em] text-muted">
            Watch the sandbox enforce a policy in real time
          </p>
        </div>
      </div>
    </section>
  );
}
