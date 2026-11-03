import type { NodeItem } from "./NodeGraph";
import IconTile from "./IconTile";

export default function ToolStack({ items }: { items: NodeItem[] }) {
  return (
    <ul className="relative flex flex-col gap-3">
      <span
        className="absolute bottom-6 left-[22px] top-6 w-px bg-acid/25"
        aria-hidden
      />
      {items.map((item, i) => (
        <li key={item.label} className="relative flex items-center gap-3">
          <span className="relative z-10">
            <IconTile icon={item.icon} active />
          </span>
          <span className="font-mono text-[10px] text-offwhite/60">{item.label}</span>
          <span
            className="ml-auto h-1 w-1 rounded-full bg-acid animate-bp-node"
            style={{ animationDelay: `${i * 0.3}s` }}
            aria-hidden
          />
        </li>
      ))}
    </ul>
  );
}
