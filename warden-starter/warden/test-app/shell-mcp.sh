#!/bin/sh
# Simple shell-based MCP server for testing
while read line; do
    # Echo back what we received
    echo "Received: $line" >&2
    
    # Parse JSON (simple approach)
    case "$line" in
        *'"method":"tools/list"'*)
            echo '{"jsonrpc":"2.0","id":1,"result":{"tools":[{"name":"read_file","description":"Read a file","inputSchema":{"type":"object","properties":{"path":{"type":"string"}},"required":["path"]}}]}}'
            ;;
        *'"method":"tools/call"'*)
            # Extract path from the request
            path=$(echo "$line" | grep -o '"path":"[^"]*"' | head -1 | cut -d'"' -f4)
            if [ -n "$path" ]; then
                if [ -f "$path" ]; then
                    content=$(cat "$path" 2>&1)
                    echo "{\"jsonrpc\":\"2.0\",\"id\":2,\"result\":{\"content\":[{\"type\":\"text\",\"text\":\"$content\"}]}}"
                else
                    echo "{\"jsonrpc\":\"2.0\",\"id\":2,\"error\":{\"code\":-32602,\"message\":\"File not found: $path\"}}"
                fi
            else
                echo '{"jsonrpc":"2.0","id":2,"error":{"code":-32602,"message":"Invalid params"}}'
            fi
            ;;
        *'"method":"initialize"'*)
            echo '{"jsonrpc":"2.0","id":0,"result":{"protocolVersion":"2024-11-05","capabilities":{"tools":{}},"serverInfo":{"name":"shell-mcp","version":"1.0.0"}}}'
            ;;
        *'"jsonrpc"*')
            echo '{"jsonrpc":"2.0","id":null,"result":{}}'
            ;;
        *'{"jsonrpc":"2.0"}')
            echo '{"jsonrpc":"2.0","id":null,"result":{}}'
            ;;
    esac
done
