//go:build windows

package systray

import (
	"runtime"
	"strings"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

func TestDoubleClickSuppressesFinalRelease(t *testing.T) {
	s := trayClickState{}
	if !s.acceptRelease() {
		t.Fatal("first release must schedule a single click")
	}
	s.pendingTimer = s.nextTimerID()
	stale := s.pendingTimer
	// The double-click handler cancels the pending single and suppresses UP.
	s.pendingTimer = 0
	s.ignoreNextUp = true
	if s.acceptRelease() {
		t.Fatal("the release after DBLCLK must not schedule a tooltip")
	}
	if single, hide := s.takeTimer(stale); single || hide {
		t.Fatal("a queued cancelled timer must be ignored")
	}
	if !s.acceptRelease() {
		t.Fatal("the next independent click must be accepted")
	}
}

func TestClickTimerGenerationAndConsumption(t *testing.T) {
	s := trayClickState{}
	oldSingle := s.nextTimerID()
	s.pendingTimer = s.nextTimerID()
	s.hideTimer = s.nextTimerID()
	if single, hide := s.takeTimer(oldSingle); single || hide {
		t.Fatal("cancelled single timer fired after another click")
	}
	id := s.pendingTimer
	if single, hide := s.takeTimer(id); !single || hide || s.pendingTimer != 0 {
		t.Fatal("single timer must be consumed once")
	}
	if single, hide := s.takeTimer(id); single || hide {
		t.Fatal("repeated single timer message must be ignored")
	}
	oldHide := s.hideTimer
	s.hideTimer = s.nextTimerID()
	if single, hide := s.takeTimer(oldHide); single || hide {
		t.Fatal("old dismiss timer must not hide a newly shown tooltip")
	}
	if single, hide := s.takeTimer(s.hideTimer); single || !hide || s.hideTimer != 0 {
		t.Fatal("dismiss timer must be consumed once")
	}
}

func TestClickTooltipPosition(t *testing.T) {
	tests := []struct {
		name   string
		cursor point
		work   tooltipRect
		want   point
	}{
		{"bottom taskbar", point{1900, 1060}, tooltipRect{0, 0, 1920, 1040}, point{1520, 940}},
		{"left monitor", point{-1900, 1060}, tooltipRect{-1920, 0, 0, 1040}, point{-1920, 940}},
		{"upper monitor", point{-10, -30}, tooltipRect{-1920, -1080, 0, -40}, point{-400, -140}},
		{"top taskbar", point{20, 20}, tooltipRect{0, 40, 1920, 1080}, point{0, 40}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := clickTooltipPosition(tc.cursor, 400, 100, 8, tc.work)
			if got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestPackTooltipPositionPreservesNegativeCoordinates(t *testing.T) {
	p := point{-1920, -1080}
	packed := packTooltipPosition(p)
	if int16(packed&0xffff) != int16(p.X) || int16((packed>>16)&0xffff) != int16(p.Y) {
		t.Fatalf("signed coordinates were lost: %#x", packed)
	}
}

func TestTooltipToolInfoLayout(t *testing.T) {
	var info tooltipToolInfo
	wantSize, wantText := uintptr(48), uintptr(36)
	if unsafe.Sizeof(uintptr(0)) == 8 {
		wantSize, wantText = 72, 48
	}
	if unsafe.Sizeof(info) != wantSize || unsafe.Offsetof(info.Text) != wantText {
		t.Fatalf("TOOLINFOW layout size=%d text=%d", unsafe.Sizeof(info), unsafe.Offsetof(info.Text))
	}
}

func TestClickTooltipTextIsSeparateAndFallsBack(t *testing.T) {
	tray := winTray{}
	if got := tray.clickTooltipText(); got[0] != 0 {
		t.Fatal("text must be empty before either summary is configured")
	}
	// This setter only stores text and works before the tray HWND exists.
	if err := tray.setClickTooltip("5시간 한도: 10% 남음\n주간 한도: 18% 남음"); err != nil {
		t.Fatal(err)
	}
	tray.nid = &notifyIconData{}
	hover, _ := windows.UTF16FromString("hover 10/18")
	copy(tray.nid.Tip[:], hover)
	if got := tray.clickTooltipText(); windows.UTF16ToString(got[:]) != "5시간 한도: 10% 남음\n주간 한도: 18% 남음" {
		t.Fatal("explicit click summary was replaced by hover text")
	}
	if windows.UTF16ToString(tray.nid.Tip[:]) != "hover 10/18" {
		t.Fatal("click summary must not mutate hover text")
	}
	if err := tray.setClickTooltip(""); err != nil {
		t.Fatal(err)
	}
	if got := tray.clickTooltipText(); windows.UTF16ToString(got[:]) != "hover 10/18" {
		t.Fatal("empty click summary must restore hover fallback")
	}
}

func TestClickTooltipUTF16Boundary(t *testing.T) {
	tests := []struct{ name, input, want string }{
		{"ASCII limit", strings.Repeat("a", 1200), strings.Repeat("a", clickTipTextCapacity-1)},
		{"complete surrogate pair", strings.Repeat("a", clickTipTextCapacity-3) + "😀", strings.Repeat("a", clickTipTextCapacity-3) + "😀"},
		{"split surrogate pair", strings.Repeat("a", clickTipTextCapacity-2) + "😀", strings.Repeat("a", clickTipTextCapacity-2)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tray := winTray{}
			if err := tray.setClickTooltip(tc.input); err != nil {
				t.Fatal(err)
			}
			text := tray.clickTooltipText()
			if text[len(text)-1] != 0 || windows.UTF16ToString(text[:]) != tc.want {
				t.Fatal("UTF-16 truncation lost the terminator or split a surrogate pair")
			}
			before := text
			if err := tray.setClickTooltip("invalid\x00text"); err == nil {
				t.Fatal("embedded NUL must be rejected")
			}
			if tray.clickTooltipText() != before {
				t.Fatal("invalid update must preserve the previous summary")
			}
		})
	}
}

// Exercise registration, Unicode text marshalling and retained-buffer lifetime
// against the actual native control. This never activates/shows a tooltip.
func TestNativeTrackingTooltipHiddenLifecycle(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	instance, _, err := pGetModuleHandle.Call(0)
	if instance == 0 {
		t.Fatal(err)
	}
	className, _ := windows.UTF16PtrFromString("STATIC")
	owner, _, err := pCreateWindowEx.Call(0, uintptr(unsafe.Pointer(className)), 0, 0, 0, 0, 0, 0, 0, 0, instance, 0)
	if owner == 0 {
		t.Fatal(err)
	}
	defer pDestroyWindow.Call(owner)
	tray := winTray{instance: windows.Handle(instance), window: windows.Handle(owner)}
	tip, err := tray.newClickTooltip()
	if err != nil {
		t.Fatal(err)
	}
	tray.clickTooltip = tip
	defer tray.destroyClickTooltip()
	assertNativeCardStyle(t, tip, 96)
	for _, dpi := range []uintptr{144, 192, 96} {
		oldFont := tip.font
		if err := tip.scaleCardStyle(dpi); err != nil {
			t.Fatal(err)
		}
		assertNativeCardStyle(t, tip, dpi)
		var oldInfo tooltipLogFont
		if size, _, _ := g32.NewProc("GetObjectW").Call(uintptr(oldFont), unsafe.Sizeof(oldInfo), uintptr(unsafe.Pointer(&oldInfo))); size != 0 {
			t.Fatal("DPI replacement must release the previous font")
		}
	}
	const longSummary = "Codex\n5시간 한도: 10% 남음\n초기화: 10월 13일 18:00 (2시간 뒤)\n\n주간 한도: 18% 남음\n초기화: 10월 18일 15:00 (5일 뒤)\n\nClaude\n5시간 한도: 42% 남음\n초기화: 10월 13일 20:00 (4시간 뒤)\n\n주간 한도: 60% 남음\n초기화: 10월 19일 11:00 (6일 뒤)"
	if err := tray.setClickTooltip(longSummary); err != nil {
		t.Fatal(err)
	}
	tip.text = tray.clickTooltipText()
	tip.sendTool(ttmUpdateTipTextW, 0)
	pSendMessage.Call(uintptr(tip.window), ttmSetMaxTipWidth, 0, 420)
	runtime.GC()
	var buffer clickTooltipBuffer
	info := tip.tool
	info.Text = &buffer[0]
	// TTM_GETTEXTW retrieves the tool text without activating the control.
	pSendMessage.Call(uintptr(tip.window), 0x0400+56, uintptr(len(buffer)), uintptr(unsafe.Pointer(&info)))
	if got := windows.UTF16ToString(buffer[:]); got != longSummary {
		t.Fatalf("native Unicode text mismatch after GC: %q", got)
	}
	tray.nid = &notifyIconData{}
	hover, _ := windows.UTF16FromString("hover summary")
	copy(tray.nid.Tip[:], hover)
	if err := tray.setClickTooltip(""); err != nil {
		t.Fatal(err)
	}
	tip.text = tray.clickTooltipText()
	tip.sendTool(ttmUpdateTipTextW, 0)
	pSendMessage.Call(uintptr(tip.window), 0x0400+56, uintptr(len(buffer)), uintptr(unsafe.Pointer(&info)))
	if got := windows.UTF16ToString(buffer[:]); got != "hover summary" {
		t.Fatalf("native hover fallback mismatch: %q", got)
	}
	tray.deferSingleClick()
	if tray.clicks.pendingTimer == 0 {
		t.Fatal("single click did not create a native window timer")
	}
	stale := tray.clicks.pendingTimer
	tray.cancelSingleClick()
	tray.handleClickTimer(stale)
	if tray.clicks.pendingTimer != 0 || tip.visible {
		t.Fatal("cancelled timer must not activate the native tooltip")
	}
	tray.deferSingleClick()
	lastFont := tip.font
	tray.destroyClickTooltip()
	if tray.clickTooltip != nil || tray.clicks.pendingTimer != 0 || tray.clicks.hideTimer != 0 {
		t.Fatal("destroy must clear tooltip handle, pinned buffer and timers")
	}
	var lastInfo tooltipLogFont
	if size, _, _ := g32.NewProc("GetObjectW").Call(uintptr(lastFont), unsafe.Sizeof(lastInfo), uintptr(unsafe.Pointer(&lastInfo))); size != 0 {
		t.Fatal("destroy must release the final GDI font")
	}
}

func assertNativeCardStyle(t *testing.T, tip *trackingTooltip, dpi uintptr) {
	t.Helper()
	background, _, _ := pSendMessage.Call(uintptr(tip.window), 0x0400+22, 0, 0) // TTM_GETTIPBKCOLOR
	textColor, _, _ := pSendMessage.Call(uintptr(tip.window), 0x0400+23, 0, 0)  // TTM_GETTIPTEXTCOLOR
	if background != clickTipBackground || textColor != clickTipTextColor {
		t.Fatalf("native palette mismatch: background=%#x text=%#x", background, textColor)
	}
	var margin tooltipRect
	pSendMessage.Call(uintptr(tip.window), 0x0400+27, 0, uintptr(unsafe.Pointer(&margin))) // TTM_GETMARGIN
	padding := int32(12 * dpi / 96)
	if margin != (tooltipRect{padding, padding, padding, padding}) {
		t.Fatalf("native padding mismatch at DPI %d: %v", dpi, margin)
	}
	handle, _, _ := pSendMessage.Call(uintptr(tip.window), 0x0031, 0, 0) // WM_GETFONT
	if handle == 0 || handle != uintptr(tip.font) {
		t.Fatal("native control did not adopt the owned font")
	}
	var font tooltipLogFont
	size, _, _ := g32.NewProc("GetObjectW").Call(handle, unsafe.Sizeof(font), uintptr(unsafe.Pointer(&font)))
	if size != unsafe.Sizeof(font) || font != clickTooltipFont(dpi) {
		t.Fatalf("native Segoe UI font mismatch at DPI %d: %+v", dpi, font)
	}
	// Comctl32 v5 accepts TTM_SETTITLEW (checked by scaleCardStyle), but does
	// not implement the later TTM_GETTITLE message. Visible heading layout
	// must be checked with the running application rather than that API.
}
