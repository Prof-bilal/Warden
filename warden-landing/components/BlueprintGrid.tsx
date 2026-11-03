import {
  Bot,
  Boxes,
  Check,
  Code2,
  Cpu,
  Folder,
  Gauge,
  Globe,
  KeyRound,
  ScrollText,
  ShieldCheck,
  Terminal,
} from "lucide-react";
import Panel from "@/components/bp/Panel";
import CircuitStack from "@/components/bp/CircuitStack";
import IsometricCube from "@/components/bp/IsometricCube";
import NodeGraph from "@/components/bp/NodeGraph";
import CodePanel from "@/components/bp/CodePanel";
import LogPanel from "@/components/bp/LogPanel";
import CopyChip from "@/components/bp/CopyChip";
import { RevealGroup, RevealItem } from "@/components/bp/Reveal";

const inlineFeatures = [
  { icon: ShieldCheck, title: "Deny by default", body: "Fail closed" },
  { icon: ScrollText, title: "Fully auditable", body: "See every action" },
  { icon: Code2, title: "Open source", body: "MIT licensed" },
];

const sandboxAxes = ["Filesystem", "Network", "Environment", "Runtime limits"];

const POLICY = `# warden.yml
version: 1
command: ["npx", "your-mcp-server"]
filesystem:
  read:
    - ./project
    - ./tmp
  write:
    - ./output
network:
  allow:
    - api.github.com
    - registry.npmjs.org
  deny:
    - "*"
env:
  allow:
    - NODE_ENV
    - OPENAI_API_KEY
limits:
  memory_mb: 512
  timeout_s: 300`;

const logRows = [
  { time: "10:34:01", status: "INFO" as const, text: 'Starting sandbox for task "analyze-repo"...' },
  { time: "10:34:01", status: "INFO" as const, text: "Loading policy from warden.yml" },
  { time: "10:34:01", status: "ALLOW" as const, text: "Read file /project/package.json" },
  { time: "10:34:01", status: "ALLOW" as const, text: "Read file /project/src/*.ts" },
  { time: "10:34:01", status: "ALLOW" as const, text: "Network request https://api.github.com" },
  { time: "10:34:02", status: "ALLOW" as const, text: "Network request https://registry.npmjs.org" },
  { time: "10:34:02", status: "ALLOW" as const, text: "Environment variable OPENAI_API_KEY" },
  { time: "10:34:02", status: "DENY" as const, text: "Write file /etc/passwd" },
  { time: "10:34:02", status: "DENY" as const, text: "Network request https://malicious.example.com" },
  { time: "10:34:02", status: "DENY" as const, text: "Environment variable AWS_SECRET_KEY" },
];

const permissionLeft = [
  { icon: Folder, label: "Filesystem" },
  { icon: Globe, label: "Network" },
  { icon: KeyRound, label: "Environment" },
  { icon: Gauge, label: "Runtime limits" },
];

const permissionRight = [
  { icon: Bot, label: "AI tools" },
  { icon: Boxes, label: "MCP servers" },
  { icon: Cpu, label: "Custom agents" },
  { icon: Terminal, label: "Any runtime" },
];

export function PanelRowA() {
  return (
    <RevealGroup className="grid gap-px border-b border-hair bg-hair md:grid-cols-2 lg:grid-cols-[1fr_1.6fr_1fr_1.1fr]">
      <RevealItem className="bg-cell">
        <Panel label="AI tools">
          <CircuitStack className="mx-auto" />
        </Panel>
      </RevealItem>

      <RevealItem className="bg-cell">
        <div className="flex h-full flex-col justify-center gap-6 p-6 sm:p-8">
          <h2 className="font-mono text-[1.15rem] font-semibold leading-[1.3] tracking-[-0.01em] text-offwhite sm:text-[1.4rem]">
            Warden is an open-source permission layer and sandbox for AI tools and
            MCP servers.
          </h2>
          <p className="max-w-[36rem] font-sans text-[13px] leading-[1.75] text-offwhite/45">
            Control what AI tools can access — filesystem, network, environment
            variables, and runtime limits. Deny by default, fail closed, and keep a
            full audit trail.
          </p>
          <div className="flex flex-wrap gap-x-8 gap-y-3">
            {inlineFeatures.map(({ icon: Icon, title, body }) => (
              <div key={title} className="flex items-center gap-2.5">
                <Icon size={15} className="shrink-0 text-acid" aria-hidden />
                <div>
                  <p className="font-mono text-[11px] font-medium leading-tight text-offwhite/85">
                    {title}
                  </p>
                  <p className="font-mono text-[9px] uppercase tracking-micro text-offwhite/35">
                    {body}
                  </p>
                </div>
              </div>
            ))}
          </div>
        </div>
      </RevealItem>

      <RevealItem className="bg-cell">
        <Panel label="Install">
          <CopyChip value="npm i -g warden-sandbox-cli" />
          <p className="mt-3 font-mono text-[9.5px] uppercase tracking-micro text-offwhite/30">
            Deny by default · fail closed
          </p>
        </Panel>
      </RevealItem>

      <RevealItem className="bg-cell">
        <Panel label="Sandbox">
          <IsometricCube className="mx-auto h-[132px] w-full max-w-[210px]" />
          <ul className="mt-4 space-y-1.5">
            {sandboxAxes.map((axis) => (
              <li key={axis} className="flex items-center gap-2 font-mono text-[10px] text-offwhite/60">
                <span className="h-1 w-1 rounded-full bg-acid" aria-hidden />
                <span className="uppercase tracking-micro">{axis}</span>
              </li>
            ))}
          </ul>
        </Panel>
      </RevealItem>
    </RevealGroup>
  );
}

export function PanelRowB() {
  return (
    <RevealGroup className="grid gap-px border-b border-hair bg-hair lg:grid-cols-3">
      <RevealItem className="bg-void">
        <CodePanel label="Policy" filename="warden.yml" code={POLICY} />
      </RevealItem>

      <RevealItem className="bg-void">
        <LogPanel
          title="Execution"
          live="Running in sandbox..."
          rows={logRows}
          footer={
            <>
              <Check size={11} aria-hidden />
              Task completed · 7 actions · 5 allowed · 2 blocked
            </>
          }
        />
      </RevealItem>

      <RevealItem className="bg-cell">
        <Panel
          label="Permission layer"
          meta={
            <span className="font-mono text-[9px] uppercase tracking-micro text-offwhite/25">
              Isolated execution
            </span>
          }
        >
          <NodeGraph left={permissionLeft} right={permissionRight} className="py-2" />
        </Panel>
      </RevealItem>
    </RevealGroup>
  );
}
