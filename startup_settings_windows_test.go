package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestStartupShortcutRoundTripInIsolatedDirectory(t *testing.T) {
	if _, err := exec.LookPath("powershell.exe"); err != nil {
		t.Skip("PowerShell unavailable")
	}
	root := t.TempDir()
	t.Setenv("APPDATA", root)
	if enabled, supported := startupEnabled(); enabled || !supported {
		t.Fatal(enabled, supported)
	}
	if err := setStartupEnabled(true); err != nil {
		t.Fatal(err)
	}
	link := startupLink()
	rel, err := filepath.Rel(root, link)
	if err != nil || len(rel) < 1 || rel[0] == '.' {
		t.Fatal("shortcut escaped isolated directory", link)
	}
	if info, err := os.Stat(link); err != nil || info.Size() == 0 {
		t.Fatal("shortcut not created", err)
	}
	if enabled, _ := startupEnabled(); !enabled {
		t.Fatal("startup not enabled")
	}
	if err := setStartupEnabled(false); err != nil {
		t.Fatal(err)
	}
	if enabled, _ := startupEnabled(); enabled {
		t.Fatal("startup not removed")
	}
	if err := os.WriteFile(link, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if enabled, _ := startupEnabled(); enabled {
		t.Fatal("empty shortcut reported enabled")
	}
	_ = os.Remove(link)
	script := "$s = New-Object -ComObject WScript.Shell; $l = $s.CreateShortcut('" + psQuote(link) + "'); $l.TargetPath = '" + psQuote(filepath.Join(root, "missing.exe")) + "'; $l.Save()"
	if err := runPS("usage-tray-test-missing-target.ps1", script); err != nil {
		t.Fatal(err)
	}
	if enabled, _ := startupEnabled(); enabled {
		t.Fatal("missing shortcut target reported enabled")
	}
}
