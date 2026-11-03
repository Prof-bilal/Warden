# SQLite MCP Server Policy

Sandbox the SQLite MCP server. **Local database, no network needed.**

## What it needs

| Resource | Access | Details |
|---|---|---|
| Filesystem | Write | DB directory (for WAL + journal sidecars) |
| Network | None | It is a **local database** |
| Env | None | **No variables needed** |

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

- **Important:** SQLite needs a `write` grant on the **directory** containing the DB file
  - Granting only `db.sqlite3` **will not work**
  - WAL and journal files are created **next to** the DB file
- Use `read: []` if you only need to **write**
- Use `network: []` because it is a **local database**

## Why Write on the Directory?

In **WAL (Write-Ahead Logging)** mode, SQLite creates **multiple files**:

```
data/
  mydb.sqlite3        # Main database
  mydb.sqlite3-wal    # Write-ahead log
  mydb.sqlite3-shm    # Shared memory file
```

That is why you **must** grant write access on the **directory**.

## Troubleshooting

| Error | Fix |
|---|---|
| `database is locked` | Check the write grant, it must be on the **directory** |
| `permission denied` | Add the directory to `filesystem.write` |
| `unable to open database file` | The path is **relative** to the policy file's folder |

## Resources

- [SQLite MCP Server](https://github.com/modelcontextprotocol/servers-archived/tree/main/src/sqlite)
- [Schema Reference](schema.md)
- [Write a Policy](write-policy.md)
