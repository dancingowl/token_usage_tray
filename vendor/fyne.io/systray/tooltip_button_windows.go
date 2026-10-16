//go:build windows

package systray

import (
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
		return clickButtonPalette{clickTipBackground, clickTipBackground, 0x00afa39a}
	}
	if state&0x0001 != 0 { // ODS_SELECTED
		return clickButtonPalette{0x00ece6e1, 0x00d6cec7, clickTipTextColor}
	}
	if hovered {
		return clickButtonPalette{0x00f5f1ed, 0x00e5ddd7, clickTipTextColor}
	}
	// Ghost actions blend into the card until hover or press. Native button
	// labels and semantics remain available to accessibility tools.
	return clickButtonPalette{clickTipBackground, clickTipBackground, clickTipDetailColor}
}

func clickButtonBounds(client tooltipRect, dpi uintptr, index, count int) tooltipRect {
	if dpi == 0 {
		dpi = 96
	}
	padding, gap, size := int32(12*dpi/96), int32(4*dpi/96), int32(28*dpi/96)
	if count != 2 || index < 0 || index >= count {
		return tooltipRect{}
	}
	left := client.Right - padding - size - int32(count-1-index)*(size+gap)
	top := client.Top + padding
	return tooltipRect{left, top, left + size, top + size}
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
	if !drawClickActionIcon(uintptr(item.DC), item.Rect, dpi, clickCardActions[index].action, colors.text, colors.fill) {
		return false
	}
	if item.State&0x0010 != 0 && item.State&0x0200 == 0 { // ODS_FOCUS, ODS_NOFOCUSRECT
		inset := int32(4 * dpi / 96)
		focus := tooltipRect{item.Rect.Left + inset, item.Rect.Top + inset, item.Rect.Right - inset, item.Rect.Bottom - inset}
		pDrawFocusRect.Call(uintptr(item.DC), uintptr(unsafe.Pointer(&focus)))
	}
	return true
}
