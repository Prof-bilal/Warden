import type { Metadata } from "next";
import Link from "next/link";
import PageShell from "@/components/PageShell";
import PageHeader from "@/components/bp/PageHeader";
import SectionMarker from "@/components/bp/SectionMarker";
import JsonLd from "@/components/JsonLd";
import { createMetadata, SITE_URL } from "@/lib/seo";
import { breadcrumbSchema } from "@/lib/schema";
import { getAllTopics, type TopicSection } from "@/lib/topics";

export const metadata: Metadata = createMetadata({
  title: "How to securely run an MCP server",
  description:
    "Guides on MCP server security, sandboxing, permissions, and API keys, plus comparisons of isolation options and practical use cases.",
  path: "/guides",
});

const ORDER: { section: TopicSection; heading: string; lede: string }[] = [
  {
    section: "guides",
    heading: "Guides",
    lede: "Start here if you have an MCP server and want to know what it can touch.",
  },
  {
    section: "compare",
    heading: "Comparisons",
    lede: "Isolation options side by side, including Docker and hand-written OS sandboxes.",
  },
  {
    section: "use-cases",
    heading: "Use cases",
    lede: "Local servers, untrusted packages, and filesystem isolation.",
  },
];

export default function GuidesIndex() {
  const topics = getAllTopics();

  return (
    <PageShell>
      <JsonLd
        data={breadcrumbSchema([
          { name: "Home", url: `${SITE_URL}/` },
          { name: "Guides", url: `${SITE_URL}/guides` },
        ])}
      />

      <PageHeader
        eyebrow="Guides"
        title="How to securely run an MCP server"
        description="An MCP server is a local process with your permissions unless you put a sandbox in front of it. These pages answer that problem directly. Warden is one open-source way to enforce the resulting policy. It is not assumed in the question."
      />

      <div className="px-6 py-12 md:py-16">
        <div className="mx-auto max-w-[48rem]">
          {ORDER.map((group, gi) => {
            const items = topics.filter((topic) => topic.section === group.section);
            if (items.length === 0) return null;
            return (
              <section key={group.section} className="mt-14 first:mt-0">
                <SectionMarker
                  index={String(gi + 1).padStart(2, "0")}
                  label={group.heading}
                />
                <h2 className="mt-3 font-mono text-[1.35rem] font-semibold text-offwhite">
                  {group.heading}
                </h2>
                <p className="mt-2 font-sans text-[13.5px] leading-[1.7] text-offwhite/45">
                  {group.lede}
                </p>
                <ul className="mt-5 divide-y divide-hair border-y border-hair">
                  {items.map((topic) => (
                    <li key={topic.path} className="py-4">
                      <Link
                        href={topic.path}
                        className="font-mono text-[15px] font-medium text-offwhite transition-colors hover:text-acid"
                      >
                        {topic.title}
                      </Link>
                      <p className="mt-1.5 font-sans text-[13px] leading-[1.7] text-offwhite/45">
                        {topic.description}
                      </p>
                    </li>
                  ))}
                </ul>
              </section>
            );
          })}
        </div>
      </div>
    </PageShell>
  );
}
