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
	pInitCommonControlsEx = windows.NewLazySystemDLL("Comctl32.dll").NewProc("InitCommonControlsEx")
	pSendMessage          = u32.NewProc("SendMessageW")
	pSetTimer             = u32.NewProc("SetTimer")
	pKillTimer            = u32.NewProc("KillTimer")
	pGetDoubleClickTime   = u32.NewProc("GetDoubleClickTime")
	pMonitorFromRect      = u32.NewProc("MonitorFromRect")
	pGetMonitorInfo       = u32.NewProc("GetMonitorInfoW")
	pGetDpiForWindow      = u32.NewProc("GetDpiForWindow")
	pSetWindowPos         = u32.NewProc("SetWindowPos")
	pGetWindowRect        = u32.NewProc("GetWindowRect")
)

const (
	ttmAddToolW          = 0x0400 + 50
	ttmUpdateTipTextW    = 0x0400 + 57
	ttmTrackActivate     = 0x0400 + 17
	ttmTrackPosition     = 0x0400 + 18
	ttmSetMaxTipWidth    = 0x0400 + 24
	clickTipDuration     = 7000
	clickTipTextCapacity = 1024
)

type clickTooltipBuffer [clickTipTextCapacity]uint16

// SetClickTooltip sets the multiline summary displayed by a single left click
// on Windows. It is independent of SetTooltip's hover text. An empty string
// restores the hover text fallback. Text is snapshotted when the card opens.
func SetClickTooltip(text string) {
	if err := wt.setClickTooltip(text); err != nil {
		log.Printf("systray error: unable to set click tooltip: %s", err)
	}
}

func (t *winTray) setClickTooltip(text string) error {
	encoded, err := windows.UTF16FromString(text)
	if err != nil {
		return err
	}
	var buffer clickTooltipBuffer
	copy(buffer[:len(buffer)-1], encoded)
	// Preserve the terminating NUL and never split a supplementary character
	// at the fixed buffer boundary (a UTF-16 high/low surrogate pair).
	if last := buffer[len(buffer)-2]; last >= 0xd800 && last <= 0xdbff {
		buffer[len(buffer)-2] = 0
	}
	t.muNID.Lock()
	t.clickText = buffer
	t.muNID.Unlock()
	return nil
}

func (t *winTray) clickTooltipText() clickTooltipBuffer {
	t.muNID.RLock()
	defer t.muNID.RUnlock()
	text := t.clickText
	if text[0] == 0 && t.nid != nil {
		copy(text[:], t.nid.Tip[:])
	}
	text[len(text)-1] = 0
	return text
}

type tooltipRect struct{ Left, Top, Right, Bottom int32 }

// Matches TTTOOLINFOW including its pointer-sized members on 32/64-bit Windows.
// https://learn.microsoft.com/en-us/windows/win32/api/commctrl/ns-commctrl-tttoolinfow
type tooltipToolInfo struct {
	Size, Flags uint32
	Window      windows.Handle
	ID          uintptr
	Rect        tooltipRect
	Instance    windows.Handle
	Text        *uint16
	Param       uintptr
	Reserved    uintptr
}

type trackingTooltip struct {
	window  windows.Handle
	tool    tooltipToolInfo
	text    clickTooltipBuffer
	pinner  runtime.Pinner
	visible bool
	font    windows.Handle
	fontDPI uintptr
}

// All state and Win32 operations below belong to wndProc's OS thread.
// SetTooltip/SetClickTooltip can run on other goroutines; they update text under
// muNID without operating on HWNDs.
type trayClickState struct {
	ignoreNextUp bool
	sequence     uintptr
	pendingTimer uintptr
	hideTimer    uintptr
	position     point
}

func (s *trayClickState) acceptRelease() bool {
	if s.ignoreNextUp {
		s.ignoreNextUp = false
		return false
	}
	return true
}

func (s *trayClickState) nextTimerID() uintptr {
	// KillTimer leaves already queued WM_TIMER messages intact. Never reusing
	// a cancelled ID makes those messages harmless, including after reshowing.
	s.sequence++
	if s.sequence < 0x7000 {
		s.sequence = 0x7000
	}
	return s.sequence
}

func (s *trayClickState) takeTimer(id uintptr) (single, hide bool) {
	if id != 0 && id == s.pendingTimer {
		s.pendingTimer = 0
		return true, false
	}
	if id != 0 && id == s.hideTimer {
		s.hideTimer = 0
		return false, true
	}
	return false, false
}

