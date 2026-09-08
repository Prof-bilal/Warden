const ATTACKS_1 = [
  {
    attack: "CPU Exhaustion",
    severity: "CRITICAL",
    without: "12.2M hashes in 6.5s",
    with_: "BLOCKED (38ms)",
    protected_: "YES",
  },
  {
    attack: "Ransomware",
    severity: "CRITICAL",
    without: "5/5 files encrypted in 88ms",
    with_: "BLOCKED (33ms)",
    protected_: "YES",
  },
  {
    attack: "Credential Harvester",
    severity: "CRITICAL",
    without: "1 token + 6 SSH keys stolen",
    with_: "BLOCKED (27ms)",
    protected_: "YES",
  },
  {
    attack: "DDoS",
    severity: "HIGH",
    without: "25K connection attempts",
    with_: "BLOCKED (27ms)",
    protected_: "YES",
  },
  {
    attack: "Data Wiper",
    severity: "CRITICAL",
    without: "1 file destroyed",
    with_: "BLOCKED (29ms)",
    protected_: "YES",
  },
];

const ATTACKS_2 = [
  {
    attack: "01-filesystem-exfil",
    what: "Stole SSH keys, .env, AWS creds",
    without: "DATA EXFILTRATED",
    with_: "BLOCKED",
  },
  {
    attack: "02-network-exfil",
    what: "Collected system fingerprint, sent to 3 hosts",
    without: "EXFIL ATTEMPTED",
    with_: "BLOCKED",
  },
  {
    attack: "03-env-stealer",
    what: "Harvested 3 secrets from 59 env vars",
    without: "SECRETS STOLEN",
    with_: "BLOCKED",
  },
  {
    attack: "04-process-spawner",
    what: "Ran whoami, dir, netstat, wrote files",
    without: "FULL SYSTEM ACCESS",
    with_: "BLOCKED",
  },
  {
    attack: "05-symlink-traversal",
    what: "Path traversal, symlinks, UNC paths",
    without: "PARTIAL ACCESS",
    with_: "BLOCKED",
  },
];

const FINDINGS = [
  {
    title: "Fail-closed is the strongest defense",
    body: "Warden refuses to start without a valid sandbox backend, so attacks never execute at all.",
  },
  {
    title: "Ransomware is devastatingly fast unsandboxed",
    body: "5 files encrypted in 88ms, originals overwritten and deleted.",
  },
  {
    title: "Credential theft is silent unsandboxed",
    body: "SSH keys and secrets stolen with no detection until Warden blocks the access attempt itself.",
  },
  {
    title: "Blocking is fast, not just safe",
    body: "Blocked attempts resolve in 27\u201338ms, versus 66ms\u20136.8s for the same attacks running to completion unsandboxed. Sandboxing here isn\u2019t a performance tradeoff.",
  },
];

const SEVERITY_STYLE: Record<string, string> = {
  CRITICAL: "text-deny bg-deny-subtle",
  HIGH: "text-progress bg-progress-subtle",
};

