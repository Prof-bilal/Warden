const POLICY = `command: ["node", "server.js"]

filesystem:
  read: ["./data"]
  write: ["./output"]

network:
  allow: ["api.github.com"]

env:
  allow: ["GITHUB_TOKEN"]

limits:
  memory_mb: 512
  timeout_s: 300`;

const FIELDS = [
  { field: "command", note: "how Warden starts the server — nothing else runs." },
  {
    field: "filesystem",
    note: "read and write are separate grants; anything not listed is invisible, not just unreadable.",
  },
  {
    field: "network",
    note: "only these hostnames resolve. DNS for anything else fails before a connection is even attempted.",
  },
  { field: "env", note: "only these variables are passed through — no inherited shell environment." },
  { field: "limits", note: "the process is killed if either bound is crossed." },
];

export default function HowItWorks() {
  return (
    <section id="how-it-works" className="border-t border-ink-800">
      <div className="mx-auto max-w-content px-6 py-20">
        <h2 className="max-w-[28rem] text-[1.75rem] font-medium leading-[1.15] tracking-[-0.01em] text-paper">
          One file describes exactly what a server can touch.
        </h2>

        <div className="mt-10 grid gap-10 md:grid-cols-[1fr_20rem]">
          <pre className="overflow-x-auto rounded-sm border border-ink-700 bg-ink-900 p-5 font-mono text-[0.8125rem] leading-[1.7] text-paper">
            <code>{POLICY}</code>
          </pre>

          <dl className="space-y-5">
            {FIELDS.map((f) => (
              <div key={f.field}>
                <dt className="font-mono text-[0.8125rem] text-blueprint">{f.field}</dt>
                <dd className="mt-1 text-[0.9375rem] leading-[1.55] text-muted">{f.note}</dd>
              </div>
            ))}
          </dl>
        </div>
      </div>
    </section>
  );
}
