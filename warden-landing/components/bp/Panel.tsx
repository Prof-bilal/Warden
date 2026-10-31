import type { ReactNode } from "react";
import { cn } from "@/lib/cn";

export default function Panel({
  label,
  meta,
  children,
  className,
  bodyClassName,
}: {
  label?: ReactNode;
  meta?: ReactNode;
  children: ReactNode;
  className?: string;
  bodyClassName?: string;
}) {
  return (
    <div className={cn("relative flex h-full flex-col bg-cell", className)}>
      {(label || meta) && (
        <div className="flex items-center justify-between gap-3 border-b border-hair px-4 py-2.5">
          <span className="font-mono text-[9px] uppercase tracking-micro text-offwhite/40">
            {label}
          </span>
          {meta}
        </div>
      )}
      <div className={cn("flex-1 p-4 sm:p-5", bodyClassName)}>{children}</div>
    </div>
  );
}
