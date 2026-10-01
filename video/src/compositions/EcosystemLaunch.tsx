import React from "react";
import {
  AbsoluteFill,
  Sequence,
  interpolate,
  spring,
  useCurrentFrame,
  useVideoConfig,
} from "remotion";
import { LogoMark } from "../components/Logo";

const FPS = 30;
const DURATION = 66 * FPS;
const at = (seconds: number) => seconds * FPS;

const palette = {
  bg: "#10141A",
  panel: "#171C24",
  ink: "#1E242E",
  border: "#2A303C",
  text: "#E8EBEF",
  muted: "#8D95A5",
  blue: "#6E93E8",
  grant: "#3FB27E",
  deny: "#E2604F",
  gold: "#D6A24A",
};

const FONTS = {
  sans: '"Inter", system-ui, sans-serif',
  mono: '"JetBrains Mono", ui-monospace, monospace',
};

const enter = (frame: number, delay: number, fps: number) => {
  const value = spring({ frame: frame - delay, fps, config: { damping: 18, stiffness: 130, mass: 0.7 } });
  return { opacity: value, transform: `translateY(${(1 - value) * 42}px) scale(${0.96 + value * 0.04})` };
};

const Background: React.FC<{ accent?: string }> = ({ accent = palette.blue }) => {
  const frame = useCurrentFrame();
  return (
    <AbsoluteFill
      style={{
        backgroundColor: palette.bg,
        backgroundImage: `radial-gradient(circle at ${30 + (frame % 240) / 12}% 12%, ${accent}22 0, transparent 28%), linear-gradient(rgba(255,255,255,.035) 1px, transparent 1px), linear-gradient(90deg, rgba(255,255,255,.035) 1px, transparent 1px)`,
        backgroundSize: "auto, 48px 48px, 48px 48px",
      }}
    />
  );
};

const Eyebrow: React.FC<{ children: React.ReactNode; color?: string }> = ({ children, color = palette.blue }) => (
  <div style={{ fontFamily: FONTS.mono, fontWeight: 700, fontSize: 18, letterSpacing: ".22em", color, textTransform: "uppercase" }}>
    {children}
  </div>
);

const Pill: React.FC<{ children: React.ReactNode; color?: string }> = ({ children, color = palette.blue }) => (
  <span style={{ display: "inline-flex", alignItems: "center", border: `1px solid ${color}66`, color, background: `${color}18`, borderRadius: 999, padding: "12px 19px", fontFamily: FONTS.mono, fontSize: 17, fontWeight: 700 }}>
    {children}
  </span>
);

const Panel: React.FC<{ children: React.ReactNode; style?: React.CSSProperties }> = ({ children, style }) => (
  <div style={{ background: "linear-gradient(145deg, #1E242E 0%, #171C24 100%)", border: `1px solid ${palette.border}`, borderRadius: 26, boxShadow: "0 30px 100px rgba(0,0,0,.28)", ...style }}>{children}</div>
);

const Stage: React.FC<{ children: React.ReactNode; accent?: string }> = ({ children, accent }) => {
  const frame = useCurrentFrame();
  const { durationInFrames } = useVideoConfig();
  const out = interpolate(frame, [durationInFrames - 12, durationInFrames - 1], [1, 0], { extrapolateLeft: "clamp", extrapolateRight: "clamp" });
  return <AbsoluteFill style={{ opacity: out }}><Background accent={accent} />{children}</AbsoluteFill>;
};

