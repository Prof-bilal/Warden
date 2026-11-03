import type { Metadata } from "next";
import { SITE_URL } from "@/lib/seo";
import JsonLd from "@/components/JsonLd";
import { breadcrumbSchema } from "@/lib/schema";
import Link from "next/link";
import PageShell from "@/components/PageShell";
import PageHeader from "@/components/bp/PageHeader";
import FeatureNav from "@/components/FeatureNav";
import Diagram from "@/components/Diagram";
import CodeBlock from "@/components/CodeBlock";
import CopyableCode from "@/components/CopyableCode";

export const metadata: Metadata = {
  title: { absolute: "Every Warden feature, explained in depth." },
  description:
    "Every Warden feature in depth: sandboxed runs, policy tooling, audit logs, gateway, proxy, Kubernetes rendering, and self-update, with exact commands.",
  alternates: {
    canonical: `${SITE_URL}/features`,
  },
  openGraph: {
    title: "Every Warden feature, explained in depth.",
    description:
      "run, init, trace, logs, doctor, gateway, proxy, k8s, and update: the complete feature surface of the Warden sandbox runtime, with usage commands.",
    url: `${SITE_URL}/features`,
    type: "article",
    images: [
      {
        url: `${SITE_URL}/og-image.png`,
        width: 1200,
        height: 630,
        alt: "Every Warden feature, explained in depth.",
      },
    ],
  },
  twitter: {
    card: "summary_large_image",
    title: "Every Warden feature, explained in depth.",
    description:
      "run, init, trace, logs, doctor, gateway, proxy, k8s, and update: the complete feature surface of the Warden sandbox runtime.",
    images: [`${SITE_URL}/og-image.png`],
  },
};

function Flag({ name, desc }: { name: string; desc: string }) {
  return (
    <div className="flex flex-col gap-0.5 border-b border-hair py-2.5 last:border-b-0 sm:flex-row sm:gap-6">
      <span className="shrink-0 sm:w-56">
        <CopyableCode text={name} />
      </span>
      <span className="font-sans text-[13px] leading-[1.65] text-offwhite/45">{desc}</span>
    </div>
  );
}

function Note({
  tone,
  title,
  children,
}: {
  tone: "grant" | "deny" | "progress";
  title: string;
  children: React.ReactNode;
}) {
  const styles = {
    grant: "border-grant/30 bg-grant-subtle text-grant",
    deny: "border-deny/30 bg-deny-subtle text-deny",
    progress: "border-progress/30 bg-progress-subtle text-progress",
  }[tone];
  return (
    <div className={`border px-4 py-3 font-sans text-[13px] leading-[1.7] ${styles}`}>
      <span className="font-semibold">{title}</span>{" "}
      <span className="text-offwhite/45">{children}</span>
    </div>
  );
}

type Feature = {
  id: string;
  name: string;
  tagline: string;
  badge?: string;
  body: string;
  bullets: string[];
  usage: string;
  flags: { name: string; desc: string }[];
  example: string;
  note?: { tone: "grant" | "deny" | "progress"; title: string; text: string };
  image: { src: string; alt: string };
};

