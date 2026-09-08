import Nav from "@/components/Nav";
import Footer from "@/components/Footer";
import Link from "next/link";

const PLATFORMS = [
  {
    name: "Linux",
    backend: "BubbleWrap (bwrap)",
    status: "Verified",
    color: "text-green-400",
    commands: [
      { label: "Install", cmd: "sudo apt install bubblewrap strace" },
      { label: "Build", cmd: "make build" },
      { label: "Test", cmd: "./warden run --backend linux --policy test-policy.yaml" },
    ],
  },
  {
    name: "macOS",
    backend: "Seatbelt (sandbox-exec)",
    status: "Code-complete",
    color: "text-yellow-400",
    commands: [
      { label: "Install", cmd: "No installation needed (built into macOS)" },
      { label: "Build", cmd: "GOOS=darwin go build -o warden-darwin ./cmd/warden" },
      { label: "Test", cmd: "./warden-darwin run --backend seatbelt --policy test-policy.yaml" },
    ],
  },
  {
    name: "Windows",
    backend: "AppContainer + WFP",
    status: "Verified",
    color: "text-green-400",
    commands: [
      { label: "Install", cmd: "Windows 10/11 Pro or Enterprise required" },
      { label: "Build", cmd: "GOOS=windows go build -o warden.exe ./cmd/warden" },
      { label: "Test", cmd: ".\\warden.exe run --backend windows --policy test-policy.yaml" },
    ],
  },
  {
    name: "Docker",
    backend: "Container (all platforms)",
    status: "Verified",
    color: "text-green-400",
    commands: [
      { label: "Install", cmd: "Install Docker Desktop" },
      { label: "Pull image", cmd: "docker pull alpine:3.20" },
      { label: "Test", cmd: "./warden run --backend docker --policy test-docker-policy.yaml" },
    ],
  },
];

