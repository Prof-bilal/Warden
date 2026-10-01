# MCP gateway for agents and SDKs

Build the development checkout first. `connect` speaks standard newline-delimited
stdio; `serve` exposes authenticated Streamable HTTP at `/mcp`. These commands
are separate from the legacy `proxy` command, whose listener is custom TCP.

## Supported boundary

The gateway implements a **2025-11-25 request/response subset**: initialization,
ping, tool discovery/calls, exact resource reads and prompt retrieval. Discovery
is filtered. Empty allowlists deny access. Unsupported methods return correlated
JSON-RPC errors; they never bypass the filter. Dynamic list notifications, pagination,
subscriptions, tasks, sampling, elicitation, roots and 2026-07-28/MRTR are refused.
When a client requests another version, Warden offers 2025-11-25; the client must
accept it before sending initialized. Unsupported client capabilities are removed
before upstream initialization. Every HTTP session request must include
`MCP-Protocol-Version: 2025-11-25`. This is not full protocol conformance or
universal host certification; clients that cannot use this version cannot connect.

Local servers launch through digest-pinned `warden run` with the reviewed process
policy. Missing enforcement stops launch. Remote servers receive filtered requests
but their process/filesystem cannot be sandboxed by Warden. One gateway instance
has one reviewed policy and upstream credential context. Deploy separate instances
for different policies/providers; authentication does not create new permissions.

## Review MCP rules

```json
{
  "version": 1,
  "tools": {
    "read_text_file": {
      "arguments": {"/path": ["/selected/hello.txt"]}
    }
  },
  "resources": [],
  "prompts": [],
  "max_bytes": 1048576,
  "timeout_seconds": 60
}
```

Argument keys are JSON Pointers with exact allowed string values. Missing values,
non-string values and different values deny the call. This is not a path-pattern
language. Resource URIs and prompt names are exact lists. Optional
`definition_sha256` on a tool pins its entire definition. Save the returned tool
object and run `warden creator definition-hash --manifest tool-definition.json`.
This hashes sorted object keys while preserving JSON number precision. Use the
printed SHA-256 in the rules file.
Warden refreshes discovery immediately before calls and rejects changed pinned
definitions. Unpinned definitions remain an explicit review limitation.

```bash
warden connect --rules /absolute/rules.json --policy /absolute/policy.yaml -- /absolute/node /prepared/server/dist/index.js
warden serve --rules /absolute/rules.json --policy /absolute/policy.yaml --token-env WARDEN_GATEWAY_TOKEN --listen 127.0.0.1:8788 --control-file /private/control.json -- /absolute/node /prepared/server/dist/index.js
warden stop --control-file /private/control.json
```

Set the gateway credential in the environment to a random secret of at least 32
characters, and configure it as a bearer header in the client. Never put tokens
in URLs, command arguments or rules. Each HTTP session has its own local process
or remote session; session IDs are random and bound to the authenticated principal.
The gateway caps sessions at four per principal, 32 total, with 15-minute idle expiry.
It binds only numeric loopback. Browser origins are denied unless explicitly listed
with `--origins https://trusted.example`; authority/Host is checked too.

Timeouts and HTTP disconnects cancel requests. For local pending work Warden kills
that managed connection/process group; other clients retain their sessions. A stdio
cancellation closes that managed connection. Stop tears down managed sessions.
Remote cancellation closes Warden's request; it cannot undo completed provider-side
effects or independent direct connections. Native Windows cancellation needs live
platform tests before an assurance claim.

## Remote upstreams

```bash
warden serve --rules rules.json --upstream https://provider.example/mcp --upstream-token-env PROVIDER_TOKEN --token-env WARDEN_GATEWAY_TOKEN
```

