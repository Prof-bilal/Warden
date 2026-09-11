# Warden Client Proxy

The Warden Client Proxy sits between MCP clients (Claude Desktop, Cursor, ChatGPT) and MCP servers (local or remote), providing security filtering, auditing, and sensitive pattern blocking.

## Overview

Traditional MCP setup:
```
Claude Desktop ──────► Remote MCP Server (mcp.github.com)
     │                        │
   All secrets            All requests
   exposed               unfiltered
```

With Warden Client Proxy:
```
Claude Desktop ──► Warden Proxy ──► Remote MCP Server
     │                  │                   │  
 Only allowed       Filters &           Approved
   secrets          audits all         requests only
                    requests
```

## Key Features

- **Secret Protection**: Only explicitly allowed environment variables are accessible
- **Tool Filtering**: Restrict which MCP tools can be called
- **Pattern Blocking**: Block requests/responses containing sensitive patterns (API keys, tokens, etc.)
- **Audit Logging**: Complete logging of all MCP communications
- **Transport Support**: stdio subprocesses **and** remote HTTP/SSE upstreams (Streamable HTTP). Plain `http://` is loopback-only; remote upstreams must use `https://` (TLS verified)

## Quick Start

### 1. Create MCP Proxy Policy

Any MCP server works: a local stdio command, or a remote Streamable HTTP/SSE
endpoint (directly, or bridged via `mcp-remote` if you prefer a stdio hop):

```yaml
# mcp-proxy-policy.yaml
mcp:
  # Option A — remote HTTPS upstream (Streamable HTTP / SSE):
  upstream: "https://mcp.github.com/mcp"

  # Option B — local stdio command (or a stdio bridge to a remote server):
  # upstream: "stdio:npx -y mcp-remote https://mcp.github.com/mcp"

  allow_tools:
    - "list_repositories"
    - "get_file"
    - "search_code"

  deny_patterns:
    - "ghp_[A-Za-z0-9]{36}"        # GitHub tokens
    - "sk-[A-Za-z0-9]{48}"         # OpenAI keys
    - "AKIA[A-Z0-9]{16}"           # AWS keys

env:
  allow: ["PATH", "HOME", "GITHUB_TOKEN"]  # stdio subprocess gets only these (deny-by-default)
```

### 2. Start the Proxy

```bash
warden proxy --policy mcp-proxy-policy.yaml --listen localhost:8765
```

### 3. Configure Your Client

The proxy speaks newline-delimited JSON-RPC over TCP (stdio-style framing).
Point your MCP client at the proxy through a stdio-to-TCP bridge, or use
clients that can connect directly. Example with `socat`/`ncat` for a
manual test:

```bash
# Send a JSON-RPC message to the proxy from the command line
echo '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' | \
  timeout 2 bash -c 'cat > /dev/tcp/127.0.0.1/8765 & cat < /dev/tcp/127.0.0.1/8765'
```

> The proxy's *client-facing* listener is the same for every upstream type;
> what changed is the *upstream* side: `https://` upstreams are now contacted
> over real HTTP/SSE instead of being rejected.

## Policy Configuration

### MCP Section

```yaml
mcp:
  # Required: Upstream MCP server. Three forms are accepted:
  #   "stdio:<command>"          — local subprocess bridge
  #   "https://host/path"        — remote Streamable HTTP upstream (TLS verified)
  #   "http://127.0.0.1:PORT"    — plain HTTP allowed for loopback dev servers only
  upstream: "https://mcp.github.com/mcp"
  
  # Optional: Allowed tool names (deny-by-default if specified)
  allow_tools:
    - "list_repositories"
    - "get_file"
    - "create_issue"
  
  # Optional: Patterns to block in payloads
  deny_patterns:
    - "ghp_[A-Za-z0-9]{36}"        # GitHub personal access tokens
    - "sk-[A-zA-Z0-9]{48}"         # OpenAI API keys
    - "xoxb-[0-9]{12}-[0-9]{12}-[a-zA-Z0-9]{24}"  # Slack tokens
  
  # Optional: Max payload size (KB) to prevent DoS
  max_payload_kb: 1024
  
  # Optional: Log all requests (careful - may log sensitive data)
  audit_requests: false
```

### Standard Sandbox Sections

The proxy also supports standard Warden policy sections for stdio servers:

