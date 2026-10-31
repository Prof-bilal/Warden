import { cn } from "@/lib/cn";

// Space Grotesk is proportional, so the size is tuned empirically so the
// ink of "warden" spans the container width (~96%, with a small margin).
const SIZE = "text-[28.4cqw]";

export default function Wordmark({
  text = "warden",
  className,
}: {
  text?: string;
  className?: string;
}) {
  return (
    <div className={cn("w-full [container-type:inline-size]", className)}>
      <div className="flex h-[32cqw] w-full items-center justify-center">
        <span
          className={cn(
            "font-wordmark font-bold lowercase leading-none tracking-[-0.02em] text-offwhite/90",
            SIZE,
          )}
        >
          {text}
        </span>
      </div>
    </div>
  );
}
