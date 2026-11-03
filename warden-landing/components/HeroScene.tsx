import {
  CircleCheck,
  Code2,
  Database,
  FileText,
  Folder,
  Globe,
  KeyRound,
  Shield,
  ShieldCheck,
  SquareTerminal,
  Wrench,
  X,
} from "lucide-react";

export default function HeroScene() {
  return (
    <div
      className="relative mx-auto mt-8 h-[330px] w-full max-w-[1100px] sm:h-[380px] lg:h-[420px]"
      aria-hidden
    >
      {/* atmosphere */}
      <div className="pointer-events-none absolute left-1/2 top-4 h-[260px] w-[560px] -translate-x-1/2 rounded-[50%] bg-mint/[0.10] blur-[80px]" />
      <div className="pointer-events-none absolute -left-16 bottom-4 h-[260px] w-[320px] rounded-[50%] bg-deny/[0.12] blur-[80px]" />
      <div className="pointer-events-none absolute -right-16 bottom-4 h-[260px] w-[320px] rounded-[50%] bg-mint/[0.10] blur-[80px]" />

      {/* floor grid */}
      <div className="absolute inset-x-10 bottom-4 h-[150px] opacity-40 [background-image:linear-gradient(rgba(59,240,160,0.14)_1px,transparent_1px),linear-gradient(90deg,rgba(59,240,160,0.14)_1px,transparent_1px)] [background-size:46px_36px] [mask-image:radial-gradient(ellipse_55%_75%_at_50%_100%,black,transparent)]" />

      <SceneWires />
      <BlockedChips />
      <AllowedChips />
      <Chamber />
    </div>
  );
}

function SceneWires() {
  return (
    <svg
      className="pointer-events-none absolute inset-0 hidden h-full w-full lg:block"
      viewBox="0 0 1100 420"
      preserveAspectRatio="none"
      fill="none"
    >
      <defs>
        <linearGradient id="wire-grant" x1="0" y1="0" x2="1" y2="0">
          <stop offset="0%" stopColor="#3BF0A0" stopOpacity="0.05" />
          <stop offset="55%" stopColor="#3BF0A0" stopOpacity="0.5" />
          <stop offset="100%" stopColor="#3BF0A0" stopOpacity="0.25" />
        </linearGradient>
        <linearGradient id="wire-deny" x1="0" y1="0" x2="1" y2="0">
          <stop offset="0%" stopColor="#E2604F" stopOpacity="0.25" />
          <stop offset="45%" stopColor="#E2604F" stopOpacity="0.5" />
          <stop offset="100%" stopColor="#E2604F" stopOpacity="0.05" />
        </linearGradient>
      </defs>

      {/* allowed paths */}
      <path
        d="M 590 300 C 700 300 780 210 900 120"
        stroke="url(#wire-grant)"
        strokeWidth="1.5"
        strokeDasharray="2 6"
      />
      <path
        d="M 600 315 C 730 320 820 250 960 190"
        stroke="url(#wire-grant)"
        strokeWidth="1.5"
        strokeDasharray="2 6"
      />
      <path
        d="M 610 335 C 750 350 870 330 1010 285"
        stroke="url(#wire-grant)"
        strokeWidth="1.5"
        strokeDasharray="2 6"
      />
      {[
        [760, 250],
        [810, 285],
        [690, 296],
        [700, 318],
        [880, 230],
      ].map(([cx, cy], i) => (
        <circle key={`g${i}`} cx={cx} cy={cy} r="2.5" fill="#3BF0A0" opacity="0.85" />
      ))}

      {/* blocked paths */}
      <path
        d="M 520 300 C 440 300 360 260 292 208"
        stroke="url(#wire-deny)"
        strokeWidth="1.5"
        strokeDasharray="2 6"
      />
      <path
        d="M 505 330 C 400 345 300 330 210 300"
        stroke="url(#wire-deny)"
        strokeWidth="1.5"
        strokeDasharray="2 6"
      />
      {[
        [430, 292],
        [380, 284],
        [340, 260],
        [300, 322],
        [262, 316],
      ].map(([cx, cy], i) => (
        <circle key={`r${i}`} cx={cx} cy={cy} r="2.5" fill="#E2604F" opacity="0.85" />
      ))}
    </svg>
  );
}

