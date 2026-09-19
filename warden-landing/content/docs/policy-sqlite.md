# SQLite MCP Server Policy

SQLite MCP server ko sandbox karo. Local database, no network needed.

## What it needs

| Resource | Access | Details |
|---|---|---|
| Filesystem | Write | DB directory (WAL + journal sidecars ke liye) |
| Network | None | Local database hai |
| Env | None | Variables nahi chahiye |

## Complete Policy

```yaml
command: ["node", "@modelcontextprotocol/server-sqlite"]

filesystem:
  read: []
  write:
    - "./data"               # DB file + WAL + journal sidecars

network:
  allow: []                  # No network

env:
  allow: []                  # No env vars

limits:
  memory_mb: 128
  timeout_s: 60
```

## How to run

```bash
# Step 1: Install the server
npm install -g @modelcontextprotocol/server-sqlite

# Step 2: Run sandboxed
warden run --policy policy-sqlite.yaml
```

## Tips

- **Important:** SQLite ko `write` grant chahiye DB file ke **directory** pe
  - Sirf `db.sqlite3` pe nahi chalega
  - WAL aur journal files next to DB file banti hain
- `read: []` rakho agar sirf write karna hai
- `network: []` rakho kyunki local database hai

## Why Write on Directory?

SQLite WAL (Write-Ahead Logging) mode mein multiple files banti hain:

```
data/
  mydb.sqlite3        # Main database
  mydb.sqlite3-wal    # Write-ahead log
  mydb.sqlite3-shm   # Shared memory file
```

Isliye directory pe write grant dena zaroori hai.

## Troubleshooting

| Error | Fix |
|---|---|
| `database is locked` | Write grant check karo - directory pe hona chahiye |
| `permission denied` | `filesystem.write` mein directory add karo |
| `unable to open database file` | Path relative hai policy file ke folder ke against |

## Resources

- [SQLite MCP Server](https://github.com/modelcontextprotocol/servers/tree/main/src/sqlite)
- [Schema Reference](schema.md)
- [Write a Policy](write-policy.md)
