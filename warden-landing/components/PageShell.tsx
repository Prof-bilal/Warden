import type { ReactNode } from "react";

export default function PageShell({ children }: { children: ReactNode }) {
  return <main id="main-content" className="site-main">{children}</main>;
}