const Hook: React.FC = () => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();
  return <Stage accent={palette.deny}>
    <div style={{ position: "absolute", left: 140, top: 112, ...enter(frame, 0, fps) }}><LogoMark size={86} glow /></div>
    <div style={{ position: "absolute", left: 140, top: 270, width: 1110 }}>
      <div style={{ ...enter(frame, 8, fps) }}><Eyebrow color={palette.deny}>MCP IS MOVING FAST</Eyebrow></div>
      <h1 style={{ ...enter(frame, 14, fps), margin: "22px 0 0", fontFamily: FONTS.sans, fontWeight: 800, fontSize: 92, lineHeight: .97, letterSpacing: "-.055em", color: palette.text }}>Your agent needs tools.</h1>
      <h1 style={{ ...enter(frame, 24, fps), margin: "4px 0 0", fontFamily: FONTS.sans, fontWeight: 800, fontSize: 92, lineHeight: .97, letterSpacing: "-.055em", color: palette.deny }}>Not your whole machine.</h1>
    </div>
    <Panel style={{ position: "absolute", right: 130, top: 230, width: 520, padding: 28, ...enter(frame, 26, fps) }}>
      <div style={{ display: "flex", gap: 8, marginBottom: 26 }}><i style={{ width: 11, height: 11, borderRadius: 50, background: palette.deny }} /><i style={{ width: 11, height: 11, borderRadius: 50, background: palette.gold }} /><i style={{ width: 11, height: 11, borderRadius: 50, background: palette.grant }} /></div>
      <div style={{ fontFamily: FONTS.mono, color: palette.muted, fontSize: 21, lineHeight: 1.8 }}>
        <div><span style={{ color: palette.blue }}>$</span> mcp-server start</div>
        <div style={{ color: palette.grant }}>✓ tools connected</div>
        <div style={{ marginTop: 22, color: palette.text }}>request: <span style={{ color: palette.deny }}>~/.ssh/id_ed25519</span></div>
        <div style={{ color: palette.deny }}>✕ too much access</div>
      </div>
    </Panel>
    <div style={{ position: "absolute", left: 140, bottom: 112, ...enter(frame, 40, fps) }}><Pill color={palette.grant}>WARDEN MAKES MCP ACCESS REVIEWABLE</Pill></div>
  </Stage>;
};

const Packs: React.FC = () => {
  const frame = useCurrentFrame(); const { fps } = useVideoConfig();
  const packs = [
    ["Filesystem", "Read a selected directory", "FILES"], ["Local Git", "Inspect a selected repo", "GIT"], ["Memory", "Store in one location", "MEM"],
    ["GitHub", "Restricted read access", "GH"], ["Brave Search", "One reviewed API", "BR"], ["Context7", "Library context", "C7"],
  ];
  return <Stage accent={palette.blue}>
    <div style={{ position: "absolute", left: 140, top: 112 }}>
      <div style={enter(frame, 0, fps)}><Eyebrow>01 / START WITH A POLICY</Eyebrow></div>
      <h2 style={{ ...enter(frame, 6, fps), color: palette.text, fontFamily: FONTS.sans, fontSize: 74, lineHeight: 1, letterSpacing: "-.045em", margin: "22px 0 0" }}>Popular MCPs.<br /><span style={{ color: palette.blue }}>Ready-to-review access.</span></h2>
    </div>
    <div style={{ position: "absolute", left: 140, right: 140, top: 435, display: "grid", gridTemplateColumns: "repeat(3, 1fr)", gap: 18 }}>
      {packs.map(([name, detail, glyph], index) => <Panel key={name} style={{ padding: 24, minHeight: 160, ...enter(frame, 14 + index * 4, fps) }}>
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "start" }}><div style={{ color: palette.blue, fontFamily: FONTS.mono, fontWeight: 800, fontSize: 18 }}>{glyph}</div><Pill color={palette.gold}>CANDIDATE</Pill></div>
        <div style={{ marginTop: 14, fontFamily: FONTS.sans, fontWeight: 700, fontSize: 27, color: palette.text }}>{name}</div>
        <div style={{ marginTop: 7, fontFamily: FONTS.sans, fontSize: 16, color: palette.muted }}>{detail}</div>
      </Panel>)}</div>
    <div style={{ position: "absolute", bottom: 105, left: 140, ...enter(frame, 44, fps) }}><Pill>packs list  →  review  →  generate</Pill></div>
  </Stage>;
};

