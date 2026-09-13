import Diagram from "@/components/Diagram";

const GROUPS = [
  {
    verdict: "14 pass",
    detail:
      "Filesystem, GitHub, Slack, Postgres, SQLite, Brave Search, Drive, Git + more",
    tone: "grant" as const,
    icon: (
      <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
        <path
          d="M3.5 8.5l3 3 6-7"
          stroke="currentColor"
          strokeWidth="1.6"
          strokeLinecap="round"
          strokeLinejoin="round"
        />
      </svg>
    ),
    iconClass: "text-grant border-grant/30 bg-grant-subtle",
  },
  {
    verdict: "2 conditional",
    detail: "Fetch and Kubernetes — need per-deployment hosts",
    tone: "progress" as const,
    icon: (
      <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
        <circle cx="8" cy="8" r="6.2" stroke="currentColor" strokeWidth="1.4" />
        <path d="M8 4.8v3.8" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
        <circle cx="8" cy="11" r="0.9" fill="currentColor" />
      </svg>
    ),
    iconClass: "text-progress border-progress/30 bg-progress-subtle",
  },
  {
    verdict: "2 fail — documented",
    detail:
      "Docker socket and Playwright wildcards can't be sandboxed honestly",
    tone: "deny" as const,
    icon: (
      <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
        <path d="M4.2 4.2l7.6 7.6M11.8 4.2l-7.6 7.6" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" />
      </svg>
    ),
    iconClass: "text-deny border-deny/30 bg-deny-subtle",
  },
];

const MATRIX_URL =
  "https://github.com/Prof-bilal/Warden/blob/main/warden-starter/warden/docs/compatibility.md";

export default function Compatibility() {
  return (
    <section id="compatibility" className="border-t border-ink-800">
      <div className="mx-auto max-w-content px-6 py-20">
        <div className="grid gap-10 lg:grid-cols-[24rem_1fr] lg:gap-14">
          {/* Left: heading + description + CTA */}
          <div className="lg:sticky lg:top-24 lg:self-start">
            <span className="inline-block rounded-full border border-ink-700 bg-ink-900 px-3 py-1 font-mono text-[0.6875rem] uppercase tracking-[0.08em] text-muted">
              Compatibility
            </span>
            <h2 className="mt-5 text-[1.75rem] font-medium leading-[1.15] tracking-[-0.01em] text-paper">
              Tested against 18 real servers, not just our own fixtures.
            </h2>
            <p className="mt-4 text-[1rem] leading-[1.65] text-muted">
              Every server ships with the exact policy it needed, pinned as a
              regression fixture — so an update can&apos;t silently break what
              used to work.
            </p>
            <a
              href={MATRIX_URL}
              className="mt-6 inline-flex items-center gap-2 rounded-[8px] border border-ink-600 bg-ink-900 px-4 py-2.5 text-[0.875rem] font-medium text-paper transition-colors hover:border-blueprint/50 hover:text-white"
            >
              Read the full compatibility matrix
              <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
                <path
                  d="M2.5 7h9m0 0L8 3.5M11.5 7L8 10.5"
                  stroke="currentColor"
                  strokeWidth="1.4"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                />
              </svg>
            </a>
          </div>

          {/* Right: 3 status cards */}
          <div className="flex flex-col gap-4">
            {GROUPS.map((g) => (
              <div
                key={g.verdict}
                className="group rounded-[12px] border border-ink-700 bg-ink-900 p-5 transition-colors hover:border-ink-600"
              >
                <div className="flex items-start gap-4">
                  <span
                    className={`flex h-9 w-9 shrink-0 items-center justify-center rounded-full border ${g.iconClass}`}
                  >
                    {g.icon}
                  </span>
                  <div className="min-w-0 flex-1">
                    <div className="flex flex-wrap items-baseline justify-between gap-x-3">
                      <h3 className={`text-[1.0625rem] font-medium text-${g.tone}`}>
                        {g.verdict}
                      </h3>
                    </div>
                    <p className="mt-1 text-[0.9375rem] leading-[1.55] text-muted">
                      {g.detail}
                    </p>
                  </div>
                </div>
              </div>
            ))}

            {/* Footer image strip */}
            <div className="mt-2 overflow-hidden rounded-[12px] border border-ink-800">
              <Diagram
                src="/diagrams/architecture-security.jpeg"
                alt="Security architecture across filesystem, network, and environment layers"
              />
              <div className="border-t border-ink-800 bg-ink-900 px-5 py-3">
                <p className="text-[0.75rem] leading-[1.5] text-muted">
                  <span className="text-paper/85">One policy, every layer.</span>{" "}
                  Filesystem, network, environment, and audit are enforced
                  together — the same policy shape protects all 18 servers.
                </p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
