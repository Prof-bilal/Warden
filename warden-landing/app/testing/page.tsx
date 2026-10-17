import type { Metadata } from "next";
import type { ReactNode } from "react";
import { SITE_URL } from "@/lib/seo";
import JsonLd from "@/components/JsonLd";
import { breadcrumbSchema } from "@/lib/schema";
import Nav from "@/components/Nav";
import Footer from "@/components/Footer";
import Link from "next/link";
import {
  AUDIT_SUMMARY,
  GO_CHECKS,
  HARNESS,
  RUN_DATE,
  TEST_FUNCTION_COUNT,
} from "@/lib/testing-evidence";

const DESCRIPTION =
  "Reproduced Warden test results: the Linux proof harness and the Go test suite, with the platforms still to run clearly marked.";

export const metadata: Metadata = {
  title: { absolute: "Warden test proof, by platform." },
  description: DESCRIPTION,
  alternates: {
    canonical: `${SITE_URL}/testing`,
  },
  openGraph: {
    title: "Warden test proof, by platform.",
    description: DESCRIPTION,
    url: `${SITE_URL}/testing`,
    images: [
      {
        url: `${SITE_URL}/og-image.png`,
        width: 1200,
        height: 630,
        alt: "Warden test proof, by platform.",
      },
    ],
  },
  twitter: {
    card: "summary_large_image",
    title: "Warden test proof, by platform.",
    description: DESCRIPTION,
    images: [`${SITE_URL}/og-image.png`],
  },
};

type Status = "Verified" | "Partially verified" | "Not yet run";

const PLATFORMS: {
  name: string;
  backend: string;
  status: Status;
  note: string;
  commands: { label: string; cmd: string }[];
}[] = [
  {
    name: "Linux",
    backend: "BubbleWrap (bwrap)",
    status: "Verified",
    note: `Proof harness and Go suite run on ${RUN_DATE}.`,
    commands: [
      { label: "Install", cmd: "sudo apt install bubblewrap strace" },
      { label: "Build", cmd: "go build -o warden ./cmd/warden" },
      { label: "Prove", cmd: "bash testdata/proof/run-proof.sh ./warden" },
    ],
  },
  {
    name: "Docker",
    backend: "Container (all platforms)",
    status: "Partially verified",
    note: "Docker backend test passes in the Go suite on the Linux host. Not run on macOS or Windows.",
    commands: [
      { label: "Install", cmd: "Install Docker Desktop or Docker Engine" },
      { label: "Pull image", cmd: "docker pull alpine:3.20" },
      { label: "Test", cmd: "go test ./internal/sandbox/docker/ -count=1" },
    ],
  },
  {
    name: "macOS",
    backend: "Seatbelt (sandbox-exec)",
    status: "Not yet run",
    note: "No proof run has been recorded on macOS.",
    commands: [
      { label: "Install", cmd: "No installation needed (built into macOS)" },
      { label: "Build", cmd: "GOOS=darwin go build -o warden-darwin ./cmd/warden" },
      { label: "Prove", cmd: "bash testdata/proof/run-proof.sh ./warden-darwin" },
    ],
  },
  {
    name: "Windows",
    backend: "AppContainer + WFP",
    status: "Not yet run",
    note: "No proof run has been recorded on Windows. Requires Windows 10/11 Pro or Enterprise.",
    commands: [
      { label: "Build", cmd: "GOOS=windows go build -o warden.exe ./cmd/warden" },
      { label: "Prove", cmd: "bash testdata/proof/run-proof.sh ./warden.exe" },
    ],
  },
];

const STATUS_STYLE: Record<Status, { box: string; dot: string }> = {
  Verified: {
    box: "border-grant/30 bg-grant-subtle text-grant",
    dot: "bg-grant",
  },
  "Partially verified": {
    box: "border-progress/30 bg-progress-subtle text-progress",
    dot: "bg-progress",
  },
  "Not yet run": {
    box: "border-ink-600 bg-ink-950 text-muted",
    dot: "bg-ink-600",
  },
};

const TH = "px-4 py-3 text-[11px] font-bold uppercase tracking-[0.1em] text-muted";
const TD = "px-4 py-2.5 text-[13px] text-paper/90";

function Badge({ children, tone }: { children: ReactNode; tone: "pass" | "block" }) {
  return (
    <span
      className={
        "inline-flex items-center gap-1 rounded px-1.5 py-0.5 text-[10px] font-bold tracking-[0.06em] " +
        (tone === "pass" ? "bg-grant text-ink-950" : "bg-deny text-ink-950")
      }
    >
      {children}
    </span>
  );
}

