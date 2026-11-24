import Link from "next/link";
import { ArrowRight } from "lucide-react";
import CopyChip from "@/components/bp/CopyChip";

const installCommand = "npm i -g warden-sandbox-cli@beta";

const policy = `command: ["/bin/cat", "./data/notes.txt"]
filesystem:
  read: ["./data"]
  write: []
network:
  allow: []
env:
  allow: []`;

export default function Hero() {
  return (
    <section className="home-hero">
      <div>
        <p className="home-meta">Open source / MIT licensed</p>
        <h1>Give AI tools only the access they need.</h1>
        <p className="home-intro">
          Warden is a local sandbox for AI tools and MCP servers. Review packages
          before installing and set explicit permissions for files, networks, and secrets.
        </p>
        <div className="home-actions">
          <Link href="/docs/install" className="site-cta home-cta">
            Get started <ArrowRight size={16} aria-hidden="true" />
          </Link>
          <a href="https://github.com/Prof-bilal/Warden">
            View on GitHub <span aria-hidden="true">↗</span>
          </a>
        </div>
        <CopyChip value={installCommand} className="home-copy" />
      </div>

      <div className="home-terminal">
        <div className="home-terminal-label">
          <span>policy.yaml</span>
          <span>Example policy</span>
        </div>
        <pre><code>{policy}</code></pre>
        <div className="home-terminal-command">
          <span aria-hidden="true">$ </span>
          <code>warden run --policy ./policy.yaml</code>
        </div>
      </div>
    </section>
  );
}
