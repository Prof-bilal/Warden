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
      eyebrow="MCP client setup"
      title="Your agent. Your tools. Your boundaries."
      description="Choose a client, review a server policy, and generate commands to preview, apply, or undo a local MCP wrapper. Other local processes can use warden run directly."
    >
      <SetupWizard initialClient={client} initialPack={pack} />
    </EcosystemShell>
  );
}
