import Link from "next/link";
import WardenMark from "@/components/bp/WardenMark";

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

export default function Footer({ variant = "dark" }: { variant?: "light" | "dark" }) {
  const light = variant === "light";

  return (
    <footer
      className={
        light
          ? "border-t border-[#E9E7E0] bg-[#FAFAF7]"
          : "border-t border-hair bg-void"
      }
    >
      <div className="mx-auto max-w-[1280px] px-6 py-14 md:py-16">
        <div className="grid gap-12 md:grid-cols-[1.3fr_2fr] md:gap-20">
          <div className="max-w-xs">
            <Link
              href="/"
              className={`inline-flex items-center gap-2.5 font-mono text-[15px] font-semibold ${
                light ? "text-[#14171A]" : "text-offwhite"
              }`}
            >
              <WardenMark size={20} className={light ? "text-[#0B6B4A]" : "text-acid"} />
              <span className="lowercase">warden</span>
            </Link>
            <p
              className={`mt-4 font-sans text-sm leading-6 ${
                light ? "text-[#616761]" : "text-offwhite/45"
              }`}
            >
              A permission layer for local AI tools and MCP servers.
            </p>
            <p
              className={`mt-5 font-mono text-[10.5px] uppercase tracking-micro ${
                light ? "text-[#8A908A]" : "text-offwhite/30"
              }`}
            >
              Open source · MIT licensed
            </p>
          </div>

          <div className="grid grid-cols-2 gap-x-8 gap-y-10 sm:grid-cols-3">
            {columns.map((column) => (
              <div key={column.title}>
                <h2
                  className={`font-mono text-[9.5px] font-medium uppercase tracking-micro ${
                    light ? "text-[#8A908A]" : "text-offwhite/40"
                  }`}
                >
                  {column.title}
                </h2>
                <nav
                  aria-label={`${column.title} links`}
                  className="mt-4 flex flex-col items-start gap-3"
                >
                  {column.links.map(([label, href]) =>
                    href.startsWith("http") ? (
                      <a
                        key={label}
                        href={href}
                        target="_blank"
                        rel="noopener noreferrer"
                        className={`font-mono text-[12.5px] transition-colors ${
                          light
                            ? "text-[#5A615B] hover:text-[#14171A]"
                            : "text-offwhite/50 hover:text-offwhite"
                        }`}
                      >
                        {label}
                      </a>
                    ) : (
                      <Link
                        key={label}
                        href={href}
                        className={`font-mono text-[12.5px] transition-colors ${
                          light
                            ? "text-[#5A615B] hover:text-[#14171A]"
                            : "text-offwhite/50 hover:text-offwhite"
                        }`}
                      >
                        {label}
                      </Link>
                    ),
                  )}
                </nav>
              </div>
            ))}
          </div>
        </div>

        <div
          className={`mt-12 flex flex-col gap-4 border-t pt-5 font-mono text-[11px] sm:flex-row sm:items-center sm:justify-between ${
            light
              ? "border-[#E9E7E0] text-[#8A908A]"
              : "border-hair text-offwhite/35"
          }`}
        >
          <span>Built for safer local development.</span>
          <div className="flex gap-5">
            <Link
              href="/privacy"
              className={light ? "transition-colors hover:text-[#14171A]" : "transition-colors hover:text-offwhite"}
            >
              Privacy
            </Link>
            <Link
              href="/terms"
              className={light ? "transition-colors hover:text-[#14171A]" : "transition-colors hover:text-offwhite"}
            >
              Terms
            </Link>
            <Link
              href="/contact"
              className={light ? "transition-colors hover:text-[#14171A]" : "transition-colors hover:text-offwhite"}
            >
              Contact
            </Link>
          </div>
        </div>
      </div>
    </footer>
  );
}
