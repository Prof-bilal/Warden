import type { Metadata } from "next";
import type { ReactNode } from "react";
import { SITE_URL } from "@/lib/seo";
import JsonLd from "@/components/JsonLd";
import { breadcrumbSchema } from "@/lib/schema";
import Link from "next/link";
import PageShell from "@/components/PageShell";
import PageHeader from "@/components/bp/PageHeader";
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
    box: "border-acid/30 bg-acid/10 text-acid",
    dot: "bg-acid",
  },
  "Partially verified": {
    box: "border-progress/30 bg-progress-subtle text-progress",
    dot: "bg-progress",
  },
  "Not yet run": {
    box: "border-hair bg-void text-offwhite/35",
    dot: "bg-offwhite/20",
  },
};

const TH =
  "px-4 py-3 text-left font-mono text-[9.5px] uppercase tracking-micro text-offwhite/35";
const TD = "px-4 py-2.5 text-[12.5px] text-offwhite/75";

function Badge({ children, tone }: { children: ReactNode; tone: "pass" | "block" }) {
  return (
    <span
      className={
        "inline-flex items-center gap-1 px-1.5 py-0.5 font-mono text-[10px] font-semibold uppercase tracking-micro " +
        (tone === "pass" ? "bg-acid text-void" : "bg-deny text-void")
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
    <PageShell>
      <JsonLd
        data={breadcrumbSchema([
          { name: "Home", url: `${SITE_URL}/` },
          { name: "Testing", url: `${SITE_URL}/testing` },
        ])}
      />

      <PageHeader eyebrow="Test proof" title="Warden test proof, by platform.">
        <p className="mt-4 max-w-2xl font-sans text-[14.5px] leading-[1.75] text-offwhite/45">
          Each result below comes from a recorded run. Platforms without a
          recorded run are marked as not yet run.
        </p>
        <div className="mt-8 flex flex-wrap gap-3">
          <Link
            href="/docs/testing-platforms"
            className="rounded-md bg-acid px-5 py-2.5 font-sans text-[13px] font-semibold text-void transition-colors hover:bg-acid-bright"
          >
            Full testing guide
          </Link>
          <Link
            href="/docs/quickstart"
            className="rounded-md border border-hair bg-cell px-5 py-2.5 font-sans text-[13px] font-semibold text-offwhite/85 transition-colors hover:border-offwhite/20"
          >
            Quickstart
          </Link>
        </div>
      </PageHeader>

      {/* platforms */}
      <section className="border-b border-hair px-6 py-12 md:py-14">
        <div className="grid gap-px border border-hair bg-hair md:grid-cols-2">
          {PLATFORMS.map((p) => (
            <div key={p.name} className="bg-cell p-5 md:p-6">
              <div className="flex items-center justify-between gap-3">
                <h3 className="font-mono text-[15px] font-semibold text-offwhite">
                  {p.name}
                </h3>
                <span
                  className={
                    "flex items-center gap-1.5 border px-2.5 py-1 font-mono text-[10px] uppercase tracking-micro " +
                    STATUS_STYLE[p.status].box
                  }
                >
                  <span className={"h-1.5 w-1.5 rounded-full " + STATUS_STYLE[p.status].dot} />
                  {p.status}
                </span>
              </div>
              <p className="mt-2 inline-block border border-hair bg-void px-2 py-0.5 font-mono text-[11px] text-offwhite/70">
                {p.backend}
              </p>
              <p className="mt-3 font-sans text-[12.5px] leading-[1.7] text-offwhite/45">
                {p.note}
              </p>
              <div className="mt-4 space-y-2">
                {p.commands.map((c) => (
                  <div key={c.label} className="flex items-start gap-3">
                    <span className="mt-0.5 w-20 shrink-0 font-mono text-[9px] uppercase tracking-micro text-offwhite/30">
                      {c.label}
                    </span>
                    <code className="block min-w-0 flex-1 overflow-x-auto bg-void px-2.5 py-1.5 font-mono text-[11.5px] text-acid">
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
      <section className="border-b border-hair px-6 py-12 md:py-14">
        <div className="border border-hair bg-cell">
          <div className="flex flex-wrap items-center gap-3 border-b border-hair px-5 py-4">
            <h2 className="font-mono text-[14px] font-semibold text-offwhite">
              Linux proof harness
            </h2>
            <span className="border border-acid/30 bg-acid/10 px-2.5 py-0.5 font-mono text-[10px] uppercase tracking-micro text-acid">
              {stepsPass}/{HARNESS.stepsPassed + HARNESS.stepsFailed} steps pass
            </span>
          </div>
          <p className="border-b border-hair px-5 py-4 font-sans text-[12.5px] leading-[1.75] text-offwhite/45">
            Run <span className="text-offwhite/80">{HARNESS.runId}</span> on {HARNESS.platform} (
            {HARNESS.backend}). The positive control (the target process starts) is{" "}
            {HARNESS.positiveControl}, and the run exited with code {HARNESS.runExit}. Each
            step&apos;s observed result is compared with its expected result.
          </p>
          <div className="overflow-x-auto">
            <table className="w-full min-w-[36rem] border-collapse text-left">
              <thead>
                <tr className="border-b border-hair bg-void/60">
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
                      className="border-b border-hair transition-colors last:border-b-0 hover:bg-offwhite/[0.03]"
                    >
                      <td className={TD + " font-mono text-[11.5px]"}>{s.step}</td>
                      <td className={TD}>{s.what}</td>
                      <td className={TD + " text-offwhite/45"}>{s.expected}</td>
                      <td className={TD + " text-offwhite/45"}>{s.observed}</td>
                      <td className="px-4 py-2.5 text-center">
                        <Badge tone={ok ? "pass" : "block"}>{ok ? "PASS" : "FAIL"}</Badge>
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
      <section className="border-b border-hair px-6 py-12 md:py-14">
        <div className="border border-hair bg-cell">
          <div className="flex flex-wrap items-center gap-3 border-b border-hair px-5 py-4">
            <h2 className="font-mono text-[14px] font-semibold text-offwhite">
              Go build, vet and test
            </h2>
            <span className="border border-acid/30 bg-acid/10 px-2.5 py-0.5 font-mono text-[10px] uppercase tracking-micro text-acid">
              {TEST_FUNCTION_COUNT} test functions
            </span>
          </div>
          <p className="border-b border-hair px-5 py-4 font-sans text-[12.5px] leading-[1.75] text-offwhite/45">
            Project checks from AGENTS.md, run on {RUN_DATE} on Linux. The suite runs
            serially (<code className="text-acid">-p 1</code>); parallel runs can exhaust
            the process limit and fail for reasons unrelated to Warden.
          </p>
          <div className="overflow-x-auto">
            <table className="w-full min-w-[36rem] border-collapse text-left">
              <thead>
                <tr className="border-b border-hair bg-void/60">
                  <th className={TH}>Check</th>
                  <th className={TH}>Exit</th>
                  <th className={TH}>Result</th>
                </tr>
              </thead>
              <tbody>
                {GO_CHECKS.map((g) => (
                  <tr
                    key={g.check}
                    className="border-b border-hair transition-colors last:border-b-0 hover:bg-offwhite/[0.03]"
                  >
                    <td className={TD + " font-mono text-[11.5px]"}>{g.check}</td>
                    <td className={TD}>
                      <Badge tone={g.exit === 0 ? "pass" : "block"}>
                        {g.exit === 0 ? "0 PASS" : `${g.exit} FAIL`}
                      </Badge>
                    </td>
                    <td className={TD + " text-offwhite/45"}>{g.result}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      </section>

      {/* Audit log */}
      <section className="border-b border-hair px-6 py-12 md:py-14">
        <div className="border border-hair bg-cell p-5 md:p-6">
          <h2 className="font-mono text-[14px] font-semibold text-offwhite">
            Audit log from the proof run
          </h2>
          <p className="mt-2 font-sans text-[12.5px] leading-[1.75] text-offwhite/45">
            The run wrote {AUDIT_SUMMARY.total} audit entries. Every blocked access is
            logged, not silently dropped.
          </p>
          <div className="mt-5 grid gap-px border border-hair bg-hair sm:grid-cols-3">
            <div className="bg-void p-4">
              <p className="font-mono text-[9px] uppercase tracking-micro text-offwhite/35">
                Allowed
              </p>
              <p className="mt-1 font-mono text-[22px] font-semibold text-acid">
                {auditAllowed}
              </p>
            </div>
            <div className="bg-void p-4">
              <p className="font-mono text-[9px] uppercase tracking-micro text-offwhite/35">
                Blocked
              </p>
              <p className="mt-1 font-mono text-[22px] font-semibold text-deny">
                {auditBlocked}
              </p>
            </div>
            <div className="bg-void p-4">
              <p className="font-mono text-[9px] uppercase tracking-micro text-offwhite/35">
                File / network
              </p>
              <p className="mt-1 font-mono text-[12px] text-offwhite/70">
                File {AUDIT_SUMMARY.file.allowed} allowed · {AUDIT_SUMMARY.file.blocked} blocked
              </p>
              <p className="font-mono text-[12px] text-offwhite/70">
                Network {AUDIT_SUMMARY.network.allowed} allowed ·{" "}
                {AUDIT_SUMMARY.network.blocked} blocked
              </p>
            </div>
          </div>
        </div>
      </section>

      <section className="px-6 py-10">
        <div className="flex flex-wrap gap-6 font-mono text-[11px] uppercase tracking-micro">
          <Link href="/docs/testing-platforms" className="text-acid transition-colors hover:text-acid-bright">
            Full testing guide
          </Link>
          <Link href="/docs/install" className="text-acid transition-colors hover:text-acid-bright">
            Install guide
          </Link>
          <Link href="/docs/quickstart" className="text-acid transition-colors hover:text-acid-bright">
            Quickstart
          </Link>
        </div>
      </section>
    </PageShell>
  );
}
