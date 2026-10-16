package main

import (
	"image"
	"image/color"
	"image/draw"
	"math"
	"runtime"
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
	iconSetTextColor = iconGDI.NewProc("SetTextColor")
	iconSetBkMode    = iconGDI.NewProc("SetBkMode")
	iconGdiFlush     = iconGDI.NewProc("GdiFlush")
	iconDrawText     = iconUser.NewProc("DrawTextW")
	iconTextMasks    sync.Map // 불변 Alpha 이미지. 0~100 숫자만 캐시한다.
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

// 실제 잉크 경계로 맞춘다. 폰트의 행간/내부 여백 때문에 숫자가 작아지는 것을 피한다.
func drawNativeIconNumber(dst *image.NRGBA, label string, ink color.NRGBA, size int) bool {
	var mask *image.Alpha
	if cached, ok := iconTextMasks.Load(label); ok {
		mask = cached.(*image.Alpha)
	} else {
		mask = renderWindowsIconText(label)
		if mask == nil {
			return false
		}
		cached, _ := iconTextMasks.LoadOrStore(label, mask)
		mask = cached.(*image.Alpha)
	}
	height, maxWidth := 11.0, 13.0 // 16px 실제 표시에서 숫자 높이.
	if label == "-" {
		height, maxWidth = 2, 6
	}
	width := height * float64(mask.Bounds().Dx()) / float64(mask.Bounds().Dy())
	if width > maxWidth {
		height *= maxWidth / width
		width = maxWidth
	}
	scale := float64(size*4) / 16
	w, h := int(math.Round(width*scale)), int(math.Round(height*scale))
	x0, y0 := (dst.Bounds().Dx()-w)/2, (dst.Bounds().Dy()-h)/2
	scaled := image.NewAlpha(image.Rect(x0, y0, x0+w, y0+h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			// 글꼴 마스크를 선형 보간하고 기존 4배 슈퍼샘플링으로 마지막 축소한다.
			sx := (float64(x)+0.5)*float64(mask.Bounds().Dx())/float64(w) - 0.5
			sy := (float64(y)+0.5)*float64(mask.Bounds().Dy())/float64(h) - 0.5
			ix, iy := int(math.Floor(sx)), int(math.Floor(sy))
			fx, fy := sx-float64(ix), sy-float64(iy)
			alpha := func(px, py int) float64 {
				px = max(0, min(px, mask.Bounds().Dx()-1))
				py = max(0, min(py, mask.Bounds().Dy()-1))
				return float64(mask.AlphaAt(px, py).A)
			}
			a := alpha(ix, iy)*(1-fx)*(1-fy) + alpha(ix+1, iy)*fx*(1-fy) + alpha(ix, iy+1)*(1-fx)*fy + alpha(ix+1, iy+1)*fx*fy
			scaled.SetAlpha(x0+x, y0+y, color.Alpha{A: uint8(math.Round(a))})
		}
	}
	draw.DrawMask(dst, scaled.Bounds(), image.NewUniform(ink), image.Point{}, scaled, scaled.Bounds().Min, draw.Over)
	return true
}

func renderWindowsIconText(label string) *image.Alpha {
	// GDI 배치와 DIB 읽기를 같은 스레드에서 끝낸다. 핸들은 캐시하지 않는다.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	dc, _, _ := iconCreateDC.Call(0)
	if dc == 0 {
		return nil
	}
	defer iconDeleteDC.Call(dc)
	const width, height = 384, 128
	info := iconBitmapInfo{Size: 40, Width: width, Height: -height, Planes: 1, BitCount: 32}
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
	face, _ := syscall.UTF16PtrFromString("Segoe UI")
	fontHeight := int32(-96)
	font, _, _ := iconCreateFont.Call(uintptr(fontHeight), 0, 0, 0, 600, 0, 0, 0, 1, 0, 0, 4, 0, uintptr(unsafe.Pointer(face))) // ANTIALIASED_QUALITY
	runtime.KeepAlive(face)
	if font == 0 {
		return nil
	}
	defer iconDeleteObject.Call(font)
	oldFont, _, _ := iconSelectObject.Call(dc, font)
	if oldFont == 0 || oldFont == ^uintptr(0) {
		return nil
	}
	defer iconSelectObject.Call(dc, oldFont)
	pixels := unsafe.Slice((*byte)(bits), width*height*4)
	clear(pixels)
	iconSetTextColor.Call(dc, 0xffffff)
	iconSetBkMode.Call(dc, 1) // TRANSPARENT; 회색조를 알파로 바꾸므로 ClearType 색 번짐이 없다.
	text, _ := syscall.UTF16PtrFromString(label)
	rect := iconTextRect{Right: width, Bottom: height}
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
	mask := image.NewAlpha(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	for y := 0; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			mask.SetAlpha(x, y, color.Alpha{A: pixels[((y+bounds.Min.Y)*width+x+bounds.Min.X)*4+2]})
		}
	}
	return mask
}
