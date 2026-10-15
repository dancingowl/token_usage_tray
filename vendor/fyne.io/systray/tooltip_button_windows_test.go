//go:build windows

package systray

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"runtime"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Suppress only native activation for the refresh test. The actual refresh
// code, child buttons, size/layout, and timer behavior still execute, while
// the test cannot put a popup on the user's desktop.
var clickButtonTestHideActivation = windows.NewCallback(func(hwnd uintptr, message uint32, wParam, lParam, subclassID, data uintptr) uintptr {
	if message == ttmTrackActivate && wParam == 1 {
		return 0
	}
	result, _, _ := pDefSubclassProc.Call(hwnd, uintptr(message), wParam, lParam)
	return result
})

func TestNativeClickCardRefreshCancelsPress(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	instance, _, _ := pGetModuleHandle.Call(0)
	class, _ := windows.UTF16PtrFromString("STATIC")
	owner, _, _ := pCreateWindowEx.Call(0, uintptr(unsafe.Pointer(class)), 0, 0, 0, 0, 0, 0, 0, 0, instance, 0)
	if owner == 0 {
		t.Fatal("create offscreen owner")
	}
	defer pDestroyWindow.Call(owner)
	tray := winTray{instance: windows.Handle(instance), window: windows.Handle(owner)}
	if err := tray.setClickTooltip("갱신된 사용량 요약"); err != nil {
		t.Fatal(err)
	}
	if err := tray.setClickTooltipRows([]ClickTooltipRow{{"Codex · 5시간 한도", "0% 남음", "초기화: 2026년 10월 15일 · 2시간 뒤", 0}}); err != nil {
		t.Fatal(err)
	}
	tip, err := tray.newClickTooltip()
	if err != nil {
		t.Fatal(err)
	}
	tray.clickTooltip = tip
	defer tray.destroyClickTooltip()
	if ok, _, _ := pSetWindowSubclass.Call(uintptr(tip.window), clickButtonTestHideActivation, 2, 0); ok == 0 {
		t.Fatal("suppress desktop activation")
	}
	defer pRemoveWindowSubclass.Call(uintptr(tip.window), clickButtonTestHideActivation, 2)
	pSetWindowPos.Call(uintptr(tip.window), 0, 0, 0, 340, 300, 0x0002|0x0004|0x0010)
	tip.layoutActionButtons()
	for len(ClickActionCh) > 0 {
		<-ClickActionCh
	}
	button := tip.buttons[0]
	pSendMessage.Call(uintptr(button), 0x0200, 0, packTooltipPosition(point{10, 10}))
	pSendMessage.Call(uintptr(button), 0x0201, 1, packTooltipPosition(point{10, 10}))
	if tip.pressed != button || tip.hovered != button {
		t.Fatal("press was not armed before refresh")
	}
	deadline := time.Now().Add(7 * time.Second)
	tip.visible, tip.dismissAt = true, deadline
	tray.refreshVisibleClickTooltip()
	if !tip.visible || tip.pressed != 0 || tip.hovered != 0 || tip.dismissAt != deadline {
		t.Fatal("refresh must keep card/deadline while clearing button interaction")
	}
	if state, _, _ := pSendMessage.Call(uintptr(button), 0x00f2, 0, 0); state&4 != 0 { // BM_GETSTATE / BST_PUSHED
		t.Fatal("refreshed button retained pressed appearance")
	}
	if captured, _, _ := pGetCapture.Call(); captured == uintptr(button) {
		t.Fatal("refreshed button retained capture")
	}
	pSendMessage.Call(uintptr(button), 0x0202, 0, packTooltipPosition(point{10, 10}))
	select {
	case <-ClickActionCh:
		t.Fatal("release after refresh dispatched cancelled action")
	default:
	}
	// A new click after the refresh still works normally.
	pSendMessage.Call(uintptr(button), 0x0201, 1, packTooltipPosition(point{10, 10}))
	pSendMessage.Call(uintptr(button), 0x0202, 0, packTooltipPosition(point{10, 10}))
	select {
	case action := <-ClickActionCh:
		if action != "refresh" {
			t.Fatal(action)
		}
	default:
		t.Fatal("new click after refresh did not dispatch")
	}
}

