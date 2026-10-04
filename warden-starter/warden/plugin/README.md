# Warden — ChatGPT / Codex MCP Plugin

Expose Warden's existing sandbox runtime to ChatGPT, Codex, and any MCP
client. This is a **thin integration layer**: all enforcement stays in
Warden's policy engine, sandbox backends, and audit log. The plugin never
simulates a security result and never bypasses Warden.

> Run AI and MCP workloads in a sandbox you control.

## Tools

| Tool | Purpose | Side effects |
|---|---|---|
| `inspect_policy` | Show what a policy allows and denies (filesystem / network / env) | Read-only |
| `run_sandbox` | Execute a command through the real `warden run` under an explicit policy | Executes the command, sandboxed |
| `trace_execution` | Summarize a run's audit events: allowed vs denied, with denied resources | Read-only |
| `explain_denial` | Explain why Warden blocked an access, from the audit evidence | Read-only; never grants |
| `generate_policy` | Propose restrictive policy YAML from explicit requirements | Proposal only; never written or applied |

`run_sandbox` is declared `confirmation: always` in the plugin manifest, so
ChatGPT asks the user before every execution. There is no tool that widens a
policy or grants access; changing a policy is done by the user editing the
YAML file themselves.

## Security model

- **Deny by default.** Everything not explicitly granted in the policy is
  blocked. The plugin never implements `if unsure → allow`.
- **No fabricated results.** Status, exit codes, and allow/deny counts come
  from real process execution and Warden's own JSONL audit log
  (`${XDG_STATE_HOME:-~/.local/state}/warden/audit.jsonl`).
- **No secret exposure.** Denied resources are reported by path/host only —
  never contents. `explain_denial` for `.env`, SSH keys, or cloud credentials
  recommends keeping them blocked.
- **Bounded output.** Process output is capped (16 KB), event lists capped
  (200), and denial samples deduplicated (10).
- **Fail closed.** If no sandbox backend is available, `warden run` refuses
  to execute; the plugin reports the refusal instead of running unsandboxed.

## UI

**Icon** — `assets/icon.svg` (source) with rendered `icon.png` (512px,
submission-ready), `icon-192.png`, and `icon-64.png`. Minimal shield mark in
the plugin's dark palette (`#0d1117`), cyan brand (`#58a6ff`), green allow
(`#3fb950`), red deny (`#f85149`). Regenerate after editing the SVG:

```bash
cd plugin/assets
rsvg-convert -w 512 -h 512 icon.svg -o icon.png
rsvg-convert -w 192 -h 192 icon.svg -o icon-192.png
rsvg-convert -w 64  -h 64  icon.svg -o icon-64.png
```

Three focused MCP Apps widget resources (dark developer-tool aesthetic,
green = allowed, red = denied):

- `ui://warden/policy` — policy view (grants + deny-by-default markers)
- `ui://warden/execution` — run status, allowed/blocked counts, denied sample
- `ui://warden/trace` — audit trace summary

Structured results are attached via `structuredContent` and the
`ui/resourceUri` / `openai/outputTemplate` metadata keys so both the MCP Apps
standard and OpenAI Apps SDK hosts can render them. Hosts without widget
support fall back to the plain-text rendering, which carries the same
information.

## Setup

Build the CLI, then register the MCP server with your client:

```bash
cd warden-starter/warden
go build -o warden ./cmd/warden
```

Client config (`plugin/mcp-config.json`):

```json
{
  "mcpServers": {
    "warden": {
      "command": "warden",
      "args": ["mcp"],
      "env": {}
    }
  }
}
```

Example session:

```
User: What filesystem access does this server have?
  → inspect_policy { "policy_path": "policy.yaml" }

User: Run this MCP server with Warden.
  → run_sandbox { "policy_path": "policy.yaml",
                  "command": ["npx", "-y", "@modelcontextprotocol/server-everything"] }
  (ChatGPT asks for confirmation first)

User: Show me what it tried to access.
  → trace_execution { "run_id": "run-…" }

User: Why was ~/.ssh/id_rsa denied?
  → explain_denial { "resource": "~/.ssh/id_rsa", "run_id": "run-…" }

User: This server only needs to read ./data and call api.example.com.
  → generate_policy { "filesystem_read": ["./data"],
                      "network_allow": ["api.example.com"] }
```

## Local testing

```bash
cd warden-starter/warden
go test ./internal/mcpserver/        # plugin tool tests
go test ./...                        # full suite (existing tests included)

# Manual smoke test over stdio:
go build -o warden ./cmd/warden
printf '%s\n' \
  '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' \
  '{"jsonrpc":"2.0","id":2,"method":"tools/list"}' \
  | ./warden mcp
```

Test scenarios covered in `internal/mcpserver/mcpserver_test.go`: policy
inspection, safe execution, malformed policy rejection, backend validation,
missing-policy rejection (deny-by-default intact), trace reading, denial
explanation (policy-rule and audit-event grounded), generate_policy
propose-only guarantee, unknown tools, bounded output, widget registry, and
the full JSON-RPC stdio loop end-to-end.

## Packaging for directory submission

1. Build release binaries: `make build-all`
2. Fill in real `contact_email` and privacy/legal URLs in
   `plugin/manifest.json` (currently placeholders). The icon is done:
   `plugin/assets/icon.png` is referenced by `logo_url` and ships with the
   repo; once the raw GitHub URL above resolves, no further icon work is
   needed.
3. Submit per the OpenAI plugin review flow, pointing at `plugin/manifest.json`
   with the MCP stdio transport above.

## Known limitations

- Requires the `warden` binary on PATH (or `WARDEN_MCP_BIN` set) on the same
  host as the client; sandboxing needs a native backend (`bwrap`,
  `sandbox-exec`, AppContainer) or Docker.
- `run_sandbox` is synchronous and capped by Warden's own `limits.timeout_s`;
  long-lived MCP servers should be run via `warden serve`/`gateway` outside
  the plugin.
- Denied-file *contents* are never available (by design); the trace shows
  paths and hosts only.
- The audit-log offset registry holds the last 64 runs; older `run_id`s
  expire (pass a log path instead).
- Warden does not guarantee complete security; the plugin says so and never
  claims otherwise.
