const TESTIMONIALS = [
  {
    quote: "Policy as code, kernel-enforced boundaries. The fail-closed design means Warden never runs my MCP servers unsandboxed — even when the sandbox primitive is missing.",
    author: "The Warden Design Principles",
    role: "fail-closed by default, verified by tests",
  },
  {
    quote: "Every denied syscall, blocked connection, and filtered MCP message lands in a JSONL audit log — evidence you can review with `warden logs`.",
    author: "Warden Audit Trail",
    role: "every decision logged, allowed or blocked",
  },
  {
    quote: "The proxy answers blocked tool calls with a JSON-RPC error instead of forwarding them. Nothing blocked ever reaches the upstream server.",
    author: "Warden MCP Proxy",
    role: "stdio JSON-RPC filtering, deny-by-default",
  },
];

export default function Testimonials() {
  return (
    <section className="border-t border-ink-800">
      <div className="mx-auto max-w-content px-6 py-20">
        <h2 className="text-[1.75rem] font-medium leading-[1.15] tracking-[-0.01em] text-paper">
          Why teams choose Warden for MCP sandboxing.
        </h2>
        <p className="mt-3 max-w-[34rem] text-[1rem] leading-[1.65] text-muted">
          The design guarantees behind the project — taken directly from the
          code and its tests, not from customer quotes. Real user stories will
          be added here as they come in.
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