```yaml
# filesystem/network/limits do NOT sandbox the stdio subprocess today —
# see Security Notes below.
filesystem:
  read: ["."]
  write: ["./tmp"]

network:
  allow: ["api.github.com"]

env:
  allow: ["GITHUB_TOKEN", "PATH"]  # the ONLY section enforced for the subprocess
```

> ⚠️ **Honest security model (current)**: Warden filters and audits every
> JSON-RPC message in both directions, and the stdio subprocess receives only
> the variables listed in `env.allow` (deny-by-default). The subprocess is
> **not** sandboxed — its filesystem and network access are unrestricted.
> The `filesystem`/`network` sections above are not enforced by the proxy.

## Security Benefits

### Secret Protection

**Without Warden Proxy:**
```javascript
// All environment variables accessible to MCP server
process.env.GITHUB_TOKEN    // ✓ Intended
process.env.AWS_SECRET_KEY  // ❌ Exposed!
process.env.DATABASE_URL    // ❌ Exposed!
```

**With Warden Proxy:**
```javascript
// Only explicitly allowed variables accessible
process.env.GITHUB_TOKEN    // ✓ Allowed in policy
process.env.AWS_SECRET_KEY  // ❌ Blocked by proxy
process.env.DATABASE_URL    // ❌ Blocked by proxy
```

### Pattern-Based Blocking

The proxy scans all MCP messages and blocks those containing sensitive patterns:

```json
// This request would be blocked:
{
  "method": "tools/create_issue", 
  "params": {
    "title": "Bug report",
    "body": "API key: ghp_1234567890abcdef1234567890abcdef12"
  }
}
// Blocked: Contains GitHub token pattern
```

### Tool Restriction

```yaml
mcp:
  allow_tools: ["list_repos", "get_file"]
  # create_issue, delete_repo, etc. are blocked
```

## Transport Types

### HTTP/HTTPS Remote Servers (implemented)

```yaml
mcp:
  upstream: "https://mcp.github.com/api"
```

The proxy POSTs each filtered JSON-RPC message upstream (Streamable HTTP
convention: `Accept: application/json, text/event-stream`) and relays the
response through the same inbound filter. TLS certificates are verified
(TLS 1.2+ minimum); a plain `http://` upstream is accepted only for loopback
hosts (`127.0.0.1`, `localhost`), so dev servers work while remote traffic is
never sent unencrypted.

### Server-Sent Events (SSE) (implemented)

```yaml
mcp:
  upstream: "https://mcp.github.com/stream"
```

Client requests ride POST bodies; the response's SSE `data:` lines are
filtered and relayed. The upstream's GET event stream (server→client
notifications) is held open for the session and relayed through the same
inbound filter, and is torn down promptly when the client disconnects.

### Stdio Local Servers

```yaml
mcp:
  upstream: "stdio:npx @modelcontextprotocol/server-github"
```

The proxy spawns the local MCP server as a subprocess and bridges stdio
communication.

## Use Cases

### AI Development with Remote APIs

```yaml
# Protect against AI tools leaking secrets
mcp:
  upstream: "https://api.anthropic.com/mcp" 
  deny_patterns: ["sk-ant-[a-zA-Z0-9]+"]
  allow_tools: ["analyze_code"]  # Not "send_email"
```

### Multi-User Development Teams

```yaml
# Team policy for shared MCP servers
mcp:
  upstream: "https://company-mcp.internal/api"
  allow_tools: ["read_docs", "search_code"]  # No write operations
  deny_patterns: ["[a-zA-Z0-9]{32}"]  # Block long tokens
```

### Security Research

```yaml
# Analyze unknown MCP servers safely
mcp:
  upstream: "https://untrusted-mcp.example.com"
  allow_tools: []  # Block all tools
  audit_requests: true  # Log everything
  max_payload_kb: 100   # Limit payload size
```

## Command Line Options

```bash
warden proxy [OPTIONS]

Options:
  --policy <file>     MCP proxy policy file (required)
  --listen <addr>     Listen address (default: localhost:8765)  
  --upstream <url>    Override upstream from policy
  --help              Show help

Examples:
  warden proxy --policy mcp-policy.yaml
  warden proxy --policy mcp-policy.yaml --listen :9000
  warden proxy --policy mcp-policy.yaml --upstream "stdio:cat"
```

## Audit Logging

All MCP activity is logged to the Warden audit log:

```bash
# View MCP proxy activity
warden logs --tail 10

# Filter for blocked requests
warden logs | grep '"type":"mcp_proxy"' | grep '"allowed":false'
```

