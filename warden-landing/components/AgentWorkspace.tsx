import Link from "next/link";
import { EyeOff, FileCheck2, ShieldCheck } from "lucide-react";
import type { LucideIcon } from "lucide-react";
import SectionMarker from "@/components/bp/SectionMarker";
import CodePanel from "@/components/bp/CodePanel";
import Reveal from "@/components/bp/Reveal";

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
    <section id="agent-workspace" className="border-b border-hair">
      <div className="grid lg:grid-cols-[0.9fr_1.1fr]">
        <div className="border-b border-hair p-6 sm:p-9 lg:border-b-0 lg:border-r">
          <SectionMarker index="05" />
          <h2 className="mt-4 font-mono text-[1.55rem] font-semibold leading-[1.18] tracking-[-0.01em] text-offwhite md:text-[1.95rem]">
            Let the agent edit code. Keep credentials out of sight.
          </h2>
          <p className="mt-5 max-w-[34rem] font-sans text-[14.5px] leading-[1.75] text-offwhite/45">
            Warden gives Claude, Codex, OpenCode, or another local coding tool a
            temporary sanitized project. The agent can work on the code it needs
            without receiving the values you declare as protected.
          </p>

          <div className="mt-7 flex flex-col gap-4">
            <Feature
              icon={EyeOff}
              title="Hide selected values."
              body="Environment names and file fields become opaque placeholders."
            />
            <Feature
              icon={FileCheck2}
              title="Review before applying."
              body="Warden shows sanitized changes and waits for your approval."
            />
            <Feature
              icon={ShieldCheck}
              title="Fail safely."
              body="Failed sessions, tampered placeholders, and rejected reviews leave the host unchanged."
            />
          </div>

          <Link
            href="/docs/agent-workspace"
            className="mt-8 inline-flex items-center gap-2 font-sans text-[12.5px] font-semibold text-acid transition-colors hover:text-acid-bright"
          >
            Read the agent workspace guide <span aria-hidden>→</span>
          </Link>
        </div>

        <Reveal className="flex flex-col">
          <CodePanel
            label="Agent workspace"
            filename="agent-policy.yaml"
            code={POLICY}
            className="border-b border-hair"
            meta={
              <span className="font-mono text-[9px] uppercase tracking-micro text-acid/80">
                review required
              </span>
            }
          />

          <div className="grid flex-1 gap-px bg-hair sm:grid-cols-2">
            <div className="bg-cell p-4">
              <p className="font-mono text-[9px] uppercase tracking-micro text-offwhite/30">
                Agent sees
              </p>
              <p className="mt-2 font-mono text-[10.5px] leading-[1.8] text-acid/80">
                DATABASE_URL=&lt;WARDEN_HIDDEN&gt;
              </p>
              <p className="font-mono text-[10.5px] leading-[1.8] text-acid/80">
                OPENAI_API_KEY=&lt;WARDEN_HIDDEN&gt;
              </p>
            </div>
            <div className="bg-cell p-4">
              <p className="font-mono text-[9px] uppercase tracking-micro text-offwhite/30">
                Host changes
              </p>
              <p className="mt-2 font-mono text-[10.5px] leading-[1.8] text-offwhite/50">
                sanitized diff
              </p>
              <p className="font-mono text-[10.5px] leading-[1.8] text-offwhite/50">
                human approval
              </p>
            </div>
          </div>
        </Reveal>
      </div>
    </section>
  );
}

function Feature({
  icon: Icon,
  title,
  body,
}: {
  icon: LucideIcon;
  title: string;
  body: string;
}) {
  return (
    <div className="flex gap-3">
      <span className="mt-0.5 flex h-6 w-6 shrink-0 items-center justify-center rounded border border-hair bg-void">
        <Icon size={12} className="text-acid" aria-hidden />
      </span>
      <p className="font-sans text-[12.5px] leading-[1.65] text-offwhite/45">
        <strong className="font-semibold text-offwhite/80">{title}</strong> {body}
      </p>
    </div>
  );
}
