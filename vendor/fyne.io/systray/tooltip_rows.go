package systray

// ClickTooltipRow is one quota in the Windows click card. Remaining is a
// percentage; a negative value means unknown and uses neutral text.
type ClickTooltipRow struct {
	Label, Value, Detail string
	Remaining            int
	// Countdown and ResetDate share the second line, followed by Label below.
	// Status adds a separate line for stale data, elapsed resets, or limits.
	// Detail remains the fallback for callers without structured reset data.
	Countdown, ResetDate, Status string
	// Supplement is a separate, neutral detail line, such as a credit balance.
	Supplement string
}
