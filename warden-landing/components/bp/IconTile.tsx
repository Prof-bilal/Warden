import type { LucideIcon } from "lucide-react";
import { cn } from "@/lib/cn";

export default function IconTile({
  icon: Icon,
  size = 14,
  active = false,
  className,
}: {
  icon: LucideIcon;
  size?: number;
  active?: boolean;
  className?: string;
}) {
  return (
    <span
      className={cn(
        "flex h-9 w-9 shrink-0 items-center justify-center rounded-md border bg-void",
        active ? "border-acid/45 text-acid" : "border-hair text-offwhite/55",
        className,
      )}
    >
      <Icon size={size} aria-hidden />
    </span>
  );
}
