export function ShieldFailClosed() {
  return (
    <svg viewBox="0 0 240 180" fill="none" className="h-full w-full max-h-[280px]">
      <rect x="80" y="30" width="80" height="120" rx="8" fill="#1E242E" stroke="#6E93E8" strokeWidth="1.5" />
      <path d="M120 50 L140 60 V85 C140 105 120 115 120 115 C120 115 100 105 100 85 V60 Z" fill="none" stroke="#E2604F" strokeWidth="1.5" />
      <line x1="108" y1="75" x2="132" y2="95" stroke="#E2604F" strokeWidth="1.5" />
      <line x1="132" y1="75" x2="108" y2="95" stroke="#E2604F" strokeWidth="1.5" />
      <text x="120" y="145" textAnchor="middle" fontSize="10" fill="#8D95A5" fontFamily="var(--font-geist-mono)">FAIL-CLOSED</text>
    </svg>
  );
}

export function DenyFilesystem() {
  return (
    <svg viewBox="0 0 240 180" fill="none" className="h-full w-full max-h-[280px]">
      <rect x="30" y="40" width="70" height="100" rx="4" fill="#1E242E" stroke="#3A4150" strokeWidth="1.2" />
      <text x="65" y="75" textAnchor="middle" fontSize="9" fill="#8D95A5" fontFamily="var(--font-geist-mono)">~/</text>
      <rect x="45" y="85" width="40" height="20" rx="2" fill="#1E242E" stroke="#3FB27E" strokeWidth="1.2" />
      <text x="65" y="99" textAnchor="middle" fontSize="8" fill="#3FB27E" fontFamily="var(--font-geist-mono)">./data</text>
      <line x1="45" y1="115" x2="85" y2="115" stroke="#E2604F" strokeWidth="1.2" opacity="0.5" />
      <g transform="translate(130,70)">
        <rect x="0" y="0" width="80" height="50" rx="4" fill="#1E242E" stroke="#6E93E8" strokeWidth="1.2" strokeDasharray="3 2" />
        <text x="40" y="20" textAnchor="middle" fontSize="9" fill="#E8EBEF" fontFamily="var(--font-geist-mono)">sandbox</text>
        <text x="40" y="35" textAnchor="middle" fontSize="8" fill="#3FB27E" fontFamily="var(--font-geist-mono)">granted</text>
      </g>
      <line x1="100" y1="90" x2="130" y2="85" stroke="#3FB27E" strokeWidth="1.2" />
      <text x="120" y="145" textAnchor="middle" fontSize="10" fill="#8D95A5" fontFamily="var(--font-geist-mono)">DENY-BY-DEFAULT</text>
    </svg>
  );
}

export function NetworkEnforced() {
  return (
    <svg viewBox="0 0 240 180" fill="none" className="h-full w-full max-h-[280px]">
      <circle cx="120" cy="90" r="45" fill="none" stroke="#6E93E8" strokeWidth="1.2" strokeDasharray="4 3" />
      <text x="120" y="85" textAnchor="middle" fontSize="9" fill="#E8EBEF" fontFamily="var(--font-geist-mono)">proxy</text>
      <text x="120" y="100" textAnchor="middle" fontSize="8" fill="#3FB27E" fontFamily="var(--font-geist-mono)">egress</text>
      <line x1="165" y1="90" x2="200" y2="60" stroke="#3FB27E" strokeWidth="1.2" />
      <circle cx="208" cy="55" r="8" fill="#1E242E" stroke="#3FB27E" strokeWidth="1.2" />
      <text x="208" y="58" textAnchor="middle" fontSize="7" fill="#3FB27E" fontFamily="var(--font-geist-mono)">dns</text>
      <line x1="75" y1="90" x2="40" y2="60" stroke="#E2604F" strokeWidth="1.2" opacity="0.6" />
      <g transform="translate(25,50)">
        <line x1="-3" y1="-3" x2="3" y2="3" stroke="#E2604F" strokeWidth="1.2" />
        <line x1="-3" y1="3" x2="3" y2="-3" stroke="#E2604F" strokeWidth="1.2" />
      </g>
      <text x="30" y="45" fontSize="8" fill="#8D95A5" fontFamily="var(--font-geist-mono)">evil.com</text>
      <line x1="75" y1="100" x2="40" y2="130" stroke="#E2604F" strokeWidth="1.2" opacity="0.6" />
      <g transform="translate(25,128)">
        <line x1="-3" y1="-3" x2="3" y2="3" stroke="#E2604F" strokeWidth="1.2" />
        <line x1="-3" y1="3" x2="3" y2="-3" stroke="#E2604F" strokeWidth="1.2" />
      </g>
      <text x="30" y="145" fontSize="8" fill="#8D95A5" fontFamily="var(--font-geist-mono)">malware.io</text>
      <text x="120" y="155" textAnchor="middle" fontSize="10" fill="#8D95A5" fontFamily="var(--font-geist-mono)">KERNEL-ENFORCED</text>
    </svg>
  );
}

