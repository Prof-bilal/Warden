import type { Metadata } from "next";
import { SITE_URL } from "@/lib/seo";
import JsonLd from "@/components/JsonLd";
import { breadcrumbSchema } from "@/lib/schema";
import Link from "next/link";
import PageShell from "@/components/PageShell";
import PageHeader from "@/components/bp/PageHeader";
import SectionMarker from "@/components/bp/SectionMarker";
import StatRow from "@/components/bp/StatRow";
import Reveal from "@/components/bp/Reveal";
import { Github } from "lucide-react";

export const metadata: Metadata = {
  title: { absolute: "Sandbox every server by default." },
  description:
    "Warden is an open-source sandbox runtime for MCP servers. It makes the safe path the easy path, with a policy file small enough to read in one sitting.",
  alternates: {
    canonical: `${SITE_URL}/about`,
  },
  openGraph: {
    title: "Sandbox every server by default.",
    description:
      "Warden is an open-source sandbox runtime for MCP servers. It makes the safe path the easy path, with a policy file small enough to read in one sitting.",
    url: `${SITE_URL}/about`,
    images: [
      {
        url: `${SITE_URL}/og-image.png`,
        width: 1200,
        height: 630,
        alt: "Sandbox every server by default.",
      },
    ],
  },
  twitter: {
    card: "summary_large_image",
    title: "Sandbox every server by default.",
    description:
      "Warden is an open-source sandbox runtime for MCP servers. It makes the safe path the easy path, with a policy file small enough to read in one sitting.",
    images: [`${SITE_URL}/og-image.png`],
  },
};

const REPO = "https://github.com/Prof-bilal/Warden";

const STATS: [string, string][] = [
  ["100%", "open source · MIT"],
  ["3", "OS-native backends"],
  ["18", "servers tested"],
  ["7/7", "escape tests contained"],
];

const PRINCIPLES = [
  {
    title: "Deny by default",
    body: "Unlisted paths, hosts, and env vars are invisible, not merely blocked.",
  },
  {
    title: "Fail closed",
    body: "No valid backend means no run. Warden never falls back silently.",
  },
  {
    title: "Everything logged",
    body: "Every allow and deny lands in a JSONL audit log you can grep.",
  },
  {
    title: "OS-native",
    body: "Bubblewrap, Seatbelt, AppContainer, one static binary, near-zero cost.",
  },
];

const TEAM = [
  {
    initials: "AB",
    name: "Abdullah Bilal",
    role: "Founder & Maintainer",
    github: "https://github.com/Prof-bilal",
  },
  {
    initials: "HM",
    name: "Hasnain Maroof",
    role: "Contributor",
    github: "https://github.com/HusnainMaroof",
  },
];

const ROWS = [
  { area: "Linux sandbox (bubblewrap)", state: "Filesystem, network proxy, audit, limits" },
  { area: "macOS sandbox (Seatbelt)", state: "With Docker fallback" },
  { area: "Windows sandbox", state: "AppContainer + WFP + ETW audit, fail-closed" },
  { area: "trace / init / logs", state: "Observe, generate, inspect" },
  { area: "Approval mode", state: "Prompt instead of hard-fail" },
  { area: "Gateway integration", state: "Wrap gateway-registered servers" },
  { area: "Compatibility matrix", state: "18 servers, 14 pass" },
];

