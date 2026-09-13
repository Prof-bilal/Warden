"use client";

import { useState } from "react";
import { Check, Copy } from "lucide-react";

/**
 * Code block with a copy-to-clipboard button.
 * Server-component friendly: safe to use from server pages.
 */
export default function CodeBlock({ children }: { children: string }) {
  const [copied, setCopied] = useState(false);

  function handleCopy() {
    navigator.clipboard.writeText(children);
    setCopied(true);
    setTimeout(() => setCopied(false), 1800);
  }

  return (
    <div className="group relative">
      <pre className="overflow-x-auto rounded-[8px] border border-ink-700 bg-ink-950 px-4 py-3 pr-20 font-mono text-[0.8125rem] leading-[1.7] text-paper">
        <code>{children}</code>
      </pre>
      <button
        onClick={handleCopy}
        aria-label="Copy command"
        className="absolute right-2 top-2 flex items-center gap-1.5 rounded-md border border-ink-700 bg-ink-900 px-2 py-1 font-mono text-[0.6875rem] text-muted opacity-0 transition-all hover:border-ink-500 hover:text-paper focus-visible:opacity-100 group-hover:opacity-100"
      >
        {copied ? (
          <>
            <Check size={11} className="text-grant" />
            <span className="text-grant">Copied</span>
          </>
        ) : (
          <>
            <Copy size={11} />
            <span>Copy</span>
          </>
        )}
      </button>
    </div>
  );
}
