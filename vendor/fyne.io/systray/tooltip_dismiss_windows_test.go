//go:build windows

package systray

import (
	"runtime"
	"sync"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	clickDismissTestOwners   sync.Map
	clickDismissTestSubclass = windows.NewCallback(func(hwnd uintptr, message uint32, wParam, lParam, subclassID, data uintptr) uintptr {
		if message == wmClickOutside {
			if value, ok := clickDismissTestOwners.Load(windows.Handle(hwnd)); ok {
				return value.(*winTray).wndProc(windows.Handle(hwnd), message, wParam, lParam)
			}
		}
		result, _, _ := pDefSubclassProc.Call(hwnd, uintptr(message), wParam, lParam)
		return result
	})
	clickDismissTestPeekMessage = u32.NewProc("PeekMessageW")
)

// The owner and native tooltip/buttons stay hidden. Queue tests supply only
// synthetic targets/coordinates; no mouse input or desktop popup is generated.
func newHiddenClickDismissTray(t *testing.T) *winTray {
	t.Helper()
	runtime.LockOSThread()
	t.Cleanup(runtime.UnlockOSThread)
	instance, _, _ := pGetModuleHandle.Call(0)
	class, _ := windows.UTF16PtrFromString("STATIC")
	owner, _, err := pCreateWindowEx.Call(0, uintptr(unsafe.Pointer(class)), 0, 0, 0, 0, 0, 0, 0, 0, instance, 0)
	runtime.KeepAlive(class)
	if owner == 0 {
		t.Fatalf("create hidden owner: %v", err)
	}
	t.Cleanup(func() { pDestroyWindow.Call(owner) })
	tray := &winTray{instance: windows.Handle(instance), window: windows.Handle(owner)}
	tip, err := tray.newClickTooltip()
	if err != nil {
		t.Fatal(err)
	}
	tray.clickTooltip = tip
	t.Cleanup(tray.destroyClickTooltip)
	clickDismissTestOwners.Store(tray.window, tray)
	if ok, _, err := pSetWindowSubclass.Call(owner, clickDismissTestSubclass, 3, 0); ok == 0 {
		clickDismissTestOwners.Delete(tray.window)
		t.Fatalf("subclass hidden owner: %v", err)
	}
	t.Cleanup(func() {
		pRemoveWindowSubclass.Call(owner, clickDismissTestSubclass, 3)
		clickDismissTestOwners.Delete(tray.window)
	})
	return tray
}

type clickDismissTestMessage struct {
	Window         windows.Handle
	Message        uint32
	WParam, LParam uintptr
	Time           uint32
	Point          point
	Private        uint32
}

func TestClickDismissNativeMessageLayout(t *testing.T) {
	want := uintptr(32)
	if unsafe.Sizeof(uintptr(0)) == 8 {
		want = 48
	}
	if size := unsafe.Sizeof(clickDismissTestMessage{}); size != want {
		t.Fatalf("native MSG size=%d, want %d", size, want)
	}
}

func takeClickDismissMessage(tray *winTray) (clickDismissTestMessage, bool) {
	var message clickDismissTestMessage
	found, _, _ := clickDismissTestPeekMessage.Call(uintptr(unsafe.Pointer(&message)), uintptr(tray.window), wmClickOutside, wmClickOutside, 1) // PM_REMOVE
	return message, found != 0
}

func TestClickDismissOnlyMousePresses(t *testing.T) {
	for _, message := range []uintptr{0x0201, 0x0204, 0x0207, 0x020b} {
		if !clickDismissPress(message) {
			t.Fatalf("button press %#x was ignored", message)
		}
	}
	for _, message := range []uintptr{0, 0x0200, 0x0202, 0x0203, 0x0205, 0x0208, 0x020a, 0x020c, 0x020e} {
		if clickDismissPress(message) {
			t.Fatalf("non-press %#x would dismiss the card", message)
		}
	}
}

