export function ShieldFailClosed() {
  return (
    <svg viewBox="0 0 320 200" fill="none" className="w-full h-auto max-h-[210px] sm:max-h-[270px] lg:max-h-[300px]">
      <text x="16" y="20" fontSize="8" fill="#8D95A5" fontFamily="var(--font-geist-mono)" letterSpacing="1.5">STARTUP DECISION FLOW</text>

      <rect x="16" y="82" width="76" height="30" rx="5" fill="#1E242E" stroke="#6E93E8" strokeWidth="1.1" />
      <text x="54" y="100" textAnchor="middle" fontSize="9" fill="#E8EBEF" fontFamily="var(--font-geist-mono)">backend init</text>

      <line x1="92" y1="97" x2="112" y2="97" stroke="#3A4150" strokeWidth="1.1" />
      <polyline points="108,93 112,97 108,101" stroke="#3A4150" strokeWidth="1.1" fill="none" />

      <rect x="112" y="78" width="88" height="38" rx="5" fill="none" stroke="#D6A24A" strokeWidth="1.1" strokeDasharray="3 2" />
      <text x="156" y="94" textAnchor="middle" fontSize="9" fill="#E8EBEF" fontFamily="var(--font-geist-mono)">backend</text>
      <text x="156" y="107" textAnchor="middle" fontSize="9" fill="#E8EBEF" fontFamily="var(--font-geist-mono)">verified?</text>

      <line x1="200" y1="97" x2="234" y2="97" stroke="#3FB27E" strokeWidth="1.1" />
      <polyline points="230,93 234,97 230,101" stroke="#3FB27E" strokeWidth="1.1" fill="none" />
      <text x="217" y="90" textAnchor="middle" fontSize="7.5" fill="#3FB27E" fontFamily="var(--font-geist-mono)">yes</text>
      <rect x="234" y="82" width="70" height="30" rx="5" fill="rgba(63,178,126,0.08)" stroke="#3FB27E" strokeWidth="1.1" />
      <text x="269" y="100" textAnchor="middle" fontSize="9" fill="#3FB27E" fontFamily="var(--font-geist-mono)">run ✓</text>

      <line x1="156" y1="116" x2="156" y2="142" stroke="#E2604F" strokeWidth="1.1" />
      <polyline points="152,138 156,142 160,138" stroke="#E2604F" strokeWidth="1.1" fill="none" />
      <text x="164" y="133" fontSize="7.5" fill="#E2604F" fontFamily="var(--font-geist-mono)">no</text>
      <rect x="112" y="142" width="88" height="30" rx="5" fill="rgba(226,96,79,0.08)" stroke="#E2604F" strokeWidth="1.1" />
      <text x="156" y="160" textAnchor="middle" fontSize="9" fill="#E2604F" fontFamily="var(--font-geist-mono)">refuse to run ✕</text>

      <text x="160" y="192" textAnchor="middle" fontSize="8" fill="#8D95A5" fontFamily="var(--font-geist-mono)" letterSpacing="1">NO VALID BACKEND ⇒ NO EXECUTION, EVER</text>
    </svg>
  );
}

