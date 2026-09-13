"use client";

import { useState } from "react";
import { Check } from "lucide-react";

/**
 * Inline code chip that copies its text on click — for code snippets
 * inside tables and prose. Shows a small ✓ badge on copy (no layout
 * shift: the badge is absolutely positioned).
 */
export default function CopyableCode({
  text,
  className = "",
}: {
  text: string;
  className?: string;
}) {
  const [copied, setCopied] = useState(false);

  function handleCopy() {
    navigator.clipboard.writeText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  }

  return (
    <button
      onClick={handleCopy}
      title={`Click to copy: ${text}`}
      aria-label={`Copy ${text}`}
      className={
        "group relative inline-flex max-w-full cursor-pointer items-center rounded-md border px-1.5 py-0.5 align-baseline font-mono text-[0.8125rem] transition-colors " +
        (copied
          ? "border-grant/50 text-grant "
          : "border-ink-600 text-blueprint hover:border-ink-500 hover:text-paper ") +
        className
      }
    >
      <code className="truncate">{text}</code>
      <span
        aria-hidden="true"
        className={
          "absolute -right-1.5 -top-1.5 flex h-3.5 w-3.5 items-center justify-center rounded-full border transition-opacity " +
          (copied
            ? "border-grant/50 bg-ink-950 opacity-100"
            : "border-transparent bg-transparent opacity-0")
        }
      >
        <Check size={9} strokeWidth={3} className="text-grant" />
      </span>
    </button>
  );
}
