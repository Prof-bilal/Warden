import { notFound } from "next/navigation";
import EcosystemShell from "@/components/EcosystemShell";
import SetupWizard from "@/components/SetupWizard";
import { clients } from "@/lib/ecosystem";
import { createMetadata } from "@/lib/seo";
export function generateStaticParams() {
  return clients.map((c) => ({ client: c.id }));
}
export async function generateMetadata({
  params,
}: {
  params: Promise<{ client: string }>;
}) {
  const { client } = await params;
  const c = clients.find((c) => c.id === client);
  return createMetadata({
    title: `Warden setup for ${c?.name ?? "MCP"}`,
    description: "Review and generate reversible MCP security setup commands.",
    path: `/setup/${client}`,
  });
}
export default async function Page({
  params,
}: {
  params: Promise<{ client: string }>;
}) {
  const { client } = await params;
  const c = clients.find((c) => c.id === client);
  if (!c) notFound();
  return (
    <EcosystemShell
      active="/setup"
      eyebrow="Client setup"
      title={`Set up Warden with ${c.name}`}
      description="Wrap an existing local MCP server with a reviewed policy and a reversible configuration change."
    >
      <SetupWizard initialClient={client} />
    </EcosystemShell>
  );
}
