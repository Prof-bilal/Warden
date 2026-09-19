# Slack MCP Server Policy

Slack MCP server ko sandbox karo. Slack API + web UI dono access chahiye.

## What it needs

| Resource | Access | Details |
|---|---|---|
| Filesystem | Write only | Cache for OAuth tokens, rate-limit state |
| Network | `slack.com` + `api.slack.com` | Web UI + API endpoints |
| Env | `SLACK_BOT_TOKEN`, `SLACK_TEAM_ID` | Bot authentication |

## Complete Policy

```yaml
command: ["node", "@modelcontextprotocol/server-slack"]

filesystem:
  read: []
  write:
    - "./cache"              # OAuth tokens, rate-limit state

network:
  allow:
    - "slack.com"            # Web UI
    - "api.slack.com"        # API

env:
  allow:
    - "SLACK_BOT_TOKEN"
    - "SLACK_TEAM_ID"
    - "HOME"

limits:
  memory_mb: 256
  timeout_s: 300
```

## How to run

```bash
# Step 1: Install the server
npm install -g @modelcontextprotocol/server-slack

# Step 2: Set your tokens
export SLACK_BOT_TOKEN="xoxb-xxxxxxxxxxxx"
export SLACK_TEAM_ID="T0123456789"

# Step 3: Run sandboxed
warden run --policy policy-slack.yaml
```

## Tips

- `write` grant mein sirf `cache` do - write grant read bhi subsume karta hai
- `read: []` alag se mat do agar `write` same path pe hai
- `HOME` Node.js libraries ke liye chahiye
- `SLACK_TEAM_ID` optional lagta hai lekin zaroori hai

## Troubleshooting

| Error | Fix |
|---|---|
| `token not found` | `SLACK_BOT_TOKEN` export karo shell mein |
| Network blocked | `slack.com` + `api.slack.com` dono list karo |
| OAuth errors | `cache` directory ka write grant do |

## Resources

- [Slack MCP Server](https://github.com/modelcontextprotocol/servers/tree/main/src/slack)
- [Schema Reference](schema.md)
- [Write a Policy](write-policy.md)
