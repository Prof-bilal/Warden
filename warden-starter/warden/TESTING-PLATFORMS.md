# Warden MCP Testing Guide

## Quick Start

### Build Warden
```bash
cd warden-starter/warden
make build
./warden --version
```

## Platform-Specific Testing

### 1. Linux Testing (bubblewrap)

**Install bubblewrap:**
```bash
sudo pacman -S bubblewrap  # Arch
sudo apt install bubblewrap  # Debian/Ubuntu
```

**Test Policy (test-policy.yaml):**
```yaml
command: ["/usr/bin/node", "/full/path/to/mcp-test/test-server.js", "/full/path/to/test-data"]

filesystem:
  read:
    - "/full/path/to/test-data"
  write:
    - "/full/path/to/test-data"

network:
  allow:
    - "api.github.com"

env:
  allow:
    - "HOME"
    - "PATH"

limits:
  memory_mb: 256
  timeout_s: 300
```

**Run with bwrap backend:**
```bash
./warden run --backend linux --policy test-policy.yaml
```

### 2. macOS Testing (Seatbelt)

**No additional installation needed** (sandbox-exec built into macOS)

**Build for macOS:**
```bash
GOOS=darwin GOARCH=amd64 go build -o warden-darwin-amd64 ./cmd/warden
```

**Run with Seatbelt backend:**
```bash
./warden-darwin-arm64 run --backend seatbelt --policy test-policy.yaml
```

### 3. Windows Testing (AppContainer)

**Build for Windows:**
```bash
GOOS=windows GOARCH=amd64 go build -o warden-windows-amd64.exe ./cmd/warden
```

**Run with Windows backend:**
```powershell
.\warden-windows-amd64.exe run --backend windows --policy test-policy.yaml
```

## Docker Backend (All Platforms)

```bash
docker pull alpine:3.20
./warden run --backend docker --policy test-policy.yaml
```

## Test Scenarios

### 1. Filesystem Restriction
- Allowed: /test-data directory (read/write)
- Blocked: /etc/shadow and other paths
- Verify: Audit log records blocked attempts

### 2. Network Restriction
```yaml
network:
  allow:
    - "api.github.com"
```
- Allowed: api.github.com
- Blocked: Other hosts
- Verify: Audit log records blocked connections

### 3. Environment Filtering
```yaml
env:
  allow:
    - "HOME"
    - "PATH"
    - "GITHUB_TOKEN"
```
- Only specified variables passed to server

### 4. Resource Limits
```yaml
limits:
  memory_mb: 128
  timeout_s: 60
```

## Compatibility Matrix

**Pass (14 servers):** filesystem, github, slack, postgres, sqlite, brave-search, gdrive, git, memory, time, sequential-thinking, notion, linear, tavily

**Conditional (2 servers):** fetch, kubernetes (need per-deployment config)

**Fail (2 servers):** docker, playwright (incompatible by design)

## Troubleshooting

| Issue | Solution |
|-------|----------|
| "bwrap not found" | Install bubblewrap |
| "sandbox-exec not supported" | Use Docker fallback |
| "AppContainer not available" | Requires Windows Pro/Enterprise |
| Docker errors | Ensure Docker daemon running |

## Security Notes

- **Always sandbox** untrusted MCP servers
- **Start minimal** - only grant needed permissions
- **Audit everything** - review blocked access attempts
- **Keep updated** - use latest Warden release

## Build All Platforms

```bash
make build-all
# Creates:
# - dist/warden-linux-amd64
# - dist/warden-linux-arm64
# - dist/warden-darwin-amd64
# - dist/warden-darwin-arm64
# - dist/warden-windows-amd64.exe
```

## References

- Architecture: `ARCHITECTURE.md`
- Policy Schema: `internal/policy/policy.go`
- Backends: `internal/sandbox/{linux,darwin,windows,docker}/`
- Compatibility: `docs/compatibility.md`
