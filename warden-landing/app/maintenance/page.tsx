import Link from "next/link";
import EcosystemShell from "@/components/EcosystemShell";
import CodeBlock from "@/components/CodeBlock";
import LocalPilotReport from "@/components/LocalPilotReport";
import { createMetadata } from "@/lib/seo";
export const metadata = createMetadata({
  title: "Review upgrades and local adoption",
  description:
    "Inspect policy and configuration drift and summarize explicit local pilot events.",
  path: "/maintenance",
});
export default function Page() {
  return (
    <EcosystemShell
      active="/maintenance"
      eyebrow="Maintain your boundary"
      title="Review changes. Keep evidence current."
      description="Check permission changes before upgrades, detect launcher drift, and measure your own setup experience without collecting tool conversations."
    >
      <div className="mb-8 grid gap-6 lg:grid-cols-2">
        <section className="min-w-0 rounded-xl border border-ink-700 p-6">
          <h2 className="text-xl font-semibold">Before an upgrade</h2>
          <CodeBlock className="mt-5">
            {[
              "warden policy-diff --before old.yaml --after candidate.yaml",
              "warden inventory --client cursor --scope project",
              "warden unwrap --client cursor --server filesystem --yes",
            ].join("\n")}
          </CodeBlock>
          <p className="mt-4 text-sm leading-relaxed text-muted">
            New grants, changed commands and expanded limits require review.
            Keep the original prepared artifact and configuration backup for
            rollback.
          </p>
        </section>
        <section className="min-w-0 rounded-xl border border-ink-700 p-6">
          <h2 className="text-xl font-semibold">Measure your pilot locally</h2>
          <CodeBlock className="mt-5">
            {[
              "warden report record --file ./pilot.jsonl --client cursor --event setup-completed --seconds 90",
              "warden report record --file ./pilot.jsonl --client cursor --event first-protected-task",
              "warden report summary --file ./pilot.jsonl",
            ].join("\n")}
          </CodeBlock>
          <p className="mt-4 text-sm leading-relaxed text-muted">
            Explicit events include failures, rollback, repeat use and badge
            activations. Record only test data outcomes. No telemetry runs in
            the background.
          </p>
        </section>
      </div>
      <LocalPilotReport />
      <Link
        href="/docs/ecosystem-maintenance"
        className="mt-8 block text-blueprint"
      >
        Release, contribution and incident guide →
      </Link>
    </EcosystemShell>
  );
}