export default function TestingPage() {
  const { steps } = HARNESS;
  const stepsPass = steps.filter((s) => s.observed === s.expected).length;
  const auditBlocked = AUDIT_SUMMARY.file.blocked + AUDIT_SUMMARY.network.blocked;
  const auditAllowed = AUDIT_SUMMARY.file.allowed + AUDIT_SUMMARY.network.allowed;

  return (
    <main className="min-h-screen bg-ink-950">
      <JsonLd
        data={breadcrumbSchema([
          { name: "Home", url: `${SITE_URL}/` },
          { name: "Testing", url: `${SITE_URL}/testing` },
        ])}
      />
      <Nav />

      {/* header */}
      <section className="border-t border-ink-800">
        <div className="mx-auto max-w-content px-6 pb-14 pt-16 text-center md:pt-24">
          <p className="mb-4 font-hero text-[12px] font-bold uppercase tracking-[0.12em] text-grant">
            Test proof
          </p>
          <h1 className="mx-auto max-w-3xl font-hero text-[2.25rem] font-bold leading-[1.1] tracking-[-0.02em] text-paper sm:text-[2.75rem]">
            Warden test proof, by platform.
          </h1>
          <p className="mx-auto mt-4 max-w-[38rem] font-hero text-[15px] leading-[1.7] text-muted md:text-[16px]">
            Each result below comes from a recorded run. Platforms without a recorded run are marked
            as not yet run.
          </p>
          <div className="mt-8 flex flex-wrap justify-center gap-3">
            <Link
              href="/docs/testing-platforms"
              className="rounded-[10px] bg-gradient-to-r from-grant to-blueprint px-5 py-2.5 font-hero text-[14px] font-semibold text-white transition hover:brightness-110"
            >
              Full testing guide
            </Link>
            <Link
              href="/docs/quickstart"
              className="rounded-[10px] border border-ink-600 bg-ink-900 px-5 py-2.5 font-hero text-[14px] font-semibold text-paper transition hover:-translate-y-px hover:border-grant/60"
            >
              Quickstart
            </Link>
          </div>
        </div>
      </section>

      {/* platforms */}
      <section className="mx-auto max-w-content px-6 pb-20">
        <div className="grid gap-4 md:grid-cols-2">
          {PLATFORMS.map((p) => (
            <div
              key={p.name}
              className="rounded-2xl border border-ink-700 bg-ink-900 p-5 transition-colors hover:border-ink-500 md:p-6"
            >
              <div className="flex items-center justify-between gap-3">
                <h3 className="font-hero text-[17px] font-bold text-paper">{p.name}</h3>
                <span
                  className={
                    "flex items-center gap-1.5 rounded-full border px-2.5 py-1 font-hero text-[11px] font-bold " +
                    STATUS_STYLE[p.status].box
                  }
                >
                  <span className={"h-1.5 w-1.5 rounded-full " + STATUS_STYLE[p.status].dot} />
                  {p.status}
                </span>
              </div>
              <p className="mt-2 inline-block rounded-md border border-ink-700 bg-ink-950 px-2 py-0.5 font-hero text-[12px] text-paper/90">
                {p.backend}
              </p>
              <p className="mt-3 font-hero text-[12.5px] leading-[1.6] text-muted">{p.note}</p>
              <div className="mt-4 space-y-2">
                {p.commands.map((c) => (
                  <div key={c.label} className="flex items-start gap-3">
                    <span className="mt-1 w-20 shrink-0 font-hero text-[10px] font-bold uppercase tracking-[0.1em] text-muted">
                      {c.label}
                    </span>
                    <code className="block min-w-0 flex-1 overflow-x-auto rounded-lg bg-ink-950 px-2.5 py-1.5 font-hero text-[12px] text-blueprint">
                      {c.cmd}
                    </code>
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>
      </section>

      {/* Linux proof harness */}
      <section className="mx-auto max-w-content px-6 pb-20">
        <div className="overflow-hidden rounded-2xl border border-ink-700 bg-ink-900">
          <div className="flex flex-wrap items-center gap-3 border-b border-ink-700 px-5 py-4">
            <h2 className="font-hero text-[15px] font-bold text-paper">
              Linux proof harness
            </h2>
            <span className="rounded-full border border-grant/30 bg-grant-subtle px-2.5 py-0.5 font-hero text-[11px] font-bold text-grant">
              {stepsPass}/{HARNESS.stepsPassed + HARNESS.stepsFailed} steps pass
            </span>
          </div>
          <p className="border-b border-ink-800 px-5 py-4 font-hero text-[12.5px] leading-[1.7] text-muted">
            Run <span className="text-paper/90">{HARNESS.runId}</span> on {HARNESS.platform} ({HARNESS.backend}).
            The positive control (the target process starts) is {HARNESS.positiveControl}, and the run
            exited with code {HARNESS.runExit}. Each step's observed result is compared with its expected result.
          </p>
          <div className="overflow-x-auto">
            <table className="w-full min-w-[36rem] border-collapse text-left font-hero">
              <thead>
                <tr className="border-b border-ink-700 bg-ink-950/60">
                  <th className={TH}>Step</th>
                  <th className={TH}>What it checks</th>
                  <th className={TH}>Expected</th>
                  <th className={TH}>Observed</th>
                  <th className={TH + " text-center"}>Result</th>
                </tr>
              </thead>
              <tbody>
                {steps.map((s) => {
                  const ok = s.observed === s.expected;
                  return (
                    <tr
                      key={s.step}
                      className="border-b border-ink-800 transition-colors last:border-b-0 hover:bg-ink-800/40"
                    >
                      <td className={TD + " font-mono text-[12px]"}>{s.step}</td>
                      <td className={TD}>{s.what}</td>
                      <td className={TD + " text-muted"}>{s.expected}</td>
                      <td className={TD + " text-muted"}>{s.observed}</td>
                      <td className="px-4 py-2.5 text-center">
                        <Badge tone={ok ? "pass" : "block"}>{ok ? "✓ PASS" : "✗ FAIL"}</Badge>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </div>
      </section>

      {/* Go checks */}
      <section className="mx-auto max-w-content px-6 pb-20">
        <div className="overflow-hidden rounded-2xl border border-ink-700 bg-ink-900">
          <div className="flex flex-wrap items-center gap-3 border-b border-ink-700 px-5 py-4">
            <h2 className="font-hero text-[15px] font-bold text-paper">Go build, vet and test</h2>
            <span className="rounded-full border border-grant/30 bg-grant-subtle px-2.5 py-0.5 font-hero text-[11px] font-bold text-grant">
              {TEST_FUNCTION_COUNT} test functions
            </span>
          </div>
          <p className="border-b border-ink-800 px-5 py-4 font-hero text-[12.5px] leading-[1.7] text-muted">
            Project checks from AGENTS.md, run on {RUN_DATE} on Linux. The suite runs serially
            (<code className="text-blueprint">-p 1</code>); parallel runs can exhaust the process limit
            and fail for reasons unrelated to Warden.
          </p>
          <div className="overflow-x-auto">
            <table className="w-full min-w-[36rem] border-collapse text-left font-hero">
              <thead>
                <tr className="border-b border-ink-700 bg-ink-950/60">
                  <th className={TH}>Check</th>
                  <th className={TH}>Exit</th>
                  <th className={TH}>Result</th>
                </tr>
              </thead>
              <tbody>
                {GO_CHECKS.map((g) => (
                  <tr
                    key={g.check}
                    className="border-b border-ink-800 transition-colors last:border-b-0 hover:bg-ink-800/40"
                  >
                    <td className={TD + " font-mono text-[12px]"}>{g.check}</td>
                    <td className={TD}>
                      <Badge tone={g.exit === 0 ? "pass" : "block"}>{g.exit === 0 ? "0 PASS" : `${g.exit} FAIL`}</Badge>
                    </td>
                    <td className={TD + " text-muted"}>{g.result}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      </section>

      {/* Audit log */}
      <section className="mx-auto max-w-content px-6 pb-20">
        <div className="rounded-2xl border border-ink-700 bg-ink-900 p-5 md:p-6">
          <h2 className="font-hero text-[15px] font-bold text-paper">Audit log from the proof run</h2>
          <p className="mt-2 font-hero text-[12.5px] leading-[1.7] text-muted">
            The run wrote {AUDIT_SUMMARY.total} audit entries. Every blocked access is logged, not
            silently dropped.
          </p>
          <div className="mt-5 grid gap-3 sm:grid-cols-3">
            <div className="rounded-xl border border-ink-700 bg-ink-950 p-4">
              <p className="font-hero text-[11px] font-bold uppercase tracking-[0.1em] text-muted">Allowed</p>
              <p className="mt-1 font-hero text-[22px] font-bold text-grant">{auditAllowed}</p>
            </div>
            <div className="rounded-xl border border-ink-700 bg-ink-950 p-4">
              <p className="font-hero text-[11px] font-bold uppercase tracking-[0.1em] text-muted">Blocked</p>
              <p className="mt-1 font-hero text-[22px] font-bold text-deny">{auditBlocked}</p>
            </div>
            <div className="rounded-xl border border-ink-700 bg-ink-950 p-4">
              <p className="font-hero text-[11px] font-bold uppercase tracking-[0.1em] text-muted">
                File / network
              </p>
              <p className="mt-1 font-hero text-[14px] text-paper/90">
                File {AUDIT_SUMMARY.file.allowed} allowed · {AUDIT_SUMMARY.file.blocked} blocked
              </p>
              <p className="font-hero text-[14px] text-paper/90">
                Network {AUDIT_SUMMARY.network.allowed} allowed · {AUDIT_SUMMARY.network.blocked} blocked
              </p>
            </div>
          </div>
        </div>
      </section>

      <section className="mx-auto max-w-content px-6 pb-24">
        <div className="flex flex-wrap gap-6 font-hero text-[13px]">
          <Link href="/docs/testing-platforms" className="text-blueprint transition-colors hover:text-paper">
            Full testing guide
          </Link>
          <Link href="/docs/install" className="text-blueprint transition-colors hover:text-paper">
            Install guide
          </Link>
          <Link href="/docs/quickstart" className="text-blueprint transition-colors hover:text-paper">
            Quickstart
          </Link>
        </div>
      </section>

      <Footer />
    </main>
  );
}