export function DenyFilesystem() {
  const files = [
    { p: "./data", ok: true, y: 44 },
    { p: "./output", ok: true, y: 72 },
    { p: "~/.ssh", ok: false, y: 100 },
    { p: "/etc", ok: false, y: 128 },
  ];
  return (
    <svg viewBox="0 0 320 200" fill="none" className="w-full h-auto max-h-[210px] sm:max-h-[270px] lg:max-h-[300px]">
      <text x="16" y="20" fontSize="8" fill="#8D95A5" fontFamily="var(--font-geist-mono)" letterSpacing="1.5">HOST FILESYSTEM</text>
      <text x="216" y="20" fontSize="8" fill="#8D95A5" fontFamily="var(--font-geist-mono)" letterSpacing="1.5">SERVER VIEW</text>

      {files.map((f) => (
        <g key={f.p}>
          <rect
            x="16" y={f.y - 14} width="88" height="20" rx="3"
            fill={f.ok ? "rgba(63,178,126,0.07)" : "#171C24"}
            stroke={f.ok ? "#3FB27E" : "#3A4150"}
            strokeWidth="1" strokeDasharray={f.ok ? undefined : "2.5 2"}
            opacity={f.ok ? 1 : 0.7}
          />
          <text x="26" y={f.y} fontSize="8.5" fill={f.ok ? "#3FB27E" : "#8D95A5"} fontFamily="var(--font-geist-mono)" opacity={f.ok ? 1 : 0.65}>{f.p}</text>
          {f.ok ? (
            <line x1="104" y1={f.y - 4} x2="212" y2={f.y - 4} stroke="#3FB27E" strokeWidth="1" opacity="0.55" />
          ) : (
            <g opacity="0.7">
              <line x1="104" y1={f.y - 4} x2="132" y2={f.y - 4} stroke="#E2604F" strokeWidth="1" strokeDasharray="2.5 2" />
              <line x1="129" y1={f.y - 8} x2="135" y2={f.y} stroke="#E2604F" strokeWidth="1.2" />
              <line x1="129" y1={f.y} x2="135" y2={f.y - 8} stroke="#E2604F" strokeWidth="1.2" />
            </g>
          )}
        </g>
      ))}

      <rect x="138" y="30" width="44" height="122" rx="5" fill="rgba(110,147,232,0.06)" stroke="#6E93E8" strokeWidth="1" strokeDasharray="3 2" />
      <text x="160" y="86" textAnchor="middle" fontSize="8" fill="#6E93E8" fontFamily="var(--font-geist-mono)">grant</text>
      <text x="160" y="98" textAnchor="middle" fontSize="8" fill="#6E93E8" fontFamily="var(--font-geist-mono)">filter</text>

      <rect x="216" y="30" width="88" height="20" rx="3" fill="rgba(63,178,126,0.07)" stroke="#3FB27E" strokeWidth="1" />
      <text x="226" y="44" fontSize="8.5" fill="#3FB27E" fontFamily="var(--font-geist-mono)">./data</text>
      <rect x="216" y="58" width="88" height="20" rx="3" fill="rgba(63,178,126,0.07)" stroke="#3FB27E" strokeWidth="1" />
      <text x="226" y="72" fontSize="8.5" fill="#3FB27E" fontFamily="var(--font-geist-mono)">./output</text>
      <rect x="216" y="96" width="88" height="56" rx="3" fill="none" stroke="#3A4150" strokeWidth="1" strokeDasharray="2.5 2" />
      <text x="260" y="120" textAnchor="middle" fontSize="9" fill="#8D95A5" fontFamily="var(--font-geist-mono)">∅ nothing else</text>
      <text x="260" y="134" textAnchor="middle" fontSize="7.5" fill="#8D95A5" fontFamily="var(--font-geist-mono)" opacity="0.7">exists for this process</text>

      <text x="160" y="192" textAnchor="middle" fontSize="8" fill="#8D95A5" fontFamily="var(--font-geist-mono)" letterSpacing="1">UNLISTED = INVISIBLE, NOT MERELY UNREADABLE</text>
    </svg>
  );
}

