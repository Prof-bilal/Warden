import EcosystemShell from "@/components/EcosystemShell";
import SetupWizard from "@/components/SetupWizard";
import { createMetadata } from "@/lib/seo";
export const metadata = createMetadata({
  title: "Set up Warden for your AI client",
  description:
    "Choose your MCP host, review a policy profile, and generate reversible local setup commands.",
  path: "/setup",
});
export default async function Page({
  searchParams,
}: {
  searchParams: Promise<{ pack?: string; client?: string }>;
}) {
  const { pack, client } = await searchParams;
  return (
    <EcosystemShell
      active="/setup"
      eyebrow="Local MCP setup"
      title="Your agent. Your tools. Your boundaries."
      description="Choose a client and a server profile. Get the local commands to preview, apply, and undo a Warden wrapper."
    >
      <SetupWizard initialClient={client} initialPack={pack} />
    </EcosystemShell>
  );
}