func TestClickButtonWin32StructLayouts(t *testing.T) {
	var draw clickButtonDrawItem
	var tracking clickMouseTracking
	size, hwndOffset, trackingSize := uintptr(48), uintptr(20), uintptr(16)
	if unsafe.Sizeof(uintptr(0)) == 8 {
		size, hwndOffset, trackingSize = 64, 24, 24
	}
	if unsafe.Sizeof(draw) != size || unsafe.Offsetof(draw.Window) != hwndOffset || unsafe.Sizeof(tracking) != trackingSize {
		t.Fatal("owner draw / mouse tracking structs do not match Win32 ABI")
	}
}

func TestThreeClickActionsAndDPILayout(t *testing.T) {
	if len(clickCardActions) != 3 || clickCardActions[0].action != "refresh" || clickCardActions[1].action != "detail" || clickCardActions[2].action != "settings" {
		t.Fatal("summary must expose refresh, detail, settings in that order")
	}
	for _, dpi := range []uintptr{96, 120, 144, 192} {
		client := tooltipRect{0, 0, int32(340 * dpi / 96), int32(300 * dpi / 96)}
		padding, gap := int32(12*dpi/96), int32(8*dpi/96)
		previous := padding - gap
		for i := range clickCardActions {
			bounds := clickButtonBounds(client, dpi, i, len(clickCardActions))
			if bounds.Left-previous != gap || bounds.Bottom-bounds.Top != int32(36*dpi/96) || bounds.Right-bounds.Left < int32(90*dpi/96) {
				t.Fatalf("DPI %d button %d has cramped/inconsistent bounds %+v", dpi, i, bounds)
			}
			previous = bounds.Right
		}
		if previous != client.Right-padding {
			t.Fatal("last button does not preserve outer padding")
		}
	}
}

func TestClickButtonEnabledTextContrast(t *testing.T) {
	luminance := func(c uintptr) float64 {
		channels := [3]float64{float64(c&255) / 255, float64(c>>8&255) / 255, float64(c>>16&255) / 255}
		for i, v := range channels {
			if v <= .04045 {
				channels[i] = v / 12.92
			} else {
				channels[i] = math.Pow((v+.055)/1.055, 2.4)
			}
		}
		return .2126*channels[0] + .7152*channels[1] + .0722*channels[2]
	}
	for _, state := range []struct {
		state   uint32
		hovered bool
	}{{0, false}, {0, true}, {1, true}} {
		palette := clickButtonColors(state.state, state.hovered)
		contrast := (luminance(palette.fill) + .05) / (luminance(palette.text) + .05)
		if contrast < 4.5 {
			t.Fatalf("button text contrast %.2f is below 4.5:1", contrast)
		}
	}
}

