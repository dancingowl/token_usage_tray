//go:build !windows

package systray

// SetClickTooltip sets the Windows-only click summary. It has no effect on
// other platforms.
func SetClickTooltip(string) {}

// SetClickTooltipRows has no effect outside Windows.
func SetClickTooltipRows([]ClickTooltipRow) {}

// SetClickTooltipTheme has no effect outside Windows.
func SetClickTooltipTheme(string) {}

// SetVisible has no effect outside Windows, where the shared application icon
// remains available for opening settings.
func SetVisible(bool) {}
