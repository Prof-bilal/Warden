import type { LucideIcon } from "lucide-react";
import { cn } from "@/lib/cn";
import WardenMark from "./WardenMark";

export type NodeItem = { icon: LucideIcon; label: string };

function NodeRow({ item, side }: { item: NodeItem; side: "left" | "right" }) {
  const Icon = item.icon;
  return (
    <li
      className={cn(
        "relative z-10 flex items-center gap-2 rounded-md border border-hair bg-void px-3 py-2",
        side === "right" && "flex-row-reverse text-right",
      )}
    >
      <Icon size={13} className="shrink-0 text-offwhite/55" aria-hidden />
      <span className="truncate font-mono text-[10px] text-offwhite/65">{item.label}</span>
    </li>
  );
}

export default function NodeGraph({
  left = [],
  right = [],
  centerLabel = "warden",
  className,
}: {
  left?: NodeItem[];
  right?: NodeItem[];
  centerLabel?: string;
  className?: string;
}) {
  const rows = Math.max(left.length, right.length, 1);
  const yFor = (i: number) => ((i + 0.5) / rows) * 100;

  return (
    <div
      className={cn(
        "relative grid grid-cols-[1fr_auto_1fr] items-center gap-x-6 sm:gap-x-10",
        className,
      )}
    >
      <svg
        className="pointer-events-none absolute inset-0 h-full w-full"
        viewBox="0 0 100 100"
        preserveAspectRatio="none"
        aria-hidden
      >
        {left.map((_, i) => (
          <line
            key={`l${i}`}
            x1="30"
            y1={yFor(i)}
            x2="47"
            y2="50"
            stroke="#5EF08A"
            strokeOpacity="0.4"
            strokeWidth="1"
            strokeDasharray="60"
            vectorEffect="non-scaling-stroke"
            className="animate-bp-dash"
            style={{ animationDelay: `${i * 0.1}s` }}
          />
        ))}
        {right.map((_, i) => (
          <line
            key={`r${i}`}
            x1="53"
            y1="50"
            x2="70"
            y2={yFor(i)}
            stroke="#5EF08A"
            strokeOpacity="0.4"
            strokeWidth="1"
            strokeDasharray="60"
            vectorEffect="non-scaling-stroke"
            className="animate-bp-dash"
            style={{ animationDelay: `${i * 0.1}s` }}
          />
        ))}
      </svg>

      <ul className="flex flex-col gap-2.5">
        {left.map((item) => (
          <NodeRow key={item.label} item={item} side="left" />
        ))}
      </ul>

      <div className="relative z-10 flex flex-col items-center gap-1.5 px-1">
        <span className="flex h-10 w-10 items-center justify-center rounded-lg border border-acid/50 bg-acid/10 text-acid">
          <WardenMark size={20} />
        </span>
        <span className="font-mono text-[9px] uppercase tracking-micro text-acid">
          {centerLabel}
        </span>
      </div>

      <ul className="flex flex-col gap-2.5">
        {right.map((item) => (
          <NodeRow key={item.label} item={item} side="right" />
        ))}
      </ul>
    </div>
  );
}
