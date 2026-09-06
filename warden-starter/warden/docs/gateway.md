# Gateway Integration

Warden wraps MCP servers registered in a gateway (or client) config so each
one runs sandboxed. The gateway keeps orchestrating; Warden becomes the spawn
prefix for every local server.

## Supported configs

| Family | File | Shape |
|---|---|---|
| Claude-style JSON | `mcp.json`, `claude_desktop_config.json` | `{"mcpServers": {"name": {"command","args","env","cwd"}}}` (also `"servers"`, `"type"/"url"` for remote) |
| Gateway YAML – upstreams | `config.yaml` (jonfairbanks/mcp-gateway) | `upstreams: [{id, transport, command, args, env, endpoint}]` |
| Gateway YAML – backends | `gateway.yaml` (MikkoParkkola/mcp-gateway) | `backends: {name: {command, env, http_url}}` |

Only **stdio** entries (a local command) can be sandboxed. Remote entries
(`type`/`url`, `endpoint`, `http_url`, non-stdio `transport`) are skipped with
a reason — there is no local process to confine.

## Workflow

```bash
# 1. Generate deny-by-default policies (never overwrites existing files)
warden gateway init --config mcp.json --policies ./policies
warden gateway list --config mcp.json --policies ./policies

# 2. Widen what each server needs (trace, then edit the policy)
warden trace -- /usr/bin/node ./server/dist/index.js
warden init --log <trace.jsonl> --output /tmp/starter.yaml
# ... merge observations into policies/<server>.yaml ...

# 3a. Run one server sandboxed (for debugging)
warden gateway run --config mcp.json --policies ./policies --server github

# 3b. Point the gateway at sandboxed servers (for production)
warden gateway wrap --config mcp.json --policies ./policies \
    --warden-bin /usr/local/bin/warden --output mcp.wrapped.json
# then start the gateway against mcp.wrapped.json
```

## How `wrap` works

Each stdio entry's command becomes:

```
<warden-bin> run --policy <policies>/<name>.yaml -- <original command...>
```

The wrapped file keeps the original `env` **values**: at runtime the gateway
spawns `warden run` with those values in its environment, and Warden forwards
only the names in the policy's `env.allow` to the sandboxed server. Policies
never store secrets.

`wrap` fails closed when any stdio server lacks a policy — run `gateway init`
first. Remote entries pass through untouched.

## Environment templates

Gateway `env` values may reference the parent environment:

- `${NAME}` — required; missing variable is an error, never an empty secret.
- `${NAME:-default}` — optional fallback.

For `gateway run`, pre-existing process env wins over the gateway default, so
`export GITHUB_TOKEN=...` overrides the file. For `wrap`, expansion happens
when the gateway spawns the server (i.e. inside `warden gateway run` semantics
— the gateway itself passes values through to the `warden run` child).

## Security notes

- Starter policies grant **nothing**: no filesystem paths, no network hosts.
  Widen them deliberately after `warden trace`.
- Only `env` names enter the policy; values live in the gateway config and
  the runtime environment.
- A missing policy, an unknown `--server`, or a remote `--server` is a
  hard error — Warden never runs the server unsandboxed.
