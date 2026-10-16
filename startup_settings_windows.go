package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func startupEnabled() (bool, bool) {
	info, err := os.Stat(startupLink())
	if err != nil || info.IsDir() || info.Size() == 0 {
		return false, true
	}
	exe, err := os.Executable()
	if err != nil {
		return false, true
	}
	script := "$s = New-Object -ComObject WScript.Shell; $l = $s.CreateShortcut('" + psQuote(startupLink()) + "'); [Console]::OutputEncoding = [System.Text.Encoding]::UTF8; Write-Output $l.TargetPath"
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script)
	hideWindow(cmd)
	out, err := cmd.Output()
	if err != nil {
		return false, true
	}
	target := strings.TrimSpace(string(out))
	if target == "" {
		return false, true
	}
	targetInfo, err := os.Stat(target)
	if err != nil || targetInfo.IsDir() {
		return false, true
	}
	exeInfo, err := os.Stat(exe)
	if err != nil {
		return false, true
	}
	// WScript may expand an 8.3 executable path to its long spelling. Compare
	// file identity so aliases work without accepting a different executable.
	return os.SameFile(targetInfo, exeInfo), true
}

// Only change the startup shortcut; unlike -install this never restarts the app or Explorer.
func setStartupEnabled(enabled bool) error {
	link := startupLink()
	if !enabled {
		err := os.Remove(link)
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(link), 0o700); err != nil {
		return err
	}
	script := "$s = New-Object -ComObject WScript.Shell\n" +
		"$l = $s.CreateShortcut('" + psQuote(link) + "')\n" +
		"$l.TargetPath = '" + psQuote(exe) + "'\n" +
		"$l.WorkingDirectory = '" + psQuote(filepath.Dir(exe)) + "'\n" +
		"$l.Description = 'token_usage_tray'\n$l.Save()\n"
	return runPS(fmt.Sprintf("token-usage-tray-startup-%d.ps1", os.Getpid()), script)
}