export default function Proof() {
  return (
    <section id="proof" className="border-t border-ink-800">
      <div className="mx-auto max-w-content px-6 py-20">
        <h2 className="text-[1.75rem] font-medium leading-[1.15] tracking-[-0.01em] text-paper">
          It blocks real attacks, not just hypotheticals.
        </h2>
        <p className="mt-3 max-w-[34rem] text-[1rem] leading-[1.65] text-muted">
          Two batches of attack simulations ran against Warden&apos;s Windows
          AppContainer backend. Every attempt was stopped before it could do
          damage.
        </p>

        {/* Table 1 — Attack simulation */}
        <div className="mt-10 overflow-x-auto">
          <table className="w-full border-y border-ink-800 text-left">
            <thead>
              <tr className="border-b border-ink-800 text-[0.8125rem] text-muted">
                <th className="py-3 pr-4 font-normal">Attack</th>
                <th className="py-3 pr-4 font-normal">Severity</th>
                <th className="py-3 pr-4 font-normal">Without Warden</th>
                <th className="py-3 pr-4 font-normal">With Warden</th>
                <th className="py-3 font-normal">Protected</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-ink-800 text-[0.875rem]">
              {ATTACKS_1.map((a) => (
                <tr key={a.attack}>
                  <td className="py-3.5 pr-4 font-mono text-paper">{a.attack}</td>
                  <td className="py-3.5 pr-4">
                    <span
                      className={`w-fit rounded-sm px-2.5 py-1 text-[0.75rem] ${SEVERITY_STYLE[a.severity]}`}
                    >
                      {a.severity}
                    </span>
                  </td>
                  <td className="py-3.5 pr-4 text-deny">{a.without}</td>
                  <td className="py-3.5 pr-4 font-mono text-grant">{a.with_}</td>
                  <td className="py-3.5 text-grant">{a.protected_}</td>
                </tr>
              ))}
            </tbody>
          </table>
          <p className="mt-3 text-[0.8125rem] text-muted">
            Detection rate: <span className="font-mono text-grant">5/5 (100%)</span>
          </p>
        </div>

        {/* Table 2 — Results summary */}
        <div className="mt-12 overflow-x-auto">
          <table className="w-full border-y border-ink-800 text-left">
            <thead>
              <tr className="border-b border-ink-800 text-[0.8125rem] text-muted">
                <th className="py-3 pr-4 font-normal">Attack</th>
                <th className="py-3 pr-4 font-normal">What it did</th>
                <th className="py-3 pr-4 font-normal">Without Warden</th>
                <th className="py-3 font-normal">With Warden</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-ink-800 text-[0.875rem]">
              {ATTACKS_2.map((a) => (
                <tr key={a.attack}>
                  <td className="py-3.5 pr-4 font-mono text-paper">{a.attack}</td>
                  <td className="py-3.5 pr-4 text-muted">{a.what}</td>
                  <td className="py-3.5 pr-4 text-deny">{a.without}</td>
                  <td className="py-3.5 font-mono text-grant">{a.with_}</td>
                </tr>
              ))}
            </tbody>
          </table>
          <p className="mt-3 text-[0.8125rem] text-muted">
            Verdict: <span className="font-mono text-grant">5/5 attacks contained. 0 bypasses.</span>
          </p>
        </div>

        {/* Fail-closed note */}
        <blockquote className="mt-10 border-l-2 border-ink-600 pl-4 text-[0.9375rem] leading-[1.6] text-muted">
          Warden&apos;s fail-closed behavior prevented every attack in both
          batches&nbsp;&mdash; when the AppContainer backend couldn&apos;t
          initialize (for example, no administrator privileges), Warden refused
          to run rather than executing unsandboxed. See the CLI&apos;s refusal
          message for what this looks like in practice.
        </blockquote>

        {/* Findings */}
        <ol className="mt-8 space-y-4 text-[0.9375rem] leading-[1.6] text-muted">
          {FINDINGS.map((f) => (
            <li key={f.title}>
              <span className="text-paper">{f.title}</span> &mdash; {f.body}
            </li>
          ))}
        </ol>

        {/* Honesty disclaimer */}
        <p className="mt-10 text-[0.8125rem] leading-[1.6] text-muted/80">
          Run internally against Warden&apos;s Windows backend. Not yet
          independently audited&nbsp;&mdash; see{" "}
          <a
            className="underline decoration-muted/40 hover:text-paper"
            href="https://github.com/Prof-bilal/Warden/blob/main/warden-starter/warden/ROADMAP.md"
          >
            ROADMAP.md
          </a>{" "}
          for what&apos;s still in progress. These numbers reflect the Windows
          AppContainer backend specifically; equivalent test data for the Linux
          and macOS backends is not yet available.
        </p>
      </div>
    </section>
  );
}
