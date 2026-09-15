import type { Metadata, Viewport } from "next";
import { GeistSans } from "geist/font/sans";
import { GeistMono } from "geist/font/mono";
import { Newsreader, JetBrains_Mono } from "next/font/google";
import { SITE_URL } from "@/lib/seo";
import { organizationSchema, websiteSchema } from "@/lib/schema";
import "./globals.css";

const newsreader = Newsreader({
  subsets: ["latin"],
  variable: "--font-display",
  display: "swap",
  weight: ["400", "500"],
});

const heroMono = JetBrains_Mono({
  subsets: ["latin"],
  variable: "--font-hero-mono",
  display: "swap",
  weight: ["400", "500", "600", "700", "800"],
});

export const viewport: Viewport = {
  themeColor: "#10141a",
};

export const metadata: Metadata = {
  metadataBase: new URL(SITE_URL),
  title: {
    default: "WardenSandbox Runtime for MCP Servers",
    template: "%s | Warden",
  },
  description:
    "Warden runs MCP servers in a restricted sandbox. Servers only get the files, hosts, and env vars you explicitly grant. Open-source, fail-closed, audited.",
  authors: [{ name: "Prof-bilal", url: "https://github.com/Prof-bilal" }],
  creator: "Prof-bilal",
  publisher: "Prof-bilal",
  robots: {
    index: true,
    follow: true,
    googleBot: {
      index: true,
      follow: true,
      "max-video-preview": -1,
      "max-image-preview": "large",
      "max-snippet": -1,
    },
  },
  alternates: {
    canonical: SITE_URL,
  },
  openGraph: {
    type: "website",
    locale: "en_US",
    url: SITE_URL,
    siteName: "Warden",
    title: "WardenSandbox Runtime for MCP Servers",
    description:
      "Run MCP servers in a restricted sandbox. Only the files, hosts, and env vars you explicitly grant are accessible. Open-source, fail-closed, audited.",
    images: [
      {
        url: "/og-image.png",
        width: 1200,
        height: 630,
        alt: "WardenSandbox Runtime for MCP Servers",
        type: "image/png",
      },
    ],
  },
  twitter: {
    card: "summary_large_image",
    title: "WardenSandbox Runtime for MCP Servers",
    description:
      "Run MCP servers in a restricted sandbox. Only the files, hosts, and env vars you explicitly grant are accessible.",
    images: ["/og-image.png"],
    creator: "@Prof-bilal",
  },
  icons: {
    icon: [
      { url: "/favicon.svg", type: "image/svg+xml" },
      { url: "/favicon-32.png", type: "image/png", sizes: "32x32" },
      { url: "/icon-192.png", type: "image/png", sizes: "192x192" },
    ],
    shortcut: "/favicon.ico",
    apple: "/apple-touch-icon.png",
  },
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html
      lang="en"
      className={`${GeistSans.variable} ${GeistMono.variable} ${newsreader.variable} ${heroMono.variable}`}
    >
      <head>
        <meta
          name="google-site-verification"
          content="Q2f7-V2KDOxOTZEYUebS_woBqRa1L-kgFr-Q5zBN7ks"
        />
        <link
          rel="alternate"
          type="application/rss+xml"
          title="Warden Blog"
          href="/feed.xml"
        />
        <script
          type="application/ld+json"
          dangerouslySetInnerHTML={{ __html: JSON.stringify(websiteSchema()) }}
        />
        <script
          type="application/ld+json"
          dangerouslySetInnerHTML={{ __html: JSON.stringify(organizationSchema()) }}
        />
      </head>
      <body className="font-sans antialiased">{children}</body>
    </html>
  );
}
