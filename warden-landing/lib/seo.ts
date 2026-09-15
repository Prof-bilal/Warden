import type { Metadata } from "next";

export const SITE_URL = "https://warden.blog";
export const SITE_NAME = "Warden";

const DEFAULT_OG_IMAGE = "/og-image.png";

type TitleInput = string | { absolute: string };

interface CreateMetadataOptions {
  title: TitleInput;
  description: string;
  /** Path starting with "/", e.g. "/blog" or "/" for the homepage. */
  path: string;
  /**
   * Path to a real OG image asset. Defaults to the site-wide OG image.
   * Pass null to omit og/twitter images entirely — use when a route-level
   * opengraph-image file convention supplies the image.
   */
  image?: string | null;
  type?: "website" | "article";
  publishedTime?: string;
  /** Only set when the content was actually updated. */
  modifiedTime?: string;
  /** Author names as plain strings (used for article:author meta). */
  authors?: string[];
}

/**
 * Builds complete page metadata (canonical, Open Graph, Twitter) from
 * per-page inputs so every page gets unique, correct metadata.
 */
export function createMetadata({
  title,
  description,
  path,
  image,
  type = "website",
  publishedTime,
  modifiedTime,
  authors,
}: CreateMetadataOptions): Metadata {
  const resolvedTitle = typeof title === "string" ? title : title.absolute;
  const url = path === "/" ? SITE_URL : `${SITE_URL}${path}`;
  const ogImage = image === null ? null : (image ?? DEFAULT_OG_IMAGE);

  return {
    title,
    description,
    alternates: {
      canonical: url,
    },
    openGraph: {
      title: resolvedTitle,
      description,
      url,
      siteName: SITE_NAME,
      type,
      ...(publishedTime ? { publishedTime } : {}),
      ...(modifiedTime ? { modifiedTime } : {}),
      ...(authors ? { authors } : {}),
      ...(ogImage
        ? {
            images: [
              {
                url: ogImage,
                width: 1200,
                height: 630,
                alt: resolvedTitle,
              },
            ],
          }
        : {}),
    },
    twitter: {
      card: "summary_large_image",
      title: resolvedTitle,
      description,
      ...(ogImage ? { images: [ogImage] } : {}),
    },
  };
}
