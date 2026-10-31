"use client";

import type { ReactNode } from "react";
import { cn } from "@/lib/cn";
import { RevealGroup, RevealItem } from "./Reveal";

export type LogStatus = "ALLOW" | "DENY" | "INFO";

export type LogRow = {
  time: string;
  status: LogStatus;
  text: ReactNode;
};

const tone: Record<LogStatus, string> = {
  ALLOW: "text-acid",
  DENY: "text-deny",
  INFO: "text-offwhite/30",
};

const glyph: Record<LogStatus, string> = {
  ALLOW: "✓",
  DENY: "✗",
  INFO: "·",
};

export default function LogPanel({
  title,
  live,
  rows,
  footer,
  className,
}: {
  title: string;
  live?: string;
  rows: LogRow[];
  footer?: ReactNode;
  className?: string;
}) {
  return (
    <div className={cn("flex h-full flex-col bg-void", className)}>
      <div className="flex items-center justify-between gap-3 border-b border-hair px-4 py-2.5">
        <span className="font-mono text-[9px] uppercase tracking-micro text-offwhite/40">
          {title}
        </span>
        {live && (
          <span className="flex items-center gap-1.5 font-mono text-[9px] text-acid">
            <span className="h-1.5 w-1.5 rounded-full bg-acid animate-bp-node" aria-hidden />
            {live}
          </span>
        )}
      </div>

      <RevealGroup className="flex-1 overflow-x-auto py-3 font-mono text-[10.5px] leading-[1.95]">
        {rows.map((row, i) => (
          <RevealItem key={i} y={6} className="flex gap-2.5 px-4">
            <span className="shrink-0 text-acid/60">{row.time}</span>
            <span
              className={cn(
                "w-14 shrink-0 whitespace-pre text-[9.5px] font-medium",
                tone[row.status],
              )}
            >
              {row.status === "INFO" ? "" : `${glyph[row.status]} `}
              {row.status}
            </span>
            <span className="min-w-0 flex-1 truncate text-offwhite/60">{row.text}</span>
          </RevealItem>
        ))}
      </RevealGroup>

      {footer && (
        <div className="flex items-center gap-2 border-t border-hair px-4 py-2.5 font-mono text-[10px] text-acid">
          {footer}
        </div>
      )}
    </div>
  );
}
