# Notion MCP Server Policy

Sandbox the Notion MCP server. **Single-host API egress.**

## What it needs

| Resource | Access | Details |
|---|---|---|
| Filesystem | Write | Cache for API responses |
| Network | `api.notion.com` | Notion API |
| Env | `NOTION_API_KEY` | API authentication |

## Complete Policy

```yaml
command: ["node", "notion-mcp"]

filesystem:
  read: []
  write:
    - "./cache"              # API response cache

network:
  allow:
    - "api.notion.com"

env:
  allow:
    - "NOTION_API_KEY"
    - "HOME"

limits:
  memory_mb: 256
  timeout_s: 300
```

## How to run

```bash
# Step 1: Install the server
npm install -g @makenotion/notion-mcp

# Step 2: Set your API key
export NOTION_API_KEY="secret_xxxxxxxxxxxxx"

# Step 3: Run sandboxed
warden run --policy policy-notion.yaml
```

## Tips

- This is a **single-host** server, only `api.notion.com`
- The `cache` directory is needed for API responses
- `HOME` is needed for Node.js initialization
- `NOTION_API_KEY` comes from the Notion **integration settings**

## Troubleshooting

| Error | Fix |
|---|---|
| `API key not found` | Export `NOTION_API_KEY` in the shell |
| Network blocked | Is `api.notion.com` in `network.allow`? |
| Cache errors | Grant write access to the `cache` directory |

## Resources

- [Notion MCP Server](https://github.com/makenotion/notion-mcp)
- [Schema Reference](schema.md)
- [Write a Policy](write-policy.md)
