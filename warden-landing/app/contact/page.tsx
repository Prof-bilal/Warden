import type { Metadata } from "next";
import Nav from "@/components/Nav";
import Footer from "@/components/Footer";
import JsonLd from "@/components/JsonLd";
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
  title: "Contact",
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
    <main className="min-h-screen bg-ink-950">
      <JsonLd
        data={breadcrumbSchema([
          { name: "Home", url: `${SITE_URL}/` },
          { name: "Contact", url: `${SITE_URL}/contact` },
        ])}
      />
      <Nav />

      {/* header */}
      <section className="border-t border-ink-800">
        <div className="mx-auto max-w-content px-6 pb-14 pt-16 text-center md:pt-24">
          <p className="mb-4 font-hero text-[12px] font-bold uppercase tracking-[0.12em] text-grant">
            Contact
          </p>
          <h1 className="mx-auto max-w-2xl font-hero text-[2.25rem] font-bold leading-[1.1] tracking-[-0.02em] text-paper sm:text-[2.75rem]">
            Talk to us in the open.
          </h1>
          <p className="mx-auto mt-4 max-w-[34rem] font-hero text-[15px] leading-[1.7] text-muted md:text-[16px]">
            Warden is developed in the open. The fastest way to reach the project is through one
            of these channels:
          </p>
        </div>
      </section>

      {/* channels */}
      <section className="mx-auto max-w-content px-6 pb-10">
        <div className="mx-auto grid max-w-4xl gap-4 md:grid-cols-3">
          {CHANNELS.map((channel) => (
            <a
              key={channel.name}
              href={channel.url}
              target="_blank"
              rel="noopener noreferrer"
              className="group flex flex-col rounded-2xl border border-ink-700 bg-ink-900 p-6 transition-colors hover:border-grant/50"
            >
              <p className="font-hero text-[11px] font-bold tracking-[0.14em] text-grant">
                {channel.n}
              </p>
              <h2 className="mt-2 font-hero text-[17px] font-bold text-paper">{channel.name}</h2>
              <p className="mt-1.5 flex-1 font-hero text-[12.5px] leading-[1.65] text-muted">
                {channel.description}
              </p>
              <p className="mt-4 truncate font-hero text-[12px] text-blueprint">{channel.label}</p>
              <span className="mt-3 inline-flex items-center gap-1.5 font-hero text-[13px] font-semibold text-paper transition-colors group-hover:text-grant">
                {channel.cta} <span aria-hidden>→</span>
              </span>
            </a>
          ))}
        </div>
      </section>

      {/* reach abdullah directly */}
      <section className="mx-auto max-w-content px-6 pb-24">
        <div className="mx-auto flex max-w-4xl flex-col items-center gap-4 rounded-2xl border border-ink-700 bg-ink-900 px-6 py-8 text-center">
          <p className="font-hero text-[11px] font-bold uppercase tracking-[0.14em] text-muted">
            Reach Abdullah directly
          </p>
          <div className="flex items-center gap-3">
            <a
              href="https://www.linkedin.com/in/abdullah-bilal-a7618134b/"
              target="_blank"
              rel="noopener noreferrer"
              aria-label="Abdullah Bilal on LinkedIn"
              className="flex h-11 w-11 items-center justify-center rounded-xl border border-ink-600 bg-ink-950 text-muted transition-colors hover:border-grant/60 hover:text-paper"
            >
              <Linkedin size={18} />
            </a>
            <a
              href="https://x.com/Abdullahbilal56"
              target="_blank"
              rel="noopener noreferrer"
              aria-label="Abdullah Bilal on X"
              className="flex h-11 w-11 items-center justify-center rounded-xl border border-ink-600 bg-ink-950 text-muted transition-colors hover:border-grant/60 hover:text-paper"
            >
              <XIcon size={16} />
            </a>
          </div>
          <p className="font-hero text-[12px] text-muted">
            Founder & Maintainerfastest replies on X
          </p>
        </div>
      </section>

      <Footer />
    </main>
  );
}
