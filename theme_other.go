//go:build !windows

package main

// Native click cards are Windows-only. The browser independently follows its
// platform's prefers-color-scheme media query on all other systems.
func systemAppDark() bool     { return false }
func systemTaskbarDark() bool { return false }
