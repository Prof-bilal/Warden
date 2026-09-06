const GROUPS = [
  {
    verdict: "14 pass",
    detail: "Filesystem, GitHub, Slack, Postgres, SQLite, Brave Search, Drive, Git + more",
    tone: "text-grant",
  },
  {
    verdict: "2 conditional",
    detail: "Fetch and Kubernetes — need per-deployment hosts",
    tone: "text-progress",
  },
  {
    verdict: "2 fail — documented",
    detail: "Docker socket and Playwright wildcards can't be sandboxed honestly",
    tone: "text-muted",
  },
];

const MATRIX_URL =
  "https://github.com/Prof-bilal/Warden/blob/main/warden-starter/warden/docs/compatibility.md";

export default function Compatibility() {
  return (
    <section id="compatibility" className="border-t border-ink-800">
      <div className="mx-auto max-w-content px-6 py-20">
        <div className="grid gap-10 md:grid-cols-[22rem_1fr]">
          <div>
            <h2 className="text-[1.75rem] font-medium leading-[1.15] tracking-[-0.01em] text-paper">
              Tested against 18 real servers, not just our own fixtures.
            </h2>
            <p className="mt-4 text-[1rem] leading-[1.65] text-muted">
              Every server ships with the exact policy it needed, pinned as a
              regression fixture — so an update can&apos;t silently break what
              used to work.
            </p>
            <a
              href={MATRIX_URL}
              className="mt-6 inline-block text-[0.9375rem] text-blueprint transition-colors hover:text-paper"
            >
              Read the full compatibility matrix →
            </a>
          </div>

          <ul className="divide-y divide-ink-800 border-y border-ink-800">
            {GROUPS.map((g) => (
              <li key={g.verdict} className="flex items-baseline justify-between gap-6 py-4">
                <span className="font-mono text-[0.875rem] text-paper">{g.detail}</span>
                <span className={`shrink-0 text-[0.8125rem] ${g.tone}`}>{g.verdict}</span>
              </li>
            ))}
          </ul>
        </div>
      </div>
    </section>
  );
}
