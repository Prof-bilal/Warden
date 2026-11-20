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
      { slug: "index", label: "What Warden is" },
      { slug: "install", label: "Install" },
      { slug: "quickstart", label: "Quickstart" },
      { slug: "migration", label: "Migrate to Warden 2.0" },
    ],
  },
  {
    title: "Protect a tool",
    items: [
      { slug: "write-policy", label: "Write a policy" },
      { slug: "examples", label: "Example policies" },
      { slug: "agent-workspace", label: "Protect an AI agent" },
      { slug: "ecosystem-setup", label: "Connect an MCP client" },
    ],
  },
  {
    title: "Day to day",
    items: [
      { slug: "approve", label: "Review and approve" },
      { slug: "install-guard", label: "Review package installs" },
      { slug: "claude-code-hook", label: "Claude Code tool hook" },
      { slug: "mcp-tool-pins", label: "Pin MCP tool definitions" },
      { slug: "cli", label: "CLI reference" },
      { slug: "faq", label: "Troubleshooting" },
    ],
  },
  {
    title: "Reference",
    items: [
      { slug: "schema", label: "Policy schema" },
      { slug: "compatibility", label: "Server compatibility" },
      { slug: "security", label: "Security and limits" },
      { slug: "testing-platforms", label: "Platform support" },
      { slug: "architecture", label: "How it works inside" },
    ],
  },
  {
    title: "Advanced",
    collapsible: true,
    items: [
      { slug: "client-proxy", label: "Proxy reference" },
      { slug: "mcp-gateway", label: "Gateway and HTTP" },
      { slug: "container-k8s", label: "Containers and Kubernetes" },
      { slug: "proof", label: "Test evidence" },
      { slug: "contributing", label: "Contributing" },
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
                className="mb-2 flex w-full items-center justify-between px-3 font-mono text-[9.5px] font-medium uppercase tracking-micro text-offwhite/35 transition-colors hover:text-offwhite"
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
              <h4 className="mb-2 px-3 font-mono text-[9.5px] font-medium uppercase tracking-micro text-offwhite/35">
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
                        className={`block rounded-md px-3 py-1.5 font-sans text-[13px] transition-colors ${
                          isActive
                            ? "bg-acid/10 font-medium text-acid"
                            : "text-offwhite/45 hover:bg-offwhite/5 hover:text-offwhite"
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
