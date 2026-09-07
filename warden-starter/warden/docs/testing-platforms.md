# Cross-Platform MCP Testing

Test the Warden MCP sandbox on Linux, macOS, and Windows using native backends or Docker.

## Prerequisites

| Platform | Backend | Required Tools |
|----------|---------|---------------|
| Linux | BubbleWrap (bwrap) | bwrap, strace for auditing |
| macOS | Seatbelt (sandbox-exec) | Built into macOS 10.15+ |
| Windows | AppContainer + WFP | Windows 10/11 Pro/Enterprise |
| Any | Docker | Docker daemon |

## Build All Platform Binaries

```bash
cd warden-starter/warden
make build-all
```

Creates:
- `dist/warden-linux-amd64`
- `dist/warden-linux-arm64`
- `dist/warden-darwin-amd64`
- `dist/warden-darwin-arm64`
- `dist/warden-windows-amd64.exe`

## Test Scripts

Warden includes ready-to-use test scripts in `mcp-test/`:

- `mcp-test/test-server.js` - Simple MCP server with filesystem tools
- `test-policy.yaml` - Default policy for local testing
- `test-docker-policy.yaml` - Docker-specific policy
- `test-shell-mcp-policy.yaml` - Policy for shell MCP server
- `test-network-blocked-policy.yaml` - Network blocking test
- `test-data/` - Contains test files for filesystem operations

## Testing on Each Platform

### Linux (bwrap)

```bash
# Install prerequisites
sudo apt install bubblewrap strace

# Run MCP server in sandbox
./warden run --backend linux --policy test-policy.yaml

# Test with Docker fallback
./warden run --backend docker --policy test-docker-policy.yaml
```

### macOS (Seatbelt)

```bash
# No installation needed (built into macOS)
cd warden-starter/warden

# Run MCP server with Seatbelt
./warden run --backend seatbelt --policy test-policy.yaml

# Test with Docker fallback
./warden run --backend docker --policy test-docker-policy.yaml
```

### Windows (AppContainer)

```powershell
# Run MCP server with AppContainer
.\warden-windows-amd64.exe run --backend windows --policy test-policy.yaml

# Test with Docker fallback
.\warden-windows-amd64.exe run --backend docker --policy test-docker-policy.yaml
```

## Test Scenarios

### 1. Basic Functionality

Start any MCP server in a sandbox:

```bash
./warden run --backend BACKEND --policy POLICY_FILE -- COMMAND
```

Verify the server starts and responds to MCP requests.

### 2. Filesystem Restriction Test

Policy:
```yaml
filesystem:
  read:
    - "/allowed/path"
  write:
    - "/allowed/path"
```

Tests:
- Read files in `/allowed/path` - Succeeds
- Read `/etc/shadow` - Blocked
- Read `~/.ssh/id_rsa` - Blocked

### 3. Network Restriction Test

Policy:
```yaml
network:
  allow:
    - "api.github.com"
```

Tests:
- Connect to `api.github.com` - Succeeds
- Connect to `google.com` - Blocked
- DNS resolution for blocked hosts - Fails

### 4. Environment Variable Filtering

Policy:
```yaml
env:
  allow:
    - "HOME"
    - "PATH"
    - "GITHUB_TOKEN"
```

Tests:
- `HOME`, `PATH`, `GITHUB_TOKEN` - Available
- `AWS_SECRET_KEY` - Filtered out
- `DATABASE_PASSWORD` - Filtered out

### 5. Resource Limits

Policy:
```yaml
limits:
  memory_mb: 128
  timeout_s: 60
```

Tests:
- Process within limits - Runs normally
- Memory exceeds 128MB - Process killed
- Runs longer than 60s - Process terminated

## Automated Test Commands

### Run All Tests (Docker Backend)

```bash
cd warden-starter/warden

# Test 1: Basic execution
echo "=== Test 1: Basic execution ==="
./warden run --backend docker --policy test-simple-policy.yaml

# Test 2: MCP initialize
echo "=== Test 2: MCP Initialize ==="
echo '{"jsonrpc":"2.0","id":0,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}' | \
  ./warden run --backend docker --policy test-shell-mcp-policy.yaml

# Test 3: List MCP tools
echo "=== Test 3: List tools ==="
echo '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}' | \
  ./warden run --backend docker --policy test-shell-mcp-policy.yaml

# Test 4: Read allowed file
echo "=== Test 4: Read allowed file ==="
echo '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"read_file","arguments":{"path":"/home/abdullah/Downloads/warden/warden-starter/warden/test-app/file1.txt"}}}' | \
  ./warden run --backend docker --policy test-shell-mcp-policy.yaml

# Test 5: Network blocking
echo "=== Test 5: Network blocking ==="
./warden run --backend docker --policy test-network-blocked-policy.yaml
```

## Test Results Reference

| Test | Linux (bwrap) | macOS (Seatbelt) | Windows | Docker |
|------|--------------|-----------------|---------|--------|
| Version check | Pass | Pass | Pass | Pass |
| Basic execution | Pass | Pass | Pass | Pass |
| MCP initialize | Pass | Pass | Pass | Pass |
| Filesystem read | Pass | Pass | Pass | Pass |
| Network block | Pass | Pass | Pass | Pass |
| Resource limits | Pass | Pass | Pass | Pass |

## Compatibility

The compatibility matrix covers 18 MCP servers tested with Warden:

- Pass (14): filesystem, github, slack, postgres, sqlite, brave-search, gdrive, git, memory, time, sequential-thinking, notion, linear, tavily
- Conditional (2): fetch, kubernetes
- Fail (2): docker, playwright

See [compatibility.md](./compatibility) for the full matrix.

## Troubleshooting

### "bwrap: command not found" (Linux)
Install bubblewrap:
```bash
sudo apt install bubblewrap
sudo dnf install bubblewrap
sudo pacman -S bubblewrap
```

### "sandbox-exec: Operation not permitted" (macOS)
- Requires macOS 10.15 (Catalina) or later
- Use Docker backend as fallback: `--backend docker`

### "AppContainer not available" (Windows)
- Requires Windows 10/11 Pro or Enterprise
- Use Docker backend as fallback

### Docker backend errors
```bash
# Ensure Docker is running
docker info

# Pull the required image
docker pull alpine:3.20

# For Node.js MCP servers, use node image
export WARDEN_DOCKER_IMAGE=node:20-alpine
```

## Audit Logging

Warden logs all sandbox activity. Inspect logs:

```bash
# Tail audit log
./warden logs --follow

# Show last 50 entries
./warden logs --tail 50

# View specific log file
./warden logs --log /path/to/audit.jsonl
```

Audit events include:
- `file_access` - File read/write attempts
- `network` - Connection attempts
- `env` - Environment variable access
- `limit` - Resource limit breaches

