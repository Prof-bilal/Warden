"use client";

import { useState } from "react";
import Link from "next/link";
import { Github, Menu, X } from "lucide-react";

const links = [
  { href: "/#how-it-works", label: "How it works" },
  { href: "/#how-to-use", label: "How to use" },
  { href: "/#backends", label: "Backends" },
  { href: "/#proof", label: "Proof" },
  { href: "/#compatibility", label: "Compatibility" },
  { href: "/testing", label: "Testing" },
  { href: "/about", label: "About" },
  { href: "/docs", label: "Docs" },
];

export default function Nav() {
  const [open, setOpen] = useState(false);

  return (
    <header className="border-b border-ink-700">
      <div className="mx-auto flex max-w-content items-center justify-between px-6 py-5">
        <Link href="#" className="flex items-center gap-2.5">
          <svg width="22" height="22" viewBox="0 0 22 22" fill="none" aria-hidden="true">
            <rect x="1" y="1" width="6" height="20" rx="1" stroke="#6E93E8" strokeWidth="1.4" />
            <rect x="15" y="1" width="6" height="20" rx="1" stroke="#6E93E8" strokeWidth="1.4" />
            <path d="M7 11H15" stroke="#3FB27E" strokeWidth="1.4" strokeDasharray="1.5 2.2" />
          </svg>
          <span className="text-[1.0625rem] font-medium text-paper">warden</span>
        </Link>

        <nav className="hidden items-center gap-8 md:flex">
          {links.map((l) => (
            <a
              key={l.href}
              href={l.href}
              className="text-[0.9375rem] text-muted transition-colors hover:text-paper"
            >
              {l.label}
            </a>
          ))}
        </nav>

        <div className="flex items-center gap-3">
          <a
            href="https://github.com/Prof-bilal/Warden"
            className="flex items-center gap-2 rounded-sm border border-ink-600 px-3.5 py-1.5 text-[0.875rem] text-paper transition-colors hover:border-blueprint hover:text-blueprint"
          >
            <Github size={15} />
            <span className="hidden sm:inline">GitHub</span>
          </a>
          <button
            onClick={() => setOpen(!open)}
            className="flex items-center justify-center rounded-sm border border-ink-600 p-1.5 text-muted transition-colors hover:text-paper md:hidden"
            aria-label={open ? "Close menu" : "Open menu"}
          >
            {open ? <X size={18} /> : <Menu size={18} />}
          </button>
        </div>
      </div>

      {open && (
        <nav className="border-t border-ink-700 px-6 py-4 md:hidden">
          <div className="flex flex-col gap-3">
            {links.map((l) => (
              <a
                key={l.href}
                href={l.href}
                onClick={() => setOpen(false)}
                className="text-[0.9375rem] text-muted transition-colors hover:text-paper"
              >
                {l.label}
              </a>
            ))}
          </div>
        </nav>
      )}
    </header>
  );
}
