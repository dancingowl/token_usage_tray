//go:build !windows

package main

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

func startupEnabled() (bool, bool) {
	app := appBundle()
	if runtime.GOOS != "darwin" || app == "" {
		return false, false
	}
	name := strings.TrimSuffix(filepath.Base(app), ".app")
	out, err := exec.Command("osascript", "-e", `tell application "System Events" to count (every login item whose name is "`+escapeAS(name)+`")`).Output()
	if err != nil {
		return false, true
	}
	n, _ := strconv.Atoi(strings.TrimSpace(string(out)))
	return n > 0, true
}

func setStartupEnabled(enabled bool) error {
	app := appBundle()
	if runtime.GOOS != "darwin" || app == "" {
		return fmt.Errorf("이 환경에서는 자동 실행을 설정할 수 없습니다")
	}
	name := strings.TrimSuffix(filepath.Base(app), ".app")
	remove := `tell application "System Events" to delete (every login item whose name is "` + escapeAS(name) + `")`
	if !enabled {
		return osa(remove)
	}
	_ = osa(remove)
	return osa(`tell application "System Events" to make login item at end with properties {path:"` + escapeAS(app) + `", hidden:true}`)
}
