package main

import (
	"image"
	"image/color"
	"image/draw"
	"math"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"unsafe"
)

var (
	iconGDI          = syscall.NewLazyDLL("gdi32.dll")
	iconUser         = syscall.NewLazyDLL("user32.dll")
	iconCreateDC     = iconGDI.NewProc("CreateCompatibleDC")
	iconDeleteDC     = iconGDI.NewProc("DeleteDC")
	iconCreateDIB    = iconGDI.NewProc("CreateDIBSection")
	iconSelectObject = iconGDI.NewProc("SelectObject")
	iconDeleteObject = iconGDI.NewProc("DeleteObject")
	iconCreateFont   = iconGDI.NewProc("CreateFontW")
	iconGetTextFace  = iconGDI.NewProc("GetTextFaceW")
	iconGetCharABC   = iconGDI.NewProc("GetCharABCWidthsW")
	iconSetTextColor = iconGDI.NewProc("SetTextColor")
	iconSetBkMode    = iconGDI.NewProc("SetBkMode")
	iconGdiFlush     = iconGDI.NewProc("GdiFlush")
	iconDrawText     = iconUser.NewProc("DrawTextW")
	iconFindWindow   = iconUser.NewProc("FindWindowW")
	iconWindowDPI    = iconUser.NewProc("GetDpiForWindow")
	iconMetricsDPI   = iconUser.NewProc("GetSystemMetricsForDpi")
	iconMetrics      = iconUser.NewProc("GetSystemMetrics")
	iconTextMasks    sync.Map // (숫자, 최종 픽셀 크기)별 불변 Alpha 이미지.
)

type iconBitmapInfo struct {
	Size                         uint32
	Width, Height                int32
	Planes, BitCount             uint16
	Compression, SizeImage       uint32
	XPelsPerMeter, YPelsPerMeter int32
	ColorsUsed, ColorsImportant  uint32
	Colors                       [1]uint32
}

type iconTextRect struct{ Left, Top, Right, Bottom int32 }

type iconCharABC struct {
	A int32
	B uint32
	C int32
}

type iconTextMaskKey struct {
	label string
	size  int
}

// Use the primary taskbar's physical small-icon size without changing this
// process's DPI awareness (the click card has its own DPI handling).
func nativeTrayIconSize() int {
	if iconWindowDPI.Find() == nil && iconMetricsDPI.Find() == nil {
		class, _ := syscall.UTF16PtrFromString("Shell_TrayWnd")
		window, _, _ := iconFindWindow.Call(uintptr(unsafe.Pointer(class)), 0)
		runtime.KeepAlive(class)
		if window != 0 {
			dpi, _, _ := iconWindowDPI.Call(window)
			if dpi != 0 {
				size, _, _ := iconMetricsDPI.Call(49, dpi) // SM_CXSMICON
				if size >= 16 && size <= 256 {
					return int(size)
				}
			}
		}
	}
	size, _, _ := iconMetrics.Call(49)
	if size >= 16 && size <= 256 {
		return int(size)
	}
	return 16
}

func cachedWindowsIconText(label string, size int) *image.Alpha {
	key := iconTextMaskKey{label, size}
	if cached, ok := iconTextMasks.Load(key); ok {
		return cached.(*image.Alpha)
	}
	mask := renderWindowsIconText(label, size)
	if mask == nil {
		return nil
	}
	cached, _ := iconTextMasks.LoadOrStore(key, mask)
	return cached.(*image.Alpha)
}

// Copy final-size hinted pixels directly. No interpolation or supersampling is
// applied to Windows numbers; the shell receives this same-size icon.
func drawNativeIconNumber(dst *image.NRGBA, label string, ink color.NRGBA, size int) bool {
	if dst.Bounds().Dx() != size || dst.Bounds().Dy() != size {
		return false
	}
	mask := cachedWindowsIconText(label, size)
	if mask == nil {
		return false
	}
	draw.DrawMask(dst, dst.Bounds(), image.NewUniform(ink), image.Point{}, mask, image.Point{}, draw.Over)
	return true
}