export function EnvFilter() {
  return (
    <svg viewBox="0 0 240 180" fill="none" className="h-full w-full max-h-[280px]">
      <rect x="30" y="35" width="70" height="110" rx="4" fill="#1E242E" stroke="#3A4150" strokeWidth="1.2" />
      <text x="65" y="55" textAnchor="middle" fontSize="9" fill="#E8EBEF" fontFamily="var(--font-geist-mono)">shell env</text>
      {["PATH", "HOME", "SECRET", "TOKEN", "USER", "SHELL"].map((v, i) => (
        <text key={v} x="40" y={72 + i * 12} fontSize="8" fill={i < 2 ? "#3FB27E" : "#E2604F"} fontFamily="var(--font-geist-mono)" opacity={i < 2 ? 1 : 0.5}>{v}</text>
      ))}
      <line x1="100" y1="90" x2="140" y2="90" stroke="#6E93E8" strokeWidth="1.2" strokeDasharray="3 2" />
      <text x="120" y="82" textAnchor="middle" fontSize="8" fill="#6E93E8" fontFamily="var(--font-geist-mono)">filter</text>
      <rect x="145" y="35" width="70" height="110" rx="4" fill="#1E242E" stroke="#3FB27E" strokeWidth="1.2" />
      <text x="180" y="55" textAnchor="middle" fontSize="9" fill="#E8EBEF" fontFamily="var(--font-geist-mono)">policy env</text>
      {["GITHUB_TOKEN"].map((v, i) => (
        <text key={v} x="155" y={72 + i * 12} fontSize="8" fill="#3FB27E" fontFamily="var(--font-geist-mono)">{v}</text>
      ))}
      <text x="180" y="100" textAnchor="middle" fontSize="8" fill="#3FB27E" fontFamily="var(--font-geist-mono)">allowed</text>
      <text x="120" y="160" textAnchor="middle" fontSize="10" fill="#8D95A5" fontFamily="var(--font-geist-mono)">ENV FILTERING</text>
    </svg>
  );
}

export function PlatformFailClosed() {
  return (
    <svg viewBox="0 0 240 180" fill="none" className="h-full w-full max-h-[280px]">
      {[
        { x: 40, label: "Linux", sub: "bwrap", color: "#3FB27E" },
        { x: 105, label: "macOS", sub: "seatbelt", color: "#6E93E8" },
        { x: 170, label: "Windows", sub: "appcon", color: "#E2604F" },
      ].map((p) => (
        <g key={p.label}>
          <rect x={p.x} y="40" width="55" height="80" rx="4" fill="#1E242E" stroke={p.color} strokeWidth="1.2" />
          <text x={p.x + 27} y="65" textAnchor="middle" fontSize="9" fill={p.color} fontFamily="var(--font-geist-mono)">{p.label}</text>
          <text x={p.x + 27} y="80" textAnchor="middle" fontSize="7" fill="#8D95A5" fontFamily="var(--font-geist-mono)">{p.sub}</text>
          <line x1={p.x + 10} y1="95" x2={p.x + 45} y2="95" stroke={p.color} strokeWidth="0.8" opacity="0.4" />
          <text x={p.x + 27} y="110" textAnchor="middle" fontSize="7" fill="#3FB27E" fontFamily="var(--font-geist-mono)">verified</text>
        </g>
      ))}
      <text x="120" y="150" textAnchor="middle" fontSize="10" fill="#8D95A5" fontFamily="var(--font-geist-mono)">THREE PLATFORMS, ONE BOUNDARY</text>
    </svg>
  );
}

