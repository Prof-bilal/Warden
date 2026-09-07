# Warden MCP Testing Summary

## What Was Set Up

### 1. Warden Build
- ✅ Built from source: `warden-starter/warden`
- ✅ Version: v0.1.6-2-g67d3403
- ✅ All platform binaries built in `dist/`

### 2. Test MCP Server
- ✅ Created: `mcp-test/test-server.js`
- ✅ Simple filesystem MCP server with 4 tools:
  - `read_file` - Read file contents
  - `write_file` - Write file contents
  - `list_directory` - List directory contents
  - `test_network` - Test network connectivity

### 3. Test Data
- ✅ Created: `test-data/test.txt`
- ✅ Location: `/home/abdullah/Downloads/warden/warden-starter/warden/test-data/`

### 4. Policies
- ✅ `test-policy.yaml` - For local development
- ✅ `test-docker-policy.yaml` - For Docker backend testing

### 5. Documentation
- ✅ `TESTING-PLATFORMS.md` - Cross-platform testing guide
- ✅ `mcp-test/TESTING.md` - Quick start guide

## Platform Status

### Linux ✅
- **Native Backend:** bubblewrap (bwrap)
- **Status:** Built, ready to test
- **Requirement:** bubblewrap + strace
- **Command:** `./warden run --backend linux --policy test-policy.yaml`

### macOS ✅
- **Native Backend:** sandbox-exec (Seatbelt)
- **Status:** Binary built for darwin-arm64
- **Requirement:** macOS 10.15+, no installation needed
- **Command:** `./dist/warden-darwin-arm64 run --backend seatbelt --policy test-policy.yaml`

### Windows ✅
- **Native Backend:** AppContainer + WFP
- **Status:** Binary built for windows-amd64
- **Requirement:** Windows 10/11 Pro/Enterprise
- **Command:** `.\dist\warden-windows-amd64.exe run --backend windows --policy test-policy.yaml`

### Docker (All Platforms) ✅
- **Backend:** Docker container fallback
- **Status:** Alpine image pulled, ready to test
- **Command:** `./warden run --backend docker --policy test-docker-policy.yaml`

## How to Test

### Option 1: Test on Current Linux System (Docker Backend)

```bash
cd /home/abdullah/Downloads/warden/warden-starter/warden

# Run MCP server in Docker sandbox
./warden run --backend docker --policy test-docker-policy.yaml

# In another terminal, send test requests
echo '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"read_file","arguments":{"path":"/app/test-data/test.txt"}}}' | nc localhost 3000
```

### Option 2: Test on macOS

Copy these files to macOS:
- `dist/warden-darwin-arm64`
- `mcp-test/test-server.js`
- `test-data/`
- `test-policy.yaml`

Then run:
```bash
chmod +x warden-darwin-arm64
./warden-darwin-arm64 run --backend seatbelt --policy test-policy.yaml
```

### Option 3: Test on Windows

Copy these files to Windows:
- `dist/warden-windows-amd64.exe`
- `mcp-test/test-server.js`
- `test-data/`
- `test-policy.yaml`

Then run:
```powershell
.\warden-windows-amd64.exe run --backend windows --policy test-policy.yaml
```

## Compatibility Matrix

Warden has been tested against 18 MCP servers:

### Pass (14 servers) ✅
- filesystem
- github
- slack
- postgres
- sqlite
- brave-search
- gdrive
- git
- memory
- time
- sequential-thinking
- notion
- linear
- tavily

### Conditional (2 servers) ⚠️
- fetch (needs per-deployment host list)
- kubernetes (needs cluster host per deployment)

### Fail (2 servers) ❌
- docker (requires host Docker socket)
- playwright (arbitrary web domains)

## Test Scenarios to Run

### 1. Basic Functionality Test
```bash
# Start server
./warden run --backend docker --policy test-docker-policy.yaml

# Test tool call (in MCP client)
{
  "method": "tools/call",
  "params": {
    "name": "read_file",
    "arguments": {
      "path": "/app/test-data/test.txt"
    }
  }
}
```
**Expected:** Success, returns file contents

### 2. Filesystem Boundary Test
```bash
# Try to read blocked file
{
  "method": "tools/call",
  "params": {
    "name": "read_file",
    "arguments": {
      "path": "/etc/shadow"
    }
  }
}
```
**Expected:** Error - file not accessible

### 3. Network Boundary Test
```bash
# Policy allows only api.github.com
# Try connecting to other host
```
**Expected:** Connection blocked in audit log

### 4. Write Permission Test
```bash
# Write to allowed directory
{
  "method": "tools/call",
  "params": {
    "name": "write_file",
    "arguments": {
      "path": "/app/test-data/new.txt",
      "content": "test"
    }
  }
}
```
**Expected:** Success

### 5. Resource Limit Test
```yaml
limits:
  memory_mb: 10
  timeout_s: 1
```
**Expected:** Process killed when limits exceeded

## Files Created

```
/home/abdullah/Downloads/warden/warden-starter/warden/
├── warden                           # Local build (6.2MB)
├── dist/
│   ├── warden-linux-amd64          # Linux binary (5.8MB)
│   ├── warden-linux-arm64          # Linux ARM binary (5.6MB)
│   ├── warden-darwin-amd64         # macOS Intel binary (6.0MB)
│   ├── warden-darwin-arm64         # macOS ARM binary (5.7MB)
│   └── warden-windows-amd64.exe    # Windows binary (6.2MB)
├── mcp-test/
│   ├── test-server.js              # Test MCP server
│   ├── TESTING.md                   # Quick start guide
│   └── node_modules/               # Dependencies
├── test-data/
│   └── test.txt                    # Test data file
├── test-policy.yaml                # Test policy
├── test-docker-policy.yaml         # Docker test policy
└── TESTING-PLATFORMS.md            # Cross-platform guide
```

## Next Steps

1. **Test on Linux:** Run with Docker backend ✅
2. **Test on macOS:** Copy binary and test with Seatbelt
3. **Test on Windows:** Copy binary and test with AppContainer
4. **Test real MCP servers:** Try filesystem, github, or postgres servers
5. **Review audit logs:** Check what gets blocked
6. **Customize policies:** Create policies for your specific MCP servers

## Getting Help

- Repository: https://github.com/warden-sandbox/warden
- Architecture: `ARCHITECTURE.md`
- Roadmap: `ROADMAP.md`
- Compatibility: `docs/compatibility.md`
