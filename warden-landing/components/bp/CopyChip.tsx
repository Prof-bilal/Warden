"use client";

import { useState } from "react";
import { Check, Copy, X } from "lucide-react";
import { cn } from "@/lib/cn";

export default function CopyChip({
  value,
  className,
}: {
  value: string;
  className?: string;
}) {
  const [status, setStatus] = useState<"copied" | "failed" | null>(null);

  async function copy() {
    try {
      await navigator.clipboard.writeText(value);
      setStatus("copied");
    } catch {
      setStatus("failed");
    }
    setTimeout(() => setStatus(null), 1600);
  }

  return (
    <div
      className={cn(
        "flex items-center justify-between gap-3 border border-hair bg-void px-3.5 py-3",
        className,
      )}
    >
      <code className="min-w-0 truncate font-mono text-[11.5px] text-paper">
        <span className="text-acid">&gt;</span> {value}
      </code>
      <button
        type="button"
        onClick={copy}
        aria-label="Copy command"
        className="flex h-6 w-6 shrink-0 items-center justify-center rounded border border-hair text-muted transition-colors hover:border-acid/40 hover:text-acid"
      >
        {status === "copied" && <Check size={14} aria-hidden="true" />}
        {status === "failed" && <X size={14} aria-hidden="true" />}
        {status === null && <Copy size={14} aria-hidden="true" />}
      </button>
      <span className="sr-only" role="status">
        {status === "copied" && "Command copied"}
        {status === "failed" && "Copy failed. Select the command to copy it manually."}
      </span>
    </div>
  );
}
