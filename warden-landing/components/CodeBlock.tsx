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
      <pre className="overflow-x-auto border border-hair bg-void px-4 py-3 pr-20 font-mono text-[12px] leading-[1.75] text-offwhite/80">
        <code>{children}</code>
      </pre>
      <button
        onClick={handleCopy}
        aria-label="Copy command"
        className="absolute right-2 top-2 flex items-center gap-1.5 rounded border border-hair bg-cell px-2 py-1 font-mono text-[9.5px] uppercase tracking-micro text-offwhite/40 transition-colors hover:border-acid/40 hover:text-acid"
      >
        {copied ? (
          <>
            <Check size={11} className="text-acid" />
            <span className="text-acid">Copied</span>
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