func TestNativeClickDismissTargetAndCoordinateRouting(t *testing.T) {
	tray := newHiddenClickDismissTray(t)
	tip := tray.clickTooltip
	const generation = 0x7199
	tip.visible = true
	tip.dismiss.generation = generation
	tip.dismiss.iconRect = tooltipRect{-120000, 90000, -119960, 90040}
	cases := []struct {
		name   string
		cursor point
		target windows.Handle
		want   bool
	}{
		{"card body", point{70000, -90000}, tip.window, false},
		{"refresh child", point{70000, -90000}, tip.buttons[0], false},
		{"settings child", point{70000, -90000}, tip.buttons[1], false},
		{"tray negative full-width coordinates", point{-119980, 90020}, tray.window, false},
		{"tray left top inclusive", point{-120000, 90000}, tray.window, false},
		{"tray right exclusive", point{-119960, 90020}, tray.window, true},
		{"tray bottom exclusive", point{-119980, 90040}, tray.window, true},
		{"other window", point{70000, -90000}, tray.window, true},
		{"no window", point{-90000, -70000}, 0, true},
	}
	for _, tc := range cases {
		// Keep all queue operations on the parent's locked window thread;
		// testing.T.Run would execute each case in another goroutine.
		tray.queueClickOutside(tc.cursor, tc.target)
		message, found := takeClickDismissMessage(tray)
		if found != tc.want {
			t.Fatalf("%s: queued=%v, want %v", tc.name, found, tc.want)
		}
		if found && (message.Window != tray.window || message.Message != wmClickOutside || message.WParam != generation) {
			t.Fatalf("%s: incorrect dismissal message: %+v", tc.name, message)
		}
		if !tip.visible {
			t.Fatal("hook-side queue must leave UI work to the owner thread")
		}
	}
	for _, state := range []struct {
		visible    bool
		generation uintptr
	}{{false, generation}, {true, 0}} {
		tip.visible, tip.dismiss.generation = state.visible, state.generation
		tray.queueClickOutside(point{70000, -90000}, tray.window)
		if _, found := takeClickDismissMessage(tray); found {
			t.Fatal("inactive card queued an outside dismissal")
		}
	}
}

func TestNativeClickDismissGenerationAndPendingToggle(t *testing.T) {
	tray := newHiddenClickDismissTray(t)
	tip := tray.clickTooltip
	tip.visible = true
	tip.dismiss.generation = tray.clicks.nextTimerID()
	tray.queueClickOutside(point{70000, -90000}, tray.window)
	oldMessage, found := takeClickDismissMessage(tray)
	if !found {
		t.Fatal("old card did not queue outside dismissal")
	}
	tray.hideClickTooltip()
	tip.visible = true
	tip.dismiss.generation = tray.clicks.nextTimerID()
	pDispatchMessage.Call(uintptr(unsafe.Pointer(&oldMessage)))
	if !tip.visible {
		t.Fatal("queued old dismissal hid the replacement card")
	}
	tray.dismissClickOutside(0)
	if !tip.visible {
		t.Fatal("zero generation hid the card")
	}
	tray.deferSingleClick()
	pending := tray.clicks.pendingTimer
	if pending == 0 {
		t.Fatal("native toggle timer was not installed")
	}
	if err := tray.startClickDismissTimer(); err != nil {
		t.Fatal(err)
	}
	tray.queueClickOutside(point{70000, -90000}, tray.window)
	message, found := takeClickDismissMessage(tray)
	if !found {
		t.Fatal("current card did not queue outside dismissal")
	}
	pDispatchMessage.Call(uintptr(unsafe.Pointer(&message)))
	if tip.visible || tray.clicks.pendingTimer != 0 || tray.clicks.hideTimer != 0 {
		t.Fatal("outside dismissal must hide the card and cancel both timers")
	}
	tray.handleClickTimer(pending)
	if tip.visible {
		t.Fatal("already queued toggle timer reopened the dismissed card")
	}
}

