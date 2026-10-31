export default function SectionMarker({
  index,
  label,
}: {
  index: string;
  label?: string;
}) {
  return (
    <div className="flex items-center gap-3">
      <span className="font-mono text-[10px] text-acid">{index}</span>
      <span className="h-px w-8 bg-acid/40" aria-hidden />
      {label && (
        <span className="font-mono text-[9.5px] uppercase tracking-micro text-offwhite/35">
          {label}
        </span>
      )}
    </div>
  );
}
