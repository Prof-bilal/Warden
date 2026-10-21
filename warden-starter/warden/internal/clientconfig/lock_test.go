package clientconfig

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLockReclaimsStaleLock(t *testing.T) {
	config := filepath.Join(t.TempDir(), "config")
	// A PID that cannot be alive, plus an old timestamp, so both the liveness
	// probe and the age fallback classify it as stale.
	rec, _ := json.Marshal(lockRecord{PID: 1 << 30, Created: time.Now().Add(-time.Hour).Unix()})
	if err := os.WriteFile(config+".warden-lock", rec, 0o600); err != nil {
		t.Fatal(err)
	}
	release, err := lock(config)
	if err != nil {
		t.Fatalf("lock() did not reclaim a stale lock: %v", err)
	}
	release()
	if _, err := os.Stat(config + ".warden-lock"); !os.IsNotExist(err) {
		t.Fatal("release did not remove the lock")
	}
}

func TestLockRefusesLiveLock(t *testing.T) {
	config := filepath.Join(t.TempDir(), "config")
	rec, _ := json.Marshal(lockRecord{PID: os.Getpid(), Created: time.Now().Unix()})
	if err := os.WriteFile(config+".warden-lock", rec, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := lock(config); err == nil {
		t.Fatal("lock() stole a lock owned by a live process")
	}
}
