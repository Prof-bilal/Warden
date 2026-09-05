# Example Policies

Copy any of these policies as a starting point for your own MCP server, then
adjust the `command` path, grants, and env allowlist to match your setup.
Relative paths in these files are resolved against this directory — not your
shell's current working directory.

> **Tip:** The easiest way to get a policy right is to run
> `warden trace -- <your-server-command>` first, then `warden init`. The
> auto-generated policy is conservative by design; review and tighten it
> from there.

## Policies

| Server | File | What it covers |
|---|---|---|
| **Filesystem** | [filesystem-mcp-server.yaml](./filesystem-mcp-server.yaml) | Local file read/write — no network |
| **GitHub** | [github-mcp-server.yaml](./github-mcp-server.yaml) | REST + GraphQL against `api.github.com`, needs `GITHUB_TOKEN` |
| **Slack** | [slack-mcp-server.yaml](./slack-mcp-server.yaml) | API + web UI, needs `SLACK_BOT_TOKEN` and `SLACK_TEAM_ID` |
| **PostgreSQL** | [postgres-mcp-server.yaml](./postgres-mcp-server.yaml) | Database connections, `.pgpass` / `.postgresql` credentials |
| **Brave Search** | [brave-search-mcp-server.yaml](./brave-search-mcp-server.yaml) | Search API only, needs `BRAVE_API_KEY` |
| **Comprehensive reference** | [policy.example.yaml](./policy.example.yaml) | Full schema with annotations for every field — use as a template |

## Common patterns

### Where to put the server binary

All examples assume you've installed the MCP server (usually via `npm install`)
and the compiled entry point lives under `./server/dist/index.js` relative to
the policy file. Adjust the `command` array to point at your actual binary:

```yaml
command: ["/usr/bin/node", "./path/to/your/server.js"]
```

The executable **must be an absolute path** inside the sandbox because Warden
bind-mounts only the paths you list — a bare name like `node` would resolve
nowhere.

### Home directory grants

Several examples pass `HOME` through `env.allow` and grant read access to
`~/.config`-style paths. This is intentional — many Node.js and Python
libraries expect `HOME` to be set. **Do not combine `HOME` with a broad
`filesystem.read` grant** (especially `~/.ssh`); see [Security Review](../docs/security.md#5-credential-exposure)
for details.

### Empty allowlists are explicit

Notice that `network.allow: []` and `env.allow: []` appear in the filesystem
example. An empty list is *not* the same as omitting the key — the former
explicitly denies everything, the latter falls back to the default (which
varies by field). Being explicit in your policy makes intent clear and helps
`warden init` produce a correct starter.

## Next steps

- Read the [Schema Reference](../docs/schema.md) to understand every field.
- Run `warden trace -- <your-server>` to see what a server actually touches.
- Run `warden init` to generate a starter policy from the trace output.
- Review [Security Review](../docs/security.md) before relying on any policy
  in production.
