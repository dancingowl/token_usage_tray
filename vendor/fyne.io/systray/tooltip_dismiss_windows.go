//go:build windows

package systray

import (
	"fmt"
	"log"
	"runtime"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

const wmClickOutside = 0x8000 + 42

var (
	pSetWindowsHookEx             = u32.NewProc("SetWindowsHookExW")
	pUnhookWindowsHookEx          = u32.NewProc("UnhookWindowsHookEx")
	pCallNextHookEx               = u32.NewProc("CallNextHookEx")
	pWindowFromPhysicalPoint      = u32.NewProc("WindowFromPhysicalPoint")
	pIsChild                      = u32.NewProc("IsChild")
	pSetThreadDpiAwarenessContext = u32.NewProc("SetThreadDpiAwarenessContext")
	pShellNotifyIconGetRect       = s32.NewProc("Shell_NotifyIconGetRect")
	clickDismissOwners            sync.Map // installing OS thread ID -> *winTray
	clickDismissCallback          = windows.NewCallback(clickDismissMouseProc)
)

type clickDismissState struct {
	hook       windows.Handle
	thread     uint32
	generation uintptr
	iconRect   tooltipRect // physical coordinates, refreshed outside the hook
}

// A no-activate popup never receives focus-loss messages when another app is
// clicked. Watch mouse presses only while it is visible, without consuming them.
// The callback runs on the installing tray thread and only queues UI work.
// https://learn.microsoft.com/en-us/windows/win32/winmsg/lowlevelmouseproc
func clickDismissMouseProc(code int32, message, data uintptr) uintptr {
	if code == 0 && clickDismissPress(message) && data != 0 {
		if value, ok := clickDismissOwners.Load(windows.GetCurrentThreadId()); ok {
			tray := value.(*winTray)
			var cursor point // MSLLHOOKSTRUCT starts with a full-width POINT.
			var read uintptr
			if err := windows.ReadProcessMemory(windows.CurrentProcess(), data, (*byte)(unsafe.Pointer(&cursor)), unsafe.Sizeof(cursor), &read); err == nil && read == unsafe.Sizeof(cursor) {
				tray.queueClickOutside(cursor, clickWindowAtPhysicalPoint(cursor))
			}
		}
	}
	result, _, _ := pCallNextHookEx.Call(0, uintptr(code), message, data)
	return result
}

func clickDismissPress(message uintptr) bool {
	return message == 0x0201 || message == 0x0204 || message == 0x0207 || message == 0x020b
}

func clickWindowAtPhysicalPoint(cursor point) windows.Handle {
	var hwnd uintptr
	if unsafe.Sizeof(uintptr(0)) == 8 {
		packed := uint64(uint32(cursor.X)) | uint64(uint32(cursor.Y))<<32
		hwnd, _, _ = pWindowFromPhysicalPoint.Call(uintptr(packed))
	} else {
		hwnd, _, _ = pWindowFromPhysicalPoint.Call(uintptr(uint32(cursor.X)), uintptr(uint32(cursor.Y)))
	}
	return windows.Handle(hwnd)
}

func (t *winTray) queueClickOutside(cursor point, target windows.Handle) {
	tip := t.clickTooltip
	if tip == nil || !tip.visible || tip.dismiss.generation == 0 || target == tip.window {
		return
	}
	if child, _, _ := pIsChild.Call(uintptr(tip.window), uintptr(target)); child != 0 {
		return
	}
	// Preserve the tray's own single/double-click handling, including its
	// delayed release; hiding on its DOWN would otherwise reopen on its UP.
	if tooltipContainsPoint(tip.dismiss.iconRect, cursor) {
		return
	}
	pPostMessage.Call(uintptr(t.window), wmClickOutside, tip.dismiss.generation, 0)
}

func (t *winTray) dismissClickOutside(generation uintptr) {
	if tip := t.clickTooltip; tip != nil && tip.visible && generation != 0 && generation == tip.dismiss.generation {
		t.cancelSingleClick()
		t.hideClickTooltip()
	}
}

func (t *winTray) startClickOutsideWatch() error {
	tip := t.clickTooltip
	if tip.dismiss.hook != 0 {
		return nil
	}
	tip.dismiss.generation = t.clicks.nextTimerID()
	t.refreshClickIconRect()
	if rect := tip.dismiss.iconRect; rect.Right <= rect.Left || rect.Bottom <= rect.Top {
		tip.dismiss = clickDismissState{}
		return fmt.Errorf("locate tray icon for outside-click watch")
	}
	tip.dismiss.thread = windows.GetCurrentThreadId()
	clickDismissOwners.Store(tip.dismiss.thread, t)
	hook, _, err := pSetWindowsHookEx.Call(14, clickDismissCallback, uintptr(t.instance), 0) // WH_MOUSE_LL
	if hook == 0 {
		clickDismissOwners.Delete(tip.dismiss.thread)
		tip.dismiss = clickDismissState{}
		return fmt.Errorf("watch click outside tooltip: %w", err)
	}
	tip.dismiss.hook = windows.Handle(hook)
	return nil
}

func (t *winTray) stopClickOutsideWatch() {
	tip := t.clickTooltip
	if tip == nil {
		return
	}
	hook, thread := tip.dismiss.hook, tip.dismiss.thread
	tip.dismiss = clickDismissState{}
	if hook == 0 {
		return
	}
	clickDismissOwners.Delete(thread)
	if ok, _, err := pUnhookWindowsHookEx.Call(uintptr(hook)); ok == 0 {
		log.Printf("systray error: unable to stop outside-click watch: %s", err)
	}
}

func (t *winTray) refreshClickIconRect() {
	if t.nid == nil || t.clickTooltip == nil {
		return
	}
	// Query Shell under per-monitor awareness so the cached rectangle uses
	// the hook's physical coordinate space, even in a DPI-unaware process.
	if pSetThreadDpiAwarenessContext.Find() != nil {
		return
	}
	previous, _, _ := pSetThreadDpiAwarenessContext.Call(^uintptr(3)) // DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2 (-4)
	if previous == 0 {
		return
	}
	defer pSetThreadDpiAwarenessContext.Call(previous)
	t.muNID.RLock()
	identifier := struct {
		Size   uint32
		Window windows.Handle
		ID     uint32
		GUID   windows.GUID
	}{Window: t.nid.Wnd, ID: t.nid.ID, GUID: t.nid.GuidItem}
	t.muNID.RUnlock()
	identifier.Size = uint32(unsafe.Sizeof(identifier))
	var rect tooltipRect
	result, _, _ := pShellNotifyIconGetRect.Call(uintptr(unsafe.Pointer(&identifier)), uintptr(unsafe.Pointer(&rect)))
	runtime.KeepAlive(identifier)
	if int32(result) >= 0 && rect.Right > rect.Left && rect.Bottom > rect.Top {
		t.clickTooltip.dismiss.iconRect = rect
	}
}
