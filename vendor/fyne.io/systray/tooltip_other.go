//go:build !windows

package systray

// SetClickTooltip sets the Windows-only click summary. It has no effect on
// other platforms.
func SetClickTooltip(string) {}
