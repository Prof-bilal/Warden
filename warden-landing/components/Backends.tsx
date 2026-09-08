const BACKENDS = [
  { platform: "Linux", mechanism: "bubblewrap (unprivileged namespaces)", status: "Verified" },
  { platform: "macOS", mechanism: "sandbox-exec, with Docker fallback", status: "Code-complete" },
  { platform: "Windows", mechanism: "AppContainer + WFP + Job Objects + ETW audit", status: "Verified" },
];

const STATUS_STYLE: Record<string, string> = {
  Verified: "text-grant bg-grant-subtle",
  "Code-complete": "text-progress bg-progress-subtle",
};

export default function Backends() {
  return (
    <section id="backends" className="border-t border-ink-800">
      <div className="mx-auto max-w-content px-6 py-20">
        <h2 className="text-[1.75rem] font-medium leading-[1.15] tracking-[-0.01em] text-paper">
          One policy, three sandboxing backends.
        </h2>
        <p className="mt-3 max-w-[34rem] text-[1rem] leading-[1.65] text-muted">
          Warden picks the strongest backend your OS supports automatically.
          If none of them can be applied correctly, it refuses to run
          unsandboxed rather than falling back silently.
        </p>

        <div className="mt-10 divide-y divide-ink-800 border-y border-ink-800">
          {BACKENDS.map((b) => (
            <div key={b.platform} className="grid grid-cols-[8rem_1fr_7rem] items-center gap-4 py-4">
              <span className="text-[0.9375rem] text-paper">{b.platform}</span>
              <span className="font-mono text-[0.8125rem] text-muted">{b.mechanism}</span>
              <span className={`w-fit rounded-sm px-2.5 py-1 text-[0.75rem] ${STATUS_STYLE[b.status]}`}>
                {b.status}
              </span>
            </div>
          ))}
        </div>

        <p className="mt-6 max-w-[36rem] text-[0.8125rem] leading-[1.6] text-muted/80">
          Linux and Windows are fully verified with escape tests passing on
          real hardware. macOS is code-complete with unit tests; real-machine
          verification requires macOS hardware. Windows requires an elevated
          (Administrator) shell for WFP + ETW. Warden fails closed
          with a clear message rather than running unaudited. See the
          {" "}<a className="underline decoration-muted/40 hover:text-paper" href="/docs/install">install docs</a>{" "}
          and the {" "}<a className="underline decoration-muted/40 hover:text-paper" href="https://github.com/Prof-bilal/Warden/blob/main/warden-starter/warden/REMAINING_WORK.md">REMAINING_WORK</a>{" "}
          tracker for the exact CI verification state.
        </p>
      </div>
    </section>
  );
}
