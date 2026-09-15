import SandboxPlayer from "@/components/SandboxPlayer";

export default function Demo() {
  return (
    <section id="demo" className="border-t border-ink-800 bg-ink-950">
      <div className="mx-auto max-w-content px-6 py-20 md:py-28">
        <div className="mx-auto max-w-[640px] text-center">
          <p className="mb-4 font-hero text-[12px] font-bold uppercase tracking-[0.12em] text-grant">
            Live demo · rendered in React
          </p>
          <h2 className="font-hero text-[1.75rem] font-bold leading-[1.15] tracking-[-0.02em] text-paper md:text-[2.25rem]">
            Watch it enforce a policy.
          </h2>
          <p className="mt-4 font-hero text-[15px] leading-[1.7] text-muted md:text-[16px]">
            A Warden sandbox from the inside: the policy grants are applied, a granted call
            succeeds, and everything outside the grant is stopped at the boundary and written to
            the audit log.
          </p>
        </div>

        {/* Remotion-style player: the whole "recording" is rendered live in
            Reactseekable, loopable, zero video bytes. */}
        <div className="mx-auto mt-10 max-w-5xl">
          <SandboxPlayer />
        </div>
        <p className="mt-3 text-center font-hero text-[11px] uppercase tracking-[0.08em] text-muted">
          24s loop · click the stage or scrub the timeline
        </p>
      </div>
    </section>
  );
}
