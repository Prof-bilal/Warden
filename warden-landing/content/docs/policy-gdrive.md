# Google Drive MCP Server Policy

Google Drive MCP server ko sandbox karo. Multiple Google APIs access chahiye.

## What it needs

| Resource | Access | Details |
|---|---|---|
| Filesystem | Read + Write | OAuth credentials file + cache |
| Network | 3 Google API hosts | `www.googleapis.com`, `oauth2.googleapis.com`, `drive.google.com` |
| Env | Google credentials | `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET` |

## Complete Policy

```yaml
command: ["node", "@modelcontextprotocol/server-gdrive"]

filesystem:
  read:
    - "./config/credentials.json"     # OAuth client credentials
  write:
    - "./cache"                       # API response cache

network:
  allow:
    - "www.googleapis.com"            # Google APIs
    - "oauth2.googleapis.com"         # OAuth flow
    - "drive.google.com"              # Drive web UI

env:
  allow:
    - "GOOGLE_CLIENT_ID"
    - "GOOGLE_CLIENT_SECRET"
    - "HOME"

limits:
  memory_mb: 256
  timeout_s: 300
```

## How to run

```bash
# Step 1: Install the server
npm install -g @modelcontextprotocol/server-gdrive

# Step 2: Set credentials
export GOOGLE_CLIENT_ID="your-client-id"
export GOOGLE_CLIENT_SECRET="your-client-secret"

# Step 3: Run sandboxed
warden run --policy policy-gdrive.yaml
```

## Tips

- `credentials.json` file ka exact path do
- 3 Google API hosts chahiye - sab `network.allow` mein daalo
- OAuth flow `oauth2.googleapis.com` pe hota hai
- `cache` directory API responses ke liye chahiye

## Security

- `credentials.json` ko **read-only** grant do (sirf `read`)
- `GOOGLE_CLIENT_SECRET` policy mein mat likho
- `cache` directory mein sensitive data store hota hai

## Troubleshooting

| Error | Fix |
|---|---|
| `OAuth error` | `oauth2.googleapis.com` `network.allow` mein hai? |
| `credentials not found` | `credentials.json` ka path check karo |
| `API error` | `www.googleapis.com` `network.allow` mein hai? |

## Resources

- [Google Drive MCP Server](https://github.com/modelcontextprotocol/servers/tree/main/src/gdrive)
- [Schema Reference](schema.md)
- [Write a Policy](write-policy.md)
