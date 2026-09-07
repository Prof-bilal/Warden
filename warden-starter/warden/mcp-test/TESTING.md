# Warden MCP Testing Setup

## Quick Start

### Test with Docker Backend (Works Now)

Since strace is not installed for native bwrap, use Docker:

```bash
# Create policy with absolute paths
cat > test-docker-policy.yaml << 'EOF'
command: ["/usr/local/bin/node", "/app/mcp-test/test-server.js", "/app/test-data"]

filesystem:
  read:
    - "/app/test-data"
  write:
    - "/app/test-data"

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
EOF

# Run MCP server in sandbox
./warden run --backend docker --policy test-docker-policy.yaml
```

## Test MCP Tools

Once running, test these operations:

### ✅ Should Work
- Read files in /app/test-data
- Write files to /app/test-data  
- List directory contents

### ❌ Should Be Blocked
- Access /etc/shadow
- Connect to non-allowlisted hosts
- Read environment variables not in allowlist

## Platform-Specific Testing

### Linux (bwrap)
```bash
# Install bubblewrap + strace
sudo pacman -S bubblewrap strace

# Test
./warden run --backend linux --policy test-policy.yaml
```

### macOS (Seatbelt)
```bash
# Build for macOS
GOOS=darwin GOARCH=arm64 go build -o warden-darwin ./cmd/warden

# Test (no installation needed)
./warden-darwin run --backend seatbelt --policy test-policy.yaml
```

### Windows (AppContainer)
```powershell
# Build for Windows
GOOS=windows GOARCH=amd64 go build -o warden-windows.exe .\cmd\warden

# Test
.\warden-windows.exe run --backend windows --policy test-policy.yaml
```

## Build All Platforms

```bash
cd warden-starter/warden
make build-all

# Creates:
# - dist/warden-linux-amd64
# - dist/warden-linux-arm64
# - dist/warden-darwin-amd64
# - dist/warden-darwin-arm64
# - dist/warden-windows-amd64.exe
```

## Security Tests

### 1. Filesystem Restriction
- Policy: Only /app/test-data readable
- Test: Try reading /etc/shadow → BLOCKED
- Audit: Check logs for blocked attempt

### 2. Network Restriction
```yaml
network:
  allow:
    - "api.github.com"
```
- Test: Connect to google.com → BLOCKED
- Test: Connect to api.github.com → ALLOWED

### 3. Environment Filtering
```yaml
env:
  allow:
    - "HOME"
    - "GITHUB_TOKEN"
```
- HOME and GITHUB_TOKEN passed through
- AWS_SECRET_KEY filtered out

## Troubleshooting

| Error | Solution |
|-------|----------|
| "strace not found" | Install strace or use Docker |
| "sandbox-exec not supported" | Use Docker fallback |
| "AppContainer not available" | Windows Pro+ required |
| Docker permission | `sudo usermod -aG docker $USER` |

## Binaries Available

All built and ready in `dist/`:
- ✅ warden-linux-amd64 (5.8MB)
- ✅ warden-linux-arm64 (5.6MB)
- ✅ warden-darwin-amd64 (6.0MB)
- ✅ warden-darwin-arm64 (5.7MB)
- ✅ warden-windows-amd64.exe (6.2MB)

## Documentation

- Architecture: `ARCHITECTURE.md`
- Roadmap: `ROADMAP.md`
- Cross-platform guide: `TESTING-PLATFORMS.md`
- Compatibility: `docs/compatibility.md`
