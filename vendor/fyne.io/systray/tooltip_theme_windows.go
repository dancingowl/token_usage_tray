//go:build windows

package systray

// Each card keeps one UI-thread palette snapshot. Updates arrive through the
// same locked tray state and posted message as quota rows, never through HWND
// operations from a polling goroutine.
type clickThemePalette struct {
	background, border, text, detail, danger, warning                            uintptr
	buttonHover, buttonHoverBorder, buttonPressed, buttonPressedBorder, disabled uintptr
}

func clickCardColors(dark bool) clickThemePalette {
	if dark {
		return clickThemePalette{
			background: 0x002e251d, border: 0x0057473a, text: 0x00f8f2ed, detail: 0x00cebfb2,
			danger: 0x009292ff, warning: 0x0066c3f0,
			buttonHover: 0x0045372b, buttonHoverBorder: 0x006a5747,
			buttonPressed: 0x005b4837, buttonPressedBorder: 0x00927c67, disabled: 0x0091877d,
		}
	}
	return clickThemePalette{
		background: clickTipBackground, border: clickTipBorderColor, text: clickTipTextColor, detail: clickTipDetailColor,
		danger: 0x002626b9, warning: 0x00006e96,
		buttonHover: 0x00f5f1ed, buttonHoverBorder: 0x00e5ddd7,
		buttonPressed: 0x00ece6e1, buttonPressedBorder: 0x00d6cec7, disabled: 0x00afa39a,
	}
}

func (tip *trackingTooltip) cardColors() clickThemePalette {
	if tip.palette.background == 0 {
		return clickCardColors(false)
	}
	return tip.palette
}

// SetClickTooltipTheme selects the resolved light or dark card palette. The
// caller resolves a system preference, so taskbar and app modes can differ.
func SetClickTooltipTheme(theme string) { wt.setClickTooltipTheme(theme == "dark") }

func (t *winTray) setClickTooltipTheme(dark bool) {
	t.muNID.Lock()
	changed := t.clickDark != dark
	t.clickDark = dark
	t.muNID.Unlock()
	if changed && t.isReady() {
		pPostMessage.Call(uintptr(t.window), wmClickSummaryChanged, 0, 0)
	}
}

func (t *winTray) clickTooltipColors() clickThemePalette {
	t.muNID.RLock()
	defer t.muNID.RUnlock()
	return clickCardColors(t.clickDark)
}
