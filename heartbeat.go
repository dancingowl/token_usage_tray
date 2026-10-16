package main

import (
	"os"
	"path/filepath"
	"time"
)

func heartbeatPath(agent string) string { return filepath.Join(dataDir(), "heartbeat-"+agent) }

func updateHeartbeat() {
	if tag() == "" {
		return
	}
	path := heartbeatPath(tag())
	now := time.Now()
	if err := os.Chtimes(path, now, now); os.IsNotExist(err) {
		_ = os.WriteFile(path, []byte("running\n"), 0o600)
	}
}

func siblingProcessAlive(agent string, lastUpdated time.Time) bool {
	if info, err := os.Stat(heartbeatPath(agent)); err == nil {
		return time.Since(info.ModTime()) <= 15*time.Second
	}
	return time.Since(lastUpdated) <= 30*time.Minute
}
