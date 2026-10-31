import { cn } from "@/lib/cn";

export default function StatRow({
  stats,
  className,
}: {
  stats: [string, string][];
  className?: string;
}) {
  return (
    <div
      className={cn(
        "grid grid-cols-2 gap-px bg-hair sm:grid-cols-4",
        className,
      )}
    >
      {stats.map(([value, label]) => (
        <div key={label} className="bg-cell px-4 py-5">
          <p className="font-mono text-[1.5rem] font-semibold leading-none tracking-[-0.02em] text-offwhite">
            {value}
          </p>
          <p className="mt-2 font-mono text-[9px] uppercase tracking-micro text-offwhite/35">
            {label}
          </p>
        </div>
      ))}
    </div>
  );
}
