#!/bin/bash
# Acceptance test for `warden proxy` (P0-1 fix).
#
# Verifies the honest/functional proxy contract:
#   1. HTTP/SSE upstreams are rejected at startup (fail-closed)
#   2. A stdio upstream starts a real JSON-RPC bridge on TCP
#   3. Allowed tool calls are forwarded end-to-end
#   4. Disallowed tool calls are answered with a JSON-RPC error
#   5. Unparseable client input gets a parse error (-32700)
#   6. Every decision lands in the audit log
#
# Usage: bash test-mcp-proxy.sh [path-to-warden-binary]
set -u

WARDEN="${1:-}"
if [ -z "$WARDEN" ] && [ -x "$(pwd)/warden-starter/warden/warden" ]; then
    WARDEN="$(pwd)/warden-starter/warden/warden"
fi
if [ -z "$WARDEN" ]; then
    WARDEN="$(command -v warden || true)"
fi
if [ -z "$WARDEN" ] || [ ! -x "$WARDEN" ]; then
    echo "❌ warden binary not found (pass it as \$1 or build warden-starter/warden)"
    exit 1
fi
if ! "$WARDEN" proxy --help >/dev/null 2>&1 && ! "$WARDEN" help proxy 2>&1 | grep -q 'MCP'; then
    echo "❌ $WARDEN does not implement 'warden proxy' (stale/other install?)"
    exit 1
fi

TEST_DIR="/tmp/warden-mcp-proxy-test"
AUDIT_LOG="${XDG_STATE_HOME:-$HOME/.local/state}/warden/audit.jsonl"
PASS=0
FAIL=0

ok()   { PASS=$((PASS + 1)); echo "✅ $1"; }
bad()  { FAIL=$((FAIL + 1)); echo "❌ $1"; }

rm -rf "$TEST_DIR"
mkdir -p "$TEST_DIR"

# send.sh <port> <line> — send one JSON-RPC line to the proxy over a single
# TCP connection and print the first response line.
cat > "$TEST_DIR/send.sh" << 'EOF'
#!/bin/bash
PORT="$1"; LINE="$2"
exec 3<>/dev/tcp/127.0.0.1/"$PORT"
printf '%s\n' "$LINE" >&3
read -u 3 -t 5 resp
printf '%s\n' "$resp"
exec 3<&-
EOF

echo "🧪 Testing Warden MCP Proxy (stdio bridge contract)"
echo "   binary: $WARDEN"

# ---------------------------------------------------------------------------
echo ""
echo "🔍 Test 1: HTTP upstream rejected at startup (fail-closed)"
cat > "$TEST_DIR/http-policy.yaml" << 'EOF'
env:
  allow: ["PATH"]
mcp:
  upstream: "https://mcp.github.com/api"
  allow_tools: ["list_repos"]
EOF
OUT="$("$WARDEN" proxy --policy "$TEST_DIR/http-policy.yaml" 2>&1)"
if echo "$OUT" | grep -q 'only "stdio:<command>" is implemented'; then
    ok "HTTP upstream rejected with clear message"
else
    bad "HTTP upstream rejection message missing; got: $OUT"
fi

# ---------------------------------------------------------------------------
echo ""
echo "🔍 Test 2: stdio bridge starts and forwards allowed tool calls"
cat > "$TEST_DIR/stdio-policy.yaml" << 'EOF'
env:
  allow: ["PATH"]
mcp:
  upstream: "stdio:cat"
  allow_tools: ["read_file"]
EOF

PORT=18777
"$WARDEN" proxy --policy "$TEST_DIR/stdio-policy.yaml" --listen "127.0.0.1:$PORT" \
    > "$TEST_DIR/proxy.out" 2>&1 &
PROXY_PID=$!
sleep 1

if ! kill -0 "$PROXY_PID" 2>/dev/null; then
    bad "proxy did not stay up; output: $(cat "$TEST_DIR/proxy.out")"
    exit 1
fi
ok "proxy listening on 127.0.0.1:$PORT (stdio:cat upstream)"

# Allowed tool call must be forwarded and echoed back by the upstream.
ALLOWED='{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"read_file"}}'
RESP="$(bash "$TEST_DIR/send.sh" "$PORT" "$ALLOWED")"
if echo "$RESP" | grep -q '"id":1' && echo "$RESP" | grep -q 'read_file'; then
    ok "allowed tool call forwarded end-to-end"
else
    bad "allowed tool call not forwarded; got: $RESP"
fi

# ---------------------------------------------------------------------------
echo ""
echo "🔍 Test 3: disallowed tool call blocked with JSON-RPC error"
BLOCKED='{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"delete_everything"}}'
RESP="$(bash "$TEST_DIR/send.sh" "$PORT" "$BLOCKED")"
if echo "$RESP" | grep -q '"error"' && echo "$RESP" | grep -q 'blocked by Warden policy'; then
    ok "blocked tool call answered with policy error (not forwarded)"
else
    bad "blocked tool call not answered correctly; got: $RESP"
fi

# ---------------------------------------------------------------------------
echo ""
echo "🔍 Test 4: unparseable input rejected (never forwarded)"
RESP="$(bash "$TEST_DIR/send.sh" "$PORT" 'this is not json')"
if echo "$RESP" | grep -q '"code":-32700'; then
    ok "garbage input answered with parse error -32700"
else
    bad "garbage input not rejected with -32700; got: $RESP"
fi

kill "$PROXY_PID" 2>/dev/null
wait "$PROXY_PID" 2>/dev/null

# ---------------------------------------------------------------------------
echo ""
echo "🔍 Test 5: decisions audited"
if [ -f "$AUDIT_LOG" ] \
    && grep -q '"action":"stdio_proxy"' "$AUDIT_LOG" \
    && grep -q 'delete_everything' "$AUDIT_LOG"; then
    ok "audit log contains stdio_proxy and block records"
else
    bad "audit log missing proxy decisions ($AUDIT_LOG)"
fi

rm -rf "$TEST_DIR"

echo ""
echo "Summary: $PASS passed, $FAIL failed"
if [ "$FAIL" -ne 0 ]; then
    exit 1
fi
echo "🎉 warden proxy stdio bridge contract verified"

