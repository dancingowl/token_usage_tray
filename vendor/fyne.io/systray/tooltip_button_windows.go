//go:build windows

package systray

import (
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	pCreateSolidBrush = g32.NewProc("CreateSolidBrush")
	pCreatePen        = g32.NewProc("CreatePen")
	pRoundRect        = g32.NewProc("RoundRect")
	pSaveDC           = g32.NewProc("SaveDC")
	pRestoreDC        = g32.NewProc("RestoreDC")
	pSetBkMode        = g32.NewProc("SetBkMode")
	pSetTextColor     = g32.NewProc("SetTextColor")
	pFillRect         = u32.NewProc("FillRect")
	pDrawText         = u32.NewProc("DrawTextW")
	pDrawFocusRect    = u32.NewProc("DrawFocusRect")
)

// DRAWITEMSTRUCT and TRACKMOUSEEVENT have pointer-aligned members. Leaving
// their alignment to Go preserves both the 32-bit and 64-bit Win32 layouts.
// https://learn.microsoft.com/en-us/windows/win32/api/winuser/ns-winuser-drawitemstruct
type clickButtonDrawItem struct {
	Type, ID, ItemID, Action, State uint32
	Window, DC                      windows.Handle
	Rect                            tooltipRect
	Data                            uintptr
}

// LPARAM is a message-specific integer union, so it must remain uintptr in
// the shared callback. Read its native DRAWITEMSTRUCT into an owned snapshot
// rather than reconstructing a Go pointer from that integer. This structure
// contains only scalar values and Win32 handles, never retained Go pointers.
func readClickButtonDrawItem(address uintptr) (clickButtonDrawItem, bool) {
	var item clickButtonDrawItem
	var read uintptr
	size := unsafe.Sizeof(item)
	err := windows.ReadProcessMemory(windows.CurrentProcess(), address, (*byte)(unsafe.Pointer(&item)), size, &read)
	return item, err == nil && read == size
}

type clickMouseTracking struct {
	Size, Flags uint32
	Window      windows.Handle
	HoverTime   uint32
}

type clickButtonPalette struct{ fill, border, text uintptr }

func clickButtonColors(state uint32, hovered bool) clickButtonPalette {
	if state&0x0004 != 0 { // ODS_DISABLED
		return clickButtonPalette{0x00f8f6f5, 0x00eae5e1, 0x00afa39a}
	}
	if state&0x0001 != 0 { // ODS_SELECTED
		return clickButtonPalette{0x00f0e5dd, 0x00b6a89a, clickTipTextColor}
	}
	if hovered {
		return clickButtonPalette{0x00f4ede8, 0x00c5b8aa, clickTipTextColor}
	}
	// Neutral Fluent-style actions have equal weight. The popup is already
	// a compact summary, so none of refresh/detail/settings needs promotion.
	return clickButtonPalette{0x00f8f5f3, 0x00e5ddd7, clickTipTextColor}
}

func clickButtonBounds(client tooltipRect, dpi uintptr, index, count int) tooltipRect {
	if dpi == 0 {
		dpi = 96
	}
	padding, gap, height := int32(12*dpi/96), int32(8*dpi/96), int32(36*dpi/96)
	available := client.Right - client.Left - 2*padding - int32(count-1)*gap
	if available < int32(count) {
		available = int32(count)
	}
	// Divide the available span rather than a rounded width, giving the last
	// button the remainder and preserving the same outer padding at all DPI.
	left := client.Left + padding + available*int32(index)/int32(count) + gap*int32(index)
	right := client.Left + padding + available*int32(index+1)/int32(count) + gap*int32(index)
	return tooltipRect{left, client.Bottom - padding - height, right, client.Bottom - padding}
}

func (tip *trackingTooltip) drawActionButton(item *clickButtonDrawItem) bool {
	index := int(item.ID) - 0x7100
	if item.Type != 4 || index < 0 || index >= len(tip.buttons) || tip.buttons[index] != item.Window || item.DC == 0 {
		return false
	}
	colors := clickButtonColors(item.State, tip.hovered == item.Window)
	background, _, _ := pCreateSolidBrush.Call(clickTipBackground)
	fill, _, _ := pCreateSolidBrush.Call(colors.fill)
	dpi := tip.fontDPI
	if dpi == 0 {
		dpi = 96
	}
	stroke := uintptr(dpi / 96)
	if stroke == 0 {
		stroke = 1
	}
	pen, _, _ := pCreatePen.Call(0, stroke, colors.border) // PS_SOLID
	defer pDeleteObject.Call(background)
	defer pDeleteObject.Call(fill)
	defer pDeleteObject.Call(pen)
	if background == 0 || fill == 0 || pen == 0 {
		return false
	}
	saved, _, _ := pSaveDC.Call(uintptr(item.DC))
	if saved == 0 {
		return false
	}
	// Restore before deleting selected GDI objects; WM_DRAWITEM requires us
	// to return the borrowed DC in its original state.
	defer pRestoreDC.Call(uintptr(item.DC), saved)
	pFillRect.Call(uintptr(item.DC), uintptr(unsafe.Pointer(&item.Rect)), background)
	pSelectObject.Call(uintptr(item.DC), fill)
	pSelectObject.Call(uintptr(item.DC), pen)
	radius := uintptr(12 * dpi / 96) // RoundRect expects the corner ellipse diameter.
	inset := int32(stroke / 2)
	pRoundRect.Call(uintptr(item.DC), uintptr(item.Rect.Left+inset), uintptr(item.Rect.Top+inset),
		uintptr(item.Rect.Right-inset), uintptr(item.Rect.Bottom-inset), radius, radius)
	if tip.font != 0 {
		pSelectObject.Call(uintptr(item.DC), uintptr(tip.font))
	}
	pSetBkMode.Call(uintptr(item.DC), 1) // TRANSPARENT
	pSetTextColor.Call(uintptr(item.DC), colors.text)
	text, _ := windows.UTF16FromString(clickCardActions[index].label)
	textRect := item.Rect
	pDrawText.Call(uintptr(item.DC), uintptr(unsafe.Pointer(&text[0])), uintptr(len(text)-1),
		uintptr(unsafe.Pointer(&textRect)), 0x0001|0x0004|0x0020|0x0800) // center, vcenter, single line, no prefix
	runtime.KeepAlive(text)
	if item.State&0x0010 != 0 && item.State&0x0200 == 0 { // ODS_FOCUS, ODS_NOFOCUSRECT
		inset := int32(4 * dpi / 96)
		focus := tooltipRect{item.Rect.Left + inset, item.Rect.Top + inset, item.Rect.Right - inset, item.Rect.Bottom - inset}
		pDrawFocusRect.Call(uintptr(item.DC), uintptr(unsafe.Pointer(&focus)))
	}
	return true
}