// Render the actual WM_DRAWITEM handler into an offscreen native DIB. This
// checks GDI pixels, parent routing, and object lifetime without desktop UI.
// An optional path exports the same bitmap for visual QA.
func TestNativeClickButtonPaintingAndLifetime(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	instance, _, _ := pGetModuleHandle.Call(0)
	class, _ := windows.UTF16PtrFromString("STATIC")
	owner, _, _ := pCreateWindowEx.Call(0, uintptr(unsafe.Pointer(class)), 0, 0, 0, 0, 0, 0, 0, 0, instance, 0)
	if owner == 0 {
		t.Fatal("create offscreen owner")
	}
	defer pDestroyWindow.Call(owner)
	tray := winTray{instance: windows.Handle(instance), window: windows.Handle(owner)}
	tip, err := tray.newClickTooltip()
	if err != nil {
		t.Fatal(err)
	}
	tray.clickTooltip = tip
	defer tray.destroyClickTooltip()
	const width, height = 340, 204
	dc, _, _ := pCreateCompatibleDC.Call(0)
	if dc == 0 {
		t.Fatal("create bitmap DC")
	}
	defer pDeleteDC.Call(dc)
	info := bitmapInfo{BmiHeader: bitmapInfoHeader{BiSize: 40, BiWidth: width, BiHeight: -height, BiPlanes: 1, BiBitCount: 32}}
	var pixels uintptr
	bitmap, _, _ := pCreateDIBSection.Call(dc, uintptr(unsafe.Pointer(&info)), 0, uintptr(unsafe.Pointer(&pixels)), 0, 0)
	if bitmap == 0 {
		t.Fatal("create bitmap")
	}
	defer pDeleteObject.Call(bitmap)
	old, _, _ := pSelectObject.Call(dc, bitmap)
	defer pSelectObject.Call(dc, old)
	background, _, _ := pCreateSolidBrush.Call(clickTipBackground)
	rect := tooltipRect{0, 0, width, height}
	pFillRect.Call(dc, uintptr(unsafe.Pointer(&rect)), background)
	pDeleteObject.Call(background)
	bytes := make([]byte, width*height*4)
	// Win32 supplies WM_DRAWITEM with native memory. A Go heap pointer
	// round-tripped through LPARAM into a Go callback loses checkptr's
	// allocation provenance, so use the same native ownership in this test.
	itemMemory, _, _ := k32.NewProc("LocalAlloc").Call(0x0040, unsafe.Sizeof(clickButtonDrawItem{})) // LPTR
	if itemMemory == 0 {
		t.Fatal("allocate native DRAWITEMSTRUCT")
	}
	defer k32.NewProc("LocalFree").Call(itemMemory)
	pixel := func(x, y int32) uintptr {
		offset := (int(y)*width + int(x)) * 4
		return uintptr(bytes[offset+2]) | uintptr(bytes[offset+1])<<8 | uintptr(bytes[offset])<<16
	}
	states := []uint32{0, 0, 1, 4}
	render := func() {
		for row, state := range states {
			for i, button := range tip.buttons {
				tip.hovered = 0
				if row == 1 || row == 2 {
					tip.hovered = button
				}
				bounds := clickButtonBounds(tooltipRect{0, 0, width, int32(56 + row*48)}, 96, i, len(tip.buttons))
				item := clickButtonDrawItem{Type: 4, ID: uint32(0x7100 + i), Action: 1, State: state, Window: button, DC: windows.Handle(dc), Rect: bounds}
				var written uintptr
				if err := windows.WriteProcessMemory(windows.CurrentProcess(), itemMemory, (*byte)(unsafe.Pointer(&item)), unsafe.Sizeof(item), &written); err != nil || written != unsafe.Sizeof(item) {
					t.Fatalf("copy DRAWITEMSTRUCT to native memory: bytes=%d error=%v", written, err)
				}
				if result, _, _ := pSendMessage.Call(uintptr(tip.window), 0x002b, uintptr(item.ID), itemMemory); result != 1 {
					t.Fatal("parent did not handle owner draw")
				}
				g32.NewProc("GdiFlush").Call()
				var read uintptr
				if err := windows.ReadProcessMemory(windows.CurrentProcess(), pixels, &bytes[0], uintptr(len(bytes)), &read); err != nil || read != uintptr(len(bytes)) {
					t.Fatalf("read native bitmap pixels: bytes=%d error=%v", read, err)
				}
				if got, want := pixel(bounds.Left+8, bounds.Top+8), clickButtonColors(state, row == 1 || row == 2).fill; got != want {
					t.Fatalf("state %d fill pixel %#x, want %#x", row, got, want)
				}
				if pixel(bounds.Left, bounds.Top) != clickTipBackground {
					t.Fatal("rounded corner was not cleared to card background")
				}
			}
		}
	}
	render()
	process, _, _ := k32.NewProc("GetCurrentProcess").Call()
	count, _, _ := u32.NewProc("GetGuiResources").Call(process, 0)
	for i := 0; i < 30; i++ {
		render()
	}
	after, _, _ := u32.NewProc("GetGuiResources").Call(process, 0)
	if after > count {
		t.Fatalf("button repaint leaked %d GDI objects", after-count)
	}
	if path := os.Getenv("USAGE_TRAY_BUTTON_PREVIEW"); path != "" {
		out := image.NewNRGBA(image.Rect(0, 0, width, height))
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				offset := (y*width + x) * 4
				out.SetNRGBA(x, y, color.NRGBA{bytes[offset+2], bytes[offset+1], bytes[offset], 255})
			}
		}
		file, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		defer file.Close()
		if err := png.Encode(file, out); err != nil {
			t.Fatal(err)
		}
	}
}
