import type { ReactNode } from "react";
import Nav from "@/components/Nav";
import Footer from "@/components/Footer";

export default function PageShell({
  children,
  glow = true,
}: {
  children: ReactNode;
  glow?: boolean;
}) {
  return (
    <main className="relative min-h-screen bg-void text-offwhite">
      <div className="bp-grid-bg pointer-events-none absolute inset-0" aria-hidden />
      {glow && (
        <div
          className="pointer-events-none absolute inset-x-0 top-0 h-[480px] bg-[radial-gradient(ellipse_60%_100%_at_50%_0%,rgba(94,240,138,0.06),transparent_70%)]"
          aria-hidden
        />
      )}
      <Nav />

      <div className="bp-shell relative z-10">
        {children}
        <Footer />
      </div>
    </main>
  );
}
