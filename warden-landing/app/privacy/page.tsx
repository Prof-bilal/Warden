import type { Metadata } from "next";
import Nav from "@/components/Nav";
import Footer from "@/components/Footer";
import JsonLd from "@/components/JsonLd";
import { createMetadata, SITE_URL } from "@/lib/seo";
import { breadcrumbSchema } from "@/lib/schema";

export const metadata: Metadata = createMetadata({
  title: "Privacy",
  description:
    "What Warden's website collects and does not collect: no third-party analytics, no advertising trackers, and anonymous install counting with no IP addresses stored.",
  path: "/privacy",
});

export default function Privacy() {
  return (
    <main className="min-h-screen bg-ink-950">
      <JsonLd
        data={breadcrumbSchema([
          { name: "Home", url: `${SITE_URL}/` },
          { name: "Privacy", url: `${SITE_URL}/privacy` },
        ])}
      />
      <Nav />
      <div className="mx-auto max-w-[48rem] px-6 pb-20 pt-10 md:pt-16">
        <h1 className="text-[2.5rem] font-semibold leading-[1.1] tracking-[-0.02em] text-paper">
          Privacy
        </h1>

        <div className="prose-warden mt-10">
          <h2>Analytics</h2>
          <p>
            This website does not use third-party analytics services and does
            not set advertising or tracking cookies. We do not profile
            visitors.
          </p>

          <h2>Install counting</h2>
          <p>
            When someone installs the <code>warden-sandbox-cli</code> npm
            package, the install script sends a single anonymous event to this
            website. The event contains only the package name, package
            version, operating system, CPU architecture, Node.js version, and
            a timestamp. No IP addresses, user-agent strings, or request
            headers are stored. Events are appended to a server-side log that
            is not publicly accessible.
          </p>

          <h2>Public statistics</h2>
          <p>
            Star and download counts shown on this site are fetched
            server-side from the public GitHub and npm APIs. Requests to those
            services originate from our server, not your browser.
          </p>

          <h2>Open-source project</h2>
          <p>
            Warden is an open-source project. Contributions, issues, and
            discussions happen publicly on{" "}
            <a href="https://github.com/Prof-bilal/Warden">GitHub</a> and in
            the project Discord. Content you post there is governed by those
            platforms.
          </p>

          <h2>Contact</h2>
          <p>
            Questions about this policy can be raised via{" "}
            <a href="https://github.com/Prof-bilal/Warden/issues">GitHub issues</a>{" "}
            or the project <a href="/contact">contact channels</a>.
          </p>
        </div>
      </div>
      <Footer />
    </main>
  );
}
