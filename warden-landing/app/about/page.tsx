import type { Metadata } from "next";
import { SITE_URL } from "@/lib/seo";
import JsonLd from "@/components/JsonLd";
import { breadcrumbSchema } from "@/lib/schema";
import Nav from "@/components/Nav";
import Footer from "@/components/Footer";
import { Github } from "lucide-react";

export const metadata: Metadata = {
  title: "About Warden",
  description:
    "About Wardenan open-source sandbox runtime for MCP servers. Learn about the mission, security posture, and current beta status.",
  alternates: {
    canonical: `${SITE_URL}/about`,
  },
  openGraph: {
    title: "About Warden",
    description:
      "Learn about Warden's mission to make the safe path the easy path for MCP server sandboxing.",
    url: `${SITE_URL}/about`,
    images: [
      {
        url: `${SITE_URL}/og-image.png`,
        width: 1200,
        height: 630,
        alt: "About Warden",
      },
    ],
  },
  twitter: {
    card: "summary_large_image",
    title: "About Warden",
    description:
      "Learn about Warden's mission to make the safe path the easy path for MCP server sandboxing.",
    images: [`${SITE_URL}/og-image.png`],
  },
};

const REPO = "https://github.com/Prof-bilal/Warden";

const STATS = [
  { value: "100%", label: "open source · MIT" },
  { value: "3", label: "OS-native backends" },
  { value: "18", label: "servers tested" },
  { value: "7/7", label: "escape tests contained" },
];

