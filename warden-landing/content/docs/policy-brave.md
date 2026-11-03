# Brave Search MCP Server Policy

Sandbox the Brave Search MCP server. **Single-host API egress.**

## What it needs

| Resource | Access | Details |
|---|---|---|
| Filesystem | Write | Cache for search results |
| Network | `api.search.brave.com` | Brave Search API |
| Env | `BRAVE_API_KEY` | API authentication |

## Complete Policy

```yaml
command: ["node", "@modelcontextprotocol/server-brave-search"]

filesystem:
  read: []
  write:
    - "./cache"              # Search result cache

network:
  allow:
    - "api.search.brave.com"

env:
  allow:
    - "BRAVE_API_KEY"
    - "HOME"

limits:
  memory_mb: 128
  timeout_s: 60
```

## How to run

```bash
# Step 1: Install the server
npm install -g @modelcontextprotocol/server-brave-search

# Step 2: Set your API key
export BRAVE_API_KEY="BSAxxxxxxxxxxxxxx"

# Step 3: Run sandboxed
warden run --policy policy-brave.yaml
```

## Tips

- This is a **single-host** server, only `api.search.brave.com`
- Grant write access to the `cache` directory (search results are cached)
- It is a **light** server, `memory_mb: 128` is enough
- `HOME` is needed for Node.js initialization

## Troubleshooting

| Error | Fix |
|---|---|
| `API key not found` | Export `BRAVE_API_KEY` in the shell |
| Network blocked | Is `api.search.brave.com` in `network.allow`? |
| Cache errors | Grant write access to the `cache` directory |

## Resources

- [Brave Search MCP Server](https://github.com/modelcontextprotocol/servers-archived/tree/main/src/brave-search)
- [Schema Reference](schema.md)
- [Write a Policy](write-policy.md)
