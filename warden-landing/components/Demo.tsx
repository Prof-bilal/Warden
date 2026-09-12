import SandboxPlayer from "@/components/SandboxPlayer";

export default function Demo() {
  return (
    <section id="demo" className="border-t border-ink-800">
      <div className="mx-auto max-w-content px-6 py-20">
        <div className="flex flex-col gap-2 md:flex-row md:items-baseline md:justify-between">
          <h2 className="text-[1.75rem] font-medium leading-[1.15] tracking-[-0.01em] text-paper">
            Watch it enforce a policy.
          </h2>
          <span className="font-mono text-[0.6875rem] uppercase tracking-[0.08em] text-muted">
            24s · rendered live in React
          </span>
        </div>
        <p className="mt-3 max-w-[34rem] text-[1rem] leading-[1.65] text-muted">
          A Warden sandbox from the inside: the policy grants are applied, a
          granted call succeeds, and everything outside the grant is stopped at
          the boundary and written to the audit log.
        </p>

        {/* Remotion-style player: the whole "recording" is rendered live in
            React — seekable, loopable, zero video bytes. */}
        <div className="mt-10">
          <SandboxPlayer />
        </div>
      </div>
    </section>
  );
}
