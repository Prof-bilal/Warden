package autopilot

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeSuggestion(t *testing.T) {
	for _, content := range []string{ReviewedEdits, "grant-all", strings.Repeat("x", 64<<10)} {
		mode, err := decodeSuggestion(strings.NewReader(`{"choices":[{"message":{"content":"` + content + `"}}]}`))
		if content == ReviewedEdits {
			if err != nil || mode != ReviewedEdits {
				t.Fatalf("valid suggestion failed: %q, %v", mode, err)
			}
		} else if err == nil {
			t.Fatalf("unexpected response accepted: %q", mode)
		}
	}
}

func TestAdvisorAcceptsOnlyBoundedSuggestions(t *testing.T) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Skipf("local HTTP fixtures unavailable: %v", err)
	}
	server := &httptest.Server{Listener: listener, Config: &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer decoy" {
			t.Error("advisor credential missing from selected endpoint")
		}
		raw, err := io.ReadAll(r.Body)
		if err != nil || !strings.Contains(string(raw), "go.mod") || strings.Contains(string(raw), "untrusted-content") {
			t.Errorf("unexpected advisor payload: %s (%v)", raw, err)
		}
		switch r.URL.Path {
		case "/valid":
			io.WriteString(w, `{"choices":[{"message":{"content":"read-only"}}]}`)
		case "/redirect":
			http.Redirect(w, r, "/unexpected", http.StatusTemporaryRedirect)
		default:
			t.Error("advisor followed redirect")
		}
	})}}
	server.Start()
	defer server.Close()
	hints := []Hint{{Marker: "go.mod", Build: "untrusted-content"}}
	for _, path := range []string{"/valid", "/redirect"} {
		t.Run(path, func(t *testing.T) {
			advisor := CompatibleAdvisor{Endpoint: server.URL + path, Model: "fixture", APIKey: "decoy"}
			mode, err := advisor.Suggest(context.Background(), hints)
			if path == "/valid" {
				if err != nil || mode != ReadOnly {
					t.Fatalf("valid suggestion failed: %q, %v", mode, err)
				}
			} else if err == nil {
				t.Fatalf("unexpected response accepted: %q", mode)
			}
		})
	}
}
