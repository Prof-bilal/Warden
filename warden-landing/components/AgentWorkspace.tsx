import Link from "next/link";
import { Check, EyeOff, FileCheck2, ShieldCheck } from "lucide-react";

const POLICY = `agent:
  workspace: ./project
  redaction:
    apply: review
    env: [OPENAI_API_KEY]
    files:
      - path: .env
        format: dotenv
        fields: [DATABASE_URL]`;

export default function AgentWorkspace() {
  return (
    <section id="agent-workspace" className="border-t border-ink-800 bg-ink-900/30">
      <div className="mx-auto max-w-content px-6 py-20 md:py-28">
        <div className="grid items-start gap-10 lg:grid-cols-[0.9fr_1.1fr] lg:gap-16">
          <div>
            <p className="mb-4 font-sans text-xs font-semibold uppercase tracking-[0.14em] text-blueprint">
              New · AI agent workspace
            </p>
            <h2 className="font-sans text-[2rem] font-semibold leading-[1.1] tracking-[-0.03em] text-paper md:text-[2.5rem]">
              Let the agent edit code. Keep credentials out of sight.
            </h2>
            <p className="mt-5 max-w-[34rem] font-sans text-base leading-[1.75] text-muted">
              Warden gives Claude, Codex, OpenCode, or another local coding tool a temporary
              sanitized project. The agent can work on the code it needs without receiving the
              values you declare as protected.
            </p>
            <div className="mt-7 flex flex-col gap-3">
              <div className="flex gap-3">
                <EyeOff size={18} className="mt-0.5 shrink-0 text-blueprint" aria-hidden="true" />
                <p className="font-sans text-sm leading-[1.65] text-paper/85">
                  <strong className="font-semibold text-paper">Hide selected values.</strong> Environment names and file fields become opaque placeholders.
                </p>
              </div>
              <div className="flex gap-3">
                <FileCheck2 size={18} className="mt-0.5 shrink-0 text-blueprint" aria-hidden="true" />
                <p className="font-sans text-sm leading-[1.65] text-paper/85">
                  <strong className="font-semibold text-paper">Review before applying.</strong> Warden shows sanitized changes and waits for your approval.
                </p>
              </div>
              <div className="flex gap-3">
                <ShieldCheck size={18} className="mt-0.5 shrink-0 text-blueprint" aria-hidden="true" />
                <p className="font-sans text-sm leading-[1.65] text-paper/85">
                  <strong className="font-semibold text-paper">Fail safely.</strong> Failed sessions, tampered placeholders, and rejected reviews leave the host unchanged.
                </p>
              </div>
            </div>
            <Link
              href="/docs/agent-workspace"
              className="mt-8 inline-flex items-center gap-2 font-sans text-sm font-semibold text-blueprint transition-colors hover:text-paper"
            >
              Read the agent workspace guide <span aria-hidden="true">→</span>
            </Link>
          </div>

          <div className="overflow-hidden rounded-xl border border-ink-700 bg-ink-950 shadow-[0_20px_60px_rgba(0,0,0,0.2)]">
            <div className="flex items-center justify-between border-b border-ink-800 px-4 py-3">
              <span className="font-mono text-xs text-muted">agent-policy.yaml</span>
              <span className="flex items-center gap-1.5 font-mono text-[0.65rem] uppercase tracking-[0.1em] text-blueprint">
                <Check size={12} aria-hidden="true" /> review required
              </span>
            </div>
            <pre className="overflow-x-auto p-5 font-mono text-[0.76rem] leading-[1.8] text-paper/85"><code>{POLICY}</code></pre>
            <div className="grid gap-3 border-t border-ink-800 bg-ink-900/70 p-4 sm:grid-cols-2">
              <div className="rounded-lg border border-ink-700 bg-ink-950 p-4">
                <p className="font-mono text-[0.65rem] uppercase tracking-[0.12em] text-muted">Agent sees</p>
                <p className="mt-2 font-mono text-xs leading-[1.7] text-blueprint">DATABASE_URL=&lt;WARDEN_HIDDEN&gt;</p>
                <p className="font-mono text-xs leading-[1.7] text-blueprint">OPENAI_API_KEY=&lt;WARDEN_HIDDEN&gt;</p>
              </div>
              <div className="rounded-lg border border-ink-700 bg-ink-950 p-4">
                <p className="font-mono text-[0.65rem] uppercase tracking-[0.12em] text-muted">Host changes</p>
                <p className="mt-2 flex items-center gap-2 font-mono text-xs leading-[1.7] text-paper/85">
                  <Check size={13} className="text-blueprint" aria-hidden="true" /> sanitized diff
                </p>
                <p className="flex items-center gap-2 font-mono text-xs leading-[1.7] text-paper/85">
                  <Check size={13} className="text-blueprint" aria-hidden="true" /> human approval
                </p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
