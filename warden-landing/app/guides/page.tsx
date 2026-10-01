import type { Metadata } from "next";
import Link from "next/link";
import Nav from "@/components/Nav";
import Footer from "@/components/Footer";
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
    <main className="min-h-screen bg-ink-950">
      <JsonLd
        data={breadcrumbSchema([
          { name: "Home", url: `${SITE_URL}/` },
          { name: "Guides", url: `${SITE_URL}/guides` },
        ])}
      />
      <Nav />
      <div className="mx-auto max-w-[48rem] px-6 pb-20 pt-10 md:pt-16">
        <h1 className="font-hero text-[2.5rem] font-bold leading-[1.1] tracking-[-0.02em] text-paper">
          How to securely run an MCP server
        </h1>
        <p className="mt-5 font-hero text-[1.0625rem] leading-[1.7] text-muted">
          An MCP server is a local process with your permissions unless you put
          a sandbox in front of it. These pages answer that problem directly.
          Warden is one open-source way to enforce the resulting policy. It is
          not assumed in the question.
        </p>

        {ORDER.map((group) => {
          const items = topics.filter((topic) => topic.section === group.section);
          if (items.length === 0) return null;
          return (
            <section key={group.section} className="mt-12">
              <h2 className="font-hero text-[1.25rem] font-bold text-paper">{group.heading}</h2>
              <p className="mt-2 font-hero text-[0.9375rem] leading-[1.6] text-muted">{group.lede}</p>
              <ul className="mt-4 space-y-4">
                {items.map((topic) => (
                  <li key={topic.path}>
                    <Link href={topic.path} className="font-hero text-[1.0625rem] font-medium text-paper hover:text-blueprint">
                      {topic.title}
                    </Link>
                    <p className="mt-1 font-hero text-[0.875rem] leading-[1.6] text-muted">
                      {topic.description}
                    </p>
                  </li>
                ))}
              </ul>
            </section>
          );
        })}
      </div>
      <Footer />
    </main>
  );
}