const PRINCIPLES = [
  {
    title: "Deny by default",
    body: "Unlisted paths, hosts, and env vars are invisiblenot merely blocked.",
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
    body: "Bubblewrap, Seatbelt, AppContainerone static binary, near-zero cost.",
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
    <main className="min-h-screen bg-ink-950">
      <JsonLd
        data={breadcrumbSchema([
          { name: "Home", url: `${SITE_URL}/` },
          { name: "About", url: `${SITE_URL}/about` },
        ])}
      />
      <Nav />

      {/* hero */}
      <section className="border-t border-ink-800">
        <div className="mx-auto max-w-content px-6 pb-14 pt-16 text-center md:pt-24">
          <p className="mb-4 font-hero text-[12px] font-bold uppercase tracking-[0.12em] text-grant">
            About
          </p>
          <h1 className="mx-auto max-w-3xl font-hero text-[2.25rem] font-bold leading-[1.1] tracking-[-0.02em] text-paper sm:text-[2.75rem]">
            Sandbox every server by default.
          </h1>
          <p className="mx-auto mt-4 max-w-[38rem] font-hero text-[15px] leading-[1.7] text-muted md:text-[16px]">
            Modern AI tooling runs third-party code with first-party trust. MCP servers install
            with a one-liner and inherit everything you can do. Warden exists to make the safe
            path the easy pathwith a policy file small enough to read in one sitting.
          </p>
        </div>
      </section>

      {/* stats */}
      <section className="mx-auto max-w-content px-6">
        <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
          {STATS.map((s) => (
            <div
              key={s.label}
              className="rounded-2xl border border-ink-700 bg-ink-900 px-4 py-5 text-center"
            >
              <p className="font-hero text-2xl font-bold text-grant md:text-3xl">{s.value}</p>
              <p className="mt-1 font-hero text-[11px] uppercase tracking-[0.1em] text-muted">
                {s.label}
              </p>
            </div>
          ))}
        </div>
      </section>

      {/* problem + docker */}
      <section className="mx-auto max-w-content px-6 pt-16">
        <div className="grid gap-4 md:grid-cols-2">
          <div className="rounded-2xl border border-ink-700 bg-ink-900 p-6 md:p-8">
            <p className="font-hero text-[11px] font-bold uppercase tracking-[0.14em] text-deny">
              The problem
            </p>
            <h2 className="mt-2 font-hero text-[1.375rem] font-bold leading-[1.2] text-paper">
              Third-party code, first-party trust.
            </h2>
            <p className="mt-3 font-hero text-[14px] leading-[1.7] text-muted">
              Every MCP server you install runs with your full permissionsyour SSH keys, your
              cloud credentials, your entire filesystem. One malicious or compromised package is
              all it takes.
            </p>
          </div>
          <div className="rounded-2xl border border-grant/30 bg-grant-subtle p-6 md:p-8">
            <p className="font-hero text-[11px] font-bold uppercase tracking-[0.14em] text-grant">
              Why not just Docker?
            </p>
            <h2 className="mt-2 font-hero text-[1.375rem] font-bold leading-[1.2] text-paper">
              Too heavy for one script.
            </h2>
            <p className="mt-3 font-hero text-[14px] leading-[1.7] text-muted">
              Docker worksWarden uses it as a fallback. But slow cold starts, a daemon
              dependency, and a far bigger trust boundary than a namespace sandbox needs. Warden
              is a single static binary over OS-native primitives, so sandboxing costs almost
              nothing.
            </p>
          </div>
        </div>
      </section>

      {/* principles */}
      <section className="mx-auto max-w-content px-6 pt-16">
        <p className="text-center font-hero text-[12px] font-bold uppercase tracking-[0.12em] text-grant">
          Principles
        </p>
        <h2 className="mx-auto mt-2 max-w-xl text-center font-hero text-[1.75rem] font-bold leading-[1.15] tracking-[-0.02em] text-paper">
          Four rules, no exceptions.
        </h2>
        <div className="mt-8 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          {PRINCIPLES.map((p, i) => {
            return (
              <div
                key={p.title}
                className="rounded-2xl border border-ink-700 bg-ink-900 p-5 transition-colors hover:border-ink-500"
              >
                <p className="font-hero text-[11px] font-bold tracking-[0.14em] text-grant">
                  {String(i + 1).padStart(2, "0")}
                </p>
                <h3 className="mt-2 font-hero text-[15px] font-bold text-paper">{p.title}</h3>
                <p className="mt-1.5 font-hero text-[12.5px] leading-[1.65] text-muted">{p.body}</p>
              </div>
            );
          })}
        </div>
      </section>

      {/* team */}
      <section className="mx-auto max-w-content px-6 pt-20">
        <p className="text-center font-hero text-[12px] font-bold uppercase tracking-[0.12em] text-grant">
          Team
        </p>
        <h2 className="mx-auto mt-2 max-w-xl text-center font-hero text-[1.75rem] font-bold leading-[1.15] tracking-[-0.02em] text-paper">
          Built by people who run agents.
        </h2>
        <div className="mx-auto mt-8 grid max-w-3xl gap-4 sm:grid-cols-2">
          {TEAM.map((m) => (
            <a
              key={m.name}
              href={m.github}
              target="_blank"
              rel="noopener noreferrer"
              className="group flex items-center gap-4 rounded-2xl border border-ink-700 bg-ink-900 p-5 transition-colors hover:border-grant/50"
            >
              <span className="flex h-14 w-14 shrink-0 items-center justify-center rounded-full border border-grant/40 bg-grant-subtle font-hero text-[16px] font-bold text-grant">
                {m.initials}
              </span>
              <span className="min-w-0">
                <span className="block truncate font-hero text-[16px] font-bold text-paper">
                  {m.name}
                </span>
                <span className="mt-0.5 block font-hero text-[12px] text-muted">{m.role}</span>
                <span className="mt-1.5 flex items-center gap-1.5 font-hero text-[12px] text-muted transition-colors group-hover:text-paper">
                  <Github size={13} />
                  GitHub →
                </span>
              </span>
            </a>
          ))}
        </div>
      </section>

      {/* status */}
      <section className="mx-auto max-w-content px-6 pt-20">
        <div className="mx-auto max-w-3xl">
          <div className="flex items-center gap-3">
            <h2 className="font-hero text-[1.375rem] font-bold text-paper">Status: beta</h2>
            <span className="rounded-full border border-progress/30 bg-progress-subtle px-2.5 py-0.5 font-hero text-[11px] font-bold text-progress">
              v0.1.17
            </span>
          </div>
          <ul className="mt-4 overflow-hidden rounded-2xl border border-ink-700">
            {ROWS.map((r) => (
              <li
                key={r.area}
                className="flex items-baseline justify-between gap-6 border-b border-ink-800 bg-ink-900 px-5 py-3.5 last:border-b-0"
              >
                <span className="font-hero text-[13px] text-paper">{r.area}</span>
                <span className="shrink-0 text-right font-hero text-[12px] text-muted">
                  {r.state}
                </span>
              </li>
            ))}
          </ul>
        </div>
      </section>

      {/* security + links */}
      <section className="mx-auto max-w-content px-6 py-20">
        <div className="mx-auto max-w-3xl rounded-2xl border border-ink-700 bg-ink-900 p-6 text-center md:p-10">
          <h2 className="font-hero text-[1.375rem] font-bold text-paper">Security posture</h2>
          <p className="mx-auto mt-3 max-w-xl font-hero text-[13.5px] leading-[1.7] text-muted">
            Deny-by-default on filesystem, network, and environment. No silent fallback to
            unsandboxed runsa missing backend fails loudly. Every blocked access is logged.
            Known limitations (no CPU throttling, no unix-socket grants) are
            documented, not buried.
          </p>
          <div className="mt-6 flex flex-wrap justify-center gap-3">
            <a
              href={REPO}
              className="rounded-[10px] bg-gradient-to-r from-grant to-blueprint px-5 py-2.5 font-hero text-[14px] font-semibold text-white transition hover:brightness-110"
            >
              Repository →
            </a>
            <a
              href="/docs/security"
              className="rounded-[10px] border border-ink-600 bg-ink-950 px-5 py-2.5 font-hero text-[14px] font-semibold text-paper transition hover:-translate-y-px hover:border-grant/60"
            >
              Security review →
            </a>
            <a
              href="/docs"
              className="rounded-[10px] border border-ink-600 bg-ink-950 px-5 py-2.5 font-hero text-[14px] font-semibold text-paper transition hover:-translate-y-px hover:border-grant/60"
            >
              Docs →
            </a>
          </div>
          <p className="mt-6 font-hero text-[11px] uppercase tracking-[0.1em] text-muted/70">
            warden · MIT licensed
          </p>
        </div>
      </section>

      <Footer />
    </main>
  );
}
