//go:build windows

package systray

import (
	"fmt"
	"runtime"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	tooltipControls       = windows.NewLazySystemDLL("Comctl32.dll")
	pSetWindowSubclass    = tooltipControls.NewProc("SetWindowSubclass")
	pRemoveWindowSubclass = tooltipControls.NewProc("RemoveWindowSubclass")
	pDefSubclassProc      = tooltipControls.NewProc("DefSubclassProc")
	pGetClientRect        = u32.NewProc("GetClientRect")
	pSetCapture           = u32.NewProc("SetCapture")
	pReleaseCapture       = u32.NewProc("ReleaseCapture")
	pGetCapture           = u32.NewProc("GetCapture")
	pTrackMouseEvent      = u32.NewProc("TrackMouseEvent")
	pInvalidateRect       = u32.NewProc("InvalidateRect")
	pIsWindowEnabled      = u32.NewProc("IsWindowEnabled")
	clickCardWindows      sync.Map // HWND -> *trackingTooltip, no retained Go pointer in Win32.
	clickCardSubclass     uintptr
)

func init() { clickCardSubclass = windows.NewCallback(clickCardSubclassProc) }

var clickCardActions = [...]struct{ label, action string }{
	{"새로고침", "refresh"}, {"상세 보기", "detail"}, {"설정", "settings"},
}

// SetWindowSubclass must be called on the owning window thread. Win32 retains
// only a callback and integer ID; Go ownership stays in clickCardWindows.
// https://learn.microsoft.com/en-us/windows/win32/api/commctrl/nf-commctrl-setwindowsubclass
func (tip *trackingTooltip) subclassWindow(hwnd windows.Handle) error {
	clickCardWindows.Store(hwnd, tip)
	if ok, _, err := pSetWindowSubclass.Call(uintptr(hwnd), clickCardSubclass, 1, 0); ok == 0 {
		clickCardWindows.Delete(hwnd)
		return fmt.Errorf("subclass click card: %w", err)
	}
	return nil
}

func (tip *trackingTooltip) createActionButtons(instance windows.Handle) error {
	if err := tip.subclassWindow(tip.window); err != nil {
		return err
	}
	class, _ := windows.UTF16PtrFromString("BUTTON")
	for i, action := range clickCardActions {
		label, _ := windows.UTF16PtrFromString(action.label)
		// WS_CHILD | WS_VISIBLE | BS_OWNERDRAW. Keep native button semantics,
		// with the card's own flat rounded appearance and no focus activation.
		hwnd, _, err := pCreateWindowEx.Call(0, uintptr(unsafe.Pointer(class)), uintptr(unsafe.Pointer(label)),
			0x40000000|0x10000000|0x0000000b, 0, 0, 1, 1, uintptr(tip.window), uintptr(0x7100+i), uintptr(instance), 0)
		runtime.KeepAlive(label)
		if hwnd == 0 {
			return fmt.Errorf("create %s click action: %w", action.action, err)
		}
		tip.buttons[i] = windows.Handle(hwnd)
		if err := tip.subclassWindow(tip.buttons[i]); err != nil {
			return err
		}
		pSendMessage.Call(hwnd, 0x0030, uintptr(tip.font), 0) // WM_SETFONT
	}
	runtime.KeepAlive(class)
	return nil
}

func (tip *trackingTooltip) layoutActionButtons() {
	var rect tooltipRect
	if ok, _, _ := pGetClientRect.Call(uintptr(tip.window), uintptr(unsafe.Pointer(&rect))); ok == 0 {
		return
	}
	dpi := tip.fontDPI
	if dpi == 0 {
		dpi = 96
	}
	for i, button := range tip.buttons {
		bounds := clickButtonBounds(rect, dpi, i, len(tip.buttons))
		pSetWindowPos.Call(uintptr(button), 0, uintptr(bounds.Left), uintptr(bounds.Top),
			uintptr(bounds.Right-bounds.Left), uintptr(bounds.Bottom-bounds.Top), 0x0004|0x0010|0x0200)
	}
}

