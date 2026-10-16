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

func TestStartupShortcutRecognizesExecutableFileIdentity(t *testing.T) {
	root := t.TempDir()
	t.Setenv("APPDATA", root)
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	// A hard link gives us a distinct spelling of the same file even when
	// 8.3 short-name generation is disabled on the test volume.
	alias := filepath.Join(root, "executable-alias.exe")
	if err := os.Link(exe, alias); err != nil {
		t.Fatal(err)
	}
	different := filepath.Join(root, "different.exe")
	if err := os.WriteFile(different, []byte("different executable"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(startupLink()), 0o700); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, target string
		want         bool
	}{{"same file alias", alias, true}, {"different existing file", different, false}} {
		t.Run(tc.name, func(t *testing.T) {
			script := "$s = New-Object -ComObject WScript.Shell; $l = $s.CreateShortcut('" + psQuote(startupLink()) + "'); $l.TargetPath = '" + psQuote(tc.target) + "'; $l.Save()"
			if err := runPS("usage-tray-test-target-identity.ps1", script); err != nil {
				t.Fatal(err)
			}
			if enabled, supported := startupEnabled(); enabled != tc.want || !supported {
				t.Fatalf("startupEnabled() = (%v, %v), want (%v, true)", enabled, supported, tc.want)
			}
		})
	}
}
