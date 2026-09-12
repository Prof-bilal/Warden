"use client";

import { useState } from "react";
import { Check, Copy } from "lucide-react";
import StepperDetail from "@/components/StepperDetail";
import type { StepperItem } from "@/components/StepperDetail";
import { HowInstall, HowInit, HowRun, HowLogs } from "@/lib/images";

const STEPS: StepperItem[] = [
  {
    title: "Install Warden",
    summary: "npm install -g warden-sandbox-cli",
    body: "Install the CLI globally via npm. Works on Linux, macOS, and Windows.",
    image: <HowInstall />,
  },
  {
    title: "Create a policy",
    summary: "warden init",
    body: "Run the interactive wizard. It asks what the server needs and drafts a deny-by-default YAML policy.",
    image: <HowInit />,
  },
  {
    title: "Run the server sandboxed",
    summary: "warden run --policy policy.yaml -- ...",
    body: "Point Warden at the policy and the server command. The sandbox enforces every grant.",
    image: <HowRun />,
  },
  {
    title: "Inspect what was blocked",
    summary: "warden logs --tail",
    body: "The audit log records every allow and deny decision. Tail it live or review after the session.",
    image: <HowLogs />,
  },
];

function CopyButton({ text }: { text: string }) {
  const [copied, setCopied] = useState(false);

  function handleCopy() {
    navigator.clipboard.writeText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 1800);
  }

  return (
    <button
      onClick={handleCopy}
      className="ml-auto flex shrink-0 items-center gap-1.5 rounded-sm border border-ink-700 px-2 py-1 font-mono text-[0.6875rem] text-muted transition-colors hover:border-ink-500 hover:text-paper"
      aria-label="Copy command"
    >
      {copied ? (
        <>
          <Check size={12} className="text-grant" />
          <span className="text-grant">Copied</span>
        </>
      ) : (
        <>
          <Copy size={12} />
          <span>Copy</span>
        </>
      )}
    </button>
  );
}

export default function HowToUse() {
  return (
    <section id="how-to-use" className="border-t border-ink-800">
      <div className="mx-auto max-w-content px-6 py-20">
        <div className="flex flex-col gap-2 md:flex-row md:items-baseline md:justify-between">
          <h2 className="text-[1.75rem] font-medium leading-[1.15] tracking-[-0.01em] text-paper">
            How to use Warden.
          </h2>
          <span className="font-mono text-[0.6875rem] uppercase tracking-[0.08em] text-muted">
            4 commands · 2 minutes
          </span>
        </div>
        <p className="mt-3 max-w-[34rem] text-[1rem] leading-[1.65] text-muted">
          From install to a sandboxed server in four steps. No daemon, no config
          files beyond the policy, no changes to your existing MCP setup.
        </p>

        <div className="mt-10">
          <StepperDetail sectionLabel="Quick Start" items={STEPS} />
        </div>
      </div>
    </section>
  );
}
