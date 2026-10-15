//go:build windows

package systray

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"runtime"
	"sync"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

func TestClickRowsSnapshotAndModeTransitions(t *testing.T) {
	tray := winTray{}
	if err := tray.setClickTooltip("fallback"); err != nil {
		t.Fatal(err)
	}
	rows := []ClickTooltipRow{{Label: "Codex · 5시간 한도 😀", Value: "0% 남음", Detail: "초기화: 2026년 10월 15일 18:00 · 2시간 뒤", Remaining: 0}}
	if err := tray.setClickTooltipRows(rows); err != nil {
		t.Fatal(err)
	}
	rows[0].Label = "mutated caller"
	text, snapshot := tray.clickTooltipSnapshot()
	if snapshot[0].Label == rows[0].Label || windows.UTF16ToString(text[:]) != "fallback" {
		t.Fatal("snapshot lost independent rows or fallback")
	}
	snapshot[0].Value = "mutated snapshot"
	_, next := tray.clickTooltipSnapshot()
	if next[0].Value != "0% 남음" {
		t.Fatal("snapshot aliases tray state")
	}
	for _, invalid := range []ClickTooltipRow{{Label: "invalid\x00"}, {Countdown: "invalid\x00"}, {ResetDate: "invalid\x00"}, {Status: "invalid\x00"}} {
		if err := tray.setClickTooltipRows([]ClickTooltipRow{invalid}); err == nil {
			t.Fatal("NUL accepted")
		}
	}
	_, next = tray.clickTooltipSnapshot()
	if len(next) != 1 || next[0].Value != "0% 남음" {
		t.Fatal("invalid rows changed state")
	}
	if err := tray.setClickTooltip("plain"); err != nil {
		t.Fatal(err)
	}
	text, next = tray.clickTooltipSnapshot()
	if len(next) != 0 || windows.UTF16ToString(text[:]) != "plain" {
		t.Fatal("plain mode retained structured state")
	}
	if err := tray.setClickTooltipRows(rows); err != nil {
		t.Fatal(err)
	}
	if err := tray.setClickTooltipRows(nil); err != nil {
		t.Fatal(err)
	}
	text, next = tray.clickTooltipSnapshot()
	if len(next) != 0 || windows.UTF16ToString(text[:]) != "plain" {
		t.Fatal("empty rows did not restore plain fallback")
	}
}

func TestClickRowNumericSeverity(t *testing.T) {
	for _, tc := range []struct {
		value int
		want  uintptr
	}{{-1, clickTipTextColor}, {0, 0x002626b9}, {10, 0x002626b9}, {11, 0x00006e96}, {20, 0x00006e96}, {21, clickTipTextColor}, {100, clickTipTextColor}} {
		if got := clickRowValueColor(tc.value); got != tc.want {
			t.Fatalf("remaining %d color=%#x", tc.value, got)
		}
	}
}

func TestClickRowResetGroup(t *testing.T) {
	for _, tc := range []struct {
		row         ClickTooltipRow
		left, right string
	}{
		{ClickTooltipRow{Countdown: "6일 22시간 후 초기화", ResetDate: "10/22(목) 15:13"}, "6일 22시간 후 초기화", " · 10/22(목) 15:13"},
		{ClickTooltipRow{Countdown: "초기화 정보 없음"}, "초기화 정보 없음", ""},
		{ClickTooltipRow{ResetDate: "10/22(목) 15:13"}, "", "10/22(목) 15:13"},
		{ClickTooltipRow{Detail: "기존 상세 정보"}, "기존 상세 정보", ""},
	} {
		left, right := clickRowSecondLine(tc.row)
		if left != tc.left || right != tc.right {
			t.Fatalf("reset group = %q, %q; want %q, %q", left, right, tc.left, tc.right)
		}
	}
}

func TestClickRowsConcurrentSnapshots(t *testing.T) {
	tray := winTray{}
	var workers sync.WaitGroup
	for i := 0; i < 4; i++ {
		workers.Add(1)
		go func(remaining int) {
			defer workers.Done()
			for j := 0; j < 100; j++ {
				if err := tray.setClickTooltipRows([]ClickTooltipRow{{Label: "서비스 · 한도", Value: "남음", Detail: "초기화", Remaining: remaining}}); err != nil {
					t.Error(err)
				}
				_, snapshot := tray.clickTooltipSnapshot()
				if len(snapshot) != 1 || snapshot[0].Label != "서비스 · 한도" || snapshot[0].Detail != "초기화" {
					t.Error("partial snapshot")
				}
			}
		}(i)
	}
	workers.Wait()
}

