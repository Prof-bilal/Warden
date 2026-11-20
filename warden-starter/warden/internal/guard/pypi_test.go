package guard

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestReviewPyPIWheels(t *testing.T) {
	hash := strings.Repeat("a", 64)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/pypi/example-tool/1.2.0/json" {
			t.Errorf("PyPI release request path = %q", r.URL.Path)
		}
		_, _ = fmt.Fprintf(w, `{
			"info":{"name":"example_tool","version":"1.2.0"},
			"urls":[
				{"packagetype":"bdist_wheel","filename":"example_tool-1.2.0-py3-none-any.whl",
				 "upload_time_iso_8601":"2026-11-10T10:00:00Z","url":"https://files.example/old.whl","digests":{"sha256":%q}},
				{"packagetype":"bdist_wheel","filename":"example_tool-1.2.0-py3-none-win_amd64.whl",
				 "upload_time_iso_8601":"2026-11-19T10:00:00Z","url":"https://files.example/new.whl","digests":{"sha256":%q}},
				{"packagetype":"bdist_wheel","filename":"example_tool-1.2.0-py2-none-any.whl", "yanked":true},
				{"packagetype":"sdist","filename":"example_tool-1.2.0.tar.gz"}
			]}`, hash, hash)
	}))
	defer server.Close()

	release, err := ReadPyPIRelease(context.Background(), server.Client(), server.URL, "Example.Tool", "1.2.0")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC)
	wheels, decision := ReviewPyPIWheels(release, now, ReleaseRules{})
	if len(release.Wheels) != 2 || len(wheels) != 1 || wheels[0].URL != "https://files.example/old.whl" || wheels[0].SHA256 != hash {
		t.Fatalf("release = %+v, eligible wheels = %+v", release, wheels)
	}
	if len(decision.Blocks) != 0 || len(decision.Warnings) != 2 {
		t.Fatalf("wheel decision = %+v", decision)
	}
	release.Wheels = nil
	wheels, decision = ReviewPyPIWheels(release, now, ReleaseRules{})
	if len(wheels) != 0 || len(decision.Blocks) == 0 || len(decision.Warnings) == 0 {
		t.Fatalf("source-only release was not excluded: %+v", decision)
	}
}