export function ResourceLimits() {
  return (
    <svg viewBox="0 0 240 180" fill="none" className="h-full w-full max-h-[280px]">
      <rect x="30" y="35" width="80" height="110" rx="4" fill="#1E242E" stroke="#6E93E8" strokeWidth="1.2" />
      <text x="70" y="55" textAnchor="middle" fontSize="9" fill="#E8EBEF" fontFamily="var(--font-geist-mono)">timeout</text>
      <rect x="42" y="65" width="56" height="60" rx="2" fill="#0d1017" stroke="#3A4150" strokeWidth="0.8" />
      <rect x="42" y="100" width="56" height="25" rx="2" fill="#6E93E8" opacity="0.3" />
      <text x="70" y="118" textAnchor="middle" fontSize="8" fill="#6E93E8" fontFamily="var(--font-geist-mono)">300s</text>
      <rect x="130" y="35" width="80" height="110" rx="4" fill="#1E242E" stroke="#E2604F" strokeWidth="1.2" />
      <text x="170" y="55" textAnchor="middle" fontSize="9" fill="#E8EBEF" fontFamily="var(--font-geist-mono)">memory</text>
      <rect x="142" y="65" width="56" height="60" rx="2" fill="#0d1017" stroke="#3A4150" strokeWidth="0.8" />
      <rect x="142" y="85" width="56" height="40" rx="2" fill="#E2604F" opacity="0.3" />
      <text x="170" y="110" textAnchor="middle" fontSize="8" fill="#E2604F" fontFamily="var(--font-geist-mono)">512MB</text>
      <text x="120" y="160" textAnchor="middle" fontSize="10" fill="#8D95A5" fontFamily="var(--font-geist-mono)">KERNEL-ENFORCED LIMITS</text>
    </svg>
  );
}

export function CapFilesystem() {
  return (
    <svg viewBox="0 0 240 180" fill="none" className="h-full w-full max-h-[280px]">
      <rect x="50" y="30" width="140" height="120" rx="6" fill="#1E242E" stroke="#6E93E8" strokeWidth="1.2" />
      <text x="120" y="50" textAnchor="middle" fontSize="9" fill="#8D95A5" fontFamily="var(--font-geist-mono)">filesystem</text>
      {["./data", "./output", "./config"].map((p, i) => (
        <g key={p}>
          <rect x="65" y={60 + i * 28} width="110" height="20" rx="3" fill="#0d1017" stroke={i < 2 ? "#3FB27E" : "#E2604F"} strokeWidth="1" />
          <text x="120" y={74 + i * 28} textAnchor="middle" fontSize="8" fill={i < 2 ? "#3FB27E" : "#E2604F"} fontFamily="var(--font-geist-mono)">{i < 2 ? "read/write" : "invisible"}</text>
          <text x="75" y={74 + i * 28} fontSize="8" fill="#E8EBEF" fontFamily="var(--font-geist-mono)">{p}</text>
        </g>
      ))}
      <text x="120" y="160" textAnchor="middle" fontSize="10" fill="#8D95A5" fontFamily="var(--font-geist-mono)">GRANTS, NOT PROMISES</text>
    </svg>
  );
}

export function CapNetwork() {
  return (
    <svg viewBox="0 0 240 180" fill="none" className="h-full w-full max-h-[280px]">
      <circle cx="120" cy="85" r="50" fill="none" stroke="#6E93E8" strokeWidth="1.2" strokeDasharray="4 3" />
      <text x="120" y="82" textAnchor="middle" fontSize="9" fill="#E8EBEF" fontFamily="var(--font-geist-mono)">proxy</text>
      <text x="120" y="95" textAnchor="middle" fontSize="8" fill="#6E93E8" fontFamily="var(--font-geist-mono)">dns+tcp</text>
      {[
        { x: 195, y: 55, label: "api.github", ok: true },
        { x: 195, y: 115, label: "evil.com", ok: false },
        { x: 45, y: 55, label: "npmjs.org", ok: true },
        { x: 45, y: 115, label: "malware.io", ok: false },
      ].map((h) => (
        <g key={h.label}>
          <circle cx={h.x} cy={h.y} r="5" fill={h.ok ? "#3FB27E" : "#E2604F"} opacity="0.8" />
          <text x={h.x} y={h.y - 10} textAnchor="middle" fontSize="7" fill="#8D95A5" fontFamily="var(--font-geist-mono)">{h.label}</text>
        </g>
      ))}
      <text x="120" y="155" textAnchor="middle" fontSize="10" fill="#8D95A5" fontFamily="var(--font-geist-mono)">HOSTNAME ALLOWLIST</text>
    </svg>
  );
}

