"use client";

import { useState } from "react";
import { ArrowRight, Check, Copy, Github, Terminal } from "lucide-react";
import ContainmentField from "./ContainmentField";

const NPM_CMD = "npm install -g warden-sandbox-cli";

export default function Hero() {
  const [npmCopied, setNpmCopied] = useState(false);

  function handleNpmCopy() {
    navigator.clipboard.writeText(NPM_CMD);
    setNpmCopied(true);
    setTimeout(() => setNpmCopied(false), 1800);
  }

  return (
    // Centered mono hero (CodeAtlas layout) + Warden ContainmentField bg
    <section className="relative overflow-hidden border-b border-ink-700 bg-ink-950">
      {/* Warden-tinted glows (replaces CodeAtlas purple wash, same placement logic) */}
      <div
        className="pointer-events-none absolute inset-0"
        aria-hidden
        style={{
          background:
            "radial-gradient(ellipse 70% 50% at 50% -10%, rgba(63,178,126,0.16) 0%, transparent 60%), radial-gradient(ellipse 50% 35% at 30% 50%, rgba(110,147,232,0.08) 0%, transparent 50%), radial-gradient(ellipse 50% 35% at 70% 70%, rgba(63,178,126,0.06) 0%, transparent 50%)",
        }}
      />
      {/* Containment animation: trapped particles + fail-closed boundary */}
      <ContainmentField />
      {/* faint dot texture under canvas content */}
      <div className="pointer-events-none absolute inset-0 hero-dot-grid opacity-[0.12]" aria-hidden />

      <div className="relative z-[2] mx-auto max-w-[1280px] px-5 pb-16 pt-16 sm:px-6 md:pb-[120px] md:pt-[140px]">
        <div className="mx-auto max-w-[800px] text-center">
          {/* eyebrow */}
          <p className="mb-6 inline-flex items-center gap-2 font-hero text-[12px] font-bold uppercase leading-none tracking-[0.12em] text-grant">
            <span className="rounded-full border border-grant/30 bg-grant-subtle px-2 py-[3px] text-[10px] leading-none">
              Pre-1.0
            </span>
            Permissions for local AI tools
          </p>

          {/* headlineJetBrains Mono bold, gradient white, like reference */}
          <h1 className="mx-auto mb-7 bg-gradient-to-b from-white to-white/70 bg-clip-text font-sans text-[2.75rem] font-semibold leading-[1.05] tracking-[-0.045em] text-transparent sm:text-6xl md:text-7xl xl:text-[84px]">
            Don&apos;t give AI agents your whole computer.
          </h1>

          {/* sub */}
          <p className="mx-auto mb-10 max-w-[640px] font-sans text-[17px] leading-[1.7] text-muted md:text-[20px]">
            Warden gives AI tools and MCP servers access only to the files, secrets,
            and network hosts you approve. Everything else is blocked by default.
          </p>

          {/* actions */}
          <div className="mb-10 flex flex-wrap items-center justify-center gap-[14px]">
            <a
              href="/docs/install"
              className="inline-flex items-center gap-2 rounded-[10px] bg-gradient-to-r from-grant to-blueprint px-4 py-[9px] font-hero text-[14px] font-semibold text-white transition hover:brightness-110"
            >
              Set up Warden
              <ArrowRight size={15} />
            </a>
            <a
              href="https://github.com/Prof-bilal/Warden"
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-2 rounded-[10px] border border-ink-600 bg-ink-900 px-4 py-[9px] font-hero text-[14px] font-semibold text-paper transition hover:-translate-y-px hover:border-grant/60"
            >
              <Github size={15} />
              View on GitHub
            </a>
          </div>

          {/* install terminalsingle centered card like reference */}
          <div className="mx-auto max-w-[460px] text-left">
            <div className="overflow-hidden rounded-xl border border-ink-700 bg-[#0B0E13] shadow-[0_24px_60px_rgba(0,0,0,0.4)]">
              <div className="flex items-center gap-2 border-b border-ink-800 bg-ink-900 px-[14px] py-[9px] font-hero text-[11px] tracking-[0.04em] text-muted">
                <Terminal size={12} className="text-muted" />
                <span>install</span>
              </div>
              <div className="flex items-center gap-[10px] p-4">
                <span className="shrink-0 font-hero font-semibold text-grant">$</span>
                <code className="flex-1 truncate font-hero text-[14px] text-paper">
                  {NPM_CMD}
                </code>
                <button
                  onClick={handleNpmCopy}
                  className="flex shrink-0 items-center gap-[6px] rounded-lg border border-ink-600 bg-ink-900 px-[10px] py-[6px] font-hero text-[12px] text-muted transition hover:border-grant/50 hover:text-paper"
                  aria-label="Copy npm install command"
                >
                  {npmCopied ? <Check size={13} className="text-grant" /> : <Copy size={13} />}
                  <span>{npmCopied ? "Copied" : "Copy"}</span>
                </button>
              </div>
            </div>
            <p className="mt-3 text-center font-sans text-[11px] uppercase leading-[1.7] tracking-[0.06em] text-muted/80">
              Linux · macOS · Windows · Docker fallback · MIT
            </p>
          </div>
        </div>

        {/* demo visual */}
        <div className="mt-12 flex justify-center md:mt-[72px]">
          <div className="w-full max-w-5xl">
            <div className="overflow-hidden rounded-2xl border border-ink-700 bg-ink-900">
              <video
                className="aspect-video w-full"
                width={1920}
                height={1080}
                autoPlay
                muted
                loop
                controls
                playsInline
                preload="metadata"
                poster="/images/ui-preview.svg"
                aria-label="Warden demo: recorded terminal session showing a sandboxed MCP server with policy grants and denied access attempts"
              >
                <source src="/videos/video-9ReotrYUC6t1GFvpAcsf.mp4" type="video/mp4" />
                Your browser does not support the video tag.
              </video>
            </div>
            <p className="mt-3 text-center font-hero text-[11px] uppercase tracking-[0.08em] text-muted">
              Recorded terminal session · verify the boundary on your own host
            </p>
          </div>
        </div>
      </div>
    </section>
  );
}
