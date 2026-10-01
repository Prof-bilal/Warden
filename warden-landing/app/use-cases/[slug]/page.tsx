import { notFound } from "next/navigation";
import TopicArticle from "@/components/TopicArticle";
import { getTopic, topicMetadata, topicStaticParams } from "@/lib/topics";

export function generateStaticParams() {
  return topicStaticParams("use-cases");
}

export async function generateMetadata({
  params,
}: {
  params: Promise<{ slug: string }>;
}) {
  const { slug } = await params;
  return topicMetadata("use-cases", slug);
}

export default async function UseCasePage({
  params,
}: {
  params: Promise<{ slug: string }>;
}) {
  const { slug } = await params;
  const topic = getTopic("use-cases", slug);
  if (!topic) notFound();
  return <TopicArticle topic={topic} />;
}
