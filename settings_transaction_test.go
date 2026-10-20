package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestSettingsTransactionProcessHelper(t *testing.T) {
	if os.Getenv("USAGE_TRAY_LOCK_HELPER") != "1" {
		t.Skip("subprocess helper")
	}
	fmt.Println("READY")
	_, err := updateTraySettings(func(settings *traySettings) error {
		settings.Theme = "dark"
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestSettingsTransactionsPreserveChangesAcrossProcesses(t *testing.T) {
	t.Setenv("USAGE_TRAY_DATA_DIR", t.TempDir())
	if err := saveTraySettings(defaultTraySettings()); err != nil {
		t.Fatal(err)
	}
	unlock, err := acquireSettingsMutation()
	if err != nil {
		t.Fatal(err)
	}
	released := false
	defer func() {
		if !released {
			unlock()
		}
	}()
	cmd := exec.Command(os.Args[0], "-test.run=^TestSettingsTransactionProcessHelper$", "-test.timeout=15s")
	cmd.Env = append(os.Environ(), "USAGE_TRAY_LOCK_HELPER=1")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cmd.Process.Kill() }()
	if line, err := bufio.NewReader(out).ReadString('\n'); err != nil || line != "READY\n" {
		t.Fatalf("child did not start: %q %v %s", line, err, stderr.String())
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		t.Fatalf("child bypassed the parent's file lock: %v %s", err, stderr.String())
	case <-time.After(100 * time.Millisecond):
	}
	settings, err := loadTraySettings()
	if err != nil {
		t.Fatal(err)
	}
	settings.PollSeconds = 300
	if err := saveTraySettings(settings); err != nil {
		t.Fatal(err)
	}
	unlock()
	released = true
	if err := <-done; err != nil {
		t.Fatalf("child update failed: %v %s", err, stderr.String())
	}
	got, err := loadTraySettings()
	if err != nil || got.Theme != "dark" || got.PollSeconds != 300 {
		t.Fatalf("cross-process update lost a setting: %+v %v", got, err)
	}
}

func TestSettingsTransactionFailureDoesNotWriteAndReleasesLock(t *testing.T) {
	t.Setenv("USAGE_TRAY_DATA_DIR", t.TempDir())
	want := errors.New("change rejected")
	_, err := updateTraySettings(func(settings *traySettings) error {
		settings.Theme = "dark"
		return want
	})
	if !errors.Is(err, want) {
		t.Fatalf("callback error lost: %v", err)
	}
	got, err := updateTraySettings(func(settings *traySettings) error {
		if settings.Theme != "system" {
			t.Fatal("failed change was saved")
		}
		settings.PollSeconds = 120
		return nil
	})
	if err != nil || got.PollSeconds != 120 {
		t.Fatalf("lock not released: %+v %v", got, err)
	}
}
