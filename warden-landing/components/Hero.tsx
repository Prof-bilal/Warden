"use client";

import { useState } from "react";
import { Check, Copy } from "lucide-react";
import SandboxVisualizer from "@/components/SandboxVisualizer";

const INSTALL_CMD =
  "curl -LO https://github.com/Prof-bilal/Warden/releases/latest/download/warden-linux-amd64";

export default function Hero() {
  const [copied, setCopied] = useState(false);

  function handleCopy() {
    navigator.clipboard.writeText(INSTALL_CMD);
    setCopied(true);
    setTimeout(() => setCopied(false), 1800);
  }

  return (
    <section className="mx-auto max-w-content px-6 pb-20 pt-16 md:pb-28 md:pt-24">
      <div className="max-w-[38rem]">
        <h1 className="text-[2.5rem] font-medium leading-[1.08] tracking-[-0.02em] text-paper md:text-[3.25rem]">
          Your MCP servers don&apos;t need your whole filesystem.
        </h1>
        <p className="mt-5 max-w-[34rem] text-[1.0625rem] leading-[1.65] text-muted">
          Warden runs them in a sandbox that only sees what you grant — a
          folder, a hostname, nothing more. Everything else fails, and
          Warden writes down that it tried.
        </p>

        <button
          onClick={handleCopy}
          className="mt-8 flex w-full max-w-[30rem] items-center justify-between gap-4 rounded-sm border border-ink-600 bg-ink-900 px-4 py-3 text-left font-mono text-[0.875rem] text-paper transition-colors hover:border-ink-500"
        >
          <span className="truncate">
            <span className="text-muted">$ </span>
            {INSTALL_CMD}
          </span>
          {copied ? (
            <Check size={15} className="shrink-0 text-grant" />
          ) : (
            <Copy size={15} className="shrink-0 text-muted" />
          )}
        </button>
      </div>

      <div className="mt-16">
        <SandboxVisualizer />
      </div>
    </section>
  );
}
