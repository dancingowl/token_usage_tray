//go:build windows

package systray

import (
	"fmt"
	"log"
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	pSetWindowTheme     = windows.NewLazySystemDLL("UxTheme.dll").NewProc("SetWindowTheme")
	pCreateFontIndirect = g32.NewProc("CreateFontIndirectW")
	pDeleteObject       = g32.NewProc("DeleteObject")
)

const (
	clickTipTitle      = "사용량 요약"
	clickTipBackground = 0x00fdfcfc // COLORREF for #FCFCFD.
	clickTipTextColor  = 0x00382920 // COLORREF for #202938.
	ttmSetTitleW       = 0x0400 + 33
	ttmSetMargin       = 0x0400 + 26
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
	pSendMessage.Call(uintptr(tip.window), 0x0400+19, clickTipBackground, 0) // TTM_SETTIPBKCOLOR
	pSendMessage.Call(uintptr(tip.window), 0x0400+20, clickTipTextColor, 0)  // TTM_SETTIPTEXTCOLOR
}

func clickTooltipFont(dpi uintptr) tooltipLogFont {
	font := tooltipLogFont{
		Height:  -int32((11*dpi + 36) / 72), // Segoe UI 11 pt, rounded to pixels.
		Weight:  400,
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
	font := clickTooltipFont(dpi)
	handle, _, err := pCreateFontIndirect.Call(uintptr(unsafe.Pointer(&font)))
	if handle == 0 {
		return fmt.Errorf("create Segoe UI tooltip font: %w", err)
	}
	oldFont := tip.font
	// The control does not own the supplied HFONT. Replace before deleting the
	// previous font, and release the last font only after destroying the HWND.
	pSendMessage.Call(uintptr(tip.window), 0x0030, handle, 0) // WM_SETFONT; still hidden.
	tip.font, tip.fontDPI = windows.Handle(handle), dpi
	if oldFont != 0 {
		pDeleteObject.Call(uintptr(oldFont))
	}
	padding := int32(12 * dpi / 96)
	margin := tooltipRect{padding, padding, padding, padding}
	pSendMessage.Call(uintptr(tip.window), ttmSetMargin, 0, uintptr(unsafe.Pointer(&margin)))
	title, _ := windows.UTF16PtrFromString(clickTipTitle)
	// Native title rendering supplies the bold heading above the body; the
	// small blue information glyph provides a restrained color accent.
	result, _, _ := pSendMessage.Call(uintptr(tip.window), ttmSetTitleW, 1, uintptr(unsafe.Pointer(title))) // TTI_INFO
	runtime.KeepAlive(title)
	if result == 0 {
		return fmt.Errorf("set tooltip heading")
	}
	return nil
}

func (tip *trackingTooltip) releaseCardFont() {
	if tip.font != 0 {
		pDeleteObject.Call(uintptr(tip.font))
		tip.font, tip.fontDPI = 0, 0
	}
}
