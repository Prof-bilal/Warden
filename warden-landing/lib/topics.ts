import fs from "fs";
import path from "path";
import matter from "gray-matter";
import { createMetadata } from "@/lib/seo";

export type TopicSection = "guides" | "compare" | "use-cases";

export interface TopicLink {
  href: string;
  label: string;
}

export interface TopicFaq {
  question: string;
  answer: string;
}

export interface Topic {
  section: TopicSection;
  slug: string;
  title: string;
  description: string;
  /** ISO date the page was published. */
  date: string;
  /** Direct answer shown under the H1 and used as the page summary. */
  answer: string;
  tldr: string[];
  faqs: TopicFaq[];
  related: TopicLink[];
  body: string;
  path: string;
}

const TOPICS_DIR = path.join(process.cwd(), "content", "topics");

function asLinks(value: unknown): TopicLink[] {
  if (!Array.isArray(value)) return [];
  return value.flatMap((item) => {
    if (!item || typeof item !== "object") return [];
    const href = "href" in item ? String(item.href) : "";
    const label = "label" in item ? String(item.label) : "";
    if (!href.startsWith("/") || !label) return [];
    return [{ href, label }];
  });
}

function asFaqs(value: unknown): TopicFaq[] {
  if (!Array.isArray(value)) return [];
  return value.flatMap((item) => {
    if (!item || typeof item !== "object") return [];
    const question = "question" in item ? String(item.question).trim() : "";
    const answer = "answer" in item ? String(item.answer).trim() : "";
    if (!question || !answer) return [];
    return [{ question, answer }];
  });
}

function parseTopic(fileName: string): Topic | null {
  const raw = fs.readFileSync(path.join(TOPICS_DIR, fileName), "utf-8");
  const { data, content } = matter(raw);
  const section = data.section;
  const slug = typeof data.slug === "string" ? data.slug : fileName.replace(/\.md$/, "");
  if (section !== "guides" && section !== "compare" && section !== "use-cases") return null;
  if (!data.title || !data.description || !data.date || !data.answer) return null;

  return {
    section,
    slug,
    title: String(data.title),
    description: String(data.description),
    date: String(data.date),
    answer: String(data.answer),
    tldr: Array.isArray(data.tldr) ? data.tldr.map(String) : [],
    faqs: asFaqs(data.faqs),
    related: asLinks(data.related),
    body: content.trim(),
    path: `/${section}/${slug}`,
  };
}

export function getAllTopics(): Topic[] {
  let files: string[] = [];
  try {
    files = fs.readdirSync(TOPICS_DIR).filter((f) => f.endsWith(".md"));
  } catch {
    return [];
  }
  return files
    .map(parseTopic)
    .filter((topic): topic is Topic => topic !== null)
    .sort((a, b) => a.title.localeCompare(b.title));
}

export function getTopicsBySection(section: TopicSection): Topic[] {
  return getAllTopics().filter((topic) => topic.section === section);
}

export function getTopic(section: TopicSection, slug: string): Topic | null {
  return getAllTopics().find((topic) => topic.section === section && topic.slug === slug) ?? null;
}

export function topicStaticParams(section: TopicSection) {
  return getTopicsBySection(section).map((topic) => ({ slug: topic.slug }));
}

export function topicMetadata(section: TopicSection, slug: string) {
  const topic = getTopic(section, slug);
  if (!topic) return { title: "Not Found" };
  const title = /warden/i.test(topic.title) ? { absolute: topic.title } : topic.title;
  return createMetadata({
    title,
    description: topic.description,
    path: topic.path,
    type: "article",
    publishedTime: topic.date,
    authors: ["Abdullah Bilal"],
  });
}