func clickCardSubclassProc(hwnd uintptr, message uint32, wParam, lParam, subclassID, data uintptr) uintptr {
	value, exists := clickCardWindows.Load(windows.Handle(hwnd))
	if !exists {
		result, _, _ := pDefSubclassProc.Call(hwnd, uintptr(message), wParam, lParam)
		return result
	}
	tip := value.(*trackingTooltip)
	buttonIndex := -1
	for i, button := range tip.buttons {
		if uintptr(button) == hwnd {
			buttonIndex = i
			break
		}
	}
	switch message {
	case 0x000f: // WM_PAINT
		if hwnd == uintptr(tip.window) && len(tip.rows) != 0 {
			tip.paintRows()
			return 0
		}
	case 0x0318: // WM_PRINTCLIENT, also used by offscreen native render tests.
		if hwnd == uintptr(tip.window) && len(tip.rows) != 0 && wParam != 0 {
			var client tooltipRect
			pGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&client)))
			tip.drawRows(wParam, client)
			return 0
		}
	case 0x0082: // WM_NCDESTROY, including partial construction failure.
		pRemoveWindowSubclass.Call(hwnd, clickCardSubclass, subclassID)
		clickCardWindows.Delete(windows.Handle(hwnd))
	case 0x0021: // WM_MOUSEACTIVATE: receive click without taking keyboard focus.
		return 3 // MA_NOACTIVATE
	case 0x0084: // WM_NCHITTEST: tracking tooltips otherwise pass mouse input through.
		return 1 // HTCLIENT
	case 0x0005: // WM_SIZE
		if hwnd == uintptr(tip.window) {
			tip.layoutActionButtons()
		}
	case 0x002b: // WM_DRAWITEM: BS_OWNERDRAW sends painting to the parent.
		if hwnd == uintptr(tip.window) && lParam != 0 {
			if item, ok := readClickButtonDrawItem(lParam); ok && tip.drawActionButton(&item) {
				return 1
			}
		}
	case 0x0111: // WM_COMMAND, e.g. accessibility-triggered BM_CLICK.
		id := int(wParam&0xffff) - 0x7100
		if id >= 0 && id < len(clickCardActions) && uintptr(tip.buttons[id]) == lParam && wParam>>16 == 0 && clickButtonEnabled(lParam) {
			notifyClickAction(clickCardActions[id].action)
			tip.dismissAt = time.Now().Add(clickTipDuration * time.Millisecond)
			return 0
		}
	case 0x0201: // WM_LBUTTONDOWN: button default handling calls SetFocus.
		if buttonIndex >= 0 && clickButtonEnabled(hwnd) {
			tip.pressed = windows.Handle(hwnd)
			pSetCapture.Call(hwnd)
			pSendMessage.Call(hwnd, 0x00f3, 1, 0) // BM_SETSTATE
			return 0
		}
	case 0x0200: // WM_MOUSEMOVE, also while dragging outside under capture.
		if buttonIndex >= 0 {
			var rect tooltipRect
			pGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&rect)))
			p := point{int32(int16(lParam & 0xffff)), int32(int16((lParam >> 16) & 0xffff))}
			inside := tooltipContainsPoint(rect, p)
			hovered := windows.Handle(0)
			if inside && clickButtonEnabled(hwnd) {
				hovered = windows.Handle(hwnd)
				tracking := clickMouseTracking{Window: windows.Handle(hwnd), Flags: 0x00000002} // TME_LEAVE
				tracking.Size = uint32(unsafe.Sizeof(tracking))
				pTrackMouseEvent.Call(uintptr(unsafe.Pointer(&tracking)))
			}
			tip.setHoveredButton(hovered)
			if tip.pressed == windows.Handle(hwnd) {
				selected := uintptr(0)
				if inside {
					selected = 1
				}
				pSendMessage.Call(hwnd, 0x00f3, selected, 0)
			}
		}
	case 0x02a3: // WM_MOUSELEAVE
		if tip.hovered == windows.Handle(hwnd) {
			tip.setHoveredButton(0)
		}
	case 0x0202: // WM_LBUTTONUP: dispatch only on release inside the pressed button.
		if buttonIndex >= 0 {
			pressed := tip.pressed == windows.Handle(hwnd)
			tip.pressed = 0
			pSendMessage.Call(hwnd, 0x00f3, 0, 0)
			pReleaseCapture.Call()
			var rect tooltipRect
			pGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&rect)))
			p := point{int32(int16(lParam & 0xffff)), int32(int16((lParam >> 16) & 0xffff))}
			if pressed && tooltipContainsPoint(rect, p) && clickButtonEnabled(hwnd) {
				notifyClickAction(clickCardActions[buttonIndex].action)
				tip.dismissAt = time.Now().Add(clickTipDuration * time.Millisecond)
			}
			return 0
		}
	case 0x001f, 0x0215: // WM_CANCELMODE / WM_CAPTURECHANGED cancel the press.
		if buttonIndex >= 0 {
			tip.pressed = 0
			pSendMessage.Call(hwnd, 0x00f3, 0, 0)
		}
	}
	result, _, _ := pDefSubclassProc.Call(hwnd, uintptr(message), wParam, lParam)
	return result
}

func clickButtonEnabled(hwnd uintptr) bool {
	enabled, _, _ := pIsWindowEnabled.Call(hwnd)
	return enabled != 0
}

func (tip *trackingTooltip) setHoveredButton(button windows.Handle) {
	if tip.hovered == button {
		return
	}
	previous := tip.hovered
	tip.hovered = button
	for _, hwnd := range []windows.Handle{previous, button} {
		if hwnd != 0 {
			pInvalidateRect.Call(uintptr(hwnd), 0, 0)
		}
	}
}

// Summary refreshes hide and resize the card before showing it again. Cancel
// a press at that boundary so a release cannot act on the replacement card.
func (tip *trackingTooltip) cancelButtonInteraction() {
	tip.setHoveredButton(0)
	pressed := tip.pressed
	tip.pressed = 0
	if pressed == 0 {
		return
	}
	pSendMessage.Call(uintptr(pressed), 0x00f3, 0, 0) // BM_SETSTATE
	if captured, _, _ := pGetCapture.Call(); captured == uintptr(pressed) {
		pReleaseCapture.Call()
	}
}