export function CapEnv() {
  return (
    <svg viewBox="0 0 240 180" fill="none" className="h-full w-full max-h-[280px]">
      <rect x="40" y="30" width="65" height="120" rx="4" fill="#1E242E" stroke="#3A4150" strokeWidth="1.2" />
      <text x="72" y="50" textAnchor="middle" fontSize="8" fill="#8D95A5" fontFamily="var(--font-geist-mono)">process env</text>
      {["PATH", "HOME", "SECRET", "TOKEN", "USER"].map((v, i) => (
        <text key={v} x="50" y={68 + i * 13} fontSize="8" fill="#E8EBEF" fontFamily="var(--font-geist-mono)">{v}</text>
      ))}
      <line x1="105" y1="90" x2="135" y2="90" stroke="#6E93E8" strokeWidth="1.2" strokeDasharray="3 2" />
      <text x="120" y="82" textAnchor="middle" fontSize="7" fill="#6E93E8" fontFamily="var(--font-geist-mono)">filter</text>
      <rect x="140" y="30" width="65" height="120" rx="4" fill="#1E242E" stroke="#3FB27E" strokeWidth="1.2" />
      <text x="172" y="50" textAnchor="middle" fontSize="8" fill="#8D95A5" fontFamily="var(--font-geist-mono)">granted</text>
      {["GITHUB_TOKEN"].map((v, i) => (
        <text key={v} x="150" y={68 + i * 13} fontSize="8" fill="#3FB27E" fontFamily="var(--font-geist-mono)">{v}</text>
      ))}
      <text x="172" y="100" textAnchor="middle" fontSize="8" fill="#3FB27E" fontFamily="var(--font-geist-mono)">empty = none</text>
      <text x="120" y="160" textAnchor="middle" fontSize="10" fill="#8D95A5" fontFamily="var(--font-geist-mono)">NAMED VARIABLES ONLY</text>
    </svg>
  );
}

export function CapLimits() {
  return (
    <svg viewBox="0 0 240 180" fill="none" className="h-full w-full max-h-[280px]">
      <rect x="35" y="35" width="75" height="100" rx="4" fill="#1E242E" stroke="#6E93E8" strokeWidth="1.2" />
      <text x="72" y="55" textAnchor="middle" fontSize="8" fill="#E8EBEF" fontFamily="var(--font-geist-mono)">wall clock</text>
      <rect x="47" y="65" width="51" height="55" rx="2" fill="#0d1017" stroke="#3A4150" strokeWidth="0.8" />
      <rect x="47" y="95" width="51" height="25" rx="2" fill="#6E93E8" opacity="0.3" />
      <text x="72" y="112" textAnchor="middle" fontSize="8" fill="#6E93E8" fontFamily="var(--font-geist-mono)">killed</text>
      <rect x="130" y="35" width="75" height="100" rx="4" fill="#1E242E" stroke="#E2604F" strokeWidth="1.2" />
      <text x="167" y="55" textAnchor="middle" fontSize="8" fill="#E8EBEF" fontFamily="var(--font-geist-mono)">memory</text>
      <rect x="142" y="65" width="51" height="55" rx="2" fill="#0d1017" stroke="#3A4150" strokeWidth="0.8" />
      <rect x="142" y="80" width="51" height="40" rx="2" fill="#E2604F" opacity="0.3" />
      <text x="167" y="105" textAnchor="middle" fontSize="8" fill="#E2604F" fontFamily="var(--font-geist-mono)">RSS cap</text>
      <text x="120" y="155" textAnchor="middle" fontSize="10" fill="#8D95A5" fontFamily="var(--font-geist-mono)">PROCESS-TREE KILL</text>
    </svg>
  );
}

export function CapAudit() {
  return (
    <svg viewBox="0 0 240 180" fill="none" className="h-full w-full max-h-[280px]">
      <rect x="30" y="30" width="180" height="120" rx="6" fill="#1E242E" stroke="#6E93E8" strokeWidth="1.2" />
      <text x="120" y="48" textAnchor="middle" fontSize="9" fill="#8D95A5" fontFamily="var(--font-geist-mono)">audit.jsonl</text>
      {[
        { y: 62, action: "read", target: "./data/report.json", granted: true },
        { y: 80, action: "read", target: "~/.ssh/id_rsa", granted: false },
        { y: 98, action: "connect", target: "api.github.com", granted: true },
        { y: 116, action: "connect", target: "evil.com", granted: false },
      ].map((e) => (
        <g key={e.y}>
          <text x="42" y={e.y + 11} fontSize="8" fill="#8D95A5" fontFamily="var(--font-geist-mono)">{e.action}</text>
          <text x="82" y={e.y + 11} fontSize="8" fill="#E8EBEF" fontFamily="var(--font-geist-mono)">{e.target}</text>
          <text x="200" y={e.y + 11} textAnchor="end" fontSize="8" fill={e.granted ? "#3FB27E" : "#E2604F"} fontFamily="var(--font-geist-mono)">{e.granted ? "allow" : "deny"}</text>
        </g>
      ))}
      <text x="120" y="160" textAnchor="middle" fontSize="10" fill="#8D95A5" fontFamily="var(--font-geist-mono)">EVERY DECISION LOGGED</text>
    </svg>
  );
}

