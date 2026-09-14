import type { Metadata } from "next";
import Nav from "@/components/Nav";
import Footer from "@/components/Footer";
import { createMetadata } from "@/lib/seo";

export const metadata: Metadata = createMetadata({
  title: "Contact",
  description:
    "How to reach the Warden project: GitHub issues and discussions, the project Discord, and the npm package page.",
  path: "/contact",
});

const CHANNELS = [
  {
    name: "GitHub issues",
    description:
      "Bug reports, feature requests, and questions about the Warden sandbox runtime.",
    url: "https://github.com/Prof-bilal/Warden/issues",
    label: "github.com/Prof-bilal/Warden/issues",
  },
  {
    name: "Discord",
    description: "Community discussion and help with setup and policies.",
    url: "https://discord.gg/Mx5BhwNP4",
    label: "discord.gg/Mx5BhwNP4",
  },
  {
    name: "npm",
    description: "The warden-sandbox-cli package page.",
    url: "https://www.npmjs.com/package/warden-sandbox-cli",
    label: "npmjs.com/package/warden-sandbox-cli",
  },
];

export default function Contact() {
  return (
    <main className="min-h-screen bg-ink-950">
      <Nav />
      <div className="mx-auto max-w-[48rem] px-6 pb-20 pt-10 md:pt-16">
        <h1 className="text-[2.5rem] font-semibold leading-[1.1] tracking-[-0.02em] text-paper">
          Contact
        </h1>
        <p className="mt-4 text-[1.0625rem] leading-relaxed text-muted">
          Warden is developed in the open. The fastest way to reach the
          project is through one of these channels:
        </p>

        <div className="mt-10 border-t border-ink-800">
          {CHANNELS.map((channel) => (
            <div
              key={channel.name}
              className="border-b border-ink-800 py-6"
            >
              <h2 className="text-[1.125rem] font-medium text-paper">
                {channel.name}
              </h2>
              <p className="mt-1 text-[0.9375rem] text-muted">
                {channel.description}
              </p>
              <a
                href={channel.url}
                target="_blank"
                rel="noopener noreferrer"
                className="mt-2 inline-block font-mono text-[0.8125rem] text-blueprint transition-colors hover:text-paper"
              >
                {channel.label}
              </a>
            </div>
          ))}
        </div>
      </div>
      <Footer />
    </main>
  );
}
