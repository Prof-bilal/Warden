import Link from "next/link";
import Nav from "@/components/Nav";
import Footer from "@/components/Footer";
import { ArrowUpRight, ShieldCheck } from "lucide-react";

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
    <main className="min-h-screen bg-ink-950">
      <Nav />
      <div className="border-b border-ink-700 bg-ink-900/60">
        <nav
          aria-label="MCP security tools"
          className="mx-auto flex max-w-content flex-wrap gap-2 px-6 py-4"
        >
          {[
            ["/setup", "Setup"],
            ["/integrations", "Connections"],
            ["/policies", "Policy packs"],
            ["/creators", "Creators"],
            ["/verify", "Verify evidence"],
            ["/maintenance", "Maintenance"],
            ["/compatibility", "Compatibility"],
            ["/protection", "Protection boundary"],
          ].map(([href, label]) => (
            <Link
              key={href}
              href={href}
              aria-current={active === href ? "page" : undefined}
              className={`rounded-lg px-4 py-2 text-sm transition-colors ${active === href ? "bg-blueprint/15 text-blueprint" : "text-muted hover:bg-ink-800 hover:text-paper"}`}
            >
              {label}
            </Link>
          ))}
          <Link
            href="/docs/ecosystem-setup"
            className="ml-auto flex items-center gap-2 px-4 py-2 text-sm text-muted hover:text-paper"
          >
            CLI guide <ArrowUpRight size={14} />
          </Link>
        </nav>
      </div>
      <div className="mx-auto max-w-content px-6 pb-20 pt-14 sm:pt-20">
        <p className="mb-5 flex items-center gap-2 text-xs font-semibold uppercase tracking-widest text-blueprint">
          <ShieldCheck size={15} />
          {eyebrow}
        </p>
        <h1 className="max-w-4xl font-display text-4xl leading-tight tracking-tight sm:text-6xl">
          {title}
        </h1>
        <p className="mt-5 max-w-2xl text-base leading-relaxed text-muted sm:text-lg">
          {description}
        </p>
        <div className="mt-12">{children}</div>
      </div>
      <Footer />
    </main>
  );
}