export function CapPolicy() {
  return (
    <svg viewBox="0 0 240 180" fill="none" className="h-full w-full max-h-[280px]">
      <rect x="30" y="30" width="75" height="120" rx="4" fill="#1E242E" stroke="#6E93E8" strokeWidth="1.2" />
      <text x="67" y="50" textAnchor="middle" fontSize="8" fill="#E8EBEF" fontFamily="var(--font-geist-mono)">warden init</text>
      <text x="42" y="70" fontSize="7" fill="#3FB27E" fontFamily="var(--font-geist-mono)">→ policy.yaml</text>
      <rect x="135" y="30" width="75" height="120" rx="4" fill="#1E242E" stroke="#3FB27E" strokeWidth="1.2" />
      <text x="172" y="50" textAnchor="middle" fontSize="8" fill="#E8EBEF" fontFamily="var(--font-geist-mono)">warden doctor</text>
      <text x="147" y="70" fontSize="7" fill="#3FB27E" fontFamily="var(--font-geist-mono)">→ verified</text>
      <line x1="105" y1="90" x2="135" y2="90" stroke="#6E93E8" strokeWidth="1" strokeDasharray="3 2" />
      <text x="120" y="84" textAnchor="middle" fontSize="7" fill="#6E93E8" fontFamily="var(--font-geist-mono)">trace</text>
      <text x="120" y="160" textAnchor="middle" fontSize="10" fill="#8D95A5" fontFamily="var(--font-geist-mono)">INIT → TRACE → DOCTOR</text>
    </svg>
  );
}

export function CapProxy() {
  return (
    <svg viewBox="0 0 240 180" fill="none" className="h-full w-full max-h-[280px]">
      <rect x="30" y="55" width="60" height="70" rx="4" fill="#1E242E" stroke="#6E93E8" strokeWidth="1.2" />
      <text x="60" y="80" textAnchor="middle" fontSize="8" fill="#E8EBEF" fontFamily="var(--font-geist-mono)">MCP</text>
      <text x="60" y="95" textAnchor="middle" fontSize="7" fill="#6E93E8" fontFamily="var(--font-geist-mono)">client</text>
      <line x1="90" y1="90" x2="120" y2="90" stroke="#6E93E8" strokeWidth="1.2" />
      <rect x="120" y="55" width="60" height="70" rx="4" fill="#1E242E" stroke="#3FB27E" strokeWidth="1.2" />
      <text x="150" y="75" textAnchor="middle" fontSize="8" fill="#E8EBEF" fontFamily="var(--font-geist-mono)">proxy</text>
      <text x="150" y="90" textAnchor="middle" fontSize="7" fill="#3FB27E" fontFamily="var(--font-geist-mono)">filter</text>
      <text x="150" y="105" textAnchor="middle" fontSize="7" fill="#8D95A5" fontFamily="var(--font-geist-mono)">json-rpc</text>
      <line x1="180" y1="90" x2="210" y2="90" stroke="#3FB27E" strokeWidth="1.2" />
      <rect x="210" y="55" width="20" height="70" rx="4" fill="#1E242E" stroke="#3A4150" strokeWidth="1.2" />
      <text x="220" y="93" textAnchor="middle" fontSize="7" fill="#8D95A5" fontFamily="var(--font-geist-mono)">→</text>
      <text x="120" y="155" textAnchor="middle" fontSize="10" fill="#8D95A5" fontFamily="var(--font-geist-mono)">CLIENT PROXY</text>
    </svg>
  );
}

