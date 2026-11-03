import type { ReactNode } from "react";
import SectionMarker from "@/components/bp/SectionMarker";

export default function PageHeader({
  index = "01",
  eyebrow,
  title,
  description,
  children,
}: {
  index?: string;
  eyebrow?: string;
  title: ReactNode;
  description?: ReactNode;
  children?: ReactNode;
}) {
  return (
    <section className="border-b border-hair">
      <div className="px-6 py-14 md:py-20">
        <SectionMarker index={index} label={eyebrow} />
        <h1 className="mt-5 max-w-3xl font-mono text-[1.9rem] font-semibold leading-[1.12] tracking-[-0.02em] text-offwhite md:text-[2.5rem]">
          {title}
        </h1>
        {description && (
          <p className="mt-4 max-w-2xl font-sans text-[14.5px] leading-[1.75] text-offwhite/45">
            {description}
          </p>
        )}
        {children}
      </div>
    </section>
  );
}
