import Nav from "@/components/Nav";
import Footer from "@/components/Footer";

const REPO = "https://github.com/Prof-bilal/Warden";

const ROWS = [
  { area: "Linux sandbox (bubblewrap)", state: "Filesystem, network proxy, audit, limits" },
  { area: "macOS sandbox (Seatbelt)", state: "With Docker fallback" },
  { area: "Windows sandbox", state: "AppContainer + WFP, fail-closed" },
  { area: "trace / init / logs", state: "Observe, generate, inspect" },
  { area: "Approval mode", state: "Prompt instead of hard-fail" },
  { area: "Gateway integration", state: "Wrap gateway-registered servers" },
  { area: "Compatibility matrix", state: "18 servers, 14 pass" },
];

export default function About() {
  return (
    <main className="min-h-screen bg-ink-950">
      <Nav />
      <section className="mx-auto max-w-content px-6 pb-20 pt-16 md:pt-24">
        <div className="max-w-[38rem]">
          <h1 className="text-[2.5rem] font-medium leading-[1.08] tracking-[-0.02em] text-paper">
            Sandbox every server by default.
          </h1>
          <p className="mt-5 max-w-[34rem] text-[1.0625rem] leading-[1.65] text-muted">
            Modern AI tooling runs third-party code with first-party trust.
            MCP servers install with a one-liner and inherit everything you
            can do. Warden exists to make the safe path the easy path — with
            a policy file small enough to read in one sitting.
          </p>
        </div>

        <div className="mt-14 max-w-[44rem]">
          <h2 className="text-[1.375rem] font-medium text-paper">Why not just use Docker?</h2>
          <p className="mt-3 leading-[1.65] text-muted">
            You can — Warden uses it as a fallback. But Docker is heavyweight
            for “run one script with a restricted home directory”: slow cold
            starts, a daemon dependency, and a far bigger trust boundary than
            a namespace sandbox needs. Warden is a single static binary over
            OS-native primitives, so sandboxing a server costs almost nothing.
          </p>
        </div>

        <div className="mt-14">
          <h2 className="text-[1.375rem] font-medium text-paper">Status: beta</h2>
          <ul className="mt-4 divide-y divide-ink-800 border-y border-ink-800">
            {ROWS.map((r) => (
              <li key={r.area} className="flex items-baseline justify-between gap-6 py-3.5">
                <span className="font-mono text-[0.875rem] text-paper">{r.area}</span>
                <span className="shrink-0 text-right text-[0.8125rem] text-muted">{r.state}</span>
              </li>
            ))}
          </ul>
        </div>

        <div className="mt-14 max-w-[44rem]">
          <h2 className="text-[1.375rem] font-medium text-paper">Security posture</h2>
          <p className="mt-3 leading-[1.65] text-muted">
            Deny-by-default on filesystem, network, and environment. No silent
            fallback to unsandboxed runs — a missing backend fails loudly.
            Every blocked access is logged. Known limitations (no CPU
            throttling, no wildcard hosts, no unix-socket grants) are
            documented, not buried.
          </p>
          <div className="mt-6 flex flex-wrap gap-6 text-[0.9375rem]">
            <a href={REPO} className="text-blueprint transition-colors hover:text-paper">
              Repository →
            </a>
            <a
              href="/docs/security"
              className="text-blueprint transition-colors hover:text-paper"
            >
              Security review →
            </a>
            <a href="/docs" className="text-blueprint transition-colors hover:text-paper">
              Docs →
            </a>
          </div>
          <p className="mt-8 text-[0.8125rem] text-muted">warden — MIT licensed</p>
        </div>
      </section>
      <Footer />
    </main>
  );
}
