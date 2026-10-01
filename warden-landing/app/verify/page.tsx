import EcosystemShell from "@/components/EcosystemShell";
import EvidenceInspector from "@/components/EvidenceInspector";
import { createMetadata } from "@/lib/seo";
export const metadata = createMetadata({
  title: "Inspect Warden creator evidence",
  description:
    "Check a creator evidence signature locally with an independently trusted issuer key.",
  path: "/verify",
});
export default function Page() {
  return (
    <EcosystemShell
      active="/verify"
      eyebrow="Evidence verification"
      title="Check the claim behind the badge."
      description="Start with a trusted issuer. Check the signature and expiry here, then use the CLI to bind the evidence to the server you actually run."
    >
      <EvidenceInspector />
    </EcosystemShell>
  );
}