const FEATURES: Feature[] = [
  {
    id: "run",
    name: "warden run",
    tagline: "Run any MCP server inside the sandbox",
    badge: "Core",
    body: "The heart of Warden. Point it at a policy and a server command, Warden initializes an OS-native sandbox (bubblewrap on Linux, Seatbelt on macOS, AppContainer + WFP + Job Objects on Windows, Docker as a fallback), applies every grant, and only then starts the process. If the sandbox cannot be initialized, Warden refuses to run. Before launch it prints a summary of exactly what the server will and won't see.",
    bullets: [
      "Deny-by-default: unlisted paths, hosts, and env vars are invisible, not merely blocked",
      "Backend auto-detection with --backend override (auto | linux | seatbelt | windows | docker)",
      "Resource limits enforced by the kernel: wall-clock timeout + memory RSS with SIGTERM → SIGKILL of the whole tree",
      "Exit code of the sandboxed server is propagated to your shell for scripting",
    ],
    usage: "warden run --policy <file> [options] -- <command...>",
    flags: [
      { name: "--policy <file>", desc: "Security policy to enforce (required)" },
      { name: "--backend <name>", desc: "auto (default), linux, seatbelt, windows, docker" },
      { name: "--approve", desc: "Prompt interactively on first out-of-policy filesystem access; approved grants are saved to the policy and the server restarts with them" },
      { name: "--approve-timeout <dur>", desc: "Per-prompt timeout (e.g. 30s, 2m); requires --approve" },
    ],
    example:
      "warden run --policy policy.yaml -- /usr/bin/node server.js\nwarden run --policy policy.yaml --backend docker -- python app.py\nwarden run --policy policy.yaml --approve -- npx @modelcontextprotocol/server-github",
    note: {
      tone: "deny",
      title: "Fails closed.",
      text: "No valid backend ⇒ no execution, ever. Warden never falls back to running unsandboxed.",
    },
    image: {
      src: "/diagrams/sandbox-boundary.jpeg",
      alt: "A server process stopped at the Warden sandbox boundary",
    },
  },
  {
    id: "init",
    name: "warden init",
    tagline: "Draft a deny-by-default policy",
    body: "Creates policy.yaml, the single file that describes everything a server may touch. Run it after a trace and it converts the recorded accesses into a conservative starter policy; or write one by hand using the schema. Warden refuses to overwrite an existing policy, so re-running is always safe.",
    bullets: [
      "Generates filesystem read/write, network allowlist, env allowlist, and limits sections",
      "Reads the latest trace by default, or a specific log with --log",
      "Relative paths in the policy resolve against the policy file's directory",
      "Refuses to overwrite existing files (O_EXCL create)",
    ],
    usage: "warden init [--log <file>] [--output <file>]",
    flags: [
      { name: "--log <file>", desc: "Audit/trace log to read (default: latest trace)" },
      { name: "--output <file>", desc: "Output policy file (default: policy.yaml)" },
    ],
    example: "warden init\nwarden init --log trace.jsonl --output starter.yaml",
    image: {
      src: "/diagrams/create-policy.jpeg",
      alt: "Creating a Warden security policy from a trace",
    },
  },
  {
    id: "trace",
    name: "warden trace",
    tagline: "Record what a server really accesses",
    body: "Runs the server once without the sandbox (deliberately) and records every filesystem, network, and environment access to a JSONL trace. That trace is the raw material for warden init, so the policy matches the server's real behavior instead of your guesswork. One run, one trace, one honest policy.",
    bullets: [
      "Records access attempts as JSONL events in the Warden state directory",
      "The recommended flow: trace once → init → review the generated policy → run sandboxed",
      "Works with any command, including npx/uvx launchers",
    ],
    usage: "warden trace -- <command...>",
    flags: [],
    example: "warden trace -- npx @modelcontextprotocol/server-github\nwarden trace -- /usr/bin/node server.js",
    note: {
      tone: "progress",
      title: "Unsandboxed by design.",
      text: "Trace runs outside the sandbox to observe everything the server wants. Only trace servers you already trust enough to run.",
    },
    image: {
      src: "/diagrams/policy-flow.jpeg",
      alt: "Policy architecture flow from trace to enforcement",
    },
  },
  {
    id: "logs",
    name: "warden logs",
    tagline: "Every allow/deny decision, on the record",
    body: "Reads the audit log, a JSONL file where Warden records each access decision a sandboxed server made, allowed or denied. Tail it live while a server runs, or review after an incident. The audit log is the product: claims about blocking are backed by entries you can grep.",
    bullets: [
      "JSONL format: one decision per line, easy to ingest elsewhere",
      "--follow polls every 250ms for a live tail",
      "Custom log path via --log for parallel sessions",
    ],
    usage: "warden logs [--tail <n>] [--follow] [--log <file>]",
    flags: [
      { name: "--tail <n>", desc: "Show the last N lines" },
      { name: "--follow, -f", desc: "Follow the log (polls every 250ms)" },
      { name: "--log <file>", desc: "Audit log path (default: ~/.local/state/warden/audit.jsonl)" },
    ],
    example: "warden logs --tail 50\nwarden logs -f",
    image: {
      src: "/diagrams/audit-trail.jpeg",
      alt: "Security audit trail recording every allow and deny decision",
    },
  },
  {
    id: "doctor",
    name: "warden doctor",
    tagline: "Verify your system can actually sandbox",
    body: "Checks sandbox readiness before you rely on it: sandbox backend availability, namespace support, network enforcement primitives, and required runtime dependencies per platform. Use it in CI so a runner misconfiguration fails loudly instead of silently weakening enforcement.",
    bullets: [
      "Detects bwrap / sandbox-exec / AppContainer availability",
      "Validates namespace and network enforcement support",
      "A failed check is never interpreted as 'sandbox disabled', Warden fails closed",
    ],
    usage: "warden doctor",
    flags: [],
    example: "warden doctor",
    note: {
      tone: "grant",
      title: "CI tip:",
      text: "Run warden doctor as a CI step so sandbox capability regressions surface before merge.",
    },
    image: {
      src: "/diagrams/policy-inspector.jpeg",
      alt: "Developer inspecting sandbox readiness and policy state",
    },
  },
  {
    id: "gateway",
    name: "warden gateway",
    tagline: "Sandbox every server in your MCP client config",
    body: "Reads Claude Desktop / Claude Code / Cursor / VS Code style mcpServers configs (JSON) or gateway YAML registries, and sandboxes their stdio servers. Four subcommands cover the lifecycle: generate per-server policies, run one server sandboxed, wrap the whole config so the gateway itself launches every server through Warden, and list what's registered.",
    bullets: [
      "gateway init, generate deny-by-default <name>.yaml per stdio server; never overwrites; remote (SSE/HTTP) servers are skipped",
      "gateway run, launch one registered server under its policy, with the same --backend/--approve flags as warden run",
      "gateway wrap, print or write a wrapped copy of the config whose stdio commands are prefixed with 'warden run --policy ...'",
      "gateway list, show registered servers, stdio vs remote, and policy presence",
    ],
    usage:
      "warden gateway <init|run|wrap|list> --config <file> --policies <dir> [options]",
    flags: [
      { name: "--config <file>", desc: "Gateway / MCP client config to read" },
      { name: "--policies <dir>", desc: "Directory of per-server policies (<name>.yaml)" },
      { name: "--server <name>", desc: "(run) which registered server to launch" },
      { name: "--backend <name>", desc: "(run) auto | linux | seatbelt | windows | docker" },
      { name: "--approve", desc: "(run) prompt on first out-of-policy access" },
    ],
    example:
      "warden gateway init --config claude_desktop_config.json --policies ./policies\nwarden gateway run --config claude_desktop_config.json --policies ./policies --server github\nwarden gateway wrap --config claude_desktop_config.json --policies ./policies --output wrapped.json",
    note: {
      tone: "deny",
      title: "Fails closed.",
      text: "gateway run refuses when the policy is missing or the server is remote, no silent unsandboxed launch.",
    },
    image: {
      src: "/diagrams/mcp-client-proxy.jpeg",
      alt: "MCP client gateway wrapping servers with Warden",
    },
  },
  {
    id: "proxy",
    name: "warden proxy",
    tagline: "Filter MCP JSON-RPC in both directions",
    body: "A local filtering proxy that sits between your MCP client and an upstream server. Newline-delimited JSON-RPC messages are inspected both ways: tool calls are checked against an allowlist, secret deny patterns (like API-token regexes) block responses from leaking credentials, and oversized payloads are rejected. Every decision is audited. Supports local stdio upstreams and remote http(s) MCP servers (Streamable HTTP and SSE).",
    bullets: [
      "Tool allowlist, only listed tools may be called",
      "Deny patterns, regexes that block secrets (e.g. ghp_… tokens) from crossing the proxy",
      "Payload size caps and request auditing",
      "stdio upstreams receive only env.allow variables (deny-by-default)",
    ],
    usage: "warden proxy --policy <file> [options]",
    flags: [
      { name: "--policy <file>", desc: "MCP proxy policy file (must contain an mcp: section)" },
      { name: "--listen <addr>", desc: "Listen address (default: localhost:8765)" },
      { name: "--upstream <url>", desc: "Override the upstream from policy (stdio:<cmd> or https://…)" },
    ],
    example:
      'warden proxy --policy mcp-policy.yaml\nwarden proxy --policy mcp-policy.yaml --listen :9000\nwarden proxy --policy mcp-policy.yaml --upstream "stdio:cat"',
    note: {
      tone: "progress",
      title: "Experimental.",
      text: "The stdio upstream subprocess is filtered and audited at the message level but is not itself sandboxed, wrap warden proxy in warden run when it needs filesystem/network isolation.",
    },
    image: {
      src: "/diagrams/mcp-client-proxy.jpeg",
      alt: "MCP client proxy architecture filtering JSON-RPC messages",
    },
  },
  {
    id: "k8s",
    name: "warden k8s",
    tagline: "Translate policies into container + K8s hardening",
    body: "Compiles a Warden policy into hardened Deployment + NetworkPolicy manifests or a docker run command, same grants, different enforcement layer. Generated manifests ship with secure defaults: read-only root filesystem, non-root user, all capabilities dropped, and seccomp RuntimeDefault.",
    bullets: [
      "render, emit K8s YAML manifests (Deployment + NetworkPolicy egress rules)",
      "docker, emit the equivalent docker run command",
      "validate, check a policy is deployable and surface warnings (e.g. FQDN limitations)",
      "filesystem.read → readOnlyRootFilesystem + RO volumes; network.allow → NetworkPolicy egress; limits → resource limits",
    ],
    usage: "warden k8s <render|docker|validate> --policy <file> [options]",
    flags: [
      { name: "--policy <file>", desc: "Warden policy file (required)" },
      { name: "--image <image>", desc: "Container image to use (required for render/docker)" },
      { name: "--namespace <ns>", desc: "K8s namespace (default: default)" },
      { name: "--output <file>", desc: "Output file (default: stdout)" },
    ],
    example:
      "warden k8s render --policy policy.yaml --image myapp:latest --output k8s.yaml\nwarden k8s docker --policy policy.yaml --image myapp:latest\nwarden k8s validate --policy policy.yaml",
    image: {
      src: "/diagrams/ci-pipeline.jpeg",
      alt: "CI pipeline governing sandboxed builds and deployments",
    },
  },
  {
    id: "update",
    name: "warden update",
    tagline: "Checksum-verified self-update",
    body: "Updates the Warden binary in place. It queries the npm registry for warden-sandbox-cli, downloads the matching GitHub Release binary for your platform, verifies its SHA256 against the published SHA256SUMS, and installs into the versioned cache. Downgrades are refused; any verification problem aborts the install.",
    bullets: [
      "--check to compare current vs latest without installing",
      "--version <v> to pin a specific release",
      "HTTPS-only to pinned hosts; no shell interpolation of version strings",
      "The CLI also warns (once per 24h, non-blocking) when a newer release exists",
    ],
    usage: "warden update [--check] [--version <version>] [--yes]",
    flags: [
      { name: "--check", desc: "Report current vs latest without installing" },
      { name: "--version <version>", desc: "Install a specific published version" },
      { name: "--yes", desc: "Skip confirmation (required in non-TTY/CI)" },
    ],
    example: "warden update --check\nwarden update --yes\nwarden update --version 0.1.15 --yes",
    image: {
      src: "/diagrams/install-cli.jpeg",
      alt: "Installing and updating the Warden CLI",
    },
  },
];

