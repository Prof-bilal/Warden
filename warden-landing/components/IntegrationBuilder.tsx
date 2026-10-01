"use client";
import { useState } from "react";
import CodeBlock from "@/components/CodeBlock";
import { shellQuote } from "@/lib/ecosystem";

const field =
  "mt-2 w-full rounded-lg border border-ink-700 bg-ink-950 p-3 text-sm text-paper";
export default function IntegrationBuilder() {
  const [transport, setTransport] = useState("stdio");
  const [policy, setPolicy] = useState("/absolute/policy.yaml");
  const [rulesFile, setRulesFile] = useState("/absolute/rules.json");
  const [argv, setArgv] = useState(
    '["/absolute/node", "/absolute/prepared-server/dist/index.js"]',
  );
  const [tools, setTools] = useState("read_text_file");
  const [constraint, setConstraint] = useState("{}");
  const [error, setError] = useState("");
  let args: string[] = [];
  let constraints: Record<string, Record<string, string[]>> = {};
  let valid = true;
  try {
    args = JSON.parse(argv);
    if (
      !Array.isArray(args) ||
      !args.length ||
      args.some((v) => typeof v !== "string" || !v.trim())
    )
      throw Error();
    constraints = JSON.parse(constraint);
    if (
      !constraints ||
      Array.isArray(constraints) ||
      typeof constraints !== "object"
    )
      throw Error();
    for (const paths of Object.values(constraints)) {
      if (!paths || Array.isArray(paths) || typeof paths !== "object")
        throw Error();
      for (const [pointer, values] of Object.entries(paths)) {
        if (
          !pointer.startsWith("/") ||
          !Array.isArray(values) ||
          !values.length ||
          values.some((v) => typeof v !== "string")
        )
          throw Error();
      }
    }
  } catch {
    valid = false;
  }
  const names = [
    ...new Set(
      tools
        .split(/\n/)
        .map((v) => v.trim())
        .filter(Boolean),
    ),
  ];
  const rules = {
    version: 1,
    tools: Object.fromEntries(
      names.map((name) => [name, { arguments: constraints[name] ?? {} }]),
    ),
    resources: [],
    prompts: [],
    max_bytes: 1048576,
    timeout_seconds: 60,
  };
  const command = `warden ${transport === "stdio" ? "connect" : "serve"} --rules ${shellQuote(rulesFile)} --policy ${shellQuote(policy)}${transport === "http" ? " --token-env WARDEN_GATEWAY_TOKEN --listen 127.0.0.1:8788" : ""} -- ${args.map(shellQuote).join(" ")}`;
  const download = () => {
    try {
      const url = URL.createObjectURL(
        new Blob([JSON.stringify(rules, null, 2) + "\n"], {
          type: "application/json",
        }),
      );
      const a = document.createElement("a");
      a.href = url;
      a.download = "rules.json";
      a.click();
      URL.revokeObjectURL(url);
      setError("");
    } catch {
      setError("Download unavailable. Copy the rules below.");
    }
  };
  return (
    <div className="grid gap-8 lg:grid-cols-2">
      <section className="rounded-2xl border border-ink-700 p-6 sm:p-8">
        <h2 className="text-xl font-semibold">Choose a connection</h2>
        <div className="mt-5 flex gap-3">
          {[
            ["stdio", "Local command"],
            ["http", "HTTP endpoint"],
          ].map(([id, label]) => (
            <button
              key={id}
              aria-pressed={transport === id}
              onClick={() => setTransport(id)}
              className={`rounded-lg border p-3 text-sm ${transport === id ? "border-blueprint text-blueprint" : "border-ink-700 text-muted"}`}
            >
              {label}
            </button>
          ))}
        </div>
        <p className="mt-4 text-sm leading-relaxed text-muted">
          Uses the prepared local server you choose. Your agent connects to
          Warden; Warden starts the server inside its sandbox.
        </p>
        <label className="mt-5 block text-sm">
          Reviewed process policy
          <input
            value={policy}
            onChange={(e) => setPolicy(e.target.value)}
            className={field}
          />
        </label>
        <label className="mt-5 block text-sm">
          Rules file location
          <input
            value={rulesFile}
            onChange={(e) => setRulesFile(e.target.value)}
            className={field}
          />
        </label>
        <label className="mt-5 block text-sm">
          Prepared command and arguments (JSON array)
          <textarea
            value={argv}
            onChange={(e) => setArgv(e.target.value)}
            rows={3}
            className={`${field} font-mono`}
          />
        </label>
        <label className="mt-5 block text-sm">
          Allowed tools (one per line)
          <textarea
            value={tools}
            onChange={(e) => setTools(e.target.value)}
            rows={3}
            className={field}
          />
        </label>
        <label className="mt-5 block text-sm">
          Exact argument limits (optional)
          <textarea
            value={constraint}
            onChange={(e) => setConstraint(e.target.value)}
            rows={3}
            className={`${field} font-mono`}
          />
          <span className="mt-2 block text-xs leading-relaxed text-muted">
            Example: {`{"read_text_file":{"/path":["/selected/hello.txt"]}}`}.
            Missing or different values are denied.
          </span>
        </label>
      </section>
      <section className="min-w-0 rounded-2xl border border-ink-700 p-6 sm:p-8">
        <h2 className="text-xl font-semibold">Review your rules</h2>
        <p className="mt-3 text-sm leading-relaxed text-muted">
          Tools outside this list are denied. Resources and prompts start
          disabled. Keep normal agent approval controls.
        </p>
        {valid ? (
          <>
            <CodeBlock className="mt-5">
              {JSON.stringify(rules, null, 2)}
            </CodeBlock>
            <button
              onClick={download}
              className="mt-4 rounded-lg bg-blueprint px-4 py-3 text-sm font-semibold text-ink-950"
            >
              Download rules
            </button>
            <h3 className="mb-3 mt-8 text-sm font-semibold">Run locally</h3>
            <CodeBlock>{command}</CodeBlock>
            {transport === "http" && (
              <p className="mt-4 text-sm leading-relaxed text-muted">
                Set WARDEN_GATEWAY_TOKEN to a random secret of at least 32
                characters in your environment. Configure the same bearer
                credential in your client. Local address:
                http://127.0.0.1:8788/mcp. Public hosting requires HTTPS and the
                deployment guide.
              </p>
            )}
          </>
        ) : (
          <p role="status" className="mt-5 text-progress">
            Enter a valid command array and exact string argument limits.
          </p>
        )}
        {error && (
          <p role="status" className="mt-3 text-progress">
            {error}
          </p>
        )}
        <p className="mt-6 text-xs leading-relaxed text-muted">
          Rules and paths stay in this page. No credentials or computer
          configuration are uploaded. Commands use macOS/Linux quoting; use
          structured command/args for SDKs or Windows.
        </p>
      </section>
    </div>
  );
}
