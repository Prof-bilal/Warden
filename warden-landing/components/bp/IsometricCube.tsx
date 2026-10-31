export default function IsometricCube({ className }: { className?: string }) {
  const planes = [
    { y: 158, o: 0.16 },
    { y: 176, o: 0.1 },
    { y: 194, o: 0.06 },
  ];

  const markers: [number, number][] = [
    [120, 46],
    [92, 74],
    [148, 74],
  ];

  return (
    <svg viewBox="0 0 240 208" className={className} fill="none" aria-hidden>
      {/* stacked translucent grid planes */}
      {planes.map((p) => (
        <g key={p.y} opacity={p.o} stroke="#5EF08A" strokeWidth="1">
          <path d={`M120 ${p.y - 30} L176 ${p.y} L120 ${p.y + 30} L64 ${p.y} Z`} />
          <path d={`M92 ${p.y - 15} L148 ${p.y + 15}`} />
          <path d={`M92 ${p.y + 15} L148 ${p.y - 15}`} />
        </g>
      ))}

      {/* wireframe cube */}
      <g stroke="#5EF08A" strokeWidth="1.2" strokeLinejoin="round">
        <path d="M120 20 L176 52 L120 84 L64 52 Z" fill="#5EF08A" fillOpacity="0.06" />
        <path d="M64 52 L120 84 L120 130 L64 98 Z" fill="#5EF08A" fillOpacity="0.03" />
        <path d="M176 52 L120 84 L120 130 L176 98 Z" fill="#5EF08A" fillOpacity="0.08" />
        <path d="M120 20 L120 84" strokeOpacity="0.5" />
        <path d="M64 52 L120 130" strokeOpacity="0.25" />
        <path d="M176 52 L120 130" strokeOpacity="0.25" />
      </g>

      {/* vertices */}
      <g fill="#5EF08A">
        <circle cx="120" cy="20" r="1.6" />
        <circle cx="176" cy="52" r="1.6" />
        <circle cx="64" cy="52" r="1.6" />
        <circle cx="120" cy="84" r="1.6" />
        <circle cx="64" cy="98" r="1.6" />
        <circle cx="176" cy="98" r="1.6" />
        <circle cx="120" cy="130" r="1.6" />
      </g>

      {/* green square markers on the faces */}
      {markers.map(([x, y]) => (
        <rect
          key={`${x}-${y}`}
          x={x - 5}
          y={y - 5}
          width="10"
          height="10"
          rx="1.5"
          fill="#5EF08A"
          className="animate-bp-node"
        />
      ))}
    </svg>
  );
}