// Real native fonts, WM_PRINTCLIENT routing, owner draw actions and pixels are
// exercised against hidden HWNDs and a DIB. No tooltip is shown on the desktop.
func TestNativeCompactRowsPaintingDPIAndLifetime(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	instance, _, _ := pGetModuleHandle.Call(0)
	class, _ := windows.UTF16PtrFromString("STATIC")
	owner, _, _ := pCreateWindowEx.Call(0, uintptr(unsafe.Pointer(class)), 0, 0, 0, 0, 0, 0, 0, 0, instance, 0)
	if owner == 0 {
		t.Fatal("create hidden owner")
	}
	defer pDestroyWindow.Call(owner)
	tray := winTray{instance: windows.Handle(instance), window: windows.Handle(owner)}
	tip, err := tray.newClickTooltip()
	if err != nil {
		t.Fatal(err)
	}
	tray.clickTooltip = tip
	defer tray.destroyClickTooltip()
	tip.rows = []ClickTooltipRow{
		{Label: "Codex · 5시간 한도", Value: "0% 남음", Countdown: "2시간 후 초기화", ResetDate: "10/15(목) 18:00", Status: "캐시 12분 전 · 한도 도달", Remaining: 0},
		{Label: "Codex · 주간 한도", Value: "18% 남음", Countdown: "5일 3시간 후 초기화", ResetDate: "10/20(화) 13:58", Remaining: 18},
		{Label: "Claude · 5시간 한도", Value: "42% 남음", Detail: "10/15(목) 20:00 초기화 · 4시간 후", Remaining: 42},
		{Label: "Claude · 주간 한도", Value: "확인 중", Detail: "초기화 정보 없음", Remaining: -1},
	}
	exStyle, _, _ := u32.NewProc("GetWindowLongW").Call(uintptr(tip.window), ^uintptr(19)) // GWL_EXSTYLE -20
	if exStyle&0x08000000 == 0 {
		t.Fatal("card lost WS_EX_NOACTIVATE")
	}
	for _, dpi := range []uintptr{96, 120, 144, 192, 96} {
		oldFonts := []windows.Handle{tip.font, tip.valueFont, tip.detailFont, tip.countdownFont}
		oldDPI := tip.fontDPI
		if err := tip.scaleCardStyle(dpi); err != nil {
			t.Fatal(err)
		}
		if oldDPI != dpi {
			for _, font := range oldFonts {
				var info tooltipLogFont
				if size, _, _ := g32.NewProc("GetObjectW").Call(uintptr(font), unsafe.Sizeof(info), uintptr(unsafe.Pointer(&info))); size != 0 {
					t.Fatal("replaced font retained")
				}
			}
		}
		for i, font := range []windows.Handle{tip.font, tip.valueFont, tip.detailFont, tip.countdownFont} {
			var got tooltipLogFont
			g32.NewProc("GetObjectW").Call(uintptr(font), unsafe.Sizeof(got), uintptr(unsafe.Pointer(&got)))
			want := []tooltipLogFont{clickCardFont(dpi, 10, 400), clickCardFont(dpi, 18, 600), clickCardFont(dpi, 10, 400), clickCardFont(dpi, 10, 600)}[i]
			if got != want {
				t.Fatalf("DPI %d font %d differs", dpi, i)
			}
		}
		tip.rowLayout = tip.measureRows()
		layout := tip.rowLayout
		if layout.width < int32(360*dpi/96) || layout.padding != int32((16*dpi+48)/96) || layout.gap != int32((12*dpi+48)/96) {
			t.Fatal("DPI geometry incorrect")
		}
		dc, _, _ := pGetDC.Call(uintptr(tip.window))
		for _, row := range tip.rows {
			label := clickMeasureText(dc, tip.font, row.Label)
			value := clickMeasureText(dc, tip.valueFont, row.Value)
			left, right := clickRowSecondLine(row)
			detail := clickMeasureText(dc, tip.rowCountdownFont(row), left)
			date := clickMeasureText(dc, tip.detailFont, right)
			status := clickMeasureText(dc, tip.detailFont, row.Status)
			if label.Right+value.Right+layout.gap+2*layout.padding+2 > layout.width || detail.Right+date.Right+2*layout.padding+2 > layout.width || status.Right+2*layout.padding+2 > layout.width {
				t.Fatal("long quota/date clips")
			}
		}
		// Longer model labels and an explicit year also participate in width
		// measurement, even though the normal preview uses the app's dates.
		previewRows := tip.rows
		tip.rows = []ClickTooltipRow{{Label: "Claude · 주간 한도 (Sonnet)", Value: "100% 남음", Countdown: "5일 3시간 후 초기화", ResetDate: "2027/10/20(수) 13:58", Status: "캐시 120분 전 · Claude 이용 제한", Remaining: 100}}
		longLayout := tip.measureRows()
		longLabel := clickMeasureText(dc, tip.font, tip.rows[0].Label)
		longValue := clickMeasureText(dc, tip.valueFont, tip.rows[0].Value)
		longDetail := clickMeasureText(dc, tip.countdownFont, tip.rows[0].Countdown)
		longDate := clickMeasureText(dc, tip.detailFont, " · "+tip.rows[0].ResetDate)
		longStatus := clickMeasureText(dc, tip.detailFont, tip.rows[0].Status)
		if longLabel.Right+longValue.Right+longLayout.gap+2*longLayout.padding+2 > longLayout.width || longDetail.Right+longDate.Right+2*longLayout.padding+2 > longLayout.width || longStatus.Right+2*longLayout.padding+2 > longLayout.width {
			t.Fatal("long model label or year clips")
		}
		tip.rows = previewRows
		pReleaseDC.Call(uintptr(tip.window), dc)
	}
	layout := tip.rowLayout
	width, height := int(layout.width), int(layout.height)
	pSetWindowPos.Call(uintptr(tip.window), 0, 0, 0, uintptr(width), uintptr(height), 0x0002|0x0004|0x0010)
	tip.layoutActionButtons()
	var nativeClient tooltipRect
	pGetClientRect.Call(uintptr(tip.window), uintptr(unsafe.Pointer(&nativeClient)))
	width, height = int(nativeClient.Right), int(nativeClient.Bottom)
	dc, _, _ := pCreateCompatibleDC.Call(0)
	if dc == 0 {
		t.Fatal("create DIB DC")
	}
	defer pDeleteDC.Call(dc)
	info := bitmapInfo{BmiHeader: bitmapInfoHeader{BiSize: 40, BiWidth: int32(width), BiHeight: -int32(height), BiPlanes: 1, BiBitCount: 32}}
	var pixels uintptr
	bitmap, _, _ := pCreateDIBSection.Call(dc, uintptr(unsafe.Pointer(&info)), 0, uintptr(unsafe.Pointer(&pixels)), 0, 0)
	if bitmap == 0 {
		t.Fatal("create DIB")
	}
	defer pDeleteObject.Call(bitmap)
	old, _, _ := pSelectObject.Call(dc, bitmap)
	defer pSelectObject.Call(dc, old)
	render := func() {
		pSendMessage.Call(uintptr(tip.window), 0x0318, dc, 4) // WM_PRINTCLIENT PRF_CLIENT
		var client tooltipRect
		pGetClientRect.Call(uintptr(tip.window), uintptr(unsafe.Pointer(&client)))
		for i, button := range tip.buttons {
			item := clickButtonDrawItem{Type: 4, ID: uint32(0x7100 + i), Window: button, DC: windows.Handle(dc), Rect: clickButtonBounds(client, 96, i, len(tip.buttons))}
			if !tip.drawActionButton(&item) {
				t.Fatal("button paint failed")
			}
		}
	}
	render()
	process, _, _ := k32.NewProc("GetCurrentProcess").Call()
	before, _, _ := u32.NewProc("GetGuiResources").Call(process, 0)
	for i := 0; i < 30; i++ {
		render()
	}
	after, _, _ := u32.NewProc("GetGuiResources").Call(process, 0)
	if after > before {
		t.Fatalf("row paints leaked %d GDI objects", after-before)
	}
	g32.NewProc("GdiFlush").Call()
	bytes := make([]byte, width*height*4)
	var read uintptr
	if err := windows.ReadProcessMemory(windows.CurrentProcess(), pixels, &bytes[0], uintptr(len(bytes)), &read); err != nil || read != uintptr(len(bytes)) {
		t.Fatal("read DIB", err)
	}
	y0 := int(layout.padding)
	for index, row := range tip.rows {
		count := 0
		for y := y0; y < y0+int(layout.firstHeight); y++ {
			for x := int(layout.padding); x < width/2; x++ {
				o := (y*width + x) * 4
				pixel := uintptr(bytes[o+2]) | uintptr(bytes[o+1])<<8 | uintptr(bytes[o])<<16
				if pixel == clickRowValueColor(row.Remaining) {
					count++
				}
			}
		}
		if count == 0 {
			t.Fatalf("row %d numeric value color absent from native pixels", index)
		}
		y0 += int(layout.rowHeight(row))
	}
	if path := os.Getenv("USAGE_TRAY_CARD_PREVIEW"); path != "" {
		out := image.NewNRGBA(image.Rect(0, 0, width, height))
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				o := (y*width + x) * 4
				out.SetNRGBA(x, y, color.NRGBA{bytes[o+2], bytes[o+1], bytes[o], 255})
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
	fonts := []windows.Handle{tip.font, tip.valueFont, tip.detailFont, tip.countdownFont}
	tray.destroyClickTooltip()
	for _, font := range fonts {
		var info tooltipLogFont
		if size, _, _ := g32.NewProc("GetObjectW").Call(uintptr(font), unsafe.Sizeof(info), uintptr(unsafe.Pointer(&info))); size != 0 {
			t.Fatal("final font retained")
		}
	}
}
