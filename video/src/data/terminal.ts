/**
 * Terminal content types used by the Terminal component.
 *
 * Every scripted block must mirror real Warden output formats from
 * docs/cli.md / docs/quickstart.md. No invented commands.
 */

export type Tone = "text" | "muted" | "faint" | "green" | "red" | "cyan" | "amber" | "heading";

export interface TermSegment {
  text: string;
  tone?: Tone;
  /** Draw an highlight box around this segment (sensitive-file callout). */
  box?: boolean;
}

export type TermLine =
  | { kind: "cmd"; segments: TermSegment[]; typeFrom?: number; typeSpeed?: number }
  | { kind: "out"; text: string; tone?: Tone; gapBefore?: number; highlight?: boolean }
  | { kind: "out-block"; lines: string[]; tone?: Tone; gapBefore?: number };

export interface TermScript {
  /** Total time budget for the whole script, in seconds (scene-local). */
  duration: number;
  /** Frames to hold after the last line before scene end. */
  tailHold?: number;
  lines: TermLine[];
}

/** Progress helper: distributes typing of each cmd across the script. */
export const typePlan = (script: TermScript) => {
  const cmds = script.lines.filter((l) => l.kind === "cmd").length;
  const outs = script.lines.length - cmds;
  return { cmds, outs };
};
