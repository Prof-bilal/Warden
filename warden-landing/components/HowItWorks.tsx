"use client";

import { useState } from "react";
import {
  Check,
  Copy,
  FolderLock,
  Gauge,
  Globe,
  KeyRound,
  Terminal,
} from "lucide-react";

const POLICY_TEXT = `command: ["node", "server.js"]

filesystem:
  read: ["./data"]
  write: ["./output"]

network:
  allow: ["api.github.com"]

env:
  allow: ["GITHUB_TOKEN"]

limits:
  memory_mb: 512
  timeout_s: 300`;

type FieldId = "command" | "filesystem" | "network" | "env" | "limits";

type Line = { field: FieldId | null; indent: number; html: React.ReactNode };

function K({ children }: { children: React.ReactNode }) {
  return <span className="text-blueprint">{children}</span>;
}
function S({ children }: { children: React.ReactNode }) {
  return <span className="text-grant">{children}</span>;
}
function N({ children }: { children: React.ReactNode }) {
  return <span className="text-progress">{children}</span>;
}
function P({ children }: { children: React.ReactNode }) {
  return <span className="text-muted">{children}</span>;
}

const LINES: Line[] = [
  { field: "command", indent: 0, html: <><K>command</K><P>: [</P><S>"node"</S><P>, </P><S>"server.js"</S><P>]</P></> },
  { field: null, indent: 0, html: <>&nbsp;</> },
  { field: "filesystem", indent: 0, html: <><K>filesystem</K><P>:</P></> },
  { field: "filesystem", indent: 1, html: <><K>read</K><P>: [</P><S>"./data"</S><P>]</P></> },
  { field: "filesystem", indent: 1, html: <><K>write</K><P>: [</P><S>"./output"</S><P>]</P></> },
  { field: null, indent: 0, html: <>&nbsp;</> },
  { field: "network", indent: 0, html: <><K>network</K><P>:</P></> },
  { field: "network", indent: 1, html: <><K>allow</K><P>: [</P><S>"api.github.com"</S><P>]</P></> },
  { field: null, indent: 0, html: <>&nbsp;</> },
  { field: "env", indent: 0, html: <><K>env</K><P>:</P></> },
  { field: "env", indent: 1, html: <><K>allow</K><P>: [</P><S>"GITHUB_TOKEN"</S><P>]</P></> },
  { field: null, indent: 0, html: <>&nbsp;</> },
  { field: "limits", indent: 0, html: <><K>limits</K><P>:</P></> },
  { field: "limits", indent: 1, html: <><K>memory_mb</K><P>: </P><N>512</N></> },
  { field: "limits", indent: 1, html: <><K>timeout_s</K><P>: </P><N>300</N></> },
];

const FIELDS: { id: FieldId; note: string; icon: React.ReactNode }[] = [
  {
    id: "command",
    note: "How Warden starts the servernothing else runs.",
    icon: <Terminal size={15} />,
  },
  {
    id: "filesystem",
    note: "Read and write are separate grants; anything not listed is invisible, not just unreadable.",
    icon: <FolderLock size={15} />,
  },
  {
    id: "network",
    note: "Only these hostnames resolve. DNS for anything else fails before a connection is even attempted.",
    icon: <Globe size={15} />,
  },
  {
    id: "env",
    note: "Only these variables are passed throughno inherited shell environment.",
    icon: <KeyRound size={15} />,
  },
  {
    id: "limits",
    note: "The process is killed if either bound is crossed.",
    icon: <Gauge size={15} />,
  },
];

const STEPS = [
  { n: "01", title: "Write a policy", desc: "warden.yaml lists every grant" },
  { n: "02", title: "Run Warden", desc: "warden run --policy warden.yaml" },
  { n: "03", title: "Stay sandboxed", desc: "denied calls fail · all logged" },
];