export function NetworkEnforced() {
  const hosts = [
    { label: "api.github.com", y: 48, ok: true },
    { label: "evil.com", y: 100, ok: false },
    { label: "malware.io", y: 152, ok: false },
  ];
  return (
    <svg viewBox="0 0 320 200" fill="none" className="w-full h-auto max-h-[210px] sm:max-h-[270px] lg:max-h-[300px]">
      <text x="16" y="20" fontSize="8" fill="#8D95A5" fontFamily="var(--font-geist-mono)" letterSpacing="1.5">EGRESS CONTROL</text>

      <rect x="16" y="78" width="64" height="36" rx="5" fill="#1E242E" stroke="#6E93E8" strokeWidth="1.1" />
      <text x="48" y="99" textAnchor="middle" fontSize="9" fill="#E8EBEF" fontFamily="var(--font-geist-mono)">server</text>

      <line x1="80" y1="96" x2="114" y2="96" stroke="#3A4150" strokeWidth="1.1" />
      <polyline points="110,92 114,96 110,100" stroke="#3A4150" strokeWidth="1.1" fill="none" />

      <rect x="114" y="70" width="78" height="52" rx="6" fill="rgba(110,147,232,0.07)" stroke="#6E93E8" strokeWidth="1.1" />
      <text x="153" y="92" textAnchor="middle" fontSize="9" fill="#6E93E8" fontFamily="var(--font-geist-mono)">egress</text>
      <text x="153" y="106" textAnchor="middle" fontSize="9" fill="#6E93E8" fontFamily="var(--font-geist-mono)">proxy</text>

      {hosts.map((h) => (
        <g key={h.label}>
          <line
            x1="192" y1={h.y < 100 ? 86 : h.y === 100 ? 96 : 106} x2="240" y2={h.y}
            stroke={h.ok ? "#3FB27E" : "#E2604F"} strokeWidth="1.1"
            strokeDasharray={h.ok ? undefined : "2.5 2"} opacity={h.ok ? 0.9 : 0.6}
          />
          <circle cx="246" cy={h.y} r="5" fill={h.ok ? "rgba(63,178,126,0.12)" : "rgba(226,96,79,0.12)"} stroke={h.ok ? "#3FB27E" : "#E2604F"} strokeWidth="1.1" />
          <text x="258" y={h.y + 3} fontSize="8" fill={h.ok ? "#3FB27E" : "#E2604F"} fontFamily="var(--font-geist-mono)" opacity={h.ok ? 1 : 0.75}>{h.label}</text>
          {!h.ok && (
            <g stroke="#E2604F" strokeWidth="1.2">
              <line x1={216} y1={(86 + h.y) / 2 - 4} x2={222} y2={(86 + h.y) / 2 + 2} />
              <line x1={216} y1={(86 + h.y) / 2 + 2} x2={222} y2={(86 + h.y) / 2 - 4} />
            </g>
          )}
        </g>
      ))}

      <line x1="153" y1="122" x2="153" y2="142" stroke="#6E93E8" strokeWidth="0.9" strokeDasharray="2 2" opacity="0.6" />
      <text x="153" y="154" textAnchor="middle" fontSize="7.5" fill="#6E93E8" fontFamily="var(--font-geist-mono)" opacity="0.85">dns resolves only after</text>
      <text x="153" y="164" textAnchor="middle" fontSize="7.5" fill="#6E93E8" fontFamily="var(--font-geist-mono)" opacity="0.85">the allowlist check</text>

      <text x="160" y="192" textAnchor="middle" fontSize="8" fill="#8D95A5" fontFamily="var(--font-geist-mono)" letterSpacing="1">NO GRANT ⇒ NO CONNECTION — NOT EVEN DNS</text>
    </svg>
  );
}

