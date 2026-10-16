//go:build windows

package systray

import (
	"image"
	"runtime"
	"strings"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

func TestLucideActionIconAssetsAndCoverage(t *testing.T) {
	if !strings.Contains(clickActionIconLicense, "ISC License") {
		t.Fatal("Lucide attribution must remain bundled")
	}
	for name, mask := range map[string]*image.Alpha{"refresh": clickRefreshIconMask, "settings": clickSettingsIconMask} {
		if mask == nil || mask.Bounds() != image.Rect(0, 0, 128, 128) {
			t.Fatalf("%s source PNG did not decode at its authored size", name)
		}
		edges := 0
		for _, alpha := range mask.Pix {
			if alpha > 0 && alpha < 255 {
				edges++
			}
		}
		if edges < 100 || mask.AlphaAt(0, 0).A != 0 || mask.AlphaAt(64, 64).A != 0 {
			t.Fatalf("%s must retain antialiased edges, a transparent corner and a hollow center", name)
		}
	}
	if decodeClickActionIcon([]byte("invalid PNG")) != nil {
		t.Fatal("invalid PNG must fail gracefully")
	}
	// A known 2x2 alpha mask averages to 112/255 coverage. White behind
	// black therefore becomes 143 in every DIB color channel.
	mask := image.NewAlpha(image.Rect(0, 0, 2, 2))
	copy(mask.Pix, []byte{0, 64, 128, 255})
	pixel := clickActionIconPixels(mask, 1, 0, 0xffffff)
	if len(pixel) != 4 || pixel[0] != 143 || pixel[1] != 143 || pixel[2] != 143 || pixel[3] != 255 {
		t.Fatalf("area coverage did not blend into an opaque BGRA pixel: %v", pixel)
	}
	// COLORREF red is the low byte, while DIB red is the third byte.
	mask.Pix = []byte{255, 255, 255, 255}
	pixel = clickActionIconPixels(mask, 1, 0x0000ff, 0)
	if pixel[0] != 0 || pixel[1] != 0 || pixel[2] != 255 {
		t.Fatalf("COLORREF/DIB channel order changed: %v", pixel)
	}
}

func TestNativeActionIconDPIAndDCState(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	const width, height = 128, 128
	dc, _, _ := pCreateCompatibleDC.Call(0)
	if dc == 0 {
		t.Fatal("create icon DC")
	}
	defer pDeleteDC.Call(dc)
	info := bitmapInfo{BmiHeader: bitmapInfoHeader{BiSize: 40, BiWidth: width, BiHeight: -height, BiPlanes: 1, BiBitCount: 32}}
	var pixels uintptr
	bitmap, _, _ := pCreateDIBSection.Call(dc, uintptr(unsafe.Pointer(&info)), 0, uintptr(unsafe.Pointer(&pixels)), 0, 0)
	if bitmap == 0 {
		t.Fatal("create icon bitmap")
	}
	defer pDeleteObject.Call(bitmap)
	old, _, _ := pSelectObject.Call(dc, bitmap)
	defer pSelectObject.Call(dc, old)
	brush, _, _ := pCreateSolidBrush.Call(clickTipBackground)
	if brush == 0 {
		t.Fatal("create icon background")
	}
	defer pDeleteObject.Call(brush)
	getObject := g32.NewProc("GetCurrentObject")
	penBefore, _, _ := getObject.Call(dc, 1)   // OBJ_PEN
	brushBefore, _, _ := getObject.Call(dc, 2) // OBJ_BRUSH
	fontBefore, _, _ := getObject.Call(dc, 6)  // OBJ_FONT
	getTextColor := g32.NewProc("GetTextColor")
	getBkMode := g32.NewProc("GetBkMode")
	textBefore, _, _ := getTextColor.Call(dc)
	modeBefore, _, _ := getBkMode.Call(dc)
	bytes := make([]byte, width*height*4)
	for _, dpi := range []uintptr{0, 96, 120, 144, 192} {
		actualDPI := dpi
		if actualDPI == 0 {
			actualDPI = 96
		}
		var previous []byte
		for _, action := range clickCardActions {
			all := tooltipRect{0, 0, width, height}
			pFillRect.Call(dc, uintptr(unsafe.Pointer(&all)), brush)
			size := int32(28 * actualDPI / 96)
			bounds := tooltipRect{10, 10, 10 + size, 10 + size}
			if !drawClickActionIcon(dc, bounds, dpi, action.action, clickTipTextColor, clickTipBackground) {
				t.Fatalf("DPI %d %s icon paint failed", dpi, action.action)
			}
			penAfter, _, _ := getObject.Call(dc, 1)
			brushAfter, _, _ := getObject.Call(dc, 2)
			fontAfter, _, _ := getObject.Call(dc, 6)
			textAfter, _, _ := getTextColor.Call(dc)
			modeAfter, _, _ := getBkMode.Call(dc)
			if penAfter != penBefore || brushAfter != brushBefore || fontAfter != fontBefore || textAfter != textBefore || modeAfter != modeBefore {
				t.Fatal("icon paint changed borrowed DC state")
			}
			g32.NewProc("GdiFlush").Call()
			var read uintptr
			if err := windows.ReadProcessMemory(windows.CurrentProcess(), pixels, &bytes[0], uintptr(len(bytes)), &read); err != nil || read != uintptr(len(bytes)) {
				t.Fatal("read icon pixels", err)
			}
			ink, edges, different := 0, 0, false
			iconSize := int32(20 * actualDPI / 96)
			iconLeft, iconTop := bounds.Left+(size-iconSize)/2, bounds.Top+(size-iconSize)/2
			for y := int32(0); y < height; y++ {
				for x := int32(0); x < width; x++ {
					o := (y*width + x) * 4
					color := uintptr(bytes[o+2]) | uintptr(bytes[o+1])<<8 | uintptr(bytes[o])<<16
					if color != clickTipBackground {
						ink++
						if x < iconLeft || x >= iconLeft+iconSize || y < iconTop || y >= iconTop+iconSize {
							t.Fatalf("DPI %d icon escaped its 20-DIP bounds", dpi)
						}
						if color != clickTipTextColor {
							edges++
						}
					}
					if previous != nil && bytes[o] != previous[o] {
						different = true
					}
				}
			}
			if ink < 20 || ink > int(size*size/2) {
				t.Fatalf("DPI %d %s has implausible ink coverage %d", dpi, action.action, ink)
			}
			if edges < 10 {
				t.Fatalf("DPI %d %s lost antialiased edge coverage: %d", dpi, action.action, edges)
			}
			if previous != nil && !different {
				t.Fatal("refresh and settings must have distinct icon pixels")
			}
			previous = append(previous[:0], bytes...)
		}
	}
	for _, invalid := range []struct {
		dc     uintptr
		bounds tooltipRect
		action string
	}{{0, tooltipRect{0, 0, 28, 28}, "refresh"}, {dc, tooltipRect{0, 0, 28, 28}, "unknown"}, {dc, tooltipRect{0, 0, 1, 1}, "settings"}} {
		if drawClickActionIcon(invalid.dc, invalid.bounds, 96, invalid.action, clickTipTextColor, clickTipBackground) {
			t.Fatal("invalid icon draw must fail")
		}
	}
}
