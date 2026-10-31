import SectionMarker from "@/components/bp/SectionMarker";
import Reveal from "@/components/bp/Reveal";

const features = [
  {
    title: "Explicit permissions",
    body: "Every file path, network host, and environment variable a tool can access is declared in a single policy file.",
  },
  {
    title: "Fail-closed security",
    body: "Blocked by default. If an action is not explicitly allowed, it cannot run — no prompts, no surprises.",
  },
  {
    title: "Full audit logs",
    body: "Every tool call is logged with timestamps, target, and decision, so you can trace exactly what happened.",
  },
  {
    title: "Open source",
    body: "MIT licensed and fully transparent. Inspect the code, audit the security model, and contribute.",
  },
];

export default function EcosystemFeatures() {
  return (
    <section className="border-b border-hair">
      <div className="px-6 py-16 md:py-20">
        <div className="max-w-2xl">
          <SectionMarker index="07" />
          <h2 className="mt-4 font-mono text-[1.55rem] font-semibold leading-[1.18] tracking-[-0.01em] text-offwhite md:text-[1.95rem]">
            Built for a safer AI ecosystem.
          </h2>
          <p className="mt-4 max-w-[470px] font-sans text-[14.5px] leading-[1.75] text-offwhite/45">
            Run AI tools and MCP servers with clear boundaries, fail-closed
            security, and zero trust by default.
          </p>
        </div>

        <Reveal className="mt-10 grid gap-px border border-hair bg-hair sm:grid-cols-2 lg:grid-cols-4">
          {features.map(({ title, body }) => (
            <article key={title} className="bg-cell p-5">
              <h3 className="font-mono text-[12px] font-semibold text-offwhite/85">{title}</h3>
              <p className="mt-2 font-sans text-[11.5px] leading-[1.7] text-offwhite/35">{body}</p>
            </article>
          ))}
        </Reveal>
      </div>
    </section>
  );
}
