interface EyebrowProps {
  children: React.ReactNode;
  className?: string;
}

export default function Eyebrow({ children, className }: EyebrowProps) {
  return (
    <div className={`flex items-center gap-2.5 ${className || ""}`}>
      <span className="h-[6px] w-[6px] shrink-0 rounded-full bg-blueprint" aria-hidden />
      <span className="font-mono text-[0.6875rem] font-medium uppercase tracking-[0.14em] text-muted md:text-[0.75rem] md:tracking-[0.12em]">
        {children}
      </span>
    </div>
  );
}