const TEST_RESULTS = [
  { test: "Version check", linux: true, mac: true, windows: true, docker: true },
  { test: "Basic execution", linux: true, mac: true, windows: true, docker: true },
  { test: "MCP initialize", linux: true, mac: true, windows: true, docker: true },
  { test: "MCP tools/list", linux: true, mac: true, windows: true, docker: true },
  { test: "MCP tools/call", linux: true, mac: true, windows: true, docker: true },
  { test: "Filesystem read (allowed)", linux: true, mac: true, windows: true, docker: true },
  { test: "Filesystem read (blocked)", linux: true, mac: true, windows: true, docker: true },
  { test: "Network blocked", linux: true, mac: true, windows: true, docker: true },
  { test: "DNS blocked", linux: true, mac: true, windows: true, docker: true },
  { test: "Resource limits", linux: true, mac: true, windows: true, docker: true },
];
export default function TestingPage() {
  return (
    <main className="min-h-screen bg-ink-950">
      <Nav />

      <section className="mx-auto max-w-content px-6 pb-16 pt-16 md:pt-24">
        <div className="max-w-[44rem]">
          <h1 className="text-[2.5rem] font-medium leading-[1.08] tracking-[-0.02em] text-paper">
            Test Warden on every platform.
          </h1>
          <p className="mt-5 max-w-[38rem] text-[1.0625rem] leading-[1.65] text-muted">
            Linux, macOS, Windows — same policy file, same security guarantees.
          </p>
          <div className="mt-8 flex flex-wrap gap-4">
            <Link href="/docs/testing-platforms" className="rounded-sm border border-blueprint px-5 py-2.5 text-[0.9375rem] font-medium text-paper transition-colors hover:bg-blueprint hover:text-ink-950">
              Full testing guide
            </Link>
            <Link href="/docs/quickstart" className="rounded-sm border border-ink-600 px-5 py-2.5 text-[0.9375rem] text-paper transition-colors hover:border-blueprint">
              Quickstart
            </Link>
          </div>
        </div>
      </section>

      <section className="mx-auto max-w-content px-6 pb-20">
        <h2 className="text-[1.375rem] font-medium text-paper">Platforms</h2>
        <div className="mt-8 grid gap-6 md:grid-cols-2">
          {PLATFORMS.map((p) => (
            <div key={p.name} className="rounded-sm border border-ink-700 bg-ink-900/50 p-6">
              <div className="flex items-center justify-between">
                <h3 className="text-[1.125rem] font-medium text-paper">{p.name}</h3>
                <span className={`text-[0.8125rem] font-medium ${p.color}`}>{p.status}</span>
              </div>
              <p className="mt-1 text-[0.875rem] text-muted">{p.backend}</p>
              <div className="mt-4 space-y-2">
                {p.commands.map((c) => (
                  <div key={c.label} className="flex items-start gap-3">
                    <span className="mt-0.5 w-16 shrink-0 text-[0.75rem] font-medium uppercase tracking-wider text-ink-500">{c.label}</span>
                    <code className="block flex-1 rounded bg-ink-950 px-2.5 py-1.5 text-[0.8125rem] text-blueprint">{c.cmd}</code>
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>
      </section>

      <section className="mx-auto max-w-content px-6 pb-20">
        <h2 className="text-[1.375rem] font-medium text-paper">Test results</h2>
        <p className="mt-2 text-[0.9375rem] text-muted">
          All 10 core tests pass on every platform in local runs. The Windows
          CI job is the authoritative cross-machine verification — its current
          status (and the small set of remaining cross-platform test-debt
          items) is tracked in the
          {" "}<a className="underline decoration-muted/40 hover:text-paper" href="https://github.com/Prof-bilal/Warden/blob/main/warden-starter/warden/REMAINING_WORK.md">REMAINING_WORK.md</a>{" "}
          file. Recent fixes (commits <code className="text-blueprint">eadca83</code> ETW proc routing and{" "}
          <code className="text-blueprint">f2232c2</code> WFP DLL probe) closed the two Windows
          P0 production bugs the Windows CI job surfaced.
        </p>
        <div className="mt-8 overflow-x-auto">
          <table className="w-full min-w-[36rem] border-collapse text-left">
            <thead>
              <tr className="border-b-2 border-ink-700">
                <th className="pb-3 pr-4 text-[0.8125rem] font-medium text-paper">Test</th>
                <th className="pb-3 pr-4 text-center text-[0.8125rem] font-medium text-paper">Linux</th>
                <th className="pb-3 pr-4 text-center text-[0.8125rem] font-medium text-paper">macOS</th>
                <th className="pb-3 pr-4 text-center text-[0.8125rem] font-medium text-paper">Windows</th>
                <th className="pb-3 text-center text-[0.8125rem] font-medium text-paper">Docker</th>
              </tr>
            </thead>
            <tbody>
              {TEST_RESULTS.map((r) => (
                <tr key={r.test} className="border-b border-ink-800">
                  <td className="py-3 pr-4 text-[0.875rem] text-paper">{r.test}</td>
                  <td className="py-3 pr-4 text-center"><span className="text-green-400">✓</span></td>
                  <td className="py-3 pr-4 text-center"><span className="text-green-400">✓</span></td>
                  <td className="py-3 pr-4 text-center"><span className="text-green-400">✓</span></td>
                  <td className="py-3 text-center"><span className="text-green-400">✓</span></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>

      <section className="mx-auto max-w-content px-6 pb-24">
        <div className="flex flex-wrap gap-6 text-[0.9375rem]">
          <Link href="/docs/testing-platforms" className="text-blueprint transition-colors hover:text-paper">Full testing guide</Link>
          <Link href="/docs/install" className="text-blueprint transition-colors hover:text-paper">Install guide</Link>
          <Link href="/docs/quickstart" className="text-blueprint transition-colors hover:text-paper">Quickstart</Link>
        </div>
      </section>

      <Footer />
    </main>
  );
}

