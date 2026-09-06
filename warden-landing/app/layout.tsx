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

export const metadata: Metadata = {
  title: "Warden — a sandbox runtime for MCP servers",
  description:
    "Warden runs MCP servers in a restricted sandbox, so a server only ever gets the files, hosts, and environment variables you explicitly grant it.",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html
      lang="en"
      className={`${GeistSans.variable} ${GeistMono.variable} ${newsreader.variable}`}
    >
      <body className="font-sans antialiased">{children}</body>
    </html>
  );
}