const Setup: React.FC = () => {
  const frame = useCurrentFrame(); const { fps } = useVideoConfig();
  const clients = ["Claude Desktop", "Cursor", "Codex", "VS Code", "Gemini", "Cline"];
  return <Stage accent={palette.grant}>
    <div style={{ position: "absolute", left: 140, top: 110, width: 785 }}>
      <div style={enter(frame, 0, fps)}><Eyebrow color={palette.grant}>02 / REMOVE THE FRICTION</Eyebrow></div>
      <h2 style={{ ...enter(frame, 7, fps), fontFamily: FONTS.sans, fontSize: 78, lineHeight: .98, letterSpacing: "-.052em", color: palette.text, margin: "22px 0 0" }}>One command.<br />Your MCP, <span style={{ color: palette.grant }}>wrapped.</span></h2>
      <p style={{ ...enter(frame, 18, fps), fontFamily: FONTS.sans, color: palette.muted, fontSize: 24, lineHeight: 1.42, marginTop: 28 }}>Preview first. Back up the configuration. Apply only after the sandbox is ready.</p>
    </div>
    <Panel style={{ position: "absolute", right: 125, top: 150, width: 780, padding: 30, ...enter(frame, 16, fps) }}>
      <div style={{ color: palette.muted, fontFamily: FONTS.mono, fontSize: 18, marginBottom: 18 }}>TERMINAL / SAFE SETUP</div>
      <div style={{ borderLeft: `3px solid ${palette.grant}`, padding: "15px 18px", background: "#10141A", fontFamily: FONTS.mono, fontSize: 20, color: palette.text, lineHeight: 1.65 }}>
        <span style={{ color: palette.blue }}>warden wrap</span> --client cursor<br />--server filesystem --policy policy.yaml<br /><span style={{ color: palette.grant }}>--dry-run</span>
      </div>
      <div style={{ marginTop: 20, display: "flex", gap: 12, flexWrap: "wrap" }}><Pill color={palette.grant}>✓ preview</Pill><Pill color={palette.grant}>✓ backup</Pill><Pill color={palette.grant}>✓ reversible</Pill></div>
    </Panel>
    <div style={{ position: "absolute", left: 140, right: 140, bottom: 120, display: "flex", justifyContent: "space-between", gap: 12 }}>
      {clients.map((client, index) => <div key={client} style={{ ...enter(frame, 34 + index * 3, fps), border: `1px solid ${palette.border}`, borderRadius: 999, padding: "16px 22px", fontFamily: FONTS.mono, fontSize: 18, color: palette.text, background: "#171C24" }}>{client}</div>)}
    </div>
  </Stage>;
};

const Gateway: React.FC = () => {
  const frame = useCurrentFrame(); const { fps } = useVideoConfig();
  const flow = ["YOUR AGENT", "WARDEN", "REVIEWED MCP"]; 
  return <Stage accent={palette.blue}>
    <div style={{ position: "absolute", top: 112, left: 0, right: 0, textAlign: "center" }}>
      <div style={enter(frame, 0, fps)}><Eyebrow>03 / ONE BOUNDARY. MANY AGENTS.</Eyebrow></div>
      <h2 style={{ ...enter(frame, 7, fps), margin: "18px 0 0", fontFamily: FONTS.sans, color: palette.text, fontSize: 78, letterSpacing: "-.052em" }}>Bring your agent.<br /><span style={{ color: palette.blue }}>Choose its access.</span></h2>
    </div>
    <div style={{ position: "absolute", left: 165, right: 165, top: 510, display: "flex", alignItems: "center", justifyContent: "space-between" }}>
      {flow.map((name, index) => <React.Fragment key={name}><Panel style={{ width: 380, height: 180, display: "flex", alignItems: "center", justifyContent: "center", textAlign: "center", ...enter(frame, 19 + index * 10, fps) }}><div><div style={{ fontFamily: FONTS.mono, color: index === 1 ? palette.grant : palette.muted, fontSize: 18, letterSpacing: ".16em" }}>{index === 1 ? "POLICY + SANDBOX" : "MCP COMPATIBLE"}</div><div style={{ marginTop: 13, fontFamily: FONTS.sans, fontWeight: 750, fontSize: 31, color: palette.text }}>{name}</div></div></Panel>{index < 2 && <div style={{ ...enter(frame, 28 + index * 10, fps), width: 140, height: 2, background: `linear-gradient(90deg, ${palette.blue}, ${palette.grant})`, position: "relative" }}><span style={{ position: "absolute", right: -4, top: -7, borderLeft: `14px solid ${palette.grant}`, borderTop: "8px solid transparent", borderBottom: "8px solid transparent" }} /></div>}</React.Fragment>)}
    </div>
    <div style={{ position: "absolute", left: 0, right: 0, bottom: 120, textAlign: "center", ...enter(frame, 52, fps) }}><Pill color={palette.gold}>STANDARD STDIO + AUTHENTICATED HTTP</Pill></div>
  </Stage>;
};