Downstream tokens are never passed to the upstream. Local policies cannot grant
gateway credential environment names to their server. Remote endpoints require
HTTPS, reject embedded credentials/query strings, reject redirects, and resolve
to checked public addresses on each dial. Connections use the checked IP, preserving
TLS hostname verification. HTTP loopback testing requires an explicit numeric
loopback URL and `--allow-loopback-upstream`. There is no automatic remote metadata
fetch, legacy HTTP+SSE fallback or shared session across clients.

## HTTPS hosting and OAuth

Terminate TLS with your existing reverse proxy and forward to loopback. Preserve
the configured public Host, bearer header and MCP headers; set appropriate request
timeouts. Configure `--public-url https://gateway.example/mcp`. Do not expose the
private control file or make `/__warden/stop` publicly routable. Keep upstream tokens
in the gateway environment. Journals, control credentials and issuer private keys
must remain outside resolved server grants, the executable's directory and
readable runtime paths. On macOS this also excludes host temporary directories.

For an existing OAuth issuer:

```bash
warden serve --rules rules.json --policy policy.yaml --public-url https://gateway.example/mcp --issuer https://issuer.example --jwks-file /trusted/issuer-jwks.json -- /prepared/server
```

The gateway publishes protected-resource metadata at
`/.well-known/oauth-protected-resource/mcp` and challenges unauthorized requests
with that discovery URL. It validates trusted RS256 signatures, issuer, exact
resource audience, expiration/not-before, subject and `mcp` scope. Signing keys
come from a trusted local JWKS snapshot; rotate it and restart. Issuer configuration
is trusted authority. Token-env authentication and OAuth are mutually exclusive.
Warden does not implement an authorization server, browser consent flow, dynamic
client registration, remote JWKS discovery, automatic revocation or upstream OAuth
token exchange. These remain release work; do not advertise arbitrary OAuth-provider
compatibility based only on this verifier.

## SDK recipes

These Python examples follow current primary documentation; they are recipes,
not live Python SDK certifications. No model/API call is required to test tools.
Use an absolute Warden path and structured arguments on all platforms.

OpenAI Agents SDK:

```python
from agents.mcp import MCPServerStdio, MCPServerStreamableHttp
import os

local = MCPServerStdio(params={
    "command": "/absolute/warden",
    "args": ["connect", "--rules", "/absolute/rules.json",
             "--policy", "/absolute/policy.yaml", "--", "/prepared/server"]
})
remote = MCPServerStreamableHttp(params={
    "url": "https://gateway.example/mcp",
    "headers": {"Authorization": "Bearer " + os.environ["GATEWAY_TOKEN"]}
})
```

[OpenAI SDK reference](https://openai.github.io/openai-agents-python/mcp/).

LangChain/LangGraph:

```python
from langchain_mcp_adapters.client import MultiServerMCPClient
import os
client = MultiServerMCPClient({"warden": {
    "transport": "streamable_http", "url": "https://gateway.example/mcp",
    "headers": {"Authorization": "Bearer " + os.environ["GATEWAY_TOKEN"]}
}})
tools = await client.get_tools()
```

[LangChain reference](https://docs.langchain.com/oss/python/langchain/mcp).

Pydantic AI:

```python
from pydantic_ai.mcp import MCPServerStdio
server = MCPServerStdio("/absolute/warden", args=[
    "connect", "--rules", "/absolute/rules.json", "--policy",
    "/absolute/policy.yaml", "--", "/prepared/server"
])
```

[Pydantic AI reference](https://pydantic.dev/docs/ai/mcp/client/).

Cline CLI uses `~/.cline/mcp.json`; extension configuration needs the file opened
by Configure MCP Servers. The `cline` adapter preserves auto-approval settings.
The `cascade` adapter requires an explicit active legacy `mcp_config.json`; Devin
Local uses another configuration. [Cline](https://docs.cline.bot/mcp/mcp-overview),
[legacy Cascade](https://docs.devin.ai/desktop/cascade/mcp). Hosted agents need a
reachable HTTPS endpoint and a compatible protocol/authentication route; localhost
is not reachable from a cloud execution environment.
