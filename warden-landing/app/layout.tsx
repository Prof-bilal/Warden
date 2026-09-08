import type { Metadata } from "next";
import { GeistSans } from "geist/font/sans";
import { GeistMono } from "geist/font/mono";
import { Newsreader } from "next/font/google";
import "./globals.css";

const newsreader = Newsreader({
  subsets: ["latin"],
  variable: "--font-display",
  display: "swap",
  weight: ["400", "500"],
});

const SITE_URL = "https://warden-six-rouge.vercel.app";

export const metadata: Metadata = {
  metadataBase: new URL(SITE_URL),
  title: {
    default: "Warden — A Sandbox Runtime for MCP Servers",
    template: "%s | Warden",
  },
  description:
    "Warden runs MCP servers in a restricted sandbox, so a server only ever gets the files, hosts, and environment variables you explicitly grant it. Open-source, fail-closed, audited.",
  keywords: [
    "MCP sandbox",
    "MCP server security",
    "Model Context Protocol",
    "AI tooling security",
    "sandbox runtime",
    "MCP server isolation",
    "bubblewrap",
    "seatbelt",
    "AppContainer",
    "Warden",
    "open source",
    "developer tools",
  ],
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
    title: "Warden — A Sandbox Runtime for MCP Servers",
    description:
      "Run MCP servers in a restricted sandbox. Only the files, hosts, and env vars you explicitly grant are accessible. Open-source, fail-closed, audited.",
    images: [
      {
        url: `${SITE_URL}/og-image.png`,
        width: 1200,
        height: 630,
        alt: "Warden — A Sandbox Runtime for MCP Servers",
        type: "image/png",
      },
    ],
  },
  twitter: {
    card: "summary_large_image",
    title: "Warden — A Sandbox Runtime for MCP Servers",
    description:
      "Run MCP servers in a restricted sandbox. Only the files, hosts, and env vars you explicitly grant are accessible.",
    images: [`${SITE_URL}/og-image.png`],
    creator: "@Prof-bilal",
  },
  icons: {
    icon: "/icon.png",
    shortcut: "/favicon.ico",
    apple: "/apple-touch-icon.png",
  },
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const websiteSchema = {
    "@context": "https://schema.org",
    "@type": "WebSite",
    name: "Warden",
    url: SITE_URL,
    description:
      "A sandbox runtime for MCP servers. Run third-party code with first-party trust.",
    publisher: {
      "@type": "Organization",
      name: "Warden",
      url: SITE_URL,
      logo: {
        "@type": "ImageObject",
        url: `${SITE_URL}/icon.png`,
      },
    },
    potentialAction: {
      "@type": "SearchAction",
      target: {
        "@type": "EntryPoint",
        urlTemplate: `${SITE_URL}/docs?q={search_term_string}`,
      },
      "query-input": "required name=search_term_string",
    },
  };

  const organizationSchema = {
    "@context": "https://schema.org",
    "@type": "Organization",
    name: "Warden",
    url: SITE_URL,
    logo: {
      "@type": "ImageObject",
      url: `${SITE_URL}/icon.png`,
    },
    sameAs: ["https://github.com/Prof-bilal/Warden"],
    description:
      "Open-source sandbox runtime for MCP servers. Run third-party AI tooling code safely.",
  };

  return (
    <html
      lang="en"
      className={`${GeistSans.variable} ${GeistMono.variable} ${newsreader.variable}`}
    >
      <head>
        <meta name="theme-color" content="#10141a" />
        <meta name="google-site-verification" content="Q2f7-V2KDOxOTZEYUebS_woBqRa1L-kgFr-Q5zBN7ks" />
        <script
          type="application/ld+json"
          dangerouslySetInnerHTML={{ __html: JSON.stringify(websiteSchema) }}
        />
        <script
          type="application/ld+json"
          dangerouslySetInnerHTML={{
            __html: JSON.stringify(organizationSchema),
          }}
        />
      </head>
      <body className="font-sans antialiased">{children}</body>
    </html>
  );
}
