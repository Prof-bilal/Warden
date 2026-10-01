"use client";
import { useState } from "react";
import Link from "next/link";
import {
  ArrowLeft,
  ArrowRight,
  Check,
  Terminal,
  FolderLock,
  ShieldCheck,
} from "lucide-react";
import CodeBlock from "@/components/CodeBlock";
import {
  clients,
  policyPacks,
  packCommand,
  wrapCommand,
  shellQuote,
} from "@/lib/ecosystem";

const inputClass =
  "mt-2 w-full rounded-lg border border-ink-700 bg-ink-950 px-4 py-3 text-sm text-paper placeholder:text-muted/60";
export default function SetupWizard({
  initialClient = "",
  initialPack = "filesystem",
}: {
  initialClient?: string;
  initialPack?: string;
}) {
  const [clientId, setClientId] = useState(
    clients.some((c) => c.id === initialClient) ? initialClient : "cursor",
  );
  const [packId, setPackId] = useState(
    policyPacks.some((p) => p.id === initialPack) ? initialPack : "filesystem",
  );
  const [step, setStep] = useState(0);
  const [scope, setScope] = useState(
    clients.find((c) => c.id === initialClient)?.scopes[0] ?? "project",
  );
  const [server, setServer] = useState("");
  const [runtimeDir, setRuntimeDir] = useState("");
  const [dataDir, setDataDir] = useState("");
  const [policyFile, setPolicyFile] = useState("");
  const [configFile, setConfigFile] = useState("");
  const [shell, setShell] = useState("posix");
  const [reviewed, setReviewed] = useState(false);
  const client = clients.find((c) => c.id === clientId)!;
  const pack = policyPacks.find((p) => p.id === packId)!;
  const isNpmProfile = pack.artifact.startsWith("@");
  const actualScope = client.scopes.includes(scope) ? scope : client.scopes[0];
  const valid = !!(
    server.trim() &&
    runtimeDir.trim() &&
    policyFile.trim() &&
    (pack.pathMode === "none" || dataDir.trim()) &&
    (actualScope !== "explicit" || configFile.trim())
  );
  // Commands are built from a structured argv; quoting prevents shell
  // interpretation of user paths, including quotes, dollar signs and spaces.
  const posixGenerate = packCommand(packId, runtimeDir, dataDir, policyFile);
  const posixWrap = wrapCommand(
    clientId,
    actualScope,
    server,
    policyFile,
    configFile,
    "--dry-run",
    isNpmProfile,
  );
  function formatCommand(command: string, argv: string[]) {
    if (shell === "posix") return command;
    return `warden ${argv.map((v) => `'${v.replaceAll("'", "''")}'`).join(" ")}`;
  }
  const prepare = formatCommand(
    `warden packs prepare ${shellQuote(packId)} --output ${shellQuote(runtimeDir)}`,
    ["packs", "prepare", packId, "--output", runtimeDir],
  );
  const generate = formatCommand(posixGenerate, [
    "packs",
    "generate",
    packId,
    "--runtime",
    runtimeDir,
    ...(pack.pathMode !== "none" ? ["--path", dataDir] : []),
    "--output",
    policyFile,
  ]);
  const common = [
    "--client",
    clientId,
    "--server",
    server,
    "--policy",
    policyFile,
    ...(actualScope !== "explicit" ? ["--scope", actualScope] : []),
    ...(configFile ? ["--config", configFile] : []),
    ...(isNpmProfile ? ["--use-policy-command"] : []),
  ];
  const preview = formatCommand(posixWrap, ["wrap", ...common, "--dry-run"]);
  const apply = formatCommand(
    wrapCommand(
      clientId,
      actualScope,
      server,
      policyFile,
      configFile,
      "--yes",
      isNpmProfile,
    ),
    ["wrap", ...common, "--yes"],
  );
  const undoArgs = [
    "unwrap",
    "--client",
    clientId,
    "--server",
    server,
    ...(actualScope !== "explicit" ? ["--scope", actualScope] : []),
    ...(configFile ? ["--config", configFile] : []),
    "--yes",
  ];
  const undo =
    shell === "posix"
      ? `warden ${undoArgs.map((v) => `'${v.replaceAll("'", `'"'"'`)}'`).join(" ")}`
      : formatCommand("", undoArgs);
  const chooseClient = (id: string) => {
    setClientId(id);
    setScope(clients.find((c) => c.id === id)!.scopes[0]);
    setReviewed(false);
  };
  return (
    <div className="grid items-start gap-8 lg:grid-cols-[1fr_300px]">
      <section className="min-w-0 rounded-2xl border border-ink-700 bg-ink-900">
        <ol
          aria-label="Setup progress"
          className="grid grid-cols-3 border-b border-ink-700"
        >
          {["Choose client", "Choose access", "Review & set up"].map(
            (label, i) => (
              <li
                key={label}
                className={`flex items-center gap-2 border-r border-ink-700 px-3 py-5 text-xs last:border-r-0 sm:px-5 ${step === i ? "text-blueprint" : "text-muted"}`}
                aria-current={step === i ? "step" : undefined}
              >
                <span
                  className={`flex h-6 w-6 shrink-0 items-center justify-center rounded-full ${step >= i ? "bg-blueprint/15 text-blueprint" : "bg-ink-800"}`}
                >
                  {step > i ? <Check size={12} /> : i + 1}
                </span>
                <span>{label}</span>
              </li>
            ),
          )}
        </ol>
        <div className="p-6 sm:p-8">
          {step === 0 && (
            <>
              <h2 className="text-2xl font-semibold">
                Where does your agent run?
              </h2>
              <p className="mt-3 text-sm leading-relaxed text-muted">
                Wrap an existing local MCP server. Any host that launches a
                stdio command can use Warden&apos;s execution boundary.
              </p>
              <div className="mt-7 grid gap-3 sm:grid-cols-2">
                {clients.map((c) => (
                  <button
                    key={c.id}
                    onClick={() => chooseClient(c.id)}
                    aria-pressed={clientId === c.id}
                    className={`flex items-center justify-between rounded-xl border p-5 text-left transition-colors ${clientId === c.id ? "border-blueprint bg-blueprint/10" : "border-ink-700 hover:border-ink-600"}`}
                  >
                    <span>
                      <span className="block text-sm font-medium">
                        {c.name}
                      </span>
                      <span className="mt-2 block text-xs text-muted">
                        {c.format} ·{" "}
                        {c.id === "generic"
                          ? "Explicit config / SDK"
                          : "Local stdio adapter"}
                      </span>
                    </span>
                    {clientId === c.id && (
                      <Check size={16} className="text-blueprint" />
                    )}
                  </button>
                ))}
              </div>
              <p className="mt-6 text-xs leading-relaxed text-muted">
                Hosted agents and remote-only connections need the standard HTTP
                gateway planned in batch 4.{" "}
                <Link
                  href="/compatibility"
                  className="text-blueprint underline underline-offset-4"
                >
                  See integration coverage
                </Link>
                .
              </p>
            </>
          )}
          {step === 1 && (
            <>
              <h2 className="text-2xl font-semibold">
                Choose the server&apos;s access
              </h2>
              <p className="mt-3 text-sm leading-relaxed text-muted">
                Prepare the pinned upstream separately, then run it with a
                narrow policy. Four npm profiles have a sandboxed preparation
                command.
              </p>
              <label className="mt-6 block text-sm">
                Policy profile
                <select
                  value={packId}
                  onChange={(e) => {
                    setPackId(e.target.value);
                    setReviewed(false);
                  }}
                  className={inputClass}
                >
                  {policyPacks.map((p) => (
                    <option key={p.id} value={p.id}>
                      {p.name} · {p.profile}
                    </option>
                  ))}
                </select>
              </label>
              <div className="mt-5 rounded-lg border border-progress/30 bg-progress-subtle p-4 text-xs leading-relaxed text-muted">
                <span className="text-progress">Candidate · </span>
                {pack.note}
                <span className="mt-2 block break-all font-mono">
                  {pack.artifact}@{pack.version}
                </span>
              </div>
              <div className="mt-6 grid gap-5 sm:grid-cols-2">
                <label className="text-sm">
                  Existing MCP server name
                  <input
                    className={inputClass}
                    value={server}
                    onChange={(e) => setServer(e.target.value)}
                    placeholder="e.g. filesystem"
                    autoComplete="off"
                  />
                  <span className="mt-2 block text-xs text-muted">
                    Match the name already in your client config.
                  </span>
                </label>
                <label className="text-sm">
                  Configuration scope
                  <select
                    className={inputClass}
                    value={actualScope}
                    onChange={(e) => setScope(e.target.value)}
                  >
                    {client.scopes.map((s) => (
                      <option key={s} value={s}>
                        {s === "explicit"
                          ? "Explicit file"
                          : s === "project"
                            ? "This project"
                            : "Your user account"}
                      </option>
                    ))}
                  </select>
                </label>
                <label className="text-sm sm:col-span-2">
                  Dedicated runtime directory
                  <input
                    className={inputClass}
                    value={runtimeDir}
                    onChange={(e) => setRuntimeDir(e.target.value)}
                    placeholder="/absolute/path/to/prepared-server"
                    autoComplete="off"
                  />
                  <span className="mt-2 block text-xs text-muted">
                    {isNpmProfile
                      ? "Use a new directory for preparation, or an existing prepared installation of the recorded version."
                      : "Prepare the pinned local binary/Python environment here first. Your existing launcher must use it."}{" "}
                    Runtime files are read only.
                  </span>
                </label>
                {pack.pathMode !== "none" && (
                  <label className="text-sm sm:col-span-2">
                    {pack.pathMode === "write"
                      ? "Memory storage directory"
                      : "Data / repository directory"}
                    <input
                      className={inputClass}
                      value={dataDir}
                      onChange={(e) => setDataDir(e.target.value)}
                      placeholder="/absolute/path/to/selected-folder"
                      autoComplete="off"
                    />
                    <span className="mt-2 block text-xs text-muted">
                      {pack.pathMode === "write"
                        ? "Writes are allowed here. Keep it separate from the runtime."
                        : "Read only. Select just the directory your task needs."}
                    </span>
                  </label>
                )}
                <label className="text-sm sm:col-span-2">
                  New policy file location
                  <input
                    className={inputClass}
                    value={policyFile}
                    onChange={(e) => setPolicyFile(e.target.value)}
                    placeholder="/absolute/path/to/warden-policy.yaml"
                    autoComplete="off"
                  />
                </label>
                <label className="text-sm sm:col-span-2">
                  {actualScope === "explicit"
                    ? "Configuration file (required)"
                    : "Configuration file (optional override)"}
                  <input
                    className={inputClass}
                    value={configFile}
                    onChange={(e) => setConfigFile(e.target.value)}
                    placeholder="Leave empty to use the selected scope"
                    autoComplete="off"
                  />
                </label>
                <label className="text-sm sm:col-span-2">
                  Your terminal
                  <select
                    className={inputClass}
                    value={shell}
                    onChange={(e) => setShell(e.target.value)}
                  >
                    <option value="posix">macOS / Linux shell</option>
                    <option value="powershell">Windows PowerShell</option>
                  </select>
                </label>
              </div>
              {!valid && (
                <p role="status" className="mt-5 text-xs text-progress">
                  Fill in the server name, runtime directory, policy location,
                  and any required data/config paths.
                </p>
              )}
            </>
          )}
          {step === 2 && (
            <>
              <h2 className="text-2xl font-semibold">
                Review, preview, then apply
              </h2>
              <p className="mt-3 text-sm leading-relaxed text-muted">
                Run these commands locally with the CLI from this checkout. A
                website cannot inspect or change your host configuration.
              </p>
              <div className="mt-7 space-y-6">
                {isNpmProfile && (
                  <div>
                    <h3 className="mb-3 text-sm font-medium">
                      Prepare the pinned runtime (once)
                    </h3>
                    <CodeBlock>{prepare}</CodeBlock>
                    <p className="mt-2 text-xs leading-relaxed text-muted">
                      Installs inside the selected sandbox, with scripts
                      disabled and registry-only egress. Requires Node/npm and a
                      new output directory. If this exact version is already
                      prepared, continue below.
                    </p>
                  </div>
                )}
                <div>
                  <h3 className="mb-3 text-sm font-medium">
                    1. Generate the candidate policy
                  </h3>
                  <CodeBlock>{generate}</CodeBlock>
                  <p className="mt-2 text-xs text-muted">
                    Refuses whole-home/root grants and existing output files.
                    Review the generated policy before continuing.
                  </p>
                </div>
                <div>
                  <h3 className="mb-3 text-sm font-medium">
                    2. Preview the launcher change
                  </h3>
                  <CodeBlock>{preview}</CodeBlock>
                  <p className="mt-2 text-xs text-muted">
                    No file writes or server launch. Preview redacts command
                    arguments and credentials.
                    {isNpmProfile &&
                      " The explicit --use-policy-command option selects the prepared local launcher and keeps the original command for undo."}
                  </p>
                </div>
                <label className="flex items-start gap-3 rounded-lg border border-ink-700 p-4 text-sm leading-relaxed">
                  <input
                    type="checkbox"
                    checked={reviewed}
                    onChange={(e) => setReviewed(e.target.checked)}
                    className="mt-1 accent-blueprint"
                  />
                  <span>
                    I have reviewed the policy grants and the local preview.
                  </span>
                </label>
                {reviewed ? (
                  <div>
                    <h3 className="mb-3 text-sm font-medium">
                      3. Apply with a backup
                    </h3>
                    <CodeBlock>{apply}</CodeBlock>
                    <p className="mt-2 text-xs leading-relaxed text-muted">
                      The CLI probes sandbox readiness before writing, keeps a
                      private backup, and leaves remote connections untouched.
                      Restart the client, then check an allowed and blocked
                      task.
                    </p>
                  </div>
                ) : (
                  <p className="text-sm text-muted">
                    Review the files and preview to reveal the apply command.
                  </p>
                )}
                <div className="border-t border-ink-700 pt-6">
                  <h3 className="mb-3 text-sm font-medium">
                    Undo the launcher change
                  </h3>
                  <CodeBlock>{undo}</CodeBlock>
                  <p className="mt-2 text-xs text-muted">
                    Keeps unrelated edits. Refuses to overwrite a launcher
                    changed after setup.
                  </p>
                </div>
              </div>
            </>
          )}
          <div className="mt-8 flex items-center justify-between border-t border-ink-700 pt-6">
            <button
              disabled={step === 0}
              onClick={() => {
                setReviewed(false);
                setStep(step - 1);
              }}
              className="flex items-center gap-2 text-sm text-muted disabled:opacity-30"
            >
              <ArrowLeft size={16} />
              Back
            </button>
            {step < 2 && (
              <button
                disabled={step === 1 && !valid}
                onClick={() => setStep(step + 1)}
                className="flex items-center gap-2 rounded-lg bg-blueprint px-5 py-3 text-sm font-semibold text-ink-950 disabled:cursor-not-allowed disabled:opacity-40"
              >
                {step === 0 ? "Choose access" : "Review setup"}
                <ArrowRight size={16} />
              </button>
            )}
            {step === 2 && (
              <Link
                href="/docs/ecosystem-setup"
                className="text-sm text-blueprint"
              >
                Verification guide →
              </Link>
            )}
          </div>
        </div>
      </section>
      <aside className="space-y-5 lg:sticky lg:top-24">
        <div className="rounded-2xl border border-ink-700 p-6">
          <p className="text-xs uppercase tracking-widest text-muted">
            Your setup
          </p>
          <h2 className="mt-4 text-xl font-medium">{client.name}</h2>
          <p className="mt-2 text-sm text-blueprint">
            {pack.name} · {actualScope} scope
          </p>
          <dl className="mt-6 space-y-5 text-xs leading-relaxed">
            <div>
              <dt className="flex items-center gap-2 text-muted">
                <FolderLock size={14} />
                File access
              </dt>
              <dd className="mt-2">
                {pack.pathMode === "read"
                  ? "Selected data directory: read only"
                  : pack.pathMode === "write"
                    ? "Selected storage directory: read & write"
                    : "No data directory"}
                <span className="mt-1 block text-muted">
                  Prepared runtime: read only
                </span>
              </dd>
            </div>
            <div>
              <dt className="text-muted">Network hosts</dt>
              <dd className="mt-2 break-all">
                {pack.hosts.join(", ") || "None"}
              </dd>
            </div>
            <div>
              <dt className="text-muted">Environment names</dt>
              <dd className="mt-2 break-all">
                {pack.env.join(", ") || "None"}
              </dd>
            </div>
          </dl>
        </div>
        <div className="rounded-2xl bg-ink-900 p-6">
          <Terminal size={18} className="text-blueprint" />
          <p className="mt-4 text-xs leading-relaxed text-muted">
            {client.config}
          </p>
          <a
            href={client.source}
            target="_blank"
            rel="noopener noreferrer"
            className="mt-4 block text-xs text-blueprint"
          >
            Official client documentation ↗
          </a>
        </div>
        <div className="flex items-start gap-3 px-1 text-xs leading-relaxed text-muted">
          <ShieldCheck size={18} className="shrink-0 text-grant" />
          <p>
            Protection applies to local servers launched through Warden.{" "}
            <Link href="/protection" className="text-blueprint">
              Read the boundary
            </Link>
            .
          </p>
        </div>
      </aside>
    </div>
  );
}
