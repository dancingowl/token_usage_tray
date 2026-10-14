package main

import (
	"os"
	"testing"
	"time"
)

func TestSiblingHeartbeatSurvivesLongPollButExpiresAfterExit(t *testing.T) {
	t.Setenv("USAGE_TRAY_DATA_DIR", t.TempDir())
	path := heartbeatPath("claude")
	if err := os.WriteFile(path, []byte("running"), 0o600); err != nil {
		t.Fatal(err)
	}
	if !siblingProcessAlive("claude", time.Now().Add(-time.Hour)) {
		t.Fatal("live slow-poll sibling disappeared")
	}
	old := time.Now().Add(-time.Minute)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}
	if siblingProcessAlive("claude", time.Now()) {
		t.Fatal("exited sibling retained by recent quota")
	}
}
