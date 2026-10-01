"use client";

import { useState } from "react";
import { Check, Copy } from "lucide-react";

/**
 * Code block with a copy-to-clipboard button.
 * Server-component friendly: safe to use from server pages.
 */
export default function CodeBlock({
  children,
  className = "",
}: {
  children: string;
  className?: string;
}) {
  const [copied, setCopied] = useState(false);
  const [failed, setFailed] = useState(false);

  async function handleCopy() {
    try {
      await navigator.clipboard.writeText(children);
      setCopied(true);
      setFailed(false);
      setTimeout(() => setCopied(false), 1800);
    } catch {
      setFailed(true);
    }
  }

  return (
    <div className={`group relative ${className}`}>
      <pre className="overflow-x-auto rounded-[8px] border border-ink-700 bg-ink-950 px-4 py-3 pr-20 font-hero text-[0.8125rem] leading-[1.7] text-paper">
        <code>{children}</code>
      </pre>
      <button
        onClick={handleCopy}
        aria-label="Copy command"
        className="absolute right-2 top-2 flex items-center gap-1.5 rounded-md border border-ink-700 bg-ink-900 px-2 py-1 font-hero text-[0.6875rem] text-muted transition-all hover:border-ink-500 hover:text-paper"
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
      {failed && (
        <p role="status" className="mt-2 text-xs text-progress">
          Copy unavailable. Select the text to copy it manually.
        </p>
      )}
    </div>
  );
}
