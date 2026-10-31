import { Check, X } from "lucide-react";
import SectionMarker from "@/components/bp/SectionMarker";
import Reveal from "@/components/bp/Reveal";

const withoutWarden = [
  { command: "read ~/.ssh/id_rsa", result: "readable" },
  { command: "read .env", result: "readable" },
  { command: "connect api.anywhere.com", result: "reachable" },
  { command: "run rm -rf ~", result: "executes" },
];

const withWarden = [
  { command: "read ./src/**", result: "allowed" },
  { command: "read .env", result: "denied" },
  { command: "connect api.github.com", result: "allowed" },
  { command: "connect api.openai.com", result: "denied" },
  { command: "run rm -rf ~", result: "denied" },
];

export default function BoundaryExample() {
  return (
    <section id="boundary" className="border-b border-hair">
      <div className="px-6 py-16 md:py-20">
        <div className="max-w-2xl">
          <SectionMarker index="04" />
          <h2 className="mt-4 font-mono text-[1.55rem] font-semibold leading-[1.18] tracking-[-0.01em] text-offwhite md:text-[1.95rem]">
            Same tool. Smaller boundary.
          </h2>
          <p className="mt-4 font-sans text-[14.5px] leading-[1.75] text-offwhite/45">
            A local tool inherits the permissions of the user who started it. Warden
            replaces that implicit access with an explicit allowlist, so a request
            that is not written down cannot happen.
          </p>
        </div>

        <Reveal className="mt-10 grid gap-px border border-hair bg-hair md:grid-cols-2">
          <ComparisonCard
            eyebrow="Without warden"
            title="Inherits your session"
            rows={withoutWarden}
            tone="deny"
          />
          <ComparisonCard
            eyebrow="With warden"
            title="Explicit policy, deny by default"
            rows={withWarden}
            tone="grant"
          />
        </Reveal>

        <p className="mt-4 font-mono text-[10px] leading-[1.7] text-offwhite/25">
          Illustration of policy scope, not a live security test. The verification
          harness and platform notes are published separately.
        </p>
      </div>
    </section>
  );
}

function ComparisonCard({
  eyebrow,
  title,
  rows,
  tone,
}: {
  eyebrow: string;
  title: string;
  rows: { command: string; result: string }[];
  tone: "grant" | "deny";
}) {
  const granted = tone === "grant";

  return (
    <div className="flex h-full flex-col bg-cell">
      <div className="flex items-center justify-between border-b border-hair px-5 py-3">
        <span
          className={`font-mono text-[9.5px] font-medium uppercase tracking-micro ${
            granted ? "text-acid" : "text-deny"
          }`}
        >
          {eyebrow}
        </span>
        <span
          className={`h-1.5 w-1.5 rounded-full ${granted ? "bg-acid" : "bg-deny"}`}
          aria-hidden
        />
      </div>

      <p className="px-5 pt-4 font-mono text-[12.5px] font-medium text-offwhite/80">{title}</p>

      <ul className="mt-2 flex-1 px-5">
        {rows.map((row) => {
          const denied = row.result === "denied" || tone === "deny";
          return (
            <li
              key={row.command}
              className="flex items-center gap-3 border-b border-hair-soft py-2.5 last:border-b-0"
            >
              {denied ? (
                <X size={11} className="shrink-0 text-deny" aria-hidden />
              ) : (
                <Check size={11} className="shrink-0 text-acid" aria-hidden />
              )}
              <code className="min-w-0 flex-1 truncate font-mono text-[10.5px] text-offwhite/50">
                {row.command}
              </code>
              <span
                className={`shrink-0 font-mono text-[9px] uppercase tracking-micro ${
                  denied ? "text-deny" : "text-acid"
                }`}
              >
                {row.result}
              </span>
            </li>
          );
        })}
      </ul>

      <div className="border-t border-hair px-5 py-3">
        <p className="font-sans text-[10.5px] leading-[1.6] text-offwhite/30">
          {granted
            ? "Everything outside the policy fails closed before execution."
            : "Access is limited only by what your user account can already reach."}
        </p>
      </div>
    </div>
  );
}
