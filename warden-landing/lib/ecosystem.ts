import catalog from "../../warden-starter/warden/internal/packs/catalog.json";

export const policyPacks = catalog;
export const clients = [
  {
    id: "cline",
    name: "Cline CLI / extension",
    format: "JSON",
    scopes: ["user", "explicit"],
    config:
      "CLI: ~/.cline/mcp.json. For extensions select the file opened by Configure MCP Servers with --config.",
    source: "https://docs.cline.bot/mcp/mcp-overview",
  },
  {
    id: "cascade",
    name: "Legacy Cascade",
    format: "JSON",
    scopes: ["explicit"],
    config:
      "Select the active mcp_config.json through the host UI and supply --config. The Devin Local agent uses a different configuration.",
    source: "https://docs.devin.ai/desktop/cascade/mcp",
  },
  {
    id: "claude-desktop",
    name: "Claude Desktop",
    format: "JSON",
    scopes: ["user"],
    config:
      "macOS: ~/Library/Application Support/Claude/claude_desktop_config.json · Windows: %APPDATA%/Claude/claude_desktop_config.json",
    source:
      "https://modelcontextprotocol.io/docs/develop/connect-local-servers",
  },
  {
    id: "claude-code",
    name: "Claude Code",
    format: "JSON",
    scopes: ["project", "user"],
    config: "Project: .mcp.json · User: ~/.claude.json (top-level servers)",
    source: "https://code.claude.com/docs/en/mcp",
  },
  {
    id: "cursor",
    name: "Cursor",
    format: "JSON",
    scopes: ["project", "user"],
    config: "Project: .cursor/mcp.json · User: ~/.cursor/mcp.json",
    source: "https://cursor.com/docs/mcp",
  },
  {
    id: "codex",
    name: "Codex",
    format: "TOML",
    scopes: ["project", "user"],
    config: "Project: .codex/config.toml · User: ~/.codex/config.toml",
    source: "https://learn.chatgpt.com/docs/extend/mcp?surface=cli",
  },
  {
    id: "vscode",
    name: "VS Code / Copilot",
    format: "JSONC",
    scopes: ["project"],
    config:
      "Project: .vscode/mcp.json · Profile: select your file with --config",
    source:
      "https://code.visualstudio.com/docs/agent-customization/mcp-servers",
  },
  {
    id: "gemini",
    name: "Gemini CLI",
    format: "JSON",
    scopes: ["project", "user"],
    config: "Project: .gemini/settings.json · User: ~/.gemini/settings.json",
    source:
      "https://google-gemini.github.io/gemini-cli/docs/tools/mcp-server.html",
  },
  {
    id: "generic",
    name: "Other MCP host",
    format: "JSON",
    scopes: ["explicit"],
    config:
      "Select an existing JSON config with mcpServers or servers. SDKs can use the generated command and argument array directly.",
    source:
      "https://modelcontextprotocol.io/specification/2026-07-28/basic/transports",
  },
];
export type Client = (typeof clients)[number];
export type PolicyPack = (typeof policyPacks)[number];
export const shellQuote = (value: string) =>
  `'${value.replaceAll("'", `'"'"'`)}'`;

export function packCommand(
  id: string,
  runtime: string,
  data: string,
  output: string,
) {
  const pack = policyPacks.find((p) => p.id === id);
  return `warden packs generate ${shellQuote(id)} --runtime ${shellQuote(runtime)}${pack?.pathMode !== "none" ? ` --path ${shellQuote(data)}` : ""} --output ${shellQuote(output)}`;
}

export function wrapCommand(
  client: string,
  scope: string,
  server: string,
  policy: string,
  config: string,
  action = "--dry-run",
  usePolicyCommand = false,
) {
  return `warden wrap --client ${shellQuote(client)} --server ${shellQuote(server)} --policy ${shellQuote(policy)}${scope !== "explicit" ? ` --scope ${shellQuote(scope)}` : ""}${config ? ` --config ${shellQuote(config)}` : ""}${usePolicyCommand ? " --use-policy-command" : ""} ${action}`;
}
