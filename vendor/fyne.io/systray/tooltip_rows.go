package systray

// ClickTooltipRow is one two-line quota in the Windows click card. Remaining
// is a percentage; a negative value means unknown and uses neutral text.
type ClickTooltipRow struct {
	Label, Value, Detail string
	Remaining            int
}
