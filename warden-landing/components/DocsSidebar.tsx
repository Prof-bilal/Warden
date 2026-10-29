"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useState } from "react";

type SidebarItem = { slug: string; label: string };

type Section = {
  title: string;
  items: SidebarItem[];
  collapsible?: boolean;
};

const sections: Section[] = [
  {
    title: "Start here",
    items: [
      { slug: "index", label: "Introduction" },
      { slug: "install", label: "Install Warden" },
      { slug: "quickstart", label: "Quickstart" },
      { slug: "how-to-use", label: "How Warden works" },
    ],
  },
  {
    title: "Protect a tool",
    items: [
      { slug: "policy-filesystem", label: "Files" },
      { slug: "write-policy", label: "Network and environment" },
      { slug: "agent-workspace", label: "AI agent workspace" },
      { slug: "client-proxy", label: "MCP servers" },
      { slug: "examples", label: "Example policies" },
    ],
  },
  {
    title: "Use Warden",
    items: [
      { slug: "ecosystem-setup", label: "Connect a client" },
      { slug: "mcp-gateway", label: "Gateway and HTTP" },
      { slug: "approve", label: "Review and approve" },
      { slug: "how-to-use", label: "Review logs" },
      { slug: "cli", label: "CLI reference" },
      { slug: "faq", label: "Troubleshooting" },
    ],
  },
  {
    title: "Platforms",
    items: [
      { slug: "install", label: "Platform requirements" },
      { slug: "testing-platforms", label: "Platform verification" },
      { slug: "security", label: "Limitations" },
    ],
  },
  {
    title: "Reference",
    items: [
      { slug: "schema", label: "Policy schema" },
      { slug: "compatibility", label: "Compatibility" },
      { slug: "proof", label: "Test evidence" },
      { slug: "architecture", label: "Architecture" },
    ],
  },
  {
    title: "Project",
    collapsible: true,
    items: [
      { slug: "contributing", label: "Contributing" },
      { slug: "testing", label: "Testing guide" },
      { slug: "license", label: "License" },
    ],
  },
];

export default function DocsSidebar() {
  const pathname = usePathname();
  const [expanded, setExpanded] = useState<Record<string, boolean>>({
    Project: false,
  });

  const toggle = (title: string) => {
    setExpanded((previous) => ({ ...previous, [title]: !previous[title] }));
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
                className="mb-2 flex w-full items-center justify-between px-3 font-sans text-[0.75rem] font-semibold uppercase tracking-[0.08em] text-muted transition-colors hover:text-paper"
                aria-expanded={isOpen}
              >
                {section.title}
                <svg
                  className={`h-3 w-3 transition-transform duration-200 ${isOpen ? "rotate-180" : ""}`}
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                  strokeWidth={2}
                  aria-hidden="true"
                >
                  <path strokeLinecap="round" strokeLinejoin="round" d="M19 9l-7 7-7-7" />
                </svg>
              </button>
            ) : (
              <h4 className="mb-2 px-3 font-sans text-[0.75rem] font-semibold uppercase tracking-[0.08em] text-muted">
                {section.title}
              </h4>
            )}
            {(!section.collapsible || isOpen) && (
              <ul className="space-y-0.5">
                {section.items.map((item) => {
                  const href = `/docs/${item.slug}`;
                  const isActive = pathname === href;
                  return (
                    <li key={item.slug}>
                      <Link
                        href={href}
                        className={`block rounded-md px-3 py-1.5 font-sans text-[0.875rem] transition-colors ${
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
