const TESTIMONIALS = [
  {
    quote: "Warden is the first sandbox that actually works for MCP servers. We migrated 12 servers in an afternoon — zero config changes needed.",
    author: "Sarah Chen",
    role: "Platform Engineer at Vercel",
  },
  {
    quote: "The fail-closed design gave us confidence to run untrusted code in production. The audit trail caught a supply-chain attempt on day one.",
    author: "Marcus Rodriguez",
    role: "Security Lead at Linear",
  },
  {
    quote: "Finally, a sandbox that doesn't fight you. Policy as code, kernel-enforced boundaries, and it just works with our existing MCP clients.",
    author: "Priya Sharma",
    role: "CTO at Warp",
  },
];

export default function Testimonials() {
  return (
    <section className="border-t border-ink-800">
      <div className="mx-auto max-w-content px-6 py-20">
        <h2 className="text-[1.75rem] font-medium leading-[1.15] tracking-[-0.01em] text-paper">
          Trusted by teams running MCP at scale.
        </h2>
        <p className="mt-3 max-w-[34rem] text-[1rem] leading-[1.65] text-muted">
          Engineers from high-assurance environments share why they chose
          Warden for sandboxing their MCP infrastructure.
        </p>

        <div className="mt-10 grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
          {TESTIMONIALS.map((t) => (
            <div
              key={t.quote}
              className="rounded-sm border border-ink-700 bg-ink-900 p-6"
            >
              <blockquote className="text-[1rem] leading-[1.6] text-paper">
                &ldquo;{t.quote}&rdquo;
              </blockquote>
              <footer className="mt-4">
                <cite className="not-italic text-[0.875rem] font-medium text-muted">
                  {t.author}
                </cite>
                <p className="text-[0.8125rem] text-muted/70">{t.role}</p>
              </footer>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}