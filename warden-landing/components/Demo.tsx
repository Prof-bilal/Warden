const VIDEO_SRC = "/videos/video-9ReotrYUC6t1GFvpAcsf.mp4";

export default function Demo() {
  return (
    <section id="demo" className="border-t border-ink-800">
      <div className="mx-auto max-w-content px-6 py-20">
        <div className="flex flex-col gap-2 md:flex-row md:items-baseline md:justify-between">
          <h2 className="text-[1.75rem] font-medium leading-[1.15] tracking-[-0.01em] text-paper">
            Watch it enforce a policy.
          </h2>
          <span className="font-mono text-[0.6875rem] uppercase tracking-[0.08em] text-muted">
            30s · recorded terminal session
          </span>
        </div>
        <p className="mt-3 max-w-[34rem] text-[1rem] leading-[1.65] text-muted">
          A Warden sandbox from the inside: the policy grants are applied, a
          granted call succeeds, and everything outside the grant is stopped at
          the boundary and written to the audit log.
        </p>

        {/* 16:9 reserve via aspect-[16/9] + intrinsic dimensions: no layout
            shift when the video loads. preload="none" defers the 4.3MB
            download until the visitor presses play. */}
        <div className="mt-10 overflow-hidden rounded-[12px] border border-ink-700 bg-ink-900">
          <video
            className="aspect-video w-full"
            width={1920}
            height={1080}
            controls
            preload="none"
            playsInline
            aria-label="Warden demo: a 30-second recorded session showing a sandboxed MCP server run under a policy, with blocked access attempts stopped and logged"
          >
            <source src={VIDEO_SRC} type="video/mp4" />
            Your browser does not support the video tag. Download the demo:
            <a href={VIDEO_SRC}>warden demo (MP4)</a>
          </video>
        </div>
      </div>
    </section>
  );
}
