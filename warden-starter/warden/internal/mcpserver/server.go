package mcpserver

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/warden-sandbox/warden/internal/version"
)

// Server holds per-session state for the MCP plugin. It deliberately keeps
// no security state: policies are read from disk on every call (so user
// edits take effect) and enforcement is delegated to the warden CLI and
// policy engine. The only state is a bounded registry of recent runs so
// trace_execution can address a specific execution by run_id without the
// client handling byte offsets.
type Server struct {
	mu   sync.Mutex
	runs map[string]runRecord
}

// runRecord points at where one run's audit events begin, so the trace for
// that execution can be extracted from Warden's shared audit log.
type runRecord struct {
	LogPath    string    `json:"log_path"`
	StartOffet int64     `json:"start_offset"`
	PolicyPath string    `json:"policy_path"`
	Command    []string  `json:"command"`
	StartedAt  time.Time `json:"started_at"`
}

// maxRunRegistry bounds in-memory run records: the plugin never becomes an
// unbounded log store. Old records remain inspectable through log_path.
const maxRunRegistry = 64

func NewServer() *Server {
	return &Server{runs: map[string]runRecord{}}
}

func (s *Server) version() string {
	if version.Version == "" {
		return "dev"
	}
	return version.Version
}

func (s *Server) rememberRun(id string, rec runRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.runs) >= maxRunRegistry {
		// Drop the oldest record (registry is small; linear scan is fine).
		var oldestID string
		var oldest time.Time
		for k, v := range s.runs {
			if oldestID == "" || v.StartedAt.Before(oldest) {
				oldestID, oldest = k, v.StartedAt
			}
		}
		delete(s.runs, oldestID)
	}
	s.runs[id] = rec
}

func (s *Server) getRun(id string) (runRecord, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.runs[id]
	return rec, ok
}

// toolCallParams is the MCP tools/call request envelope.
type toolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

// toolResult is what a tool handler produces. Text is the human/LLM-readable
// summary; Structured is the machine-readable payload mirrored into the UI
// widget; Widget names the resource used to render it.
type toolResult struct {
	Text       string
	Structured map[string]any
	Widget     string
}

// toMCPResult shapes a handler result per the MCP tools/call schema, with
// OpenAI Apps widget metadata attached so ChatGPT can render the focused
// policy/execution/trace views.
func (r *toolResult) toMCPResult() map[string]any {
	out := map[string]any{
		"content": []map[string]any{{"type": "text", "text": r.Text}},
	}
	if r.Structured != nil {
		out["structuredContent"] = r.Structured
	}
	if r.Widget != "" {
		out["_meta"] = map[string]any{
			// MCP Apps standard key plus the OpenAI Apps SDK alias, so both
			// hosts can locate the widget resource.
			"ui/resourceUri":        r.Widget,
			"openai/outputTemplate": r.Widget,
		}
	}
	return out
}

func (s *Server) handleToolsCall(raw json.RawMessage) (any, *rpcError) {
	var p toolCallParams
	if err := json.Unmarshal(raw, &p); err != nil || p.Name == "" {
		return nil, &rpcError{Code: codeInvalidParams, Message: "tools/call requires {name, arguments}"}
	}
	res, err := s.callTool(p.Name, p.Arguments)
	if err != nil {
		// Tool-level errors are reported as tool results with isError, not
		// transport errors, so the client can show the message to the user.
		return map[string]any{
			"content": []map[string]any{{"type": "text", "text": err.Error()}},
			"isError": true,
		}, nil
	}
	return res.toMCPResult(), nil
}

func (s *Server) callTool(name string, rawArgs json.RawMessage) (*toolResult, error) {
	args := map[string]any{}
	if len(rawArgs) > 0 {
		if err := json.Unmarshal(rawArgs, &args); err != nil {
			return nil, fmt.Errorf("arguments must be a JSON object: %v", err)
		}
	}
	switch name {
	case "inspect_policy":
		return s.toolInspectPolicy(args)
	case "run_sandbox":
		return s.toolRunSandbox(args)
	case "trace_execution":
		return s.toolTraceExecution(args)
	case "explain_denial":
		return s.toolExplainDenial(args)
	case "generate_policy":
		return s.toolGeneratePolicy(args)
	default:
		return nil, fmt.Errorf("unknown tool %q (supported: inspect_policy, run_sandbox, trace_execution, explain_denial, generate_policy)", name)
	}
}

type resourceReadParams struct {
	URI string `json:"uri"`
}

func (s *Server) handleResourcesRead(raw json.RawMessage) (any, *rpcError) {
	var p resourceReadParams
	if err := json.Unmarshal(raw, &p); err != nil || p.URI == "" {
		return nil, &rpcError{Code: codeInvalidParams, Message: "resources/read requires {uri}"}
	}
	html, ok := widgetHTML(p.URI)
	if !ok {
		return nil, &rpcError{Code: codeInvalidParams, Message: fmt.Sprintf("unknown resource %q", p.URI)}
	}
	return map[string]any{
		"contents": []map[string]any{{
			"uri":      p.URI,
			"mimeType": "text/html",
			"text":     html,
		}},
	}, nil
}

// wardenBin resolves the warden CLI executable used to spawn sandboxed runs.
// In production this is the running binary itself (`warden mcp` is a
// subcommand of `warden`), so the plugin always drives the same tested
// runtime. Tests can override it with an env var.
func wardenBin() (string, error) {
	if p := os.Getenv("WARDEN_MCP_BIN"); p != "" {
		return p, nil
	}
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("locate warden binary: %w", err)
	}
	return exe, nil
}
