export default function Footer() {
  return (
    <footer className="border-t border-ink-800">
      <div className="mx-auto flex max-w-content flex-col items-start justify-between gap-4 px-6 py-10 text-[0.8125rem] text-muted md:flex-row md:items-center">
        <span>warden — MIT licensed</span>
        <div className="flex gap-6">
          <a href="/#how-it-works" className="transition-colors hover:text-paper">
            How it works
          </a>
          <a href="/#how-to-use" className="transition-colors hover:text-paper">
            How to use
          </a>
          <a href="/about" className="transition-colors hover:text-paper">
            About
          </a>
          <a href="/docs" className="transition-colors hover:text-paper">
            Docs
          </a>
          <a
            href="/docs/roadmap"
            className="transition-colors hover:text-paper"
          >
            Roadmap
          </a>
          <a href="https://github.com/Prof-bilal/Warden" className="transition-colors hover:text-paper">
            GitHub
          </a>
        </div>
      </div>
    </footer>
  );
}
