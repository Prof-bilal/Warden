# Brave Search MCP Server Policy

Brave Search MCP server ko sandbox karo. Single-host API egress.

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

- Single-host server hai - sirf `api.search.brave.com`
- `cache` directory ka write grant do (search results cache hoti hain)
- Light server hai - `memory_mb: 128` sufficient hai
- `HOME` Node.js initialization ke liye chahiye

## Troubleshooting

| Error | Fix |
|---|---|
| `API key not found` | `BRAVE_API_KEY` shell mein export karo |
| Network blocked | `api.search.brave.com` `network.allow` mein hai? |
| Cache errors | `cache` directory ka write grant do |

## Resources

- [Brave Search MCP Server](https://github.com/modelcontextprotocol/servers/tree/main/src/brave-search)
- [Schema Reference](schema.md)
- [Write a Policy](write-policy.md)
