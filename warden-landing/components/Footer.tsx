import Link from "next/link";
import Image from "next/image";

const columns = [
  {
    title: "Product",
    links: [
      ["Warden 2.0", "/v2"],
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
    <footer className="site-footer">
      <div className="grid gap-10 py-14 md:grid-cols-[1fr_2fr] md:gap-16">
        <div>
          <Link href="/" className="site-brand" aria-label="Warden home">
            <Image src="/warden-mark.png" alt="" width={22} height={22} />
            warden
          </Link>
          <p className="mt-4 max-w-[28ch]">
            A permission layer for local AI tools and MCP servers.
          </p>
          <p className="mt-4 text-muted">Open source · MIT licensed</p>
        </div>
        <div className="grid grid-cols-2 gap-6 sm:grid-cols-3">
          {columns.map(({ title, links }) => (
            <nav key={title} aria-label={`${title} links`}>
              <h2 className="font-semibold text-offwhite">{title}</h2>
              <ul className="mt-4 space-y-2">
                {links.map(([label, href]) => (
                  <li key={href}>
                    <Link href={href}>{label}</Link>
                  </li>
                ))}
              </ul>
            </nav>
          ))}
        </div>
      </div>
      <div className="flex flex-wrap gap-5 border-t border-hair py-5">
        <Link href="/privacy">Privacy</Link>
        <Link href="/terms">Terms</Link>
        <Link href="/contact">Contact</Link>
      </div>
    </footer>
  );
}