function BlockedChips() {
  return (
    <div className="pointer-events-none absolute inset-0 hidden lg:block">
      <div className="absolute left-6 top-[58px] w-[148px] rounded-lg border border-deny/30 bg-[#150808]/90 p-2.5 shadow-[0_0_30px_rgba(226,96,79,0.18)]">
        <p className="font-mono text-[9px] uppercase tracking-[0.14em] text-deny/70">env</p>
        <p className="mt-1.5 font-mono text-[9px] leading-[1.6] text-deny/85">
          OPENAI_API_KEY=****
        </p>
        <p className="font-mono text-[9px] leading-[1.6] text-deny/85">DATABASE_URL=****</p>
        <p className="font-mono text-[9px] leading-[1.6] text-deny/85">AWS_SECRET=****</p>
      </div>

      <BlockedIcon className="left-[64px] top-[196px]">
        <KeyRound size={17} />
      </BlockedIcon>
      <BlockedIcon className="left-[132px] top-[222px]">
        <Database size={17} />
      </BlockedIcon>
      <BlockedIcon className="left-[34px] top-[268px]">
        <Globe size={17} />
      </BlockedIcon>
      <BlockedIcon className="left-[122px] top-[300px]">
        <SquareTerminal size={17} />
      </BlockedIcon>

      <div className="absolute left-[248px] top-[150px] flex h-12 w-12 items-center justify-center rounded-full border border-deny/40 bg-[#1A0908]/90 shadow-[0_0_36px_rgba(226,96,79,0.35)]">
        <X size={22} className="text-deny" />
      </div>
    </div>
  );
}

function BlockedIcon({
  className = "",
  children,
}: {
  className?: string;
  children: React.ReactNode;
}) {
  return (
    <div
      className={`absolute flex h-[52px] w-[52px] items-center justify-center rounded-lg border border-deny/30 bg-[#150808]/90 text-deny/85 shadow-[0_0_24px_rgba(226,96,79,0.14)] ${className}`}
    >
      {children}
    </div>
  );
}

function AllowedChips() {
  return (
    <div className="pointer-events-none absolute inset-0 hidden lg:block">
      <AllowedChip className="right-8 top-[38px] w-[152px]" icon={<Folder size={13} />}>
        /project
      </AllowedChip>
      <AllowedChip className="right-6 top-[104px] w-[176px]" icon={<Globe size={13} />}>
        api.anthropic.com
      </AllowedChip>
      <AllowedChip
        className="right-[74px] top-[160px] w-[178px]"
        icon={<SquareTerminal size={13} />}
      >
        ANTHROPIC_API_KEY
      </AllowedChip>
      <AllowedChip className="right-10 top-[240px] w-[150px]" icon={<FileText size={13} />}>
        Read files
      </AllowedChip>
      <AllowedChip className="right-[60px] top-[300px] w-[140px]" icon={<Wrench size={13} />}>
        Run tool
      </AllowedChip>
    </div>
  );
}

function AllowedChip({
  className = "",
  icon,
  children,
}: {
  className?: string;
  icon: React.ReactNode;
  children: React.ReactNode;
}) {
  return (
    <div
      className={`absolute flex items-center gap-2 rounded-lg border border-mint/30 bg-[#07130E]/90 px-2.5 py-2 shadow-[0_0_26px_rgba(59,240,160,0.14)] ${className}`}
    >
      <span className="flex h-5 w-5 shrink-0 items-center justify-center rounded bg-mint/10 text-mint">
        {icon}
      </span>
      <span className="min-w-0 flex-1 truncate font-mono text-[10px] text-white/80">
        {children}
      </span>
      <CircleCheck size={13} className="shrink-0 text-mint" />
    </div>
  );
}

