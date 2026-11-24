import Link from "next/link";

const POLICY = `agent:
  workspace: ./project
  redaction:
    apply: review
    env: [OPENAI_API_KEY]
    files:
      - path: .env
        format: dotenv
        fields: [DATABASE_URL]`;

export default function AgentWorkspace() {
  return (
    <section id="agent-workspace" className="home-section home-workspace">
      <div>
        <h2 className="accent-heading">Keep credentials out of the agent workspace</h2>
        <p className="home-intro">
          Give Claude, Codex, OpenCode, or another local coding tool a temporary
          project where selected secrets are replaced with placeholders.
          Review the sanitized diff before any edits reach your real project.
        </p>
        <Link className="home-section-link" href="/docs/agent-workspace">
          Read the workspace guide <span aria-hidden="true">→</span>
        </Link>
      </div>

      <div className="home-terminal">
        <div className="home-terminal-label">
          <span>agent-policy.yaml</span>
          <span>Review required</span>
        </div>
        <pre><code>{POLICY}</code></pre>
      </div>
    </section>
  );
}
