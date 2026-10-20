//go:build windows

package systray

import (
	"math"
	"runtime"
	"sync"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

func clickThemeContrast(a, b uintptr) float64 {
	luminance := func(c uintptr) float64 {
		linear := func(v uintptr) float64 {
			f := float64(v&255) / 255
			if f <= .04045 {
				return f / 12.92
			}
			return math.Pow((f+.055)/1.055, 2.4)
		}
		return .2126*linear(c) + .7152*linear(c>>8) + .0722*linear(c>>16)
	}
	l1, l2 := luminance(a), luminance(b)
	if l1 < l2 {
		l1, l2 = l2, l1
	}
	return (l1 + .05) / (l2 + .05)
}

func TestClickThemePalettes(t *testing.T) {
	for _, dark := range []bool{false, true} {
		colors := clickCardColors(dark)
		for _, foreground := range []uintptr{colors.text, colors.detail, colors.danger, colors.warning} {
			if ratio := clickThemeContrast(foreground, colors.background); ratio < 4.5 {
				t.Errorf("dark=%v foreground=%#x contrast=%.2f", dark, foreground, ratio)
			}
		}
		if clickRowValueThemeColor(-1, colors) != colors.text || clickRowValueThemeColor(8, colors) != colors.danger || clickRowValueThemeColor(18, colors) != colors.warning || clickRowValueThemeColor(75, colors) != colors.text {
			t.Fatal("theme changed severity semantics")
		}
		for _, state := range []uint32{0, 1, 4} {
			for _, hover := range []bool{false, true} {
				button := clickButtonThemeColors(state, hover, colors)
				if state != 4 && clickThemeContrast(button.text, button.fill) < 4.5 {
					t.Fatal("action contrast too low")
				}
			}
		}
	}
}

func TestClickThemeConcurrentSnapshots(t *testing.T) {
	tray := winTray{}
	var workers sync.WaitGroup
	for i := 0; i < 4; i++ {
		workers.Add(1)
		go func(i int) {
			defer workers.Done()
			for j := 0; j < 100; j++ {
				tray.setClickTooltipTheme((i+j)%2 == 0)
				colors := tray.clickTooltipColors()
				if colors != clickCardColors(false) && colors != clickCardColors(true) {
					t.Error("partial theme palette")
				}
			}
		}(i)
	}
	workers.Wait()
}

// Render with real GDI fonts into a DIB without creating or activating a
// desktop window. Both palettes must paint their own body/value/action colors.
func TestNativeClickThemePixels(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	tip := trackingTooltip{rows: []ClickTooltipRow{{Label: "Codex · 주간", Value: "75% 남음", Countdown: "2시간 후 초기화", ResetDate: "10/20(화) 18:00", Remaining: 75}}}
	if err := tip.scaleCardStyle(96); err != nil {
		t.Fatal(err)
	}
	defer tip.releaseCardFont()
	tip.rowLayout = clickRowMetrics(96)
	const width, height = 400, 130
	dc, _, _ := pCreateCompatibleDC.Call(0)
	if dc == 0 {
		t.Fatal("create DIB DC")
	}
	defer pDeleteDC.Call(dc)
	info := bitmapInfo{BmiHeader: bitmapInfoHeader{BiSize: 40, BiWidth: width, BiHeight: -height, BiPlanes: 1, BiBitCount: 32}}
	var pixels uintptr
	bitmap, _, _ := pCreateDIBSection.Call(dc, uintptr(unsafe.Pointer(&info)), 0, uintptr(unsafe.Pointer(&pixels)), 0, 0)
	if bitmap == 0 {
		t.Fatal("create DIB")
	}
	defer pDeleteObject.Call(bitmap)
	old, _, _ := pSelectObject.Call(dc, bitmap)
	defer pSelectObject.Call(dc, old)
	client := tooltipRect{0, 0, width, height}
	var layout clickRowsLayout
	for _, dark := range []bool{false, true, false} {
		tip.palette = clickCardColors(dark)
		if !tip.drawRows(dc, client) {
			t.Fatal("paint rows")
		}
		tip.buttons[0] = windows.Handle(1)
		item := clickButtonDrawItem{Type: 4, ID: 0x7100, Window: tip.buttons[0], DC: windows.Handle(dc), Rect: clickButtonBounds(client, 96, 0, 2)}
		if !tip.drawActionButton(&item) {
			t.Fatal("paint action")
		}
		g32.NewProc("GdiFlush").Call()
		data := make([]byte, width*height*4)
		var read uintptr
		if err := windows.ReadProcessMemory(windows.CurrentProcess(), pixels, &data[0], uintptr(len(data)), &read); err != nil || read != uintptr(len(data)) {
			t.Fatal("read DIB", err)
		}
		pixel := func(x, y int) uintptr {
			o := (y*width + x) * 4
			return uintptr(data[o+2]) | uintptr(data[o+1])<<8 | uintptr(data[o])<<16
		}
		colors := tip.cardColors()
		if pixel(1, height-2) != colors.background {
			t.Fatalf("dark=%v background=%#x", dark, pixel(1, height-2))
		}
		foundText, foundDetail := false, false
		for y := 12; y < 80; y++ {
			for x := 12; x < 280; x++ {
				c := pixel(x, y)
				foundText = foundText || c == colors.text
				foundDetail = foundDetail || c == colors.detail
			}
		}
		if !foundText || !foundDetail {
			t.Fatalf("dark=%v native foreground absent: text=%v detail=%v", dark, foundText, foundDetail)
		}
		if layout.width != 0 && tip.rowLayout != layout {
			t.Fatal("theme changed row geometry")
		}
		layout = tip.rowLayout
	}
}
