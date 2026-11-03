"use client";

import { useState } from "react";
import { Check, Copy } from "lucide-react";
import type { ReactNode } from "react";
import { cn } from "@/lib/cn";

function renderValue(value: string): ReactNode {
  const hash = value.indexOf("#");
  if (hash >= 0) {
    return (
      <>
        <span className="text-offwhite/70">{value.slice(0, hash)}</span>
        <span className="text-offwhite/25">{value.slice(hash)}</span>
      </>
    );
  }
  return <span className="text-offwhite/70">{value}</span>;
}

function highlight(line: string): ReactNode {
  const trimmed = line.trimStart();
  if (trimmed.startsWith("#")) {
    return <span className="text-offwhite/25">{line}</span>;
  }

  const kv = line.match(/^(\s*)(-\s+)?([A-Za-z_$][\w.$-]*)(:)(\s*)(.*)$/);
  if (kv) {
    const [, indent, dash, key, colon, space, value] = kv;
    return (
      <>
        {indent}
        {dash ? <span className="text-offwhite/40">{dash}</span> : null}
        <span className="text-acid">{key}</span>
        <span className="text-offwhite/40">{colon}</span>
        {space}
        {renderValue(value)}
      </>
    );
  }

  const item = line.match(/^(\s*)(-\s+)(.+)$/);
  if (item) {
    const [, indent, dash, value] = item;
    return (
      <>
        {indent}
        <span className="text-offwhite/40">{dash}</span>
        <span className="text-acid/85">{value}</span>
      </>
    );
  }

  return <span className="text-offwhite/60">{line}</span>;
}

export default function CodePanel({
  filename,
  label,
  code,
  className,
  meta,
}: {
  filename: string;
  label?: string;
  code: string;
  className?: string;
  meta?: ReactNode;
}) {
  const [copied, setCopied] = useState(false);
  const lines = code.replace(/\n$/, "").split("\n");

  function copy() {
    navigator.clipboard.writeText(code).catch(() => {});
    setCopied(true);
    setTimeout(() => setCopied(false), 1600);
  }

  return (
    <div className={cn("flex h-full flex-col overflow-hidden bg-void", className)}>
      <div className="flex items-center justify-between gap-3 border-b border-hair px-4 py-2.5">
        <span className="flex min-w-0 items-center gap-2">
          {label && (
            <span className="font-mono text-[9px] uppercase tracking-micro text-offwhite/40">
              {label}
            </span>
          )}
          <span className="truncate font-mono text-[10px] text-offwhite/45">{filename}</span>
        </span>
        <div className="flex shrink-0 items-center gap-2">
          {meta}
          <button
            type="button"
            onClick={copy}
            aria-label="Copy code"
            className="flex h-5 w-5 items-center justify-center rounded border border-hair text-offwhite/40 transition-colors hover:border-acid/40 hover:text-acid"
          >
            {copied ? <Check size={11} className="text-acid" /> : <Copy size={11} />}
          </button>
        </div>
      </div>
      <div className="flex-1 overflow-x-auto py-3 font-mono text-[10.5px] leading-[1.9]">
        {lines.map((line, i) => (
          <div key={i} className="flex gap-3 px-4">
            <span className="w-4 shrink-0 select-none text-right text-offwhite/20">
              {i + 1}
            </span>
            <code className="whitespace-pre">{highlight(line)}</code>
          </div>
        ))}
      </div>
    </div>
  );
}
