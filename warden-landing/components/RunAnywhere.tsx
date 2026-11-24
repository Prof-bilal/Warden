import Link from "next/link";

const platforms = [
  ["Linux", "Ubuntu, Debian, Fedora, and more"],
  ["macOS", "macOS 13+ on Apple Silicon"],
  ["Windows", "Windows 11 x64"],
  ["Docker", "Docker or Docker Compose"],
];

export default function RunAnywhere() {
  return (
    <section className="home-section">
      <h2 className="accent-heading">Run Warden on your machine</h2>
      <p className="home-intro">
        Warden uses OS-native sandboxing and runs locally.
      </p>
      <dl className="home-grid home-platforms">
        {platforms.map(([name, detail]) => (
          <div key={name}>
            <dt>{name}</dt>
            <dd>{detail}</dd>
          </div>
        ))}
      </dl>
      <Link className="home-section-link" href="/compatibility">
        Check platform compatibility <span aria-hidden="true">→</span>
      </Link>
    </section>
  );
}
