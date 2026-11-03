import type { Metadata } from "next";
import PageShell from "@/components/PageShell";
import JsonLd from "@/components/JsonLd";
import PageHeader from "@/components/bp/PageHeader";
import { createMetadata, SITE_URL } from "@/lib/seo";
import { breadcrumbSchema } from "@/lib/schema";

export const metadata: Metadata = createMetadata({
  title: "Privacy policy",
  description:
    "What Warden's website collects and does not collect: no third-party analytics, no advertising trackers, and anonymous install counting with no IP addresses stored.",
  path: "/privacy",
});

export default function Privacy() {
  return (
    <PageShell>
      <JsonLd
        data={breadcrumbSchema([
          { name: "Home", url: `${SITE_URL}/` },
          { name: "Privacy", url: `${SITE_URL}/privacy` },
        ])}
      />

      <PageHeader eyebrow="Legal" title="Privacy policy" />

      <div className="px-6 py-12 md:py-16">
        <div className="prose-warden mx-auto">
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
    </PageShell>
  );
}
