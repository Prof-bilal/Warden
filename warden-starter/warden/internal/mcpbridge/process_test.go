package mcpbridge

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestCancellationUnblocksStalledInputAndReapsProcess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix fixture; native Windows process lifecycle requires its platform suite")
	}
	path := filepath.Join(t.TempDir(), "trusted-relay-fixture")
	if e := os.WriteFile(path, []byte("#!/bin/sh\nexec /bin/sleep 30\n"), 0700); e != nil {
		t.Fatal(e)
	}
	parent, cancel := context.WithCancel(context.Background())
	defer cancel()
	p, e := StartProcess(parent, path, []string{"run", "fixture"}, []string{}, io.Discard, 1024*1024, func(string) error { return nil })
	if e != nil {
		t.Fatal(e)
	}
	defer p.Close()
	ctx, c := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer c()
	params, _ := json.Marshal(map[string]string{"large": strings.Repeat("x", 512*1024)})
	done := make(chan error, 1)
	go func() {
		_, e := p.RoundTrip(ctx, Message{JSONRPC: "2.0", ID: json.RawMessage("1"), Method: "tools/call", Params: params})
		done <- e
	}()
	select {
	case e := <-done:
		if e == nil {
			t.Fatal("stalled request returned success")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("cancellation left pipe write blocked")
	}
	if e = p.Close(); e != nil {
		t.Fatal(e)
	}
}

func TestCompletedNotificationContextCleanupKeepsProcessAlive(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix fixture; native Windows process lifecycle requires its platform suite")
	}
	path := filepath.Join(t.TempDir(), "trusted-reading-fixture")
	if e := os.WriteFile(path, []byte("#!/bin/sh\nwhile IFS= read -r line; do :; done\n"), 0700); e != nil {
		t.Fatal(e)
	}
	parent, cancel := context.WithCancel(context.Background())
	defer cancel()
	p, e := StartProcess(parent, path, []string{"run", "fixture"}, []string{}, io.Discard, 1024*1024, func(string) error { return nil })
	if e != nil {
		t.Fatal(e)
	}
	defer p.Close()
	for i := 0; i < 500; i++ {
		ctx, c := context.WithCancel(context.Background())
		e = p.Notify(ctx, Message{JSONRPC: "2.0", Method: "notifications/initialized"})
		c()
		if e != nil {
			t.Fatalf("successful notification cleanup killed connection at %d: %v", i, e)
		}
	}
	select {
	case <-p.stopped:
		t.Fatal("notification cleanup stopped the connection")
	default:
	}
}