// GDI may silently substitute a requested face. Verify the selected face and
// release rejected fonts before trying the broadly available fallback.
// The caller owns the returned font and must restore oldFont before deleting it.
func selectWindowsIconNumberFont(dc uintptr, size int) (font, oldFont uintptr) {
	for _, name := range []string{"Tahoma", "Segoe UI"} {
		face, _ := syscall.UTF16PtrFromString(name)
		// 8pt at the target DPI, rounded to the actual GDI pixel grid.
		fontHeight := -int32(math.Round(8.0 * 96 / 72 * float64(size) / 16))
		for ; fontHeight < 0; fontHeight++ {
			candidate, _, _ := iconCreateFont.Call(uintptr(fontHeight), 0, 0, 0, 400, 0, 0, 0, 1, 0, 0, 4, 0, uintptr(unsafe.Pointer(face))) // FW_NORMAL, ANTIALIASED_QUALITY
			runtime.KeepAlive(face)
			if candidate == 0 {
				continue
			}
			previous, _, _ := iconSelectObject.Call(dc, candidate)
			if previous == 0 || previous == ^uintptr(0) {
				iconDeleteObject.Call(candidate)
				continue
			}
			var actual [64]uint16
			length, _, _ := iconGetTextFace.Call(dc, uintptr(len(actual)), uintptr(unsafe.Pointer(&actual[0])))
			if length != 0 && strings.EqualFold(syscall.UTF16ToString(actual[:]), name) {
				// Keep every value at one size, including the widest value, 100.
				// Segoe UI has wider digits than Tahoma at some pixel sizes.
				var abc [2]iconCharABC
				ok, _, _ := iconGetCharABC.Call(dc, '0', '1', uintptr(unsafe.Pointer(&abc[0])))
				inkWidth := int(abc[1].B) + int(abc[1].C) + 2*int(abc[0].A) + 2*int(abc[0].B) + int(abc[0].C)
				if name == "Tahoma" || (ok != 0 && inkWidth <= size) {
					return candidate, previous
				}
				iconSelectObject.Call(dc, previous)
				iconDeleteObject.Call(candidate)
				continue
			}
			iconSelectObject.Call(dc, previous)
			iconDeleteObject.Call(candidate)
			break
		}
	}
	return 0, 0
}

func renderWindowsIconText(label string, size int) *image.Alpha {
	if size < 16 || size > 256 || label == "" || label == "-" {
		return nil
	}
	// GDI 배치와 DIB 읽기를 같은 스레드에서 끝낸다. 핸들은 캐시하지 않는다.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	dc, _, _ := iconCreateDC.Call(0)
	if dc == 0 {
		return nil
	}
	defer iconDeleteDC.Call(dc)
	// Leave room for the complete glyphs before placing their unscaled ink in
	// the icon. This detects overflow rather than clipping it while drawing.
	width, height := size*2+32, size*2+32
	info := iconBitmapInfo{Size: 40, Width: int32(width), Height: -int32(height), Planes: 1, BitCount: 32}
	var bits unsafe.Pointer
	bitmap, _, _ := iconCreateDIB.Call(dc, uintptr(unsafe.Pointer(&info)), 0, uintptr(unsafe.Pointer(&bits)), 0, 0)
	if bitmap == 0 || bits == nil {
		return nil
	}
	defer iconDeleteObject.Call(bitmap)
	oldBitmap, _, _ := iconSelectObject.Call(dc, bitmap)
	if oldBitmap == 0 || oldBitmap == ^uintptr(0) {
		return nil
	}
	defer iconSelectObject.Call(dc, oldBitmap)
	font, oldFont := selectWindowsIconNumberFont(dc, size)
	if font == 0 {
		return nil
	}
	defer iconDeleteObject.Call(font)
	defer iconSelectObject.Call(dc, oldFont)
	pixels := unsafe.Slice((*byte)(bits), width*height*4)
	clear(pixels)
	iconSetTextColor.Call(dc, 0xffffff)
	iconSetBkMode.Call(dc, 1) // TRANSPARENT; 회색조를 알파로 바꾸므로 ClearType 색 번짐이 없다.
	text, _ := syscall.UTF16PtrFromString(label)
	rect := iconTextRect{Right: int32(width), Bottom: int32(height)}
	result, _, _ := iconDrawText.Call(dc, uintptr(unsafe.Pointer(text)), uintptr(len(label)), uintptr(unsafe.Pointer(&rect)), 0x20|0x800) // DT_SINGLELINE | DT_NOPREFIX
	runtime.KeepAlive(text)
	flushed, _, _ := iconGdiFlush.Call()
	if result == 0 || flushed == 0 {
		return nil
	}
	bounds := image.Rectangle{}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if pixels[(y*width+x)*4+2] != 0 {
				bounds = bounds.Union(image.Rect(x, y, x+1, y+1))
			}
		}
	}
	if bounds.Empty() {
		return nil
	}
	shift := max(1, int(math.Round(float64(size)/16)))
	x0, y0 := (size-bounds.Dx())/2, (size-bounds.Dy())/2-shift
	if bounds.Dx() > size || y0 < 0 || y0+bounds.Dy() > size {
		return nil
	}
	mask := image.NewAlpha(image.Rect(0, 0, size, size))
	for y := 0; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			mask.SetAlpha(x0+x, y0+y, color.Alpha{A: pixels[((y+bounds.Min.Y)*width+x+bounds.Min.X)*4+2]})
		}
	}
	return mask
}
