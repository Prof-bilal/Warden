"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useState } from "react";

type SidebarItem = { slug: string; label: string; external?: boolean };

type Section = {
  title: string;
  items: SidebarItem[];
  collapsible?: boolean;
};

const sections: Section[] = [
  {
    title: "Guide",
    items: [
      { slug: "install", label: "Installation" },
      { slug: "quickstart", label: "Quickstart" },
      { slug: "write-policy", label: "Write a Policy" },
      { slug: "schema", label: "Policy schema" },
      { slug: "examples", label: "Example policies" },
      { slug: "compatibility", label: "Compatibility" },
      { slug: "faq", label: "FAQ" },
    ],
  },
  {
    title: "Server Guides",
    collapsible: true,
    items: [
      { slug: "policy-github", label: "GitHub" },
      { slug: "policy-slack", label: "Slack" },
      { slug: "policy-filesystem", label: "Filesystem" },
      { slug: "policy-postgres", label: "PostgreSQL" },
      { slug: "policy-sqlite", label: "SQLite" },
      { slug: "policy-brave", label: "Brave Search" },
      { slug: "policy-gdrive", label: "Google Drive" },
      { slug: "policy-notion", label: "Notion" },
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
  {
    title: "More",
    collapsible: true,
    items: [
      { slug: "about", label: "About Warden" },
      { slug: "how-to-use", label: "How to use Warden" },
      { slug: "approve", label: "Interactive Approval Mode" },
      { slug: "gateway", label: "Gateway Integration" },
      { slug: "client-proxy", label: "Warden Client Proxy" },
      { slug: "container-k8s", label: "Container & Kubernetes Mode" },
      { slug: "proof", label: "Proof & Test Results" },
      { slug: "deploy", label: "Deploying the docs to GitHub Pages" },
      { slug: "prd", label: "Product Requirements Document" },
      { slug: "mvp", label: "Minimum Viable Product" },
      { slug: "beta", label: "External Beta Program (M8)" },
      { slug: "design", label: "Code Review Guide" },
      { slug: "codestyle", label: "Code Style Guide" },
      { slug: "license", label: "License" },
    ],
  },
];

export default function DocsSidebar() {
  const pathname = usePathname();
  const serverGuidesExpanded = sections.find((s) => s.collapsible);
  const [expanded, setExpanded] = useState<Record<string, boolean>>({
    "Server Guides": true,
  });

  const toggle = (title: string) => {
    setExpanded((prev) => ({ ...prev, [title]: !prev[title] }));
  };

  return (
    <nav className="docs-sidebar" aria-label="Documentation navigation">
      {sections.map((section) => {
        const isOpen = section.collapsible ? (expanded[section.title] ?? false) : true;
        return (
          <div key={section.title} className="mb-6">
            {section.collapsible ? (
              <button
                onClick={() => toggle(section.title)}
                className="mb-2 flex w-full items-center justify-between px-3 font-hero text-[0.75rem] font-bold uppercase tracking-[0.08em] text-muted hover:text-paper transition-colors"
              >
                {section.title}
                <svg
                  className={`h-3 w-3 transition-transform duration-200 ${isOpen ? "rotate-180" : ""}`}
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                  strokeWidth={2}
                >
                  <path strokeLinecap="round" strokeLinejoin="round" d="M19 9l-7 7-7-7" />
                </svg>
              </button>
            ) : (
              <h4 className="mb-2 px-3 font-hero text-[0.75rem] font-bold uppercase tracking-[0.08em] text-muted">
                {section.title}
              </h4>
            )}
            {(!section.collapsible || isOpen) && (
              <ul className="space-y-0.5">
                {section.items.map((item) => {
                  const href = item.external ? "/features" : `/docs/${item.slug}`;
                  const isActive = pathname === href;
                  return (
                    <li key={item.slug}>
                      <Link
                        href={href}
                        className={`block rounded-md px-3 py-1.5 font-hero text-[0.875rem] transition-colors ${
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
            )}
          </div>
        );
      })}
    </nav>
  );
}
