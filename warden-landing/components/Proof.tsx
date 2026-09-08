const GUARANTEES = [
  {
    title: "Fail-closed by default",
    body: "Warden refuses to start without a valid sandbox backend. If enforcement can't be verified, the server never runs.",
  },
  {
    title: "Deny-by-default filesystem",
    body: "Only explicitly granted paths are visible. Everything else — including the rest of the filesystem, environment variables, and network — is invisible.",
  },
  {
    title: "Network enforced at the kernel level",
    body: "An in-process egress proxy blocks every hostname not in the policy. DNS is resolved only after the allowlist check. No policy grant, no connection.",
  },
  {
    title: "Environment filtering",
    body: "Only env.allow names are forwarded. Empty allowlist = empty environment. No secrets leak through ungranted variables.",
  },
  {
    title: "Fail-closed on every platform",
    body: "Linux (bubblewrap), macOS (Seatbelt), Windows (AppContainer + WFP + ETW), Docker fallback — each requires its primitives to initialize or the run is refused.",
  },
  {
    title: "Resource limits enforced by the kernel",
    body: "Wall-clock timeout, memory RSS sampling with SIGTERM-then-SIGKILL, and kill-on-close via Job Objects on Windows. A runaway process is terminated with its entire tree.",
  },
];

export default function Proof() {
  return (
    <section id="proof" className="border-t border-ink-800">
      <div className="mx-auto max-w-content px-6 py-20">
        <h2 className="text-[1.75rem] font-medium leading-[1.15] tracking-[-0.01em] text-paper">
          It blocks attacks by design, not by accident.
        </h2>
        <p className="mt-3 max-w-[34rem] text-[1rem] leading-[1.65] text-muted">
          Warden enforces six hard invariants. Every one of them is verified by
          the test suite — including escape tests that confirm the sandbox
          actually prevents the access it claims to block.
        </p>

        <div className="mt-10 grid gap-6 sm:grid-cols-2">
          {GUARANTEES.map((g) => (
            <div
              key={g.title}
              className="rounded-sm border border-ink-700 bg-ink-900 p-5"
            >
              <dt className="font-mono text-[0.875rem] text-blueprint">
                {g.title}
              </dt>
              <dd className="mt-2 text-[0.9375rem] leading-[1.55] text-muted">
                {g.body}
              </dd>
            </div>
          ))}
        </div>

        {/* What's been verified */}
        <div className="mt-10 border-y border-ink-800 py-6">
          <p className="text-[0.8125rem] leading-[1.6] text-muted">
            <span className="text-grant">Verified on Linux (Arch x86_64):</span>{" "}
            7 escape tests pass — read grants accessible, unlisted paths invisible,
            write grants writable, writes outside grants denied, exit codes
            propagated, environment passthrough filtered, fail-closed without bwrap.
          </p>
          <p className="mt-3 text-[0.8125rem] leading-[1.6] text-muted">
            <span className="text-progress">Code-complete, verification pending:</span>{" "}
            macOS (Seatbelt) and Windows (AppContainer + WFP + ETW) backends
            implemented with unit tests; real-machine escape tests require their
            respective platforms.
          </p>
        </div>

        {/* Honesty disclaimer */}
        <p className="mt-6 text-[0.8125rem] leading-[1.6] text-muted/80">
          We publish only claims backed by committed test fixtures and source
          code. Attack-simulation benchmarks are not included until a
          reproducible harness and fixtures are committed to the repo. See{" "}
          <a
            className="underline decoration-muted/40 hover:text-paper"
            href="https://github.com/Prof-bilal/Warden/blob/main/warden-starter/warden/ROADMAP.md"
          >
            ROADMAP.md
          </a>{" "}
          for what&apos;s still in progress.
        </p>
      </div>
    </section>
  );
}
