# Warden MCP Quick Reference

## 🚀 Quick Test (Right Now!)

```bash
cd /home/abdullah/Downloads/warden/warden-starter/warden

# Test with Docker backend (works immediately)
./warden run --backend docker --policy test-docker-policy.yaml
```

## 🐧 Test on Linux

```bash
# Install bubblewrap (if not present)
sudo pacman -S bubblewrap strace

# Test with native bwrap backend
./warden run --backend linux --policy test-policy.yaml
```

## 🍎 Test on macOS

Copy `dist/warden-darwin-arm64` to your Mac, then:
```bash
chmod +x warden-darwin-arm64
./warden-darwin-arm64 run --backend seatbelt --policy test-policy.yaml
```

## 🪟 Test on Windows

Copy `dist/warden-windows-amd64.exe` to your Windows PC, then:
```powershell
.\warden-windows-amd64.exe run --backend windows --policy test-policy.yaml
```

## 📦 Test with Docker (Any Platform)

```bash
# All platforms
docker pull alpine:3.20
./warden run --backend docker --policy test-docker-policy.yaml
```

## 🔍 Test MCP Server Tools

### Read File ✅
```json
{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"read_file","arguments":{"path":"/app/test-data/test.txt"}}}
```

### Write File ✅
```json
{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"write_file","arguments":{"path":"/app/test-data/new.txt","content":"Hello!"}}}
```

### List Directory ✅
```json
{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"list_directory","arguments":{"path":"/app/test-data"}}}
```

### Access Blocked File ❌
```json
{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"read_file","arguments":{"path":"/etc/shadow"}}}
```

## 📁 Key Files

| File | Purpose |
|------|---------|
| `dist/warden-linux-amd64` | Linux binary |
| `dist/warden-darwin-arm64` | macOS ARM binary |
| `dist/warden-windows-amd64.exe` | Windows binary |
| `mcp-test/test-server.js` | Test MCP server |
| `test-policy.yaml` | Test policy |
| `TESTING-SUMMARY.md` | Full summary |
| `TESTING-PLATFORMS.md` | Cross-platform guide |

## 🔐 Security Features

- **Filesystem:** Only allowed paths accessible
- **Network:** Only allowlisted hosts reachable
- **Environment:** Only specified env vars passed
- **Resources:** Memory and time limits enforced
- **Audit:** All access attempts logged

## 📊 Compatibility

- ✅ **14 servers pass:** filesystem, github, slack, postgres, sqlite, etc.
- ⚠️ **2 conditional:** fetch, kubernetes (need config)
- ❌ **2 fail:** docker, playwright (incompatible by design)

## 🎯 What to Test

1. **Basic functionality** - Does server start and respond?
2. **Filesystem boundary** - Can it access /etc/shadow?
3. **Network boundary** - Can it reach google.com?
4. **Env filtering** - Are sensitive vars blocked?
5. **Resource limits** - Does it respect memory/timeout?

## 📚 Documentation

- **Quick Start:** This file
- **Full Guide:** `TESTING-SUMMARY.md`
- **Cross-Platform:** `TESTING-PLATFORMS.md`
- **Architecture:** `ARCHITECTURE.md`
- **Compatibility:** `docs/compatibility.md`

## 🆘 Troubleshooting

| Problem | Solution |
|---------|----------|
| "strace not found" | Install strace or use Docker |
| "sandbox-exec not supported" | Use Docker fallback |
| Docker errors | Check Docker daemon |
| Path issues | Use absolute paths |

## ✅ Checklist

- [x] Warden built ✅
- [x] All platform binaries ready ✅
- [x] Test MCP server created ✅
- [x] Test data prepared ✅
- [x] Test policies created ✅
- [x] Documentation written ✅
- [ ] Test on current Linux system
- [ ] Test on macOS (if available)
- [ ] Test on Windows (if available)

## 🎉 Ready to Test!

You now have a complete Warden MCP testing setup with:
- ✅ 5 platform binaries (Linux x64/arm64, macOS x64/arm64, Windows)
- ✅ Test MCP server
- ✅ Policies for all backends
- ✅ Comprehensive documentation
- ✅ Ready to test on all three platforms!

**Start testing:** `./warden run --backend docker --policy test-docker-policy.yaml`
