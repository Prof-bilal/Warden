import type { Metadata, Viewport } from "next";
import Script from "next/script";
import { GeistSans } from "geist/font/sans";
import { GeistMono } from "geist/font/mono";
import { Newsreader, JetBrains_Mono, Space_Grotesk } from "next/font/google";
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

const wordmark = Space_Grotesk({
  subsets: ["latin"],
  variable: "--font-wordmark",
  display: "swap",
  weight: ["500", "600", "700"],
});

export const viewport: Viewport = {
  themeColor: "#10141a",
};

export const metadata: Metadata = {
  metadataBase: new URL(SITE_URL),
  title: {
    default: "Warden — Permissions for Local AI Tools",
    template: "%s | Warden",
  },
  description:
    "Run local AI tools and MCP servers with explicit file, network, environment, and resource permissions using Warden's open-source sandbox.",
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
    title: "Warden — Permissions for Local AI Tools",
    description:
      "Give local AI tools and MCP servers only the access they need. Open source, local, and fail-closed.",
    images: [
      {
        url: "/og-image.png",
        width: 1200,
        height: 630,
        alt: "Warden",
        type: "image/png",
      },
    ],
  },
  twitter: {
    card: "summary_large_image",
    title: "Warden — Permissions for Local AI Tools",
    description:
      "Give local AI tools and MCP servers only the access they need.",
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
      className={`${GeistSans.variable} ${GeistMono.variable} ${newsreader.variable} ${heroMono.variable} ${wordmark.variable}`}
    >
      <head>
        {/* Google Tag Manager */}
        <script
          id="gtm-script"
          dangerouslySetInnerHTML={{
            __html: `(function(w,d,s,l,i){w[l]=w[l]||[];w[l].push({'gtm.start':new Date().getTime(),event:'gtm.js'});var f=d.getElementsByTagName(s)[0],j=d.createElement(s),dl=l!='dataLayer'?'&l='+l:'';j.async=true;j.src='https://www.googletagmanager.com/gtm.js?id='+i+dl;f.parentNode.insertBefore(j,f);})(window,document,'script','dataLayer','GTM-P9BH885W');`,
          }}
        />
        {/* End Google Tag Manager */}
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
      <body className="font-sans antialiased">
        {/* Google Tag Manager (noscript) */}
        <noscript>
          <iframe
            src="https://www.googletagmanager.com/ns.html?id=GTM-P9BH885W"
            height="0"
            width="0"
            style={{ display: "none", visibility: "hidden" }}
          />
        </noscript>
        {/* End Google Tag Manager (noscript) */}
        {/* Google tag (gtag.js) */}
        <Script
          src="https://www.googletagmanager.com/gtag/js?id=G-GC7ENDGEP3"
          strategy="afterInteractive"
        />
        <Script id="google-analytics" strategy="afterInteractive">
          {`
            window.dataLayer = window.dataLayer || [];
            function gtag(){dataLayer.push(arguments);}
            gtag('js', new Date());
            gtag('config', 'G-GC7ENDGEP3');
          `}
        </Script>
        {/* End Google tag (gtag.js) */}
        {children}
      </body>
    </html>
  );
}