export default function FeaturesPage() {
  const featuresSchema = {
    "@context": "https://schema.org",
    "@type": "ItemList",
    name: "Warden Features",
    itemListElement: FEATURES.map((f, i) => ({
      "@type": "ListItem",
      position: i + 1,
      name: f.name,
      description: f.tagline,
    })),
  };

  return (
    <PageShell>
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{ __html: JSON.stringify(featuresSchema) }}
      />
      <JsonLd
        data={breadcrumbSchema([
          { name: "Home", url: `${SITE_URL}/` },
          { name: "Features", url: `${SITE_URL}/features` },
        ])}
      />

      <PageHeader eyebrow="Feature Reference" title="Every Warden feature, explained in depth.">
        <p className="mt-4 max-w-2xl font-sans text-[14.5px] leading-[1.75] text-offwhite/45">
          Warden is a local sandbox runtime for MCP servers. Nine commands cover
          that surface, from sandboxing a single server to wrapping a whole MCP
          client config. Everything below ships today and is backed by the test
          suite. For the full policy schema, see{" "}
          <Link
            href="/docs/schema"
            className="text-acid transition-colors hover:text-acid-bright"
          >
            the policy schema docs
          </Link>
          .
        </p>
      </PageHeader>

      <FeatureNav />

      <div>
        {FEATURES.map((f, idx) => (
          <section
            key={f.id}
            id={f.id}
            className="scroll-mt-32 border-b border-hair px-6 py-14 last:border-b-0"
          >
            <div className="flex flex-wrap items-center gap-3">
              <span className="font-mono text-[11px] text-offwhite/35">
                {String(idx + 1).padStart(2, "0")}
              </span>
              <h2 className="font-mono text-[1.35rem] font-semibold tracking-[-0.01em] text-offwhite">
                {f.name}
              </h2>
              <span className="border border-acid/30 bg-acid/10 px-2.5 py-0.5 font-mono text-[10px] uppercase tracking-micro text-acid">
                {f.badge}
              </span>
            </div>
            <p className="mt-2 font-sans text-[14px] text-offwhite/50">{f.tagline}</p>

            <div className="mt-8 grid gap-10 lg:grid-cols-[1fr_22rem]">
              <div className="min-w-0">
                <p className="max-w-2xl font-sans text-[14px] leading-[1.75] text-offwhite/45">
                  {f.body}
                </p>

                {f.bullets.length > 0 && (
                  <ul className="mt-5 space-y-2">
                    {f.bullets.map((b) => (
                      <li
                        key={b}
                        className="flex gap-2.5 font-sans text-[13.5px] leading-[1.7] text-offwhite/45"
                      >
                        <svg
                          width="14"
                          height="14"
                          viewBox="0 0 16 16"
                          fill="none"
                          className="mt-1 shrink-0 text-acid"
                        >
                          <path
                            d="M3.5 8.5l3 3 6-7"
                            stroke="currentColor"
                            strokeWidth="1.8"
                            strokeLinecap="round"
                            strokeLinejoin="round"
                          />
                        </svg>
                        <span>{b}</span>
                      </li>
                    ))}
                  </ul>
                )}

                <div className="mt-7">
                  <span className="font-mono text-[9.5px] uppercase tracking-micro text-offwhite/35">
                    Usage
                  </span>
                  <div className="mt-2">
                    <CodeBlock>{f.usage}</CodeBlock>
                  </div>
                </div>

                {f.flags.length > 0 && (
                  <div className="mt-6">
                    <span className="font-mono text-[9.5px] uppercase tracking-micro text-offwhite/35">
                      Flags
                    </span>
                    <div className="mt-2">
                      {f.flags.map((fl) => (
                        <Flag key={fl.name} name={fl.name} desc={fl.desc} />
                      ))}
                    </div>
                  </div>
                )}

                <div className="mt-6">
                  <span className="font-mono text-[9.5px] uppercase tracking-micro text-offwhite/35">
                    Examples
                  </span>
                  <div className="mt-2">
                    <CodeBlock>{f.example}</CodeBlock>
                  </div>
                </div>

                {f.note && (
                  <div className="mt-6">
                    <Note tone={f.note.tone} title={f.note.title}>
                      {f.note.text}
                    </Note>
                  </div>
                )}
              </div>

              <div className="min-w-0">
                <div className="overflow-hidden border border-hair lg:sticky lg:top-32">
                  <Diagram src={f.image.src} alt={f.image.alt} />
                </div>
              </div>
            </div>
          </section>
        ))}
      </div>

      <div className="px-6 pb-24 pt-14">
        <div className="border border-hair bg-cell p-8 text-center">
          <h2 className="font-mono text-[1.35rem] font-semibold text-offwhite">
            Ready to sandbox your first server?
          </h2>
          <p className="mx-auto mt-2 max-w-md font-sans text-[13.5px] leading-[1.7] text-offwhite/45">
            Install the CLI, trace once, and run under a policy, in about two
            minutes.
          </p>
          <div className="mt-6 flex flex-wrap items-center justify-center gap-3">
            <Link
              href="/docs/install"
              className="rounded-md bg-acid px-5 py-2.5 font-sans text-[13px] font-semibold text-void transition-colors hover:bg-acid-bright"
            >
              Install Warden
            </Link>
            <Link
              href="/docs/quickstart"
              className="rounded-md border border-hair bg-void px-5 py-2.5 font-sans text-[13px] font-semibold text-offwhite/85 transition-colors hover:border-offwhite/20"
            >
              Read the quickstart
            </Link>
          </div>
        </div>
      </div>
    </PageShell>
  );
}
