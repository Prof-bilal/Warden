import EcosystemShell from "@/components/EcosystemShell";
import PolicyCatalog from "@/components/PolicyCatalog";
import { createMetadata } from "@/lib/seo";
export const metadata = createMetadata({
  title: "MCP policy packs",
  description:
    "Choose a narrow candidate security profile for your MCP server. Review files, API hosts, credentials and evidence before setup.",
  path: "/policies",
});
export default function Page() {
  return (
    <EcosystemShell
      active="/policies"
      eyebrow="Policy library"
      title="Start with the task. Keep the grants small."
      description="Six starting profiles for everyday MCP workflows. Choose your server, review its access, then build a policy around the directories you select."
    >
      <PolicyCatalog />
    </EcosystemShell>
  );
}