export function EnvFilter() {
  const vars = [
    { name: "PATH", ok: false },
    { name: "HOME", ok: false },
    { name: "GITHUB_TOKEN", ok: true },
    { name: "AWS_SECRET", ok: false },
    { name: "SHELL", ok: false },
  ];
  return (
    <svg viewBox="0 0 320 200" fill="none" className="w-full h-auto max-h-[210px] sm:max-h-[270px] lg:max-h-[300px]">
      <text x="16" y="20" fontSize="8" fill="#8D95A5" fontFamily="var(--font-geist-mono)" letterSpacing="1.5">PROCESS ENV</text>
      <text x="216" y="20" fontSize="8" fill="#8D95A5" fontFamily="var(--font-geist-mono)" letterSpacing="1.5">SANDBOX ENV</text>

      <rect x="16" y="30" width="112" height="130" rx="5" fill="#171C24" stroke="#3A4150" strokeWidth="1" />
      {vars.map((v, i) => (
        <text key={v.name} x="28" y={54 + i * 22} fontSize="8.5" fill={v.ok ? "#3FB27E" : "#8D95A5"} fontFamily="var(--font-geist-mono)" opacity={v.ok ? 1 : 0.6}>{v.name}</text>
      ))}

      <line x1="160" y1="36" x2="160" y2="154" stroke="#6E93E8" strokeWidth="1.1" strokeDasharray="3 2" />
      <text x="160" y="28" textAnchor="middle" fontSize="8" fill="#6E93E8" fontFamily="var(--font-geist-mono)">env.allow</text>

      {vars.map((v, i) => {
        const y = 50 + i * 22;
        return v.ok ? (
          <line key={v.name} x1="128" y1={y} x2="216" y2={y} stroke="#3FB27E" strokeWidth="1.1" opacity="0.8" />
        ) : (
          <g key={v.name} opacity="0.65">
            <line x1="128" y1={y} x2="154" y2={y} stroke="#E2604F" strokeWidth="1" strokeDasharray="2.5 2" />
            <line x1="151" y1={y - 4} x2="157" y2={y + 2} stroke="#E2604F" strokeWidth="1.2" />
            <line x1="151" y1={y + 2} x2="157" y2={y - 4} stroke="#E2604F" strokeWidth="1.2" />
          </g>
        );
      })}

      <rect x="216" y="30" width="88" height="130" rx="5" fill="rgba(63,178,126,0.05)" stroke="#3FB27E" strokeWidth="1" />
      <text x="228" y="54" fontSize="8.5" fill="#3FB27E" fontFamily="var(--font-geist-mono)">GITHUB_TOKEN</text>
      <text x="260" y="100" textAnchor="middle" fontSize="8" fill="#8D95A5" fontFamily="var(--font-geist-mono)">1 of 5 forwarded</text>
      <text x="260" y="114" textAnchor="middle" fontSize="7.5" fill="#8D95A5" fontFamily="var(--font-geist-mono)" opacity="0.7">nothing inherited</text>

      <text x="160" y="192" textAnchor="middle" fontSize="8" fill="#8D95A5" fontFamily="var(--font-geist-mono)" letterSpacing="1">EMPTY ALLOWLIST ⇒ EMPTY ENVIRONMENT</text>
    </svg>
  );
}