export default function About() {
  return (
    <PageShell>
      <JsonLd
        data={breadcrumbSchema([
          { name: "Home", url: `${SITE_URL}/` },
          { name: "About", url: `${SITE_URL}/about` },
        ])}
      />

      <PageHeader
        eyebrow="About"
        title="Sandbox every server by default."
        description="Warden is an open-source sandbox runtime for MCP servers, maintained by Abdullah Bilal. Modern AI tooling runs third-party code with first-party trust. MCP servers install with a one-liner and inherit everything you can do. Warden exists to make the safe path the easy path, with a policy file small enough to read in one sitting."
      />

      {/* stats */}
      <section className="border-b border-hair">
        <div className="px-6 py-12 md:py-14">
          <Reveal>
            <StatRow stats={STATS} className="border border-hair" />
          </Reveal>
        </div>
      </section>

      {/* problem + docker */}
      <section className="border-b border-hair">
        <div className="grid lg:grid-cols-2">
          <div className="border-b border-hair px-6 py-10 lg:border-b-0 lg:border-r md:py-12">
            <p className="font-mono text-[9.5px] uppercase tracking-micro text-deny">
              The problem
            </p>
            <h2 className="mt-3 font-mono text-[1.35rem] font-semibold leading-[1.2] text-offwhite">
              Third-party code, first-party trust.
            </h2>
            <p className="mt-3 max-w-xl font-sans text-[14px] leading-[1.75] text-offwhite/45">
              Every MCP server you install runs with your full permissions, your
              SSH keys, your cloud credentials, your entire filesystem. One
              malicious or compromised package is all it takes.
            </p>
          </div>
          <div className="px-6 py-10 md:py-12">
            <p className="font-mono text-[9.5px] uppercase tracking-micro text-acid">
              Why not just Docker?
            </p>
            <h2 className="mt-3 font-mono text-[1.35rem] font-semibold leading-[1.2] text-offwhite">
              Too heavy for one script.
            </h2>
            <p className="mt-3 max-w-xl font-sans text-[14px] leading-[1.75] text-offwhite/45">
              Docker works, Warden uses it as a fallback. But slow cold starts, a
              daemon dependency, and a far bigger trust boundary than a namespace
              sandbox needs. Warden is a single static binary over OS-native
              primitives, so sandboxing costs almost nothing.
            </p>
          </div>
        </div>
      </section>

      {/* principles */}
      <section className="border-b border-hair">
        <div className="px-6 py-14 md:py-16">
          <SectionMarker index="02" label="Principles" />
          <h2 className="mt-4 font-mono text-[1.55rem] font-semibold leading-[1.18] tracking-[-0.01em] text-offwhite md:text-[1.95rem]">
            Four rules, no exceptions.
          </h2>
          <div className="mt-8 grid gap-px border border-hair bg-hair sm:grid-cols-2 lg:grid-cols-4">
            {PRINCIPLES.map((p, i) => (
              <div key={p.title} className="bg-cell p-5">
                <p className="font-mono text-[10px] text-acid">
                  {String(i + 1).padStart(2, "0")}
                </p>
                <h3 className="mt-3 font-mono text-[14px] font-semibold text-offwhite">
                  {p.title}
                </h3>
                <p className="mt-1.5 font-sans text-[12.5px] leading-[1.7] text-offwhite/45">
                  {p.body}
                </p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* team */}
      <section className="border-b border-hair">
        <div className="px-6 py-14 md:py-16">
          <SectionMarker index="03" label="Team" />
          <h2 className="mt-4 font-mono text-[1.55rem] font-semibold leading-[1.18] tracking-[-0.01em] text-offwhite md:text-[1.95rem]">
            Built by people who run agents.
          </h2>
          <div className="mx-auto mt-8 grid max-w-3xl gap-px border border-hair bg-hair sm:grid-cols-2">
            {TEAM.map((m) => (
              <a
                key={m.name}
                href={m.github}
                target="_blank"
                rel="noopener noreferrer"
                className="group flex items-center gap-4 bg-cell p-5 transition-colors hover:bg-offwhite/[0.03]"
              >
                <span className="flex h-14 w-14 shrink-0 items-center justify-center rounded-full border border-acid/40 bg-acid/10 font-mono text-[15px] font-semibold text-acid">
                  {m.initials}
                </span>
                <span className="min-w-0">
                  <span className="block truncate font-mono text-[15px] font-semibold text-offwhite">
                    {m.name}
                  </span>
                  <span className="mt-0.5 block font-mono text-[10.5px] uppercase tracking-micro text-offwhite/35">
                    {m.role}
                  </span>
                  <span className="mt-2 flex items-center gap-1.5 font-mono text-[11px] text-offwhite/40 transition-colors group-hover:text-acid">
                    <Github size={12} />
                    GitHub →
                  </span>
                </span>
              </a>
            ))}
          </div>
        </div>
      </section>

      {/* status */}
      <section className="border-b border-hair">
        <div className="px-6 py-14 md:py-16">
          <div className="mx-auto max-w-3xl">
            <div className="flex items-center gap-3">
              <h2 className="font-mono text-[1.35rem] font-semibold text-offwhite">
                Status: beta
              </h2>
              <span className="border border-progress/30 bg-progress-subtle px-2.5 py-0.5 font-mono text-[10px] uppercase tracking-micro text-progress">
                v0.1.17
              </span>
            </div>
            <ul className="mt-4 border border-hair">
              {ROWS.map((r) => (
                <li
                  key={r.area}
                  className="flex items-baseline justify-between gap-6 border-b border-hair bg-cell px-5 py-3.5 last:border-b-0"
                >
                  <span className="font-mono text-[12.5px] text-offwhite/80">
                    {r.area}
                  </span>
                  <span className="shrink-0 text-right font-mono text-[11px] text-offwhite/40">
                    {r.state}
                  </span>
                </li>
              ))}
            </ul>
          </div>
        </div>
      </section>

      {/* security + links */}
      <section>
        <div className="px-6 py-14 md:py-16">
          <div className="mx-auto max-w-3xl border border-hair bg-cell p-6 text-center md:p-10">
            <h2 className="font-mono text-[1.35rem] font-semibold text-offwhite">
              Security posture
            </h2>
            <p className="mx-auto mt-3 max-w-xl font-sans text-[13.5px] leading-[1.75] text-offwhite/45">
              Deny-by-default on filesystem, network, and environment. No silent
              fallback to unsandboxed runs, a missing backend fails loudly. Every
              blocked access is logged. Known limitations (no CPU throttling, no
              unix-socket grants) are documented, not buried.
            </p>
            <div className="mt-6 flex flex-wrap justify-center gap-3">
              <a
                href={REPO}
                className="rounded-md bg-acid px-5 py-2.5 font-sans text-[13px] font-semibold text-void transition-colors hover:bg-acid-bright"
              >
                Repository →
              </a>
              <Link
                href="/docs/security"
                className="rounded-md border border-hair bg-void px-5 py-2.5 font-sans text-[13px] font-semibold text-offwhite/85 transition-colors hover:border-offwhite/20"
              >
                Security review →
              </Link>
              <Link
                href="/docs"
                className="rounded-md border border-hair bg-void px-5 py-2.5 font-sans text-[13px] font-semibold text-offwhite/85 transition-colors hover:border-offwhite/20"
              >
                Docs →
              </Link>
            </div>
            <p className="mt-6 font-mono text-[10px] uppercase tracking-micro text-offwhite/30">
              warden · MIT licensed
            </p>
          </div>
        </div>
      </section>
    </PageShell>
  );
}
