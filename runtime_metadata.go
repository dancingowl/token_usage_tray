package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"time"
)

const dashboardRevision = "2026-10-23-auto-update-instant-settings"

var processInstanceID = newInstanceID()

func newInstanceID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err == nil {
		return hex.EncodeToString(b[:])
	}
	return fmt.Sprintf("%d-%d", os.Getpid(), time.Now().UnixNano())
}

func dashboardInstanceID() string { return processInstanceID }
func dashboardUIRevision() string { return dashboardRevision }
