"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

type SidebarItem = { slug: string; label: string; external?: boolean };

const sections: { title: string; items: SidebarItem[] }[] = [
  {
    title: "Guide",
    items: [
      { slug: "install", label: "Installation" },
      { slug: "quickstart", label: "Quickstart" },
      { slug: "schema", label: "Policy schema" },
      { slug: "examples", label: "Example policies" },
      { slug: "compatibility", label: "Compatibility" },
      { slug: "faq", label: "FAQ" },
    ],
  },
  {
    title: "Deep Dives",
    items: [
      { slug: "architecture", label: "Architecture" },
      { slug: "security", label: "Security review" },
      { slug: "cli", label: "CLI reference" },
      { slug: "roadmap", label: "Roadmap" },
    ],
  },
  {
    title: "Reference",
    items: [
      { slug: "features", label: "Features (in depth)", external: true },
    ],
  },


  {
    title: "Contributing",
    items: [
      { slug: "contributing", label: "Contributing" },
      { slug: "testing", label: "Testing guide" },
      { slug: "testing-platforms", label: "Cross-platform testing" },
    ],
  },
];

export default function DocsSidebar() {
  const pathname = usePathname();

  return (
    <nav className="docs-sidebar" aria-label="Documentation navigation">
      {sections.map((section) => (
        <div key={section.title} className="mb-6">
          <h4 className="mb-2 px-3 text-[0.75rem] font-semibold uppercase tracking-[0.08em] text-muted">
            {section.title}
          </h4>
          <ul className="space-y-0.5">
            {section.items.map((item) => {
              const href = item.external ? "/features" : `/docs/${item.slug}`;
              const isActive = pathname === href;
              return (
                <li key={item.slug}>
                  <Link
                    href={href}
                    className={`block rounded-md px-3 py-1.5 text-[0.875rem] transition-colors ${
                      isActive
                        ? "bg-blueprint/10 font-medium text-blueprint"
                        : "text-muted hover:bg-ink-800 hover:text-paper"
                    }`}
                  >
                    {item.label}
                  </Link>
                </li>
              );
            })}
          </ul>
        </div>
      ))}
    </nav>
  );
}
