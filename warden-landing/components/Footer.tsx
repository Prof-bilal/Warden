import Link from "next/link";

const columns = [
  {
    title: "Product",
    links: [
      ["How it works", "/#how-it-works"],
      ["How to use", "/#how-to-use"],
      ["Setup", "/setup"],
      ["Compatibility", "/compatibility"],
      ["Testing", "/testing"],
    ],
  },
  {
    title: "Learn",
    links: [
      ["Docs", "/docs"],
      ["Guides", "/guides"],
      ["Policy packs", "/policies"],
      ["Connections", "/integrations"],
      ["Roadmap", "/docs/roadmap"],
    ],
  },
  {
    title: "Project",
    links: [
      ["About", "/about"],
      ["Blog", "/blog"],
      ["Verify evidence", "/verify"],
      ["Maintenance", "/maintenance"],
      ["GitHub", "https://github.com/Prof-bilal/Warden"],
    ],
  },
] as const;

export default function Footer() {
  return (
    <footer className="border-t border-ink-800 bg-ink-950">
      <div className="mx-auto max-w-content px-6 py-14 md:py-16">
        <div className="grid gap-12 md:grid-cols-[1.3fr_2fr] md:gap-20">
          <div className="max-w-xs">
            <Link href="/" className="inline-flex items-center gap-2 font-hero text-lg font-bold text-paper">
              <span className="flex h-7 w-7 items-center justify-center rounded-lg border border-blueprint/40 bg-blueprint/10 text-sm text-blueprint">
                W
              </span>
              warden
            </Link>
            <p className="mt-4 font-hero text-sm leading-6 text-muted">
              A permission layer for local AI tools and MCP servers.
            </p>
            <p className="mt-5 font-hero text-xs text-muted/70">Open source · MIT licensed</p>
          </div>

          <div className="grid grid-cols-2 gap-x-8 gap-y-10 sm:grid-cols-3">
            {columns.map((column) => (
              <div key={column.title}>
                <h2 className="font-hero text-xs font-semibold uppercase tracking-[0.14em] text-paper">
                  {column.title}
                </h2>
                <nav aria-label={`${column.title} links`} className="mt-4 flex flex-col items-start gap-3">
                  {column.links.map(([label, href]) =>
                    href.startsWith("http") ? (
                      <a
                        key={label}
                        href={href}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="font-hero text-sm text-muted transition-colors hover:text-paper"
                      >
                        {label}
                      </a>
                    ) : (
                      <Link key={label} href={href} className="font-hero text-sm text-muted transition-colors hover:text-paper">
                        {label}
                      </Link>
                    ),
                  )}
                </nav>
              </div>
            ))}
          </div>
        </div>

        <div className="mt-12 flex flex-col gap-4 border-t border-ink-800 pt-5 font-hero text-xs text-muted sm:flex-row sm:items-center sm:justify-between">
          <span>Built for safer local development.</span>
          <div className="flex gap-5">
            <Link href="/privacy" className="transition-colors hover:text-paper">Privacy</Link>
            <Link href="/terms" className="transition-colors hover:text-paper">Terms</Link>
            <Link href="/contact" className="transition-colors hover:text-paper">Contact</Link>
          </div>
        </div>
      </div>
    </footer>
  );
}