const Trust: React.FC = () => {
  const frame = useCurrentFrame(); const { fps } = useVideoConfig();
  return <Stage accent={palette.gold}>
    <div style={{ position: "absolute", left: 150, top: 146, width: 760 }}>
      <div style={enter(frame, 0, fps)}><Eyebrow color={palette.gold}>04 / MAKE TRUST VISIBLE</Eyebrow></div>
      <h2 style={{ ...enter(frame, 7, fps), fontFamily: FONTS.sans, color: palette.text, fontSize: 78, lineHeight: .98, letterSpacing: "-.05em", margin: "24px 0 0" }}>Let your users see<br /><span style={{ color: palette.gold }}>the evidence.</span></h2>
      <p style={{ ...enter(frame, 18, fps), fontFamily: FONTS.sans, color: palette.muted, fontSize: 25, lineHeight: 1.45, marginTop: 30 }}>Run allowed and denied checks. Bind them to the artifact, policy and workflow. Ship a badge they can verify.</p>
    </div>
    <Panel style={{ position: "absolute", top: 155, right: 170, width: 630, padding: 42, ...enter(frame, 16, fps) }}>
      <div style={{ display: "flex", alignItems: "center", gap: 16 }}><LogoMark size={62} /><div style={{ fontFamily: FONTS.mono, fontSize: 18, color: palette.muted }}>SECURED BY</div></div>
      <div style={{ fontFamily: FONTS.sans, color: palette.text, fontWeight: 800, fontSize: 65, letterSpacing: "-.045em", marginTop: 23 }}>Warden</div>
      <div style={{ marginTop: 30, padding: 20, borderRadius: 16, background: `${palette.grant}18`, border: `1px solid ${palette.grant}77`, fontFamily: FONTS.mono, color: palette.grant, fontSize: 22 }}>✓ checks passing</div>
      <div style={{ marginTop: 16, fontFamily: FONTS.mono, fontSize: 16, color: palette.muted }}>artifact · policy · rules · expiry</div>
    </Panel>
    <div style={{ position: "absolute", left: 150, bottom: 120, ...enter(frame, 44, fps) }}><Pill color={palette.gold}>SIGN IT. VERIFY IT. SHARE IT.</Pill></div>
  </Stage>;
};

const Finale: React.FC = () => {
  const frame = useCurrentFrame(); const { fps } = useVideoConfig();
  return <Stage accent={palette.grant}>
    <div style={{ position: "absolute", left: 0, right: 0, top: 160, display: "flex", justifyContent: "center", ...enter(frame, 4, fps) }}><LogoMark size={144} glow /></div>
    <div style={{ position: "absolute", left: 0, right: 0, top: 335, textAlign: "center" }}>
      <div style={{ ...enter(frame, 10, fps), fontFamily: FONTS.sans, color: palette.text, fontWeight: 800, fontSize: 100, letterSpacing: "-.06em" }}>Make MCP <span style={{ color: palette.grant }}>safe to adopt.</span></div>
      <div style={{ ...enter(frame, 20, fps), marginTop: 25, fontFamily: FONTS.sans, color: palette.muted, fontSize: 28 }}>Open-source sandboxing for the agentic ecosystem.</div>
    </div>
    <Panel style={{ position: "absolute", left: 300, right: 300, bottom: 160, padding: "24px 34px", display: "flex", alignItems: "center", justifyContent: "space-between", ...enter(frame, 30, fps) }}>
      <div><div style={{ fontFamily: FONTS.mono, fontSize: 17, color: palette.muted }}>BETA · AVAILABLE AFTER PUBLISH</div><div style={{ fontFamily: FONTS.mono, fontSize: 25, color: palette.text, marginTop: 7 }}>npm install -g warden-sandbox-cli@beta</div></div>
      <Pill color={palette.grant}>github.com/Prof-bilal/Warden</Pill>
    </Panel>
  </Stage>;
};

export const WardenEcosystemLaunch: React.FC = () => (
  <AbsoluteFill style={{ backgroundColor: palette.bg }}>
    <Sequence from={at(0)} durationInFrames={at(9)}><Hook /></Sequence>
    <Sequence from={at(9)} durationInFrames={at(12)}><Packs /></Sequence>
    <Sequence from={at(21)} durationInFrames={at(13)}><Setup /></Sequence>
    <Sequence from={at(34)} durationInFrames={at(12)}><Gateway /></Sequence>
    <Sequence from={at(46)} durationInFrames={at(11)}><Trust /></Sequence>
    <Sequence from={at(57)} durationInFrames={at(9)}><Finale /></Sequence>
  </AbsoluteFill>
);

export { DURATION as ECOSYSTEM_LAUNCH_DURATION, FPS as ECOSYSTEM_LAUNCH_FPS };
