#!/bin/sh
# Test network connectivity
echo "Testing network connectivity..."
if ping -c 1 -W 2 8.8.8.8 > /dev/null 2>&1; then
    echo '{"jsonrpc":"2.0","id":1,"result":{"content":[{"type":"text","text":"Network: Connected to 8.8.8.8"}]}}'
else
    echo '{"jsonrpc":"2.0","id":1,"result":{"content":[{"type":"text","text":"Network: Blocked or unreachable"}]}}'
fi
