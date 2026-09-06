export default function Problem() {
  return (
    <section className="border-t border-ink-800">
      <div className="mx-auto grid max-w-content gap-10 px-6 py-20 md:grid-cols-[22rem_1fr] md:gap-16">
        <div>
          <h2 className="text-[1.75rem] font-medium leading-[1.15] tracking-[-0.01em] text-paper">
            MCP has no concept of a boundary.
          </h2>
          <p className="mt-4 text-[1rem] leading-[1.65] text-muted">
            A server you installed five minutes ago runs with your full
            user permissions by default — every file you can read, every
            host you can reach. Nothing in the protocol stops it. Warden
            adds the boundary the protocol doesn&apos;t: a policy the
            server can&apos;t see past.
          </p>
        </div>

        <div className="overflow-x-auto">
          <svg
            viewBox="0 0 600 240"
            className="h-auto w-full min-w-[560px]"
            role="img"
            aria-label="Diagram comparing an MCP server with no sandbox, whose access reaches everywhere, to one running under Warden, where only granted paths and hosts are reachable and everything else is stopped at the boundary."
          >
            <line x1="300" y1="10" x2="300" y2="230" stroke="#2A303C" strokeWidth="1" />

            <rect x="98" y="90" width="70" height="50" rx="3" fill="#1E242E" stroke="#3A4150" strokeWidth="1.3" />
            <text x="133" y="118" textAnchor="middle" fontSize="10" fill="#E8EBEF" fontFamily="var(--font-geist-mono)">
              server
            </text>

            {[
              { x2: 22, y2: 24, label: "~/.ssh" },
              { x2: 252, y2: 30, label: "$ENV" },
              { x2: 18, y2: 200, label: "/etc" },
              { x2: 258, y2: 206, label: "*.internal" },
            ].map((l) => (
              <g key={l.label}>
                <line
                  x1="133"
                  y1="115"
                  x2={l.x2}
                  y2={l.y2}
                  stroke="#E2604F"
                  strokeWidth="1.2"
                  strokeDasharray="3 3"
                  opacity="0.75"
                />
                <circle cx={l.x2} cy={l.y2} r="2.5" fill="#E2604F" opacity="0.85" />
                <text
                  x={l.x2}
                  y={l.y2 - 8}
                  textAnchor="middle"
                  fontSize="9"
                  fill="#8D95A5"
                  fontFamily="var(--font-geist-mono)"
                >
                  {l.label}
                </text>
              </g>
            ))}

            <text x="133" y="228" textAnchor="middle" fontSize="11" fill="#8D95A5" fontFamily="var(--font-geist-mono)">
              without a boundary
            </text>

            <rect
              x="380"
              y="35"
              width="180"
              height="150"
              rx="3"
              fill="none"
              stroke="#6E93E8"
              strokeWidth="1.2"
              strokeDasharray="4 3"
            />
            <rect x="435" y="90" width="70" height="50" rx="3" fill="#1E242E" stroke="#3A4150" strokeWidth="1.3" />
            <text x="470" y="118" textAnchor="middle" fontSize="10" fill="#E8EBEF" fontFamily="var(--font-geist-mono)">
              server
            </text>

            <line x1="470" y1="90" x2="470" y2="15" stroke="#3FB27E" strokeWidth="1.3" />
            <circle cx="470" cy="15" r="2.5" fill="#3FB27E" />
            <text x="470" y="8" textAnchor="middle" fontSize="9" fill="#8D95A5" fontFamily="var(--font-geist-mono)">
              ./data
            </text>

            <line x1="505" y1="115" x2="575" y2="115" stroke="#3FB27E" strokeWidth="1.3" />
            <circle cx="575" cy="115" r="2.5" fill="#3FB27E" />
            <text x="575" y="105" textAnchor="middle" fontSize="9" fill="#8D95A5" fontFamily="var(--font-geist-mono)">
              api.github.com
            </text>

            <line x1="435" y1="105" x2="381" y2="65" stroke="#E2604F" strokeWidth="1.2" opacity="0.8" />
            <g transform="translate(378,62)">
              <line x1="-3" y1="-3" x2="3" y2="3" stroke="#E2604F" strokeWidth="1.4" />
              <line x1="-3" y1="3" x2="3" y2="-3" stroke="#E2604F" strokeWidth="1.4" />
            </g>

            <line x1="435" y1="130" x2="381" y2="158" stroke="#E2604F" strokeWidth="1.2" opacity="0.8" />
            <g transform="translate(378,161)">
              <line x1="-3" y1="-3" x2="3" y2="3" stroke="#E2604F" strokeWidth="1.4" />
              <line x1="-3" y1="3" x2="3" y2="-3" stroke="#E2604F" strokeWidth="1.4" />
            </g>

            <text x="470" y="228" textAnchor="middle" fontSize="11" fill="#8D95A5" fontFamily="var(--font-geist-mono)">
              with warden
            </text>
          </svg>
        </div>
      </div>
    </section>
  );
}
