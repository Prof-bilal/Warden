export default function Preview() {
  return (
    <section className="border-t border-ink-800">
      <div className="mx-auto max-w-content px-6 py-20">
        <div className="flex flex-col gap-2 md:flex-row md:items-baseline md:justify-between">
          <h2 className="text-[1.75rem] font-medium leading-[1.15] tracking-[-0.01em] text-paper">
            The new Warden UI.
          </h2>
          <span className="font-mono text-[0.6875rem] uppercase tracking-[0.08em] text-muted">
            Interactive stepper · live detail panel
          </span>
        </div>
        <p className="mt-3 max-w-[34rem] text-[1rem] leading-[1.65] text-muted">
          Every section now uses a single-detail stepper: tap a step on the left,
          see the full explanation and diagram on the right. Shorter scan,
          richer context.
        </p>

        <div className="mt-10 overflow-hidden rounded-[12px] border border-ink-700 bg-ink-900">
          <img
            src="/images/ui-preview.svg"
            alt="Warden UI preview: interactive stepper with numbered steps on the left and a detail panel with security diagram on the right"
            className="w-full"
            width={1920}
            height={1080}
          />
        </div>
        <p className="mt-3 text-center font-mono text-[0.6875rem] uppercase tracking-[0.08em] text-muted">
          Hover or tap a step to explore the sandbox invariants
        </p>
      </div>
    </section>
  );
}
