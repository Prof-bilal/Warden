import Link from "next/link";
import EcosystemShell from "@/components/EcosystemShell";
import CodeBlock from "@/components/CodeBlock";
import { createMetadata } from "@/lib/seo";
export const metadata = createMetadata({
  title: "Secured by Warden creator program",
  description:
    "Include a reviewed policy and publish signed, version-specific allowed and denied MCP checks.",
  path: "/creators",
});
const commands = [
  "warden creator init --output ./warden-kit",
  "warden creator keygen --output /private/issuer.key",
  "warden creator check --policy policy.yaml --rules rules.json --artifact /prepared/runtime --manifest checks.json --key /private/issuer.key --output evidence.json -- /prepared/server",
  "warden creator verify --policy policy.yaml --rules rules.json --artifact /prepared/runtime --manifest checks.json --public-key /trusted/issuer.key.pub --revocations /trusted/revocations.json --evidence evidence.json --badge checks-passing.svg -- /prepared/server",
].join("\n");
export default function Page() {
  return (
    <EcosystemShell
      active="/creators"
      eyebrow="Secured by Warden"
      title="Give users evidence they can check."
      description="Include a policy with your server. Earn a checks-passing badge for a specific artifact, policy, platform and tested workflow."
    >
      <div className="grid gap-6 md:grid-cols-2">
        {[
          [
            "Policy included",
            "A policy artifact is available. It does not certify the server or imply tests passed.",
            "policy-included.svg",
          ],
          [
            "Checks passing",
            "A trusted issuer signed allowed and denied checks. The CLI checks signatures, artifact bindings, expiry and a fresh revocation snapshot.",
            "checks-passing.svg",
          ],
        ].map(([title, body, file]) => (
          <section
            key={title}
            className="rounded-2xl border border-ink-700 p-7"
          >
            <h2 className="text-2xl font-semibold">{title}</h2>
            <img
              src={`/badges/${file}`}
              alt={`Warden: ${title.toLowerCase()}`}
              width={218}
              height={28}
              className="mt-5"
            />
            <p className="mt-5 text-sm leading-relaxed text-muted">{body}</p>
          </section>
        ))}
      </div>
      <section className="mt-8 rounded-2xl border border-ink-700 p-7">
        <h2 className="text-2xl font-semibold">Start with disposable data</h2>
        <p className="mt-3 max-w-3xl text-sm leading-relaxed text-muted">
          The creator kit supplies rules, positive/negative check templates and
          a README badge. Run a prepared server inside the sandbox with fake
          credentials. Review permissions before signing; keep the issuer
          private key outside all server grants.
        </p>
        <CodeBlock className="mt-5">{commands}</CodeBlock>
        <p className="mt-4 text-xs leading-relaxed text-muted">
          Distribute issuer trust and revocation snapshots independently.
          Changed artifacts, rules, launchers or policies require fresh checks.
          Evidence expires after seven days; signatures prove issuer provenance,
          not complete safety.
        </p>
        <div className="mt-6 flex flex-wrap gap-5 text-sm text-blueprint">
          <Link href="/verify">Inspect signed evidence →</Link>
          <Link href="/docs/creator-program">Creator and CI guide →</Link>
        </div>
      </section>
    </EcosystemShell>
  );
}