function Chamber() {
  return (
    <div className="absolute bottom-0 left-1/2 -translate-x-1/2">
      <div className="relative h-[238px] w-[264px] sm:h-[276px] sm:w-[318px] lg:h-[300px] lg:w-[356px]">
        {/* backlight + portal rings */}
        <div className="absolute inset-8 rounded-full bg-mint/[0.14] blur-[46px]" />
        <div className="absolute left-1/2 top-[46%] h-[210px] w-[210px] -translate-x-1/2 -translate-y-1/2 rounded-full border border-mint/15" />
        <div className="absolute left-1/2 top-[46%] h-[150px] w-[150px] -translate-x-1/2 -translate-y-1/2 rounded-full border border-mint/10" />
        <div className="absolute -top-10 left-1/2 h-16 w-28 -translate-x-1/2 rounded-[50%] bg-mint/15 blur-xl" />

        {/* side faces */}
        <div className="absolute -left-3 bottom-10 top-10 w-6 origin-right -skew-y-6 rounded-l-md border-y border-l border-mint/20 bg-gradient-to-r from-mint/[0.02] to-mint/[0.09]" />
        <div className="absolute -right-3 bottom-10 top-10 w-6 origin-left skew-y-6 rounded-r-md border-y border-r border-mint/20 bg-gradient-to-l from-mint/[0.02] to-mint/[0.09]" />

        {/* top face */}
        <div className="absolute inset-x-6 top-2 h-7 rounded-[6px] border border-mint/20 bg-gradient-to-b from-mint/[0.12] to-transparent [clip-path:polygon(8%_0,92%_0,100%_100%,0_100%)]" />

        {/* glass box */}
        <div className="absolute inset-x-1 bottom-9 top-7 rounded-[14px] border border-mint/30 bg-gradient-to-b from-mint/[0.09] via-[#07140F]/85 to-[#050907]/95 shadow-[0_0_70px_-12px_rgba(59,240,160,0.55),inset_0_0_50px_rgba(59,240,160,0.06)] backdrop-blur-[1px]">
          {/* inner rim light */}
          <div className="absolute inset-x-4 top-0 h-px bg-gradient-to-r from-transparent via-mint/60 to-transparent" />
          <div className="absolute inset-x-6 bottom-0 h-px bg-gradient-to-r from-transparent via-mint/30 to-transparent" />

          {/* shield backdrop */}
          <Shield
            size={168}
            strokeWidth={0.9}
            className="absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 text-mint/[0.16]"
          />

          {/* corner light beams */}
          <div className="absolute left-3 top-3 bottom-3 w-px bg-gradient-to-b from-mint/50 via-mint/10 to-transparent" />
          <div className="absolute right-3 top-3 bottom-3 w-px bg-gradient-to-b from-mint/50 via-mint/10 to-transparent" />

          {/* side cubes */}
          <div className="absolute bottom-7 left-[11%] flex h-9 w-9 items-center justify-center rounded-md border border-mint/30 bg-mint/[0.07] text-mint/70">
            <Code2 size={14} />
          </div>
          <div className="absolute bottom-7 right-[11%] flex h-9 w-9 items-center justify-center rounded-md border border-mint/30 bg-mint/[0.07] text-mint/70">
            <ShieldCheck size={14} />
          </div>

          <Robot />

          {/* nameplate */}
          <div className="absolute -bottom-3 left-1/2 flex -translate-x-1/2 items-center gap-1.5 rounded-md border border-mint/25 bg-[#06110C] px-2.5 py-1">
            <Shield size={10} className="text-mint" />
            <span className="font-sans text-[9px] font-medium tracking-wide text-white/70">
              Warden
            </span>
          </div>
        </div>

        {/* top rim */}
        <div className="absolute inset-x-8 top-[22px] h-[3px] rounded-full bg-mint/30 blur-[1px]" />
        <div className="absolute inset-x-12 top-[25px] h-[1px] rounded-full bg-mint/60" />

        {/* base */}
        <div className="absolute inset-x-2 bottom-3 h-8 rounded-[50%] border border-mint/20 bg-mint/[0.05]" />
        <div className="absolute inset-x-14 bottom-2 h-[3px] rounded-[50%] bg-mint/40 blur-[3px]" />
        <div className="absolute inset-x-24 bottom-4 h-10 rounded-[50%] bg-mint/[0.14] blur-xl" />
      </div>
    </div>
  );
}

function Robot() {
  return (
    <div className="absolute bottom-[46px] left-1/2 -translate-x-1/2">
      <div className="relative flex flex-col items-center">
        {/* head */}
        <div className="relative flex h-[62px] w-[58px] items-center justify-center rounded-[22px] border border-mint/60 bg-gradient-to-b from-[#0D3A26] to-[#071912] shadow-[0_0_30px_rgba(59,240,160,0.35)]">
          <span className="absolute -top-2 h-2.5 w-[2px] rounded bg-mint/70" />
          <span className="absolute -top-3.5 h-1.5 w-1.5 rounded-full bg-mint shadow-[0_0_8px_#3BF0A0]" />
          <div className="flex h-[22px] w-[40px] items-center justify-center gap-2.5 rounded-full bg-[#03100A] shadow-[inset_0_0_10px_rgba(59,240,160,0.35)]">
            <span className="h-1.5 w-1.5 rounded-full bg-mint shadow-[0_0_6px_#3BF0A0]" />
            <span className="h-1.5 w-1.5 rounded-full bg-mint shadow-[0_0_6px_#3BF0A0]" />
          </div>
        </div>

        {/* body */}
        <div className="relative -mt-1 flex h-[44px] w-[76px] items-start justify-center rounded-t-[22px] rounded-b-[9px] border border-mint/45 bg-gradient-to-b from-[#0B2A1C] to-[#061410] pt-2 shadow-[0_10px_24px_rgba(0,0,0,0.5)]">
          <ShieldCheck size={18} className="text-mint/85" />
        </div>
      </div>
    </div>
  );
}
