import type { Metadata } from "next";

/** First markdown H1, which is also the visible page heading. */
export function extractDocHeadline(markdown: string, fallback: string): string {
  const line = markdown.split("\n").find((l) => l.startsWith("# "));
  return line ? line.replace(/^#\s*/, "").trim() : fallback;
}

function toPlain(text: string): string {
  return text
    .replace(/!\[[^\]]*\]\([^)]*\)/g, "")
    .replace(/\[([^\]]+)\]\([^)]*\)/g, "$1")
    .replace(/\*\*([^*]+)\*\*/g, "$1")
    .replace(/`([^`]+)`/g, "$1")
    .replace(/\s+/g, " ")
    .trim();
}

function clip(text: string, max = 160): string {
  if (text.length <= max) return text;
  const clipped = text.slice(0, max - 1).replace(/\s+\S*$/, "").trim();
  return `${clipped}…`;
}

/**
 * Opening prose after the H1, clipped for a meta description.
 * Fenced code is ignored so a diagram cannot become the description.
 */
export function extractDocDescription(markdown: string, headline: string): string {
  const body = markdown
    .replace(/^#[^\n]*\n+/, "")
    .replace(/```[\s\S]*?```/g, "\n");

  const parts: string[] = [];
  for (const raw of body.split("\n")) {
    const line = raw.trim();
    if (!line) {
      if (parts.length) break;
      continue;
    }
    if (line.startsWith("#") || line.startsWith("|")) {
      if (parts.length) break;
      continue;
    }

    const cleaned = toPlain(
      line.replace(/^>\s?/, "").replace(/^[-*]\s+/, "").replace(/^\d+\.\s+/, ""),
    );
    if (cleaned) parts.push(cleaned);
  }

  const plain = parts.join(" ");
  if (plain.length >= 40) return clip(plain);
  return `${headline}. Documentation for the Warden sandbox runtime.`;
}

/** Title tag that matches the H1, without appending a second "Warden". */
export function docTitle(headline: string): Metadata["title"] {
  if (/warden/i.test(headline)) return { absolute: headline };
  return headline;
}

/** Open Graph / Twitter title. Same rule: one "Warden", not two. */
export function docSocialTitle(headline: string): string {
  if (/warden/i.test(headline)) return headline;
  return `${headline} | Warden`;
}

/**
 * Pull question headings (###) and the paragraph under each one.
 * Used as FAQPage structured data for answer engines.
 */
export function extractFaqItems(markdown: string): { question: string; answer: string }[] {
  const items: { question: string; answer: string }[] = [];
  let question: string | null = null;
  let answerLines: string[] = [];
  let inCode = false;

  const flush = () => {
    if (!question) return;
    const answer = toPlain(answerLines.join(" "));
    if (answer) items.push({ question, answer: answer.slice(0, 600) });
    answerLines = [];
  };

  for (const line of markdown.split("\n")) {
    if (line.startsWith("```")) {
      inCode = !inCode;
      continue;
    }
    if (inCode) continue;

    const heading = line.match(/^#{2,3}\s+(.+)/);
    if (heading) {
      flush();
      question = line.startsWith("### ")
        ? heading[1].replace(/`/g, "").trim()
        : null;
      continue;
    }

    if (question && line.trim()) answerLines.push(line.trim());
  }

  flush();
  return items;
}
