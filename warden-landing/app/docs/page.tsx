import Nav from "@/components/Nav";
import Footer from "@/components/Footer";

const CARDS = [
  {
    title: "Install",
    body: "Linux, macOS, Windows, Docker fallback — or build from source with Go 1.22+. What each platform needs before warden will run.",
    href: "/docs/install",
  },
  {
    title: "Quickstart",
    body: "Sandbox your first MCP server in five minutes: copy a fixture policy, run it, then trace-and-generate for your own servers.",
    href: "/docs/quickstart",
  },
  {
    title: "Policy schema",
    body: "Every policy.yaml field, validation rule, and enforcement note — the one reference to keep open while writing policies.",
    href: "/docs/schema",
  },
  {
    title: "CLI reference",
    body: "run, trace, init, logs, gateway, --approve — every subcommand and flag, verified against the source.",
    href: "/docs/cli",
  },
  {
    title: "Compatibility matrix",
    body: "18 tested servers with exact policies: 14 pass, 2 conditional, 2 fail with classified reasons.",
    href: "/docs/compatibility",
  },
  {
    title: "FAQ",
    body: "Missing backends, bare npx commands, blocked access, the HOME footgun, and what Warden can't do.",
    href: "/docs/faq",
  },
  {
    title: "Architecture",
    body: "How the CLI, policy engine, sandbox backends, egress proxy, and audit logger fit together — with component diagrams.",
    href: "/docs/architecture",
  },
  {
    title: "Example policies",
    body: "Copy-paste policies for filesystem, GitHub, Slack, PostgreSQL, Brave Search, and a comprehensive reference template.",
    href: "/docs/examples",
  },
  {
    title: "Security review",
    body: "Threat model, known limitations, credential exposure risks, and best practices before relying on any policy in production.",
    href: "/docs/security",
  },
  {
    title: "Roadmap",
    body: "M0 through M8 milestones — what's built, what's shipped, and what's next for Warden.",
    href: "/docs/roadmap",
  },
  {
    title: "Contributing",
    body: "Development setup, pull request guidance, and the three highest-impact contribution areas right now.",
    href: "/docs/contributing",
  },
  {
    title: "Testing guide",
    body: "Unit tests, integration tests, escape tests, fixtures, and CI expectations for the project.",
    href: "/docs/testing",
  },
];

export default function Docs() {
  return (
    <main className="min-h-screen bg-ink-950">
      <Nav />
      <section className="mx-auto max-w-content px-6 pb-20 pt-16 md:pt-24">
        <div className="max-w-[38rem]">
          <h1 className="text-[2.5rem] font-medium leading-[1.08] tracking-[-0.02em] text-paper">
            Docs.
          </h1>
          <p className="mt-5 max-w-[34rem] text-[1.0625rem] leading-[1.65] text-muted">
            Install it, run your first sandboxed server, then go deep on the
            policy schema. Start with Install and Quickstart — in that order.
          </p>
        </div>

        <div className="mt-14 grid gap-px overflow-hidden rounded-sm border border-ink-800 bg-ink-800 md:grid-cols-2">
          {CARDS.map((c) => (
            <a
              key={c.title}
              href={c.href}
              className="group bg-ink-950 p-7 transition-colors hover:bg-ink-900"
            >
              <h2 className="text-[1.125rem] font-medium text-paper">
                {c.title}
                <span className="ml-2 inline-block text-blueprint transition-transform group-hover:translate-x-0.5">
                  →
                </span>
              </h2>
              <p className="mt-2 text-[0.9375rem] leading-[1.6] text-muted">{c.body}</p>
            </a>
          ))}
        </div>
      </section>
      <Footer />
    </main>
  );
}
