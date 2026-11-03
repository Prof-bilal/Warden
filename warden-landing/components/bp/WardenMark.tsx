export default function WardenMark({
  size = 28,
  className,
}: {
  size?: number;
  className?: string;
}) {
  return (
    <svg
      viewBox="0 0 24 24"
      width={size}
      height={size}
      className={className}
      fill="none"
      aria-hidden
    >
      <rect x="2" y="2" width="8.6" height="8.6" rx="1.6" fill="currentColor" />
      <rect x="13.4" y="2" width="8.6" height="8.6" rx="1.6" fill="currentColor" fillOpacity="0.55" />
      <rect x="2" y="13.4" width="8.6" height="8.6" rx="1.6" fill="currentColor" fillOpacity="0.55" />
      <rect x="13.4" y="13.4" width="8.6" height="8.6" rx="1.6" fill="currentColor" />
    </svg>
  );
}
