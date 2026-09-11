package proxy

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/warden-sandbox/warden/internal/audit"
)

// TestOneConnServerCompletesAllowedRequest is a regression test for the
// race where serveConn's deferred conn.Close fired as soon as Serve's second
// Accept returned EOF, cancelling the handler mid-request. Allowed proxied
// requests (which dial upstream before responding) came back with an empty
// reply; the denial path was fast enough to win the race and masked the bug.
// It mirrors the real topology: one TCP conn from the bridge, client keeps
// the connection open while awaiting the response.
func TestOneConnServerCompletesAllowedRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "RELAYED")
	}))
	defer srv.Close()

	s, err := Start([]string{"127.0.0.1"}, audit.New(discardWriter{}))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	go s.serveConn(serverConn)

	// Speak HTTP directly over the single bridged connection, exactly like
	// the sandboxed curl does through the bridge.
	addr := srv.Listener.Addr().String()
	req := "GET http://" + addr + "/ HTTP/1.1\r\nHost: " + addr + "\r\nConnection: close\r\n\r\n"
	if _, err := clientConn.Write([]byte(req)); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 8192)
	var sb strings.Builder
	for sb.Len() < len(buf) {
		m, err := clientConn.Read(buf[:])
		if m > 0 {
			sb.Write(buf[:m])
		}
		if err != nil || strings.Contains(sb.String(), "RELAYED") {
			break
		}
	}
	out := sb.String()
	if !strings.Contains(out, "RELAYED") {
		t.Fatalf("allowed proxied request did not complete; got:\n%s", out)
	}
	if strings.Contains(out, "502") || strings.Contains(out, "Bad Gateway") {
		t.Fatalf("allowed proxied request degraded to 502; got:\n%s", out)
	}
}

type discardWriter struct{}

func (discardWriter) Write(p []byte) (int, error) { return len(p), io.EOF }