Example audit events:
```json
{"timestamp":"2026-09-10T00:00:00Z","type":"mcp_proxy","action":"stdio_proxy","resource":"npx @modelcontextprotocol/server-github","allowed":true,"reason":"stdio MCP upstream started"}
{"timestamp":"2026-09-10T00:00:01Z","type":"mcp_proxy","action":"mcp_block","resource":"tools/call","allowed":false,"reason":"tool \"delete_repo\" not in allow list"}
```

## Architecture

```
Client Request
      │
      ▼
┌─────────────┐
│   Policy    │ ◄── mcp: section validates
│ Validation  │     upstream, tools, patterns
└─────────────┘
      │
      ▼
┌─────────────┐
│   Message   │ ◄── Parse JSON-RPC
│   Parsing   │     Extract method, params
└─────────────┘
      │
      ▼
┌─────────────┐
│  Filtering  │ ◄── Check allow_tools
│   Engine    │     Scan deny_patterns
└─────────────┘
      │
      ▼
┌─────────────┐
│   Upstream  │ ◄── Forward to MCP server
│    Proxy    │     stdio transport only (subprocess stdin/stdout)
└─────────────┘
      │
      ▼
┌─────────────┐
│   Audit     │ ◄── Log all activity
│   Logger    │     Both allowed & blocked
└─────────────┘
```

## Implementation Status

**Working today (verified by unit and end-to-end tests):**
- `warden proxy` runs a real stdio JSON-RPC bridge: spawns the upstream
  subprocess, relays newline-delimited JSON-RPC in both directions
- **Remote HTTP/SSE upstreams** (Streamable HTTP): POST-per-message with
  SSE-framed responses, plus the server→client GET event stream — all
  filtered, all audited, TLS verified, plain http:// loopback-only
- **`Mcp-Session-Id` handling**: the id issued on the initialize response is
  captured and replayed on every subsequent request, so session-based servers
  (Microsoft Learn, GitHub, …) keep one upstream session per client connection
- Tool allowlist, deny patterns, and payload-size enforcement in both directions
- Env filtering: the stdio subprocess receives only `env.allow` variables (deny-by-default)
- JSON-RPC errors for blocked (`-32001`) and unparseable (`-32700`) messages;
  `-32603` when the upstream itself fails (fail-closed, never a silent hang)
- Audit logging for every allow/block decision (`mcp_message`, `mcp_block`,
  `stdio_proxy` events)

**Not implemented / limitations:**
- Sandbox isolation of the stdio subprocess (filesystem/network are unrestricted)
- Direct MCP-client protocols (the TCP listener speaks JSON-RPC lines, not HTTP)
- OAuth token minting: the proxy is a pass-through; it forwards whatever auth
  your client establishes (for stdio bridges, `env.allow` governs token reachability)

**Known safe by design:** every blocked message is answered with a JSON-RPC
error and audited; nothing blocked is ever forwarded — including on the HTTP
path, where a filtered payload never leaves loopback.

## Live Interop Results

The HTTP/SSE transport was verified against real, public MCP servers on
2026-09-11 (`warden-sandbox-cli` 0.1.13, local build from this tree). The
client below spoke plain newline-delimited JSON-RPC to `warden proxy`; the
proxy handled the Streamable HTTP framing, TLS, and session management.

Upstreams tested:

| Upstream | Transport notes | Result |
|---|---|---|
| `https://mcp.deepwiki.com/mcp` (DeepWiki) | No session id; POST responses are `text/event-stream` frames | 5/5 pass |
| `https://learn.microsoft.com/api/mcp` (Microsoft Learn) | Issues `Mcp-Session-Id`; session replay verified across tools/list + tools/call | 4/4 pass |
| `https://api.githubcopilot.com/mcp/` (GitHub) | Requires OAuth Bearer (401 without token) — out of proxy scope, fail-closed as designed | not proxied |
| Dead host (`*.invalid`) | DNS failure surfaces as JSON-RPC `-32603` to the client, no hang | pass (fail-closed) |

Test matrix against DeepWiki (policy: allowlist = `read_wiki_structure`,
`ask_question`; deny pattern = `ghp_[A-Za-z0-9]{36}`):

