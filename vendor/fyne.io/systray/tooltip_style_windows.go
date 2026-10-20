//go:build windows

package systray

import (
	"fmt"
	"log"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	pSetWindowTheme     = windows.NewLazySystemDLL("UxTheme.dll").NewProc("SetWindowTheme")
	pCreateFontIndirect = g32.NewProc("CreateFontIndirectW")
	pDeleteObject       = g32.NewProc("DeleteObject")
)

const (
	clickTipBackground  = 0x00fdfcfc // COLORREF for #FCFCFD.
	clickTipBorderColor = 0x00e9e4df // COLORREF for #DFE4E9.
	clickTipTextColor   = 0x00382920 // COLORREF for #202938.
	ttmSetTitleW        = 0x0400 + 33
	ttmSetMargin        = 0x0400 + 26
)

// LOGFONTW has the same layout on 32/64-bit Windows.
type tooltipLogFont struct {
	Height, Width, Escapement, Orientation, Weight int32
	Italic, Underline, StrikeOut, CharSet          uint8
	OutPrecision, ClipPrecision, Quality, Family   uint8
	FaceName                                       [32]uint16
}

func (tip *trackingTooltip) configureCardStyle() {
	// Tooltip color/margin messages are ignored when visual styles are active.
	// Disable them for this one control to keep the same card palette on both
	// Comctl32 v5 and a future v6 manifest, without affecting other windows.
	empty := [1]uint16{}
	result, _, _ := pSetWindowTheme.Call(uintptr(tip.window), uintptr(unsafe.Pointer(&empty[0])), uintptr(unsafe.Pointer(&empty[0])))
	if int32(result) < 0 {
		log.Printf("systray error: unable to customize tooltip theme: HRESULT %#x", result)
	}
	tip.configureCardColors()
}

func (tip *trackingTooltip) configureCardColors() {
	colors := tip.cardColors()
	pSendMessage.Call(uintptr(tip.window), 0x0400+19, colors.background, 0) // TTM_SETTIPBKCOLOR
	pSendMessage.Call(uintptr(tip.window), 0x0400+20, colors.text, 0)       // TTM_SETTIPTEXTCOLOR
}

func clickTooltipFont(dpi uintptr) tooltipLogFont {
	return clickCardFont(dpi, 10, 400)
}

func clickCardFont(dpi uintptr, points, weight int32) tooltipLogFont {
	font := tooltipLogFont{
		Height:  -int32((uintptr(points)*dpi + 36) / 72),
		Weight:  weight,
		Quality: 5, // CLEARTYPE_QUALITY; normal GDI font linking covers Korean.
	}
	name, _ := windows.UTF16FromString("Segoe UI")
	copy(font.FaceName[:], name)
	return font
}

func (tip *trackingTooltip) scaleCardStyle(dpi uintptr) error {
	if tip.font != 0 && tip.fontDPI == dpi {
		return nil
	}
	var handles [4]windows.Handle
	for i, font := range []tooltipLogFont{clickTooltipFont(dpi), clickCardFont(dpi, 13, 600), clickCardFont(dpi, 10, 400), clickCardFont(dpi, 10, 500)} {
		handle, _, err := pCreateFontIndirect.Call(uintptr(unsafe.Pointer(&font)))
		if handle == 0 {
			for _, allocated := range handles {
				if allocated != 0 {
					pDeleteObject.Call(uintptr(allocated))
				}
			}
			return fmt.Errorf("create Segoe UI card font: %w", err)
		}
		handles[i] = windows.Handle(handle)
	}
	oldFonts := [4]windows.Handle{tip.font, tip.valueFont, tip.detailFont, tip.countdownFont}
	handle := uintptr(handles[0])
	// The control does not own the supplied HFONT. Replace before deleting the
	// previous font, and release the last font only after destroying the HWND.
	pSendMessage.Call(uintptr(tip.window), 0x0030, handle, 0) // WM_SETFONT; still hidden.
	for _, button := range tip.buttons {
		if button != 0 {
			pSendMessage.Call(uintptr(button), 0x0030, handle, 1)
		}
	}
	tip.font, tip.valueFont, tip.detailFont, tip.fontDPI = handles[0], handles[1], handles[2], dpi
	tip.countdownFont = handles[3]
	for _, oldFont := range oldFonts {
		if oldFont != 0 {
			pDeleteObject.Call(uintptr(oldFont))
		}
	}
	padding := int32(12 * dpi / 96)
	// Plain-text fallback reserves a header for the icon actions. Structured
	// quota rows use their own measured layout and ignore this native margin.
	margin := tooltipRect{padding, padding + int32(36*dpi/96), padding, padding}
	pSendMessage.Call(uintptr(tip.window), ttmSetMargin, 0, uintptr(unsafe.Pointer(&margin)))
	// Clear both the title and its native information glyph for every mode.
	pSendMessage.Call(uintptr(tip.window), ttmSetTitleW, 0, 0)
	return nil
}

func (tip *trackingTooltip) releaseCardFont() {
	for _, font := range [4]windows.Handle{tip.font, tip.valueFont, tip.detailFont, tip.countdownFont} {
		if font != 0 {
			pDeleteObject.Call(uintptr(font))
		}
	}
	tip.font, tip.valueFont, tip.detailFont, tip.fontDPI = 0, 0, 0, 0
	tip.countdownFont = 0
}
