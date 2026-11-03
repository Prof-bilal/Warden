import Link from "next/link";
import PageShell from "@/components/PageShell";
import PageHeader from "@/components/bp/PageHeader";
import { ArrowUpRight } from "lucide-react";

const TOOLS: [string, string][] = [
  ["/setup", "Setup"],
  ["/integrations", "Connections"],
  ["/policies", "Policy packs"],
  ["/creators", "Creators"],
  ["/verify", "Verify evidence"],
  ["/maintenance", "Maintenance"],
  ["/compatibility", "Compatibility"],
  ["/protection", "Protection boundary"],
];

export default function EcosystemShell({
  active,
  eyebrow,
  title,
  description,
  children,
}: {
  active: string;
  eyebrow: string;
  title: string;
  description: string;
  children: React.ReactNode;
}) {
  return (
    <PageShell>
      <nav aria-label="MCP security tools" className="border-b border-hair">
        <div className="flex flex-wrap items-center gap-1 px-4 py-2.5 sm:px-6">
          {TOOLS.map(([href, label]) => (
            <Link
              key={href}
              href={href}
              aria-current={active === href ? "page" : undefined}
              className={`rounded-md px-3 py-1.5 font-mono text-[10.5px] uppercase tracking-micro transition-colors ${
                active === href
                  ? "bg-acid/10 text-acid"
                  : "text-offwhite/45 hover:text-offwhite"
              }`}
            >
              {label}
            </Link>
          ))}
          <Link
            href="/docs/ecosystem-setup"
            className="ml-auto hidden items-center gap-1.5 px-3 py-1.5 font-mono text-[10.5px] uppercase tracking-micro text-offwhite/35 transition-colors hover:text-offwhite sm:inline-flex"
          >
            CLI guide <ArrowUpRight size={12} />
          </Link>
        </div>
      </nav>

      <PageHeader eyebrow={eyebrow} title={title} description={description} />

      <div className="px-6 py-12 md:py-16">{children}</div>
    </PageShell>
  );
}