export function CapCI() {
  return (
    <svg viewBox="0 0 240 180" fill="none" className="h-full w-full max-h-[280px]">
      <rect x="30" y="35" width="80" height="90" rx="4" fill="#1E242E" stroke="#6E93E8" strokeWidth="1.2" />
      <text x="70" y="55" textAnchor="middle" fontSize="8" fill="#E8EBEF" fontFamily="var(--font-geist-mono)">GitHub Action</text>
      <text x="42" y="75" fontSize="7" fill="#3FB27E" fontFamily="var(--font-geist-mono)">warden run --policy</text>
      <text x="42" y="90" fontSize="7" fill="#3FB27E" fontFamily="var(--font-geist-mono)">  .warden/policy.yaml</text>
      <text x="42" y="108" fontSize="7" fill="#8D95A5" fontFamily="var(--font-geist-mono)">-- ci-job</text>
      <line x1="110" y1="80" x2="140" y2="80" stroke="#6E93E8" strokeWidth="1" strokeDasharray="3 2" />
      <rect x="140" y="35" width="75" height="90" rx="4" fill="#1E242E" stroke="#E2604F" strokeWidth="1.2" />
      <text x="177" y="55" textAnchor="middle" fontSize="8" fill="#E8EBEF" fontFamily="var(--font-geist-mono)">K8s Render</text>
      <text x="152" y="75" fontSize="7" fill="#3FB27E" fontFamily="var(--font-geist-mono)">→ Deployment</text>
      <text x="152" y="90" fontSize="7" fill="#3FB27E" fontFamily="var(--font-geist-mono)">→ NetPolicy</text>
      <text x="152" y="108" fontSize="7" fill="#8D95A5" fontFamily="var(--font-geist-mono)">hardened</text>
      <text x="120" y="150" textAnchor="middle" fontSize="10" fill="#8D95A5" fontFamily="var(--font-geist-mono)">CI + DEPLOY</text>
    </svg>
  );
}

export function WinAppContainer() {
  return (
    <svg viewBox="0 0 240 180" fill="none" className="h-full w-full max-h-[280px]">
      <rect x="50" y="25" width="140" height="130" rx="6" fill="#1E242E" stroke="#6E93E8" strokeWidth="1.2" />
      <text x="120" y="45" textAnchor="middle" fontSize="9" fill="#E8EBEF" fontFamily="var(--font-geist-mono)">AppContainer Token</text>
      <rect x="65" y="55" width="110" height="80" rx="4" fill="#0d1017" stroke="#3A4150" strokeWidth="0.8" />
      <text x="120" y="80" textAnchor="middle" fontSize="8" fill="#8D95A5" fontFamily="var(--font-geist-mono)">LowBox token</text>
      <text x="120" y="95" textAnchor="middle" fontSize="8" fill="#3FB27E" fontFamily="var(--font-geist-mono)">denies all by default</text>
      <text x="120" y="115" textAnchor="middle" fontSize="8" fill="#8D95A5" fontFamily="var(--font-geist-mono)">no syscall crosses it</text>
      <text x="120" y="155" textAnchor="middle" fontSize="10" fill="#8D95A5" fontFamily="var(--font-geist-mono)">HARD BOUNDARY</text>
    </svg>
  );
}

export function WinWFP() {
  return (
    <svg viewBox="0 0 240 180" fill="none" className="h-full w-full max-h-[280px]">
      <rect x="30" y="40" width="180" height="100" rx="6" fill="#1E242E" stroke="#6E93E8" strokeWidth="1.2" />
      <text x="120" y="58" textAnchor="middle" fontSize="9" fill="#E8EBEF" fontFamily="var(--font-geist-mono)">WFP Egress Filters</text>
      {[
        { x: 55, label: "loopback", ok: true },
        { x: 120, label: "tcp block", ok: false },
        { x: 185, label: "dns deny", ok: false },
      ].map((f) => (
        <g key={f.label}>
          <rect x={f.x - 25} y="70" width="50" height="25" rx="3" fill="#0d1017" stroke={f.ok ? "#3FB27E" : "#E2604F"} strokeWidth="1" />
          <text x={f.x} y="86" textAnchor="middle" fontSize="7" fill={f.ok ? "#3FB27E" : "#E2604F"} fontFamily="var(--font-geist-mono)">{f.label}</text>
        </g>
      ))}
      <text x="120" y="120" textAnchor="middle" fontSize="8" fill="#8D95A5" fontFamily="var(--font-geist-mono)">only proxy bridge allowed</text>
      <text x="120" y="155" textAnchor="middle" fontSize="10" fill="#8D95A5" fontFamily="var(--font-geist-mono)">NETWORK FILTER</text>
    </svg>
  );
}

