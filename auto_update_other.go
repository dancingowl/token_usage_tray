//go:build !windows

package main

// Installation and replacement are deliberately unsupported outside Windows.
func automaticUpdateSupported() bool    { return false }
func handleAutomaticUpdateMode() bool   { return false }
func startAutomaticUpdates(quit func()) {}
func automaticUpdateHealthy()           {}
func automaticUpdateRestart() bool      { return false }
