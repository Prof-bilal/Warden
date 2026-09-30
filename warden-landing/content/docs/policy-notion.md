# Notion MCP Server Policy

Notion MCP server ko sandbox karo. Single-host API egress.

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

- Single-host server hai - sirf `api.notion.com`
- `cache` directory API responses ke liye chahiye
- `HOME` Node.js initialization ke liye chahiye
- `NOTION_API_KEY` Notion integration settings se milta hai

## Troubleshooting

| Error | Fix |
|---|---|
| `API key not found` | `NOTION_API_KEY` shell mein export karo |
| Network blocked | `api.notion.com` `network.allow` mein hai? |
| Cache errors | `cache` directory ka write grant do |

## Resources

- [Notion MCP Server](https://github.com/makenotion/notion-mcp)
- [Schema Reference](schema.md)
- [Write a Policy](write-policy.md)
