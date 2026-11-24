package autopilot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"
)

// Advisor may suggest a workspace mode. Only the CLI can ask for approval.
type Advisor interface {
	Suggest(context.Context, []Hint) (string, error)
}

type CompatibleAdvisor struct {
	Endpoint string
	Model    string
	APIKey   string
}

func (a CompatibleAdvisor) Suggest(ctx context.Context, hints []Hint) (string, error) {
	u, err := url.Parse(a.Endpoint)
	if err != nil || u.Hostname() == "" || u.User != nil || u.Fragment != "" || u.RawQuery != "" || a.Model == "" {
		return "", fmt.Errorf("advisor requires a model and a clean chat-completions URL")
	}
	ip, _ := netip.ParseAddr(u.Hostname())
	if u.Scheme != "https" && !(u.Scheme == "http" && ip.IsLoopback()) {
		return "", fmt.Errorf("advisor requires HTTPS or an HTTP loopback IP")
	}
	markers := make([]string, 0, len(hints))
	for _, hint := range hints {
		markers = append(markers, hint.Marker)
	}
	body, err := json.Marshal(map[string]any{
		"model": a.Model,
		"messages": []map[string]string{
			{"role": "system", "content": "Suggest minimal workspace access for a coding project. Reply with exactly read-only or reviewed-edits. If unsure, choose read-only."},
			{"role": "user", "content": "Known project markers: " + strings.Join(markers, ", ")},
		},
	})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("could not build advisor request")
	}
	req.Header.Set("Content-Type", "application/json")
	if a.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+a.APIKey)
	}
	transport := &http.Transport{}
	defer transport.CloseIdleConnections()
	client := http.Client{
		Timeout: 15 * time.Second, Transport: transport,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("advisor request failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("advisor returned HTTP %d", resp.StatusCode)
	}
	return decodeSuggestion(resp.Body)
}

func decodeSuggestion(r io.Reader) (string, error) {
	const maxReply = 64 << 10
	reply, err := io.ReadAll(io.LimitReader(r, maxReply+1))
	if err != nil || len(reply) > maxReply {
		return "", fmt.Errorf("could not read bounded advisor response")
	}
	var result struct {
		Choices []struct {
			Message struct{ Content string }
		}
	}
	if err := json.Unmarshal(reply, &result); err != nil || len(result.Choices) != 1 {
		return "", fmt.Errorf("invalid advisor response")
	}
	mode := strings.TrimSpace(result.Choices[0].Message.Content)
	if mode != ReadOnly && mode != ReviewedEdits {
		return "", fmt.Errorf("advisor response did not name a supported mode")
	}
	return mode, nil
}
