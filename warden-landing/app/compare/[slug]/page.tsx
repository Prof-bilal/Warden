import { notFound } from "next/navigation";
import TopicArticle from "@/components/TopicArticle";
import { getTopic, topicMetadata, topicStaticParams } from "@/lib/topics";

export function generateStaticParams() {
  return topicStaticParams("compare");
}

export async function generateMetadata({
  params,
}: {
  params: Promise<{ slug: string }>;
}) {
  const { slug } = await params;
  return topicMetadata("compare", slug);
}

export default async function ComparePage({
  params,
}: {
  params: Promise<{ slug: string }>;
}) {
  const { slug } = await params;
  const topic = getTopic("compare", slug);
  if (!topic) notFound();
  return <TopicArticle topic={topic} />;
}
