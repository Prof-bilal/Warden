import { Boxes, Globe, Terminal } from "lucide-react";
import type { LucideIcon } from "lucide-react";
import { cn } from "@/lib/cn";

const items: LucideIcon[] = [Boxes, Terminal, Globe];

const traces = [
  "M48 30 H104 Q112 30 112 38 V77 Q112 85 120 85 H152",
  "M48 85 H152",
  "M48 140 H104 Q112 140 112 132 V93 Q112 85 120 85 H152",
];

export default function CircuitStack({ className }: { className?: string }) {
  return (
    <div className={cn("relative h-[176px] w-[206px]", className)}>
      <svg
        width="200"
        height="170"
        viewBox="0 0 200 170"
        className="absolute left-0 top-0 overflow-visible"
        fill="none"
        aria-hidden
      >
        {traces.map((d, i) => (
          <path key={i} d={d} stroke="#5EF08A" strokeOpacity="0.45" strokeWidth="1" />
        ))}
        <circle cx="104" cy="30" r="2.5" fill="#5EF08A" className="animate-bp-node" />
        <circle
          cx="104"
          cy="140"
          r="2.5"
          fill="#5EF08A"
          className="animate-bp-node"
          style={{ animationDelay: "0.5s" }}
        />
        <rect x="152" y="77" width="16" height="16" rx="2.5" fill="#5EF08A" />
      </svg>

      {items.map((Icon, i) => (
        <span
          key={i}
          className="absolute left-0 flex h-12 w-12 items-center justify-center rounded-full border border-hair bg-void"
          style={{ top: 6 + i * 55 }}
        >
          <Icon size={18} className="text-offwhite/60" aria-hidden />
        </span>
      ))}
    </div>
  );
}
