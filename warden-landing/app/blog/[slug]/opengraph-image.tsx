import { ImageResponse } from "next/og";
import { getArticleBySlug, getCategoryBySlug } from "@/lib/blog";

export const size = { width: 1200, height: 630 };
export const contentType = "image/png";
export const alt = "Article preview — Warden";

export default async function ArticleOgImage({
  params,
}: {
  params: Promise<{ slug: string }>;
}) {
  const { slug } = await params;
  const article = getArticleBySlug(slug);
  const category = article ? getCategoryBySlug(article.categorySlug) : null;
  const title = article?.title ?? "Warden";

  return new ImageResponse(
    (
      <div
        style={{
          width: "100%",
          height: "100%",
          display: "flex",
          flexDirection: "column",
          justifyContent: "space-between",
          backgroundColor: "#10141a",
          color: "#E8EBEF",
          padding: 72,
        }}
      >
        <div style={{ display: "flex", alignItems: "center", gap: 24 }}>
          <div
            style={{
              width: 56,
              height: 56,
              borderRadius: 28,
              backgroundColor: "#16a34a",
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
            }}
          >
            <div
              style={{
                width: 20,
                height: 28,
                backgroundColor: "#0a0f14",
                borderRadius: 4,
              }}
            />
          </div>
          <div style={{ fontSize: 32, color: "#8D95A5" }}>warden</div>
          {category && (
            <div
              style={{
                fontSize: 26,
                color: "#3B82F6",
                border: "1px solid #233043",
                borderRadius: 999,
                padding: "6px 20px",
              }}
            >
              {category.name}
            </div>
          )}
        </div>

        <div
          style={{
            display: "flex",
            fontSize: title.length > 60 ? 56 : 68,
            fontWeight: 700,
            lineHeight: 1.15,
            maxWidth: 1050,
          }}
        >
          {title}
        </div>

        <div style={{ display: "flex", fontSize: 28, color: "#8D95A5" }}>
          warden.blog
        </div>
      </div>
    ),
    size
  );
}