| # | Scenario | Expected | Result |
|---|---|---|---|
| 1 | `initialize` handshake | `serverInfo.name = DeepWiki` relayed | PASS |
| 2 | `tools/list` (discovery passes the filter) | 3 tools listed | PASS |
| 3 | `tools/call read_wiki_structure` (allowlisted) | forwarded, result relayed | PASS |
| 4 | `tools/call read_wiki_contents` (NOT allowlisted) | `-32001 blocked by Warden policy: tool ... not in allow list`; upstream never contacted | PASS |
| 5 | allowlisted call whose payload contains a `ghp_…` token | `-32001 … blocked pattern: ghp_[A-Za-z0-9]{36}`; payload never leaves loopback | PASS |

Corresponding audit trail (`warden logs`) for the same session — every
decision recorded, allowed and blocked:

```json
{"type":"mcp_proxy","action":"mcp_message","resource":"tools/call","allowed":true,"reason":"direction=outbound id=3"}
{"type":"mcp_proxy","action":"mcp_block","resource":"tools/call","allowed":false,"reason":"tool \"read_wiki_contents\" not in allow list"}
{"type":"mcp_proxy","action":"mcp_block","resource":"tools/call","allowed":false,"reason":"message contains blocked pattern: ghp_[A-Za-z0-9]{36}"}
```

What the live run changed in the implementation (interop fixes, each now
pinned by the unit/e2e test suite):

1. **SSE-framed POST responses.** Production servers answer POSTs with
   `Content-Type: text/event-stream` bodies, not raw JSON lines. The proxy
   now dispatches on the response media type and runs SSE-framed bodies
   through the same `data:`-line filter (caught live: both earlier attempts
   would have been dropped as "unparseable").
2. **`Mcp-Session-Id` replay.** Session-minting servers (Microsoft Learn)
   reject post-initialize calls that omit the id. The initialize response's
   id is captured once per proxy process and replayed on every subsequent
   POST and GET.
3. **GET-without-session is spec-legal.** Opening the server→client event
   stream on a server that issues no session id returns HTTP 406 — treated
   as "no GET stream", not a failure; POST responses still deliver results.

Reproduction sketch (any MCP client that speaks newline-delimited JSON-RPC
over TCP, e.g. a stdio bridge):

```bash
cat > mcp-proxy-policy.yaml <<'YAML'
mcp:
  upstream: "https://mcp.deepwiki.com/mcp"
  allow_tools: ["read_wiki_structure", "ask_question"]
  deny_patterns: ["ghp_[A-Za-z0-9]{36}"]
  max_payload_kb: 256
YAML
warden proxy --policy mcp-proxy-policy.yaml --listen 127.0.0.1:8765
# then send JSON-RPC lines to 127.0.0.1:8765 (initialize first)
```

## Troubleshooting

### Common Issues

1. **"Policy must contain 'mcp' section"**
   - Add `mcp:` section to your policy file
   - Ensure `upstream` is specified

2. **"Connection refused"**  
   - Check that upstream MCP server is accessible
   - Verify network.allow includes the upstream host

3. **"Tool not in allow list"**
   - Add the tool name to `mcp.allow_tools`
   - Or remove `allow_tools` to allow all tools

4. **"Blocked pattern detected"**
   - Review `mcp.deny_patterns` for overly broad rules
   - Check audit log for the specific blocked pattern

### Debug Mode

```bash
# Enable verbose logging
WARDEN_DEBUG=1 warden proxy --policy mcp-policy.yaml

# Monitor audit log in real-time
warden logs --follow &
warden proxy --policy mcp-policy.yaml
```

## Security Considerations

1. **Pattern Selection**: Choose deny patterns carefully to avoid false positives
2. **Audit Logging**: Consider storage/retention for audit logs  
3. **Env allowlist**: Only pass secrets the upstream actually needs (`env.allow` is the only section that isolates the subprocess)
4. **Tool Permissions**: Use minimal `allow_tools` for principle of least privilege
5. **Payload Limits**: Set `max_payload_kb` to prevent resource exhaustion
6. **No process isolation yet**: the stdio subprocess runs unsandboxed on your
   host — wrap `warden proxy` itself in `warden run` (or a container) if the
   upstream must be filesystem/network isolated

## Future Enhancements

- Browser extension for easy client configuration
- Machine learning-based sensitive content detection
- Integration with security information and event management (SIEM) systems
- Support for custom MCP authentication methods
- GraphQL-style query allowlisting for complex MCP operations

## Related Documentation

- [Policy Schema Reference](../docs/schema.md)
- [Security Best Practices](../docs/security.md)  
- [Audit Logging Guide](../docs/audit.md)
- [Container/K8s Mode](../docs/container.md)