export function WinJobObject() {
  return (
    <svg viewBox="0 0 240 180" fill="none" className="h-full w-full max-h-[280px]">
      <rect x="35" y="35" width="75" height="100" rx="4" fill="#1E242E" stroke="#6E93E8" strokeWidth="1.2" />
      <text x="72" y="55" textAnchor="middle" fontSize="8" fill="#E8EBEF" fontFamily="var(--font-geist-mono)">timeout</text>
      <rect x="47" y="65" width="51" height="55" rx="2" fill="#0d1017" stroke="#3A4150" strokeWidth="0.8" />
      <rect x="47" y="95" width="51" height="25" rx="2" fill="#6E93E8" opacity="0.3" />
      <text x="72" y="112" textAnchor="middle" fontSize="8" fill="#6E93E8" fontFamily="var(--font-geist-mono)">SIGKILL</text>
      <rect x="130" y="35" width="75" height="100" rx="4" fill="#1E242E" stroke="#E2604F" strokeWidth="1.2" />
      <text x="167" y="55" textAnchor="middle" fontSize="8" fill="#E8EBEF" fontFamily="var(--font-geist-mono)">memory</text>
      <rect x="142" y="65" width="51" height="55" rx="2" fill="#0d1017" stroke="#3A4150" strokeWidth="0.8" />
      <rect x="142" y="80" width="51" height="40" rx="2" fill="#E2604F" opacity="0.3" />
      <text x="167" y="105" textAnchor="middle" fontSize="8" fill="#E2604F" fontFamily="var(--font-geist-mono)">tree kill</text>
      <text x="120" y="155" textAnchor="middle" fontSize="10" fill="#8D95A5" fontFamily="var(--font-geist-mono)">KILL-ON-CLOSE</text>
    </svg>
  );
}

export function WinETW() {
  return (
    <svg viewBox="0 0 240 180" fill="none" className="h-full w-full max-h-[280px]">
      <rect x="30" y="30" width="180" height="120" rx="6" fill="#1E242E" stroke="#6E93E8" strokeWidth="1.2" />
      <text x="120" y="48" textAnchor="middle" fontSize="9" fill="#E8EBEF" fontFamily="var(--font-geist-mono)">ETW Audit Trail</text>
      {[
        { y: 62, event: "FileCreate", target: "sandbox/file.txt", ok: true },
        { y: 80, event: "FileRead", target: "./data/report.json", ok: true },
        { y: 98, event: "NetConnect", target: "api.github.com", ok: true },
        { y: 116, event: "FileAccess", target: "~/.ssh/id_rsa", ok: false },
      ].map((e) => (
        <g key={e.y}>
          <text x="42" y={e.y + 11} fontSize="7" fill="#8D95A5" fontFamily="var(--font-geist-mono)">{e.event}</text>
          <text x="110" y={e.y + 11} fontSize="7" fill="#E8EBEF" fontFamily="var(--font-geist-mono)">{e.target}</text>
          <text x="200" y={e.y + 11} textAnchor="end" fontSize="7" fill={e.ok ? "#3FB27E" : "#E2604F"} fontFamily="var(--font-geist-mono)">{e.ok ? "logged" : "denied"}</text>
        </g>
      ))}
      <text x="120" y="155" textAnchor="middle" fontSize="10" fill="#8D95A5" fontFamily="var(--font-geist-mono)">KERNEL-LEVEL TRACE</text>
    </svg>
  );
}

export function HowInstall() {
  return (
    <svg viewBox="0 0 240 180" fill="none" className="h-full w-full max-h-[280px]">
      <rect x="30" y="25" width="180" height="130" rx="6" fill="#1E242E" stroke="#3A4150" strokeWidth="1.2" />
      <text x="42" y="48" fontSize="9" fill="#8D95A5" fontFamily="var(--font-geist-mono)">$</text>
      <text x="56" y="48" fontSize="9" fill="#3FB27E" fontFamily="var(--font-geist-mono)">npm install -g warden-sandbox-cli</text>
      <text x="42" y="70" fontSize="8" fill="#8D95A5" fontFamily="var(--font-geist-mono)">added 1 package in 2s</text>
      <text x="42" y="90" fontSize="8" fill="#8D95A5" fontFamily="var(--font-geist-mono)"></text>
      <text x="42" y="110" fontSize="9" fill="#8D95A5" fontFamily="var(--font-geist-mono)">$</text>
      <text x="56" y="110" fontSize="9" fill="#3FB27E" fontFamily="var(--font-geist-mono)">warden version</text>
      <text x="42" y="132" fontSize="8" fill="#E8EBEF" fontFamily="var(--font-geist-mono)">warden version v0.1.15</text>
      <text x="120" y="165" textAnchor="middle" fontSize="10" fill="#8D95A5" fontFamily="var(--font-geist-mono)">STEP 1: INSTALL</text>
    </svg>
  );
}

