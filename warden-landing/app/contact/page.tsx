import type { Metadata } from "next";
import PageShell from "@/components/PageShell";
import JsonLd from "@/components/JsonLd";
import PageHeader from "@/components/bp/PageHeader";
import { createMetadata, SITE_URL } from "@/lib/seo";
import { breadcrumbSchema } from "@/lib/schema";
import { Linkedin } from "lucide-react";

function XIcon({ size = 16 }: { size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
      <path d="M18.244 2.25h3.308l-7.227 8.26 8.502 11.24H16.17l-5.214-6.817L4.99 21.75H1.68l7.73-8.835L1.254 2.25H8.08l4.713 6.231 5.451-6.231Zm-1.161 17.52h1.833L7.084 4.126H5.117l11.966 15.644Z" />
    </svg>
  );
}

export const metadata: Metadata = createMetadata({
  title: { absolute: "Talk to us in the open." },
  description:
    "How to reach the Warden project: GitHub issues and discussions, the project Discord, and the npm package page.",
  path: "/contact",
});

const CHANNELS = [
  {
    n: "01",
    name: "GitHub issues",
    description: "Bug reports, feature requests, and questions about the Warden sandbox runtime.",
    url: "https://github.com/Prof-bilal/Warden/issues",
    label: "github.com/Prof-bilal/Warden/issues",
    cta: "Open an issue",
  },
  {
    n: "02",
    name: "Discord",
    description: "Community discussion and help with setup and policies.",
    url: "https://discord.gg/Mx5BhwNP4",
    label: "discord.gg/Mx5BhwNP4",
    cta: "Join the server",
  },
  {
    n: "03",
    name: "npm",
    description: "The warden-sandbox-cli package page.",
    url: "https://www.npmjs.com/package/warden-sandbox-cli",
    label: "npmjs.com/package/warden-sandbox-cli",
    cta: "View the package",
  },
];

export default function Contact() {
  return (
    <PageShell>
      <JsonLd
        data={breadcrumbSchema([
          { name: "Home", url: `${SITE_URL}/` },
          { name: "Contact", url: `${SITE_URL}/contact` },
        ])}
      />

      <PageHeader
        eyebrow="Contact"
        title="Talk to us in the open."
        description="Warden is developed in the open. The fastest way to reach the project is through one of these channels."
      />

      {/* channels */}
      <section className="border-b border-hair">
        <div className="px-6 py-12 md:py-14">
          <div className="grid gap-px border border-hair bg-hair md:grid-cols-3">
            {CHANNELS.map((channel) => (
              <a
                key={channel.name}
                href={channel.url}
                target="_blank"
                rel="noopener noreferrer"
                className="group flex flex-col bg-cell p-6 transition-colors hover:bg-offwhite/[0.03]"
              >
                <p className="font-mono text-[10px] text-acid">{channel.n}</p>
                <h2 className="mt-3 font-mono text-[15px] font-semibold text-offwhite">
                  {channel.name}
                </h2>
                <p className="mt-1.5 flex-1 font-sans text-[12.5px] leading-[1.7] text-offwhite/45">
                  {channel.description}
                </p>
                <p className="mt-4 truncate font-mono text-[11px] text-offwhite/35">
                  {channel.label}
                </p>
                <span className="mt-3 inline-flex items-center gap-1.5 font-mono text-[11.5px] font-semibold text-offwhite/80 transition-colors group-hover:text-acid">
                  {channel.cta} <span aria-hidden>→</span>
                </span>
              </a>
            ))}
          </div>
        </div>
      </section>

      {/* reach abdullah directly */}
      <section>
        <div className="px-6 py-14 md:py-16">
          <div className="mx-auto flex max-w-3xl flex-col items-center gap-4 border border-hair bg-cell px-6 py-8 text-center">
            <p className="font-mono text-[9.5px] uppercase tracking-micro text-offwhite/35">
              Reach Abdullah directly
            </p>
            <div className="flex items-center gap-3">
              <a
                href="https://www.linkedin.com/in/abdullah-bilal-a7618134b/"
                target="_blank"
                rel="noopener noreferrer"
                aria-label="Abdullah Bilal on LinkedIn"
                className="flex h-11 w-11 items-center justify-center rounded-md border border-hair bg-void text-offwhite/45 transition-colors hover:border-acid/50 hover:text-acid"
              >
                <Linkedin size={18} />
              </a>
              <a
                href="https://x.com/Abdullahbilal56"
                target="_blank"
                rel="noopener noreferrer"
                aria-label="Abdullah Bilal on X"
                className="flex h-11 w-11 items-center justify-center rounded-md border border-hair bg-void text-offwhite/45 transition-colors hover:border-acid/50 hover:text-acid"
              >
                <XIcon size={16} />
              </a>
            </div>
            <p className="font-mono text-[11px] text-offwhite/35">
              Founder & Maintainer, fastest replies on X
            </p>
          </div>
        </div>
      </section>
    </PageShell>
  );
}
