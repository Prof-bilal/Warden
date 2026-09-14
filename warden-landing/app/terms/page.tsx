import type { Metadata } from "next";
import Nav from "@/components/Nav";
import Footer from "@/components/Footer";
import JsonLd from "@/components/JsonLd";
import { createMetadata, SITE_URL } from "@/lib/seo";
import { breadcrumbSchema } from "@/lib/schema";

export const metadata: Metadata = createMetadata({
  title: "Terms",
  description:
    "Terms of use for the Warden website and its published content. Warden itself is open source under the MIT license.",
  path: "/terms",
});

export default function Terms() {
  return (
    <main className="min-h-screen bg-ink-950">
      <JsonLd
        data={breadcrumbSchema([
          { name: "Home", url: `${SITE_URL}/` },
          { name: "Terms", url: `${SITE_URL}/terms` },
        ])}
      />
      <Nav />
      <div className="mx-auto max-w-[48rem] px-6 pb-20 pt-10 md:pt-16">
        <h1 className="text-[2.5rem] font-semibold leading-[1.1] tracking-[-0.02em] text-paper">
          Terms
        </h1>

        <div className="prose-warden mt-10">
          <h2>Website content</h2>
          <p>
            Content on this website is provided for informational purposes.
            While we aim for accuracy, the software it describes is in active
            development and details may change; the project documentation and
            source code are the authoritative reference.
          </p>

          <h2>The software</h2>
          <p>
            Warden is open-source software licensed under the MIT license. Use
            of the software is governed by that license, not by this page. See
            the{" "}
            <a href="https://github.com/Prof-bilal/Warden">
              project repository
            </a>{" "}
            for the full license text.
          </p>

          <h2>External links</h2>
          <p>
            Links to third-party sites (GitHub, npm, Discord, Product Hunt)
            are provided for convenience. We do not control those services and
            are not responsible for their content or policies.
          </p>

          <h2>Contact</h2>
          <p>
            Questions about these terms can be raised via{" "}
            <a href="https://github.com/Prof-bilal/Warden/issues">GitHub issues</a>{" "}
            or the project <a href="/contact">contact channels</a>.
          </p>
        </div>
      </div>
      <Footer />
    </main>
  );
}