export default function HowItWorks() {
  const [active, setActive] = useState<FieldId | null>(null);
  const [copied, setCopied] = useState(false);

  const howToSchema = {
    "@context": "https://schema.org",
    "@type": "HowTo",
    name: "How to sandbox an MCP server with Warden",
    description:
      "Write a policy file, run Warden, and your MCP server is sandboxed with deny-by-default controls.",
    step: [
      {
        "@type": "HowToStep",
        name: "Write a policy file",
        text: "Create a YAML policy that specifies which filesystem paths, network hosts, and environment variables your MCP server can access.",
        position: 1,
      },
      {
        "@type": "HowToStep",
        name: "Run Warden with the policy",
        text: "Execute warden run --policy policy.yaml to start your MCP server inside the sandbox.",
        position: 2,
      },
      {
        "@type": "HowToStep",
        name: "Server runs sandboxed",
        text: "The server can only access what the policy grants. Everything else fails, and Warden logs every blocked attempt.",
        position: 3,
      },
    ],
  };

  function handleCopy() {
    navigator.clipboard.writeText(POLICY_TEXT);
    setCopied(true);
    setTimeout(() => setCopied(false), 1800);
  }

  return (
    <section id="how-it-works" className="border-t border-ink-800 bg-ink-950">
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{ __html: JSON.stringify(howToSchema) }}
      />
      <div className="mx-auto max-w-content px-6 py-20 md:py-28">
        {/* header */}
        <div className="mx-auto max-w-[640px] text-center">
          <p className="mb-4 font-hero text-[12px] font-bold uppercase tracking-[0.12em] text-grant">
            How it works
          </p>
          <h2 className="font-hero text-[1.75rem] font-bold leading-[1.15] tracking-[-0.02em] text-paper md:text-[2.25rem]">
            One file describes exactly what a server can touch.
          </h2>
          <p className="mt-4 font-hero text-[15px] leading-[1.7] text-muted md:text-[16px]">
            No code changes, no SDK. Write the grants, run Warden, everything else disappears.
          </p>
        </div>

        {/* steps */}
        <div className="mx-auto mt-10 grid max-w-4xl gap-3 sm:grid-cols-3">
          {STEPS.map((s) => (
            <div
              key={s.n}
              className="rounded-xl border border-ink-700 bg-ink-900 px-4 py-3.5"
            >
              <p className="font-hero text-[11px] font-bold tracking-[0.14em] text-grant">
                {s.n}
              </p>
              <p className="mt-1 font-hero text-[14px] font-bold text-paper">{s.title}</p>
              <p className="mt-0.5 truncate font-hero text-[12px] text-muted">{s.desc}</p>
            </div>
          ))}
        </div>

        {/* editor + field guide */}
        <div className="mx-auto mt-6 grid max-w-4xl gap-4 lg:grid-cols-[1.15fr_0.85fr]">
          {/* code editor */}
          <div className="overflow-hidden rounded-2xl border border-ink-700 bg-[#0B0E13]">
            <div className="flex items-center gap-2 border-b border-ink-800 bg-ink-900 px-4 py-2.5">
              <span className="h-2.5 w-2.5 rounded-full bg-deny/80" />
              <span className="h-2.5 w-2.5 rounded-full bg-progress/80" />
              <span className="h-2.5 w-2.5 rounded-full bg-grant/80" />
              <span className="ml-2 rounded-md border border-ink-700 bg-ink-950 px-2 py-0.5 font-hero text-[11px] text-muted">
                warden.yaml
              </span>
              <button
                onClick={handleCopy}
                aria-label="Copy policy file"
                className="ml-auto flex items-center gap-1.5 rounded-lg border border-ink-700 px-2 py-1 font-hero text-[11px] text-muted transition-colors hover:border-grant/50 hover:text-paper"
              >
                {copied ? <Check size={12} className="text-grant" /> : <Copy size={12} />}
                {copied ? "Copied" : "Copy"}
              </button>
            </div>
            <div className="overflow-x-auto p-4">
              <code className="block min-w-max font-hero text-[12.5px] leading-[1.85] md:text-[13px]">
                {LINES.map((line, i) => {
                  const dim = active !== null && line.field !== null && line.field !== active;
                  const lit = active !== null && line.field === active;
                  return (
                    <span
                      key={i}
                      className={
                        "flex rounded px-2 transition-all " +
                        (lit ? "bg-grant-subtle" : "") +
                        (dim ? " opacity-30" : "")
                      }
                      style={line.indent ? { paddingLeft: `calc(0.5rem + ${line.indent * 1.25}rem)` } : undefined}
                    >
                      <span className="w-7 shrink-0 select-none text-right text-muted/40">
                        {line.field === null ? "" : i + 1}
                      </span>
                      <span className="pl-4">{line.html}</span>
                    </span>
                  );
                })}
              </code>
            </div>
          </div>

          {/* field guide */}
          <div className="flex flex-col gap-2.5">
            {FIELDS.map((f) => {
              const isActive = active === f.id;
              return (
                <button
                  key={f.id}
                  onMouseEnter={() => setActive(f.id)}
                  onMouseLeave={() => setActive(null)}
                  onFocus={() => setActive(f.id)}
                  onBlur={() => setActive(null)}
                  onClick={() => setActive(isActive ? null : f.id)}
                  className={
                    "rounded-xl border p-3.5 text-left transition-colors " +
                    (isActive
                      ? "border-grant/50 bg-grant-subtle"
                      : "border-ink-700 bg-ink-900 hover:border-ink-500")
                  }
                >
                  <span className="flex items-center gap-2">
                    <span className={isActive ? "text-grant" : "text-blueprint"}>{f.icon}</span>
                    <span className="font-hero text-[13px] font-bold text-paper">{f.id}</span>
                  </span>
                  <span className="mt-1 block font-hero text-[12.5px] leading-[1.6] text-muted">
                    {f.note}
                  </span>
                </button>
              );
            })}
            <p className="px-1 font-hero text-[11px] uppercase tracking-[0.08em] text-muted/70">
              Hover a grant to locate it in the file
            </p>
          </div>
        </div>
      </div>
    </section>
  );
}
