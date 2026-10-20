//go:build !windows

package main

import (
	"os"
	"syscall"
)

func lockSettingsHandle(f *os.File) error {
	return syscall.Flock(int(f.Fd()), syscall.LOCK_EX)
}

func unlockSettingsHandle(f *os.File) {
	_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
}