export function HowInit() {
  return (
    <svg viewBox="0 0 240 180" fill="none" className="h-full w-full max-h-[280px]">
      <rect x="30" y="25" width="180" height="130" rx="6" fill="#1E242E" stroke="#3A4150" strokeWidth="1.2" />
      <text x="42" y="48" fontSize="9" fill="#8D95A5" fontFamily="var(--font-geist-mono)">$</text>
      <text x="56" y="48" fontSize="9" fill="#3FB27E" fontFamily="var(--font-geist-mono)">warden init</text>
      <text x="42" y="70" fontSize="8" fill="#8D95A5" fontFamily="var(--font-geist-mono)">? Server command: npx server</text>
      <text x="42" y="88" fontSize="8" fill="#8D95A5" fontFamily="var(--font-geist-mono)">? Read paths: ./data</text>
      <text x="42" y="106" fontSize="8" fill="#8D95A5" fontFamily="var(--font-geist-mono)">? Write paths: ./output</text>
      <text x="42" y="124" fontSize="8" fill="#3FB27E" fontFamily="var(--font-geist-mono)">✓ policy.yaml created</text>
      <text x="120" y="165" textAnchor="middle" fontSize="10" fill="#8D95A5" fontFamily="var(--font-geist-mono)">STEP 2: INIT</text>
    </svg>
  );
}

export function HowRun() {
  return (
    <svg viewBox="0 0 240 180" fill="none" className="h-full w-full max-h-[280px]">
      <rect x="30" y="25" width="180" height="130" rx="6" fill="#1E242E" stroke="#3A4150" strokeWidth="1.2" />
      <text x="42" y="48" fontSize="9" fill="#8D95A5" fontFamily="var(--font-geist-mono)">$</text>
      <text x="56" y="48" fontSize="9" fill="#3FB27E" fontFamily="var(--font-geist-mono)">warden run --policy policy.yaml</text>
      <text x="62" y="62" fontSize="9" fill="#3FB27E" fontFamily="var(--font-geist-mono)">  -- npx server</text>
      <text x="42" y="85" fontSize="8" fill="#6E93E8" fontFamily="var(--font-geist-mono)">[sandbox] bwrap initialized</text>
      <text x="42" y="103" fontSize="8" fill="#3FB27E" fontFamily="var(--font-geist-mono)">[grant] ./data → read</text>
      <text x="42" y="121" fontSize="8" fill="#E2604F" fontFamily="var(--font-geist-mono)">[deny]  ~/.ssh → invisible</text>
      <text x="42" y="139" fontSize="8" fill="#3FB27E" fontFamily="var(--font-geist-mono)">[grant] api.github.com → net</text>
      <text x="120" y="165" textAnchor="middle" fontSize="10" fill="#8D95A5" fontFamily="var(--font-geist-mono)">STEP 3: RUN SANDBOXED</text>
    </svg>
  );
}

export function HowLogs() {
  return (
    <svg viewBox="0 0 240 180" fill="none" className="h-full w-full max-h-[280px]">
      <rect x="30" y="25" width="180" height="130" rx="6" fill="#1E242E" stroke="#3A4150" strokeWidth="1.2" />
      <text x="42" y="48" fontSize="9" fill="#8D95A5" fontFamily="var(--font-geist-mono)">$</text>
      <text x="56" y="48" fontSize="9" fill="#3FB27E" fontFamily="var(--font-geist-mono)">warden logs --tail</text>
      {[
        { y: 68, action: "read", target: "./data", granted: true },
        { y: 86, action: "read", target: "~/.ssh/id_rsa", granted: false },
        { y: 104, action: "connect", target: "api.github.com", granted: true },
        { y: 122, action: "connect", target: "evil.com", granted: false },
      ].map((e) => (
        <g key={e.y}>
          <text x="42" y={e.y + 11} fontSize="8" fill="#8D95A5" fontFamily="var(--font-geist-mono)">{e.action}</text>
          <text x="90" y={e.y + 11} fontSize="8" fill="#E8EBEF" fontFamily="var(--font-geist-mono)">{e.target}</text>
          <text x="200" y={e.y + 11} textAnchor="end" fontSize="8" fill={e.granted ? "#3FB27E" : "#E2604F"} fontFamily="var(--font-geist-mono)">{e.granted ? "allow" : "deny"}</text>
        </g>
      ))}
      <text x="120" y="160" textAnchor="middle" fontSize="10" fill="#8D95A5" fontFamily="var(--font-geist-mono)">STEP 4: INSPECT</text>
    </svg>
  );
}