func (t *winTray) deferSingleClick() {
	if !t.clicks.acceptRelease() {
		return
	}
	t.cancelSingleClick()
	var cursor point
	if ok, _, err := pGetCursorPos.Call(uintptr(unsafe.Pointer(&cursor))); ok == 0 {
		log.Printf("systray error: unable to locate click tooltip: %s", err)
		return
	}
	t.clicks.position = cursor
	id := t.clicks.nextTimerID()
	delay, _, _ := pGetDoubleClickTime.Call()
	// Waiting through the OS double-click interval preserves the second click.
	if ok, _, err := pSetTimer.Call(uintptr(t.window), id, delay, 0); ok == 0 {
		log.Printf("systray error: unable to defer single click: %s", err)
		return
	}
	t.clicks.pendingTimer = id
}

func (t *winTray) cancelSingleClick() {
	if id := t.clicks.pendingTimer; id != 0 {
		t.clicks.pendingTimer = 0
		pKillTimer.Call(uintptr(t.window), id)
	}
}

func (t *winTray) handleClickTimer(id uintptr) {
	single, hide := t.clicks.takeTimer(id)
	if !single && !hide {
		return
	}
	pKillTimer.Call(uintptr(t.window), id)
	if hide {
		t.hideClickTooltip()
		return
	}
	if t.clickTooltip != nil && t.clickTooltip.visible {
		t.hideClickTooltip()
		return
	}
	if err := t.showClickTooltip(t.clicks.position); err != nil {
		log.Printf("systray error: unable to show click tooltip: %s", err)
	}
}

func (t *winTray) newClickTooltip() (*trackingTooltip, error) {
	// ICC_WIN95_CLASSES registers the tooltip class without requiring CGO.
	controls := struct{ Size, Classes uint32 }{Size: 8, Classes: 0x000000ff}
	if ok, _, err := pInitCommonControlsEx.Call(uintptr(unsafe.Pointer(&controls))); ok == 0 {
		return nil, fmt.Errorf("initialize tooltip controls: %w", err)
	}
	className, _ := windows.UTF16PtrFromString("tooltips_class32")
	const (
		wsExTopmost    = 0x00000008
		wsExNoActivate = 0x08000000
		wsPopup        = 0x80000000
		ttsAlwaysTip   = 0x01
		ttsNoPrefix    = 0x02
		ttsNoAnimate   = 0x10
		ttsNoFade      = 0x20
	)
	hwnd, _, err := pCreateWindowEx.Call(wsExTopmost|wsExNoActivate,
		uintptr(unsafe.Pointer(className)), 0,
		wsPopup|ttsAlwaysTip|ttsNoPrefix|ttsNoAnimate|ttsNoFade,
		0, 0, 0, 0, uintptr(t.window), 0, uintptr(t.instance), 0)
	runtime.KeepAlive(className)
	if hwnd == 0 {
		return nil, fmt.Errorf("create tooltip: %w", err)
	}
	tip := &trackingTooltip{window: windows.Handle(hwnd)}
	// The control retains lpszText beyond SendMessage. Pin the fixed backing
	// array until DestroyWindow, instead of retaining a movable Go pointer.
	tip.pinner.Pin(&tip.text[0])
	tip.tool = tooltipToolInfo{
		Flags:  0x0020 | 0x0080, // TTF_TRACK | TTF_ABSOLUTE; no mouse subclassing.
		Window: t.window, ID: 1, Text: &tip.text[0],
	}
	// Use the v2 structure prefix, also supported by Comctl32 v5 when the
	// process does not opt into v6 with a visual-styles manifest. lpReserved
	// is the only omitted member and is not needed for tracking tooltips.
	tip.tool.Size = uint32(unsafe.Offsetof(tip.tool.Reserved))
	if tip.sendTool(ttmAddToolW, 0) == 0 {
		pDestroyWindow.Call(hwnd)
		tip.pinner.Unpin()
		return nil, fmt.Errorf("register tracking tooltip tool")
	}
	tip.configureCardStyle()
	if err := tip.scaleCardStyle(96); err != nil {
		// Styling is optional: keep the working click tooltip if GDI cannot
		// allocate a font, while reporting the degraded appearance.
		log.Printf("systray error: unable to style click tooltip: %s", err)
	}
	return tip, nil
}

func (tip *trackingTooltip) sendTool(message, wParam uintptr) uintptr {
	result, _, _ := pSendMessage.Call(uintptr(tip.window), message, wParam, uintptr(unsafe.Pointer(&tip.tool)))
	runtime.KeepAlive(tip)
	return result
}

func (tip *trackingTooltip) move(p point) {
	// HWND_TOPMOST, SWP_NOSIZE | SWP_NOACTIVATE | SWP_NOOWNERZORDER.
	pSetWindowPos.Call(uintptr(tip.window), ^uintptr(0), uintptr(p.X), uintptr(p.Y), 0, 0, 0x0001|0x0010|0x0200)
}

