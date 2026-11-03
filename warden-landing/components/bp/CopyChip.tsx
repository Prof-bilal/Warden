"use client";

import { useState } from "react";
import { Check, Copy } from "lucide-react";
import { cn } from "@/lib/cn";

export default function CopyChip({
  value,
  className,
}: {
  value: string;
  className?: string;
}) {
  const [copied, setCopied] = useState(false);

  function copy() {
    navigator.clipboard.writeText(value).catch(() => {});
    setCopied(true);
    setTimeout(() => setCopied(false), 1600);
  }

  return (
    <div
      className={cn(
        "flex items-center justify-between gap-3 border border-hair bg-void px-3.5 py-3",
        className,
      )}
    >
      <code className="min-w-0 truncate font-mono text-[11.5px] text-offwhite/80">
        <span className="text-acid">&gt;</span> {value}
      </code>
      <button
        type="button"
        onClick={copy}
        aria-label="Copy command"
        className="flex h-6 w-6 shrink-0 items-center justify-center rounded border border-hair text-offwhite/40 transition-colors hover:border-acid/40 hover:text-acid"
      >
        {copied ? <Check size={12} className="text-acid" /> : <Copy size={12} />}
      </button>
    </div>
  );
}
