import type { ReactNode } from "react";

export default function PageHeader({
  eyebrow,
  title,
  description,
  children,
}: {
  eyebrow?: string;
  title: ReactNode;
  description?: ReactNode;
  children?: ReactNode;
}) {
  return (
    <section className="border-b border-hair">
      <div className="px-6 py-14">
        {eyebrow && <p className="text-sm text-muted">{eyebrow}</p>}
        <h1 className="accent-heading mt-4 max-w-3xl">
          {title}
        </h1>
        {description && (
          <p className="mt-4 max-w-2xl text-base text-paper">
            {description}
          </p>
        )}
        {children}
      </div>
    </section>
  );
}
