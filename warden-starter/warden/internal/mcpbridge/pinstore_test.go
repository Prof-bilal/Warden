package mcpbridge

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestPinStoreRequiresExplicitAcceptance(t *testing.T) {
	store := PinStore{Dir: filepath.Join(t.TempDir(), "pins")}
	first, second, third := strings.Repeat("a", 64), strings.Repeat("b", 64), strings.Repeat("c", 64)
	if pin, err := store.Read("source", "read"); pin != "" || err != nil {
		t.Fatalf("fresh pin = %q, %v", pin, err)
	}
	if _, err := os.Stat(store.Dir); !os.IsNotExist(err) {
		t.Fatalf("read created pin directory: %v", err)
	}
	if pin, err := store.Observe("source", "read", first); pin != first || err != nil {
		t.Fatalf("observe first = %q, %v", pin, err)
	}
	reopened := PinStore{Dir: store.Dir}
	if pin, err := reopened.Observe("source", "read", second); pin != first || err != nil {
		t.Fatalf("changed definition replaced the pin: %q, %v", pin, err)
	}
	if pin, err := reopened.Status("source", "read"); pin != (ToolPin{Accepted: first, Observed: second}) || err != nil {
		t.Fatalf("change for review = %+v, %v", pin, err)
	}
	if err := reopened.Accept("source", "read", first, third); err == nil {
		t.Fatal("unobserved hash replaced the pin")
	}
	if err := reopened.Accept("source", "read", first, second); err != nil {
		t.Fatal(err)
	}
	if err := store.Accept("source", "read", first, third); err == nil {
		t.Fatal("stale acceptance overwrote the current pin")
	}
	if pin, err := reopened.Read("source", "read"); pin != second || err != nil {
		t.Fatalf("accepted pin = %q, %v", pin, err)
	}
	if _, err := store.Observe("other/source", "read", first); err == nil {
		t.Fatal("invalid server identity accepted")
	}
}

func TestPinStoreConcurrentFirstObservation(t *testing.T) {
	store := PinStore{Dir: t.TempDir()}
	digests := []string{strings.Repeat("a", 64), strings.Repeat("b", 64)}
	chosen := make([]string, len(digests))
	var done sync.WaitGroup
	for i, digest := range digests {
		done.Add(1)
		go func() {
			defer done.Done()
			var err error
			chosen[i], err = store.Observe("source", "read", digest)
			if err != nil {
				t.Error(err)
			}
		}()
	}
	done.Wait()
	if chosen[0] == "" || chosen[0] != chosen[1] {
		t.Fatalf("observers disagreed about the first pin: %v", chosen)
	}
}