func clickTooltipWorkArea(cursor point) (tooltipRect, error) {
	rect := tooltipRect{cursor.X, cursor.Y, cursor.X + 1, cursor.Y + 1}
	monitor, _, _ := pMonitorFromRect.Call(uintptr(unsafe.Pointer(&rect)), 2) // MONITOR_DEFAULTTONEAREST
	info := struct {
		Size          uint32
		Monitor, Work tooltipRect
		Flags         uint32
	}{}
	info.Size = uint32(unsafe.Sizeof(info))
	if ok, _, err := pGetMonitorInfo.Call(monitor, uintptr(unsafe.Pointer(&info))); ok == 0 {
		return tooltipRect{}, fmt.Errorf("get tooltip monitor: %w", err)
	}
	return info.Work, nil
}

func clickTooltipPosition(cursor point, width, height, gap int32, work tooltipRect) point {
	// Prefer above the clicked icon, then fit within this monitor's work area.
	x := cursor.X - width/2
	y := cursor.Y - height - gap
	x = clampTooltipCoordinate(x, work.Left, work.Right-width)
	y = clampTooltipCoordinate(y, work.Top, work.Bottom-height)
	return point{x, y}
}

func clampTooltipCoordinate(value, lower, upper int32) int32 {
	if value > upper {
		value = upper
	}
	if value < lower {
		value = lower
	}
	return value
}

func packTooltipPosition(p point) uintptr {
	// MAKELPARAM preserves signed 16-bit coordinates on monitors left/above
	// the primary display. The final SetWindowPos also handles larger desktops.
	return uintptr(uint32(uint16(p.X)) | uint32(uint16(p.Y))<<16)
}

func (t *winTray) showClickTooltip(cursor point) error {
	text := t.clickTooltipText()
	if text[0] == 0 {
		return nil
	}
	work, err := clickTooltipWorkArea(cursor)
	if err != nil {
		return err
	}
	if t.clickTooltip == nil {
		t.clickTooltip, err = t.newClickTooltip()
		if err != nil {
			return err
		}
	}
	tip := t.clickTooltip
	tip.text = text
	tip.sendTool(ttmUpdateTipTextW, 0)
	// Move while hidden first so a per-monitor-aware control gets the DPI of
	// the clicked monitor. GetDpiForWindow returns 96 for DPI-unaware processes.
	tip.move(cursor)
	dpi := uintptr(96)
	if pGetDpiForWindow.Find() == nil {
		if value, _, _ := pGetDpiForWindow.Call(uintptr(tip.window)); value != 0 {
			dpi = value
		}
	}
	if err := tip.scaleCardStyle(dpi); err != nil {
		log.Printf("systray error: unable to scale click tooltip: %s", err)
	}
	gap := int32(8 * dpi / 96)
	maxWidth := clampTooltipCoordinate(int32(420*dpi/96), 1, work.Right-work.Left-2*gap)
	pSendMessage.Call(uintptr(tip.window), ttmSetMaxTipWidth, 0, uintptr(maxWidth))
	pSendMessage.Call(uintptr(tip.window), ttmTrackPosition, 0, packTooltipPosition(cursor))
	tip.sendTool(ttmTrackActivate, 1)
	tip.visible = true
	// Comctl32 v5 can crash in TTM_GETBUBBLESIZE before first activation.
	// Activation lays out the control synchronously; read its actual window
	// size and position it before returning to the message loop for painting.
	var rect tooltipRect
	if ok, _, err := pGetWindowRect.Call(uintptr(tip.window), uintptr(unsafe.Pointer(&rect))); ok == 0 {
		t.hideClickTooltip()
		return fmt.Errorf("measure tracking tooltip: %w", err)
	}
	position := clickTooltipPosition(cursor, rect.Right-rect.Left, rect.Bottom-rect.Top, gap, work)
	tip.move(position)
	id := t.clicks.nextTimerID()
	if ok, _, err := pSetTimer.Call(uintptr(t.window), id, clickTipDuration, 0); ok == 0 {
		t.hideClickTooltip()
		return fmt.Errorf("start tooltip dismiss timer: %w", err)
	}
	t.clicks.hideTimer = id
	return nil
}

func (t *winTray) hideClickTooltip() {
	if id := t.clicks.hideTimer; id != 0 {
		t.clicks.hideTimer = 0
		pKillTimer.Call(uintptr(t.window), id)
	}
	if tip := t.clickTooltip; tip != nil && tip.visible {
		tip.sendTool(ttmTrackActivate, 0)
		tip.visible = false
	}
}

func (t *winTray) destroyClickTooltip() {
	t.cancelSingleClick()
	t.hideClickTooltip()
	if tip := t.clickTooltip; tip != nil {
		pDestroyWindow.Call(uintptr(tip.window))
		tip.releaseCardFont()
		tip.pinner.Unpin()
		t.clickTooltip = nil
	}
}