export function PlatformFailClosed() {
  const platforms = [
    { x: 14, name: "Linux", backend: "bubblewrap", verified: true },
    { x: 116, name: "macOS", backend: "sandbox-exec", verified: false },
    { x: 218, name: "Windows", backend: "AppContainer", verified: true },
  ];
  return (
    <svg viewBox="0 0 320 200" fill="none" className="w-full h-auto max-h-[210px] sm:max-h-[270px] lg:max-h-[300px]">
      <rect x="102" y="14" width="116" height="26" rx="5" fill="#1E242E" stroke="#8D95A5" strokeWidth="1" />
      <text x="160" y="31" textAnchor="middle" fontSize="9" fill="#E8EBEF" fontFamily="var(--font-geist-mono)">policy.yaml</text>

      <line x1="160" y1="40" x2="160" y2="52" stroke="#3A4150" strokeWidth="1" />
      <line x1="58" y1="52" x2="262" y2="52" stroke="#3A4150" strokeWidth="1" />
      <line x1="58" y1="52" x2="58" y2="66" stroke="#3A4150" strokeWidth="1" />
      <line x1="160" y1="52" x2="160" y2="66" stroke="#3A4150" strokeWidth="1" />
      <line x1="262" y1="52" x2="262" y2="66" stroke="#3A4150" strokeWidth="1" />

      {platforms.map((p) => (
        <g key={p.name}>
          <rect x={p.x} y="66" width="88" height="76" rx="6" fill="#171C24" stroke={p.verified ? "#3FB27E" : "#D6A24A"} strokeWidth="1" strokeOpacity="0.6" />
          <text x={p.x + 44} y="88" textAnchor="middle" fontSize="10" fill="#E8EBEF" fontFamily="var(--font-geist-mono)">{p.name}</text>
          <text x={p.x + 44} y="104" textAnchor="middle" fontSize="8" fill="#8D95A5" fontFamily="var(--font-geist-mono)">{p.backend}</text>
          <line x1={p.x + 16} y1="114" x2={p.x + 72} y2="114" stroke="#2A303C" strokeWidth="1" />
          {p.verified ? (
            <text x={p.x + 44} y="130" textAnchor="middle" fontSize="7.5" fill="#3FB27E" fontFamily="var(--font-geist-mono)">✓ CI-verified</text>
          ) : (
            <text x={p.x + 44} y="130" textAnchor="middle" fontSize="7.5" fill="#D6A24A" fontFamily="var(--font-geist-mono)">CI green · HW pending</text>
          )}
        </g>
      ))}

      <text x="160" y="172" textAnchor="middle" fontSize="8" fill="#8D95A5" fontFamily="var(--font-geist-mono)" letterSpacing="1">ONE POLICY · NATIVE ENFORCEMENT PER OS</text>
      <text x="160" y="188" textAnchor="middle" fontSize="8" fill="#E2604F" fontFamily="var(--font-geist-mono)" opacity="0.8">backend missing? refuse to run — never fallback silently</text>
    </svg>
  );
}

export function ResourceLimits() {
  const gauges = [
    { cx: 90, value: "300s", unit: "wall clock", pct: 72, color: "#6E93E8" },
    { cx: 230, value: "512MB", unit: "memory RSS", pct: 55, color: "#E2604F" },
  ];
  return (
    <svg viewBox="0 0 320 200" fill="none" className="w-full h-auto max-h-[210px] sm:max-h-[270px] lg:max-h-[300px]">
      {gauges.map((g) => (
        <g key={g.unit}>
          <rect x={g.cx - 36} y="18" width="72" height="18" rx="4" fill="#171C24" stroke="#3A4150" strokeWidth="1" />
          <text x={g.cx} y="30" textAnchor="middle" fontSize="8" fill="#8D95A5" fontFamily="var(--font-geist-mono)">{g.unit}</text>
          <path d={`M ${g.cx - 46} 118 A 46 46 0 0 1 ${g.cx + 46} 118`} stroke="#2A303C" strokeWidth="8" strokeLinecap="round" />
          <path
            d={`M ${g.cx - 46} 118 A 46 46 0 0 1 ${g.cx + 46} 118`}
            stroke={g.color} strokeWidth="8" strokeLinecap="round"
            pathLength={100} strokeDasharray={`${g.pct} 100`}
          />
          <text x={g.cx} y="102" textAnchor="middle" fontSize="15" fill="#E8EBEF" fontFamily="var(--font-geist-mono)" fontWeight="600">{g.value}</text>
          <text x={g.cx} y="134" textAnchor="middle" fontSize="7.5" fill="#8D95A5" fontFamily="var(--font-geist-mono)">hard cap</text>
        </g>
      ))}

      <rect x="52" y="150" width="216" height="26" rx="5" fill="rgba(226,96,79,0.05)" stroke="#E2604F" strokeWidth="1" strokeDasharray="3 2" />
      <text x="160" y="167" textAnchor="middle" fontSize="8.5" fill="#E2604F" fontFamily="var(--font-geist-mono)">breach ⇒ SIGTERM → SIGKILL, whole process tree</text>

      <text x="160" y="192" textAnchor="middle" fontSize="8" fill="#8D95A5" fontFamily="var(--font-geist-mono)" letterSpacing="1">LIMITS ENFORCED BY THE KERNEL, NOT THE PROCESS</text>
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