func TestNativeClickDismissHookLifecycle(t *testing.T) {
	tray := newHiddenClickDismissTray(t)
	tip := tray.clickTooltip
	thread := windows.GetCurrentThreadId()
	var previous uintptr
	for i := 0; i < 12; i++ {
		tip.visible = true
		tip.dismiss.iconRect = tooltipRect{-120000, 90000, -119960, 90040}
		if err := tray.startClickOutsideWatch(); err != nil {
			t.Fatal(err)
		}
		if tip.dismiss.hook == 0 || tip.dismiss.thread != thread || tip.dismiss.generation <= previous {
			t.Fatalf("invalid native hook lifecycle state: %+v", tip.dismiss)
		}
		previous = tip.dismiss.generation
		installed := tip.dismiss
		if err := tray.startClickOutsideWatch(); err != nil || tip.dismiss != installed {
			t.Fatalf("duplicate start replaced an active hook: state=%+v error=%v", tip.dismiss, err)
		}
		if owner, ok := clickDismissOwners.Load(thread); !ok || owner != tray {
			t.Fatal("installing thread did not retain its tray owner")
		}
		tray.hideClickTooltip()
		if removed, _, _ := pUnhookWindowsHookEx.Call(uintptr(installed.hook)); removed != 0 {
			t.Fatal("hide had not unhooked the native hook")
		}
		if tip.visible || tip.dismiss != (clickDismissState{}) {
			t.Fatal("hide retained native hook state")
		}
		if _, ok := clickDismissOwners.Load(thread); ok {
			t.Fatal("hide retained the installing thread owner")
		}
		tray.hideClickTooltip() // Cleanup is safe after an already hidden card.
	}
	tip.visible = true
	tip.dismiss.iconRect = tooltipRect{-120000, 90000, -119960, 90040}
	if err := tray.startClickOutsideWatch(); err != nil {
		t.Fatal(err)
	}
	tray.destroyClickTooltip()
	if tray.clickTooltip != nil {
		t.Fatal("destroy retained tooltip ownership")
	}
	if _, ok := clickDismissOwners.Load(thread); ok {
		t.Fatal("destroy retained native hook owner")
	}
}

func TestClickDismissMissingShellDataRetainsCachedRectangle(t *testing.T) {
	tray := newHiddenClickDismissTray(t)
	rect := tooltipRect{-120000, 90000, -119960, 90040}
	tray.clickTooltip.dismiss.iconRect = rect
	tray.refreshClickIconRect() // No Shell registration exists for this hidden owner.
	if tray.clickTooltip.dismiss.iconRect != rect {
		t.Fatal("missing Shell data discarded the last usable icon rectangle")
	}
	tray.nid = &notifyIconData{Wnd: tray.window, ID: 0x1234}
	getDPIContext := u32.NewProc("GetThreadDpiAwarenessContext")
	before, _, _ := getDPIContext.Call()
	tray.refreshClickIconRect() // This window/icon pair has never been registered.
	after, _, _ := getDPIContext.Call()
	if tray.clickTooltip.dismiss.iconRect != rect || before != after {
		t.Fatal("failed Shell query changed cached bounds or the thread DPI context")
	}
}

func TestNativeClickDismissMissingIconFallsBackWithoutHook(t *testing.T) {
	tray := newHiddenClickDismissTray(t)
	tip := tray.clickTooltip
	tip.visible = true
	tray.deferSingleClick()
	if err := tray.startClickDismissTimer(); err != nil {
		t.Fatal(err)
	}
	if err := tray.startClickOutsideWatch(); err == nil {
		t.Fatal("missing icon bounds must refuse the hook to preserve tray toggle behavior")
	}
	if !tip.visible || tip.dismiss != (clickDismissState{}) || tray.clicks.pendingTimer == 0 || tray.clicks.hideTimer == 0 {
		t.Fatal("failed watch must retain existing visibility/timers and clear hook state")
	}
	if _, found := clickDismissOwners.Load(windows.GetCurrentThreadId()); found {
		t.Fatal("failed watch retained an owner map entry")
	}
	tray.queueClickOutside(point{-90000, -70000}, tray.window)
	if _, found := takeClickDismissMessage(tray); found {
		t.Fatal("failed watch queued an outside dismissal")
	}
}
