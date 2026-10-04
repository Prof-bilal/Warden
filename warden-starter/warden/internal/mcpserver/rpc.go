// Package mcpserver exposes a narrow, Warden-backed tool surface over the
// Model Context Protocol (stdio transport) so ChatGPT/Codex and other MCP
// clients can inspect policies, run sandboxed workloads, and read audit
// traces.
//
// The package is a thin integration layer: all enforcement stays in the
// existing Warden packages (internal/policy, internal/audit, internal/
// sandbox). It never simulates a security result — every allow/deny figure
// comes from Warden's own policy engine or audit log.
package mcpserver

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// rpcRequest is one JSON-RPC 2.0 request or notification on the MCP stdio
// channel. Notifications (no id) are acknowledged implicitly and never
// answered, per the MCP stdio transport.
type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

const (
	codeParseError     = -32700
	codeInvalidRequest = -32600
	codeMethodNotFound = -32601
	codeInvalidParams  = -32602
	codeInternalError  = -32603
)

// Serve runs the MCP stdio loop until stdin is exhausted. Each line is one
// JSON-RPC message (newline-delimited, per the MCP stdio transport).
// Requests are handled sequentially: handlers are local and bounded, and a
// long `run_sandbox` may legitimately make the client wait.
func Serve(stdin io.Reader, stdout io.Writer) error {
	s := NewServer()
	scanner := bufio.NewScanner(stdin)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	enc := json.NewEncoder(stdout)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var req rpcRequest
		if err := json.Unmarshal(line, &req); err != nil {
			// Unrecoverable per JSON-RPC: the id is unknowable, so reply with a
			// null id and let the client correlate by ordering. Not counted as
			// an id-bearing response.
			_ = enc.Encode(map[string]any{
				"jsonrpc": "2.0",
				"id":      nil,
				"error":   rpcError{Code: codeParseError, Message: "parse error: request must be a single JSON object per line"},
			})
			continue
		}
		if len(req.ID) == 0 || bytes.Equal(req.ID, []byte("null")) {
			// Notification (no id, or a parse-failed id): no response.
			continue
		}
		result, rpcErr := s.handle(&req)
		resp := map[string]any{"jsonrpc": "2.0", "id": req.ID}
		if rpcErr != nil {
			resp["error"] = *rpcErr
		} else {
			resp["result"] = result
		}
		if err := enc.Encode(resp); err != nil {
			return fmt.Errorf("write MCP response: %w", err)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read MCP request: %w", err)
	}
	return nil
}

// handle dispatches one request to the right handler, converting Go errors
// into JSON-RPC errors. It never panics across the boundary: an unexpected
// handler error is reported as an internal error result instead.
func (s *Server) handle(req *rpcRequest) (result any, rpcErr *rpcError) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(stdErr, "warden mcp: internal error in %s: %v\n", req.Method, r)
			result = nil
			rpcErr = &rpcError{Code: codeInternalError, Message: "internal error handling request"}
		}
	}()
	switch req.Method {
	case "initialize":
		return s.handleInitialize(req.Params), nil
	case "ping":
		return map[string]any{}, nil
	case "tools/list":
		return map[string]any{"tools": ToolDefinitions()}, nil
	case "tools/call":
		return s.handleToolsCall(req.Params)
	case "resources/list":
		return map[string]any{"resources": WidgetResources()}, nil
	case "resources/read":
		return s.handleResourcesRead(req.Params)
	default:
		return nil, &rpcError{Code: codeMethodNotFound, Message: fmt.Sprintf("unknown method %q (this server supports initialize, ping, tools/list, tools/call, resources/list, resources/read)", req.Method)}
	}
}

type initializeParams struct {
	ProtocolVersion string `json:"protocolVersion"`
}

// protocolVersion is the bounded MCP revision this server implements.
// Warden fails closed: an unknown client version is answered with the
// version we actually support rather than a promise we cannot keep.
const protocolVersion = "2025-06-18"

// stdErr is the server's diagnostic channel. MCP stdio carries JSON-RPC only,
// so logs must never mix into stdout. Tests can swap it.
var stdErr io.Writer = os.Stderr

func (s *Server) handleInitialize(raw json.RawMessage) map[string]any {
	var p initializeParams
	_ = json.Unmarshal(raw, &p) // unvalidated params fall back to our version
	return map[string]any{
		"protocolVersion": protocolVersion,
		"capabilities": map[string]any{
			"tools":     map[string]any{},
			"resources": map[string]any{},
		},
		"serverInfo": map[string]any{
			"name":    "warden",
			"version": s.version(),
			// Positioning, also used by client discovery surfaces.
			"title": "Warden",
		},
		"instructions": "Warden runs MCP servers and workloads in a deny-by-default sandbox. " +
			"Use inspect_policy to see what a policy allows, run_sandbox to execute a workload under an explicit policy, " +
			"trace_execution to review what a run attempted, and explain_denial to see why Warden blocked an access. " +
			"Anything a policy does not explicitly grant is denied.",
	}
}
