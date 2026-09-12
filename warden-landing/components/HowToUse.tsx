"use client";

import { useState } from "react";
import { Check, Copy } from "lucide-react";

const STEPS = [
  {
    num: "01",
    title: "Install Warden",
    desc: "Install the CLI globally via npm. Works on Linux, macOS, and Windows.",
    code: "npm install -g warden-sandbox-cli",
  },
  {
    num: "02",
    title: "Create a policy",
    desc: "Run the interactive wizard. It asks what the server needs and drafts a deny-by-default YAML policy.",
    code: "warden init",
  },
  {
    num: "03",
    title: "Run the server sandboxed",
    desc: "Point Warden at the policy and the server command. The sandbox enforces every grant.",
    code: "warden run --policy policy.yaml -- npx @modelcontextprotocol/server-filesystem",
  },
  {
    num: "04",
    title: "Inspect what was blocked",
    desc: "The audit log records every allow and deny decision. Tail it live or review after the session.",
    code: "warden logs --tail",
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

        <div className="mt-12 grid gap-6 sm:grid-cols-2">
          {STEPS.map((s) => (
            <div
              key={s.num}
              className="flex flex-col rounded-sm border border-ink-700 bg-ink-900 p-5"
            >
              <div className="flex items-baseline gap-3">
                <span className="font-mono text-[0.75rem] text-blueprint">
                  {s.num}
                </span>
                <h3 className="text-[1.0625rem] font-medium text-paper">
                  {s.title}
                </h3>
              </div>
              <p className="mt-2 text-[0.875rem] leading-[1.55] text-muted">
                {s.desc}
              </p>
              <div className="mt-auto flex items-center gap-2 rounded-sm border border-ink-700 bg-ink-950 px-4 py-3 font-mono text-[0.8125rem] leading-none text-paper">
                <span className="shrink-0 text-muted">$ </span>
                <code className="truncate">{s.code}</code>
                <CopyButton text={s.code} />
              </div>
            </div>
          ))}
        </div>

        <p className="mt-8 text-[0.8125rem] leading-[1.6] text-muted/80">
          Want to see it in action first?{" "}
          <a href="#demo" className="underline decoration-muted/40 hover:text-paper">
            Watch the demo above
          </a>{" "}
          or check the{" "}
          <a href="/docs" className="underline decoration-muted/40 hover:text-paper">
            full documentation
          </a>
          .
        </p>
      </div>
    </section>
  );
}
