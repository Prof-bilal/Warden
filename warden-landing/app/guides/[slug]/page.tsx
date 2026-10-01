import { notFound } from "next/navigation";
import TopicArticle from "@/components/TopicArticle";
import { getTopic, topicMetadata, topicStaticParams } from "@/lib/topics";

export function generateStaticParams() {
  return topicStaticParams("guides");
}

export async function generateMetadata({
  params,
}: {
  params: Promise<{ slug: string }>;
}) {
  const { slug } = await params;
  return topicMetadata("guides", slug);
}

export default async function GuidePage({
  params,
}: {
  params: Promise<{ slug: string }>;
}) {
  const { slug } = await params;
  const topic = getTopic("guides", slug);
  if (!topic) notFound();
  return <TopicArticle topic={topic} />;
}
