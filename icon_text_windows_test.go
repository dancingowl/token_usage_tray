package main

import (
	"image"
	"math"
	"runtime"
	"strconv"
	"sync"
	"syscall"
	"testing"
	"unsafe"
)

func windowsNumberInkBounds(im image.Image) image.Rectangle {
	bounds := image.Rectangle{}
	for y := im.Bounds().Min.Y; y < im.Bounds().Max.Y; y++ {
		for x := im.Bounds().Min.X; x < im.Bounds().Max.X; x++ {
			if _, _, _, a := im.At(x, y).RGBA(); a != 0 {
				bounds = bounds.Union(image.Rect(x, y, x+1, y+1))
			}
		}
	}
	return bounds
}

// TEXTMETRICW describes the realized font rather than the requested LOGFONT.
type windowsTestTextMetric struct {
	Height, Ascent, Descent, InternalLeading, ExternalLeading int32
	AveCharWidth, MaxCharWidth, Weight, Overhang              int32
	DigitizedAspectX, DigitizedAspectY                        int32
	FirstChar, LastChar, DefaultChar, BreakChar               uint16
	Italic, Underlined, StruckOut, PitchAndFamily, CharSet    byte
}

func windowsTestSelectedFont(t *testing.T, selectFont func(uintptr) (uintptr, uintptr)) (string, windowsTestTextMetric) {
	t.Helper()
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	dc, _, _ := iconCreateDC.Call(0)
	if dc == 0 {
		t.Fatal("cannot create font inspection DC")
	}
	defer iconDeleteDC.Call(dc)
	font, oldFont := selectFont(dc)
	if font == 0 || oldFont == 0 || oldFont == ^uintptr(0) {
		t.Fatal("cannot select font for inspection")
	}
	defer iconDeleteObject.Call(font)
	defer iconSelectObject.Call(dc, oldFont)
	var face [128]uint16
	n, _, _ := iconGDI.NewProc("GetTextFaceW").Call(dc, uintptr(len(face)), uintptr(unsafe.Pointer(&face[0])))
	var metrics windowsTestTextMetric
	ok, _, _ := iconGDI.NewProc("GetTextMetricsW").Call(dc, uintptr(unsafe.Pointer(&metrics)))
	if n == 0 || ok == 0 {
		t.Fatal("cannot inspect realized GDI font")
	}
	return syscall.UTF16ToString(face[:]), metrics
}

func windowsTestCreateFont(dc uintptr, name string, height int32) (uintptr, uintptr) {
	face, _ := syscall.UTF16PtrFromString(name)
	font, _, _ := iconCreateFont.Call(uintptr(-height), 0, 0, 0, 400, 0, 0, 0, 1, 0, 0, 4, 0, uintptr(unsafe.Pointer(face)))
	runtime.KeepAlive(face)
	oldFont, _, _ := iconSelectObject.Call(dc, font)
	return font, oldFont
}

func windowsTestPreferredFace(t *testing.T) string {
	t.Helper()
	face, _ := windowsTestSelectedFont(t, func(dc uintptr) (uintptr, uintptr) {
		return windowsTestCreateFont(dc, "Tahoma", 11)
	})
	if face == "Tahoma" {
		return "Tahoma"
	}
	return "Segoe UI"
}

func TestWindowsNumberFontSelection(t *testing.T) {
	wantFace := windowsTestPreferredFace(t)
	// 8pt at 96 DPI, rounded to the final icon's actual pixel grid.
	for size, pixelHeight := range map[int]int32{16: 11, 20: 13, 24: 16, 32: 21} {
		pixelHeight = windowsTestExpectedPixelHeight(t, wantFace, size, pixelHeight)
		face, metrics := windowsTestSelectedFont(t, func(dc uintptr) (uintptr, uintptr) {
			return selectWindowsIconNumberFont(dc, size)
		})
		if face != wantFace || metrics.Weight != 400 {
			t.Errorf("size %d: realized font %q weight %d, want %q weight 400", size, face, metrics.Weight, wantFace)
		}
		if em := metrics.Height - metrics.InternalLeading; em != pixelHeight {
			t.Errorf("size %d: realized em height %dpx, want native font height %dpx", size, em, pixelHeight)
		}
	}
}

func windowsTestExpectedPixelHeight(t *testing.T, face string, size int, requested int32) int32 {
	t.Helper()
	if face == "Tahoma" {
		return requested
	}
	// A substitute face may be wider. Independently rasterize 100 to find the
	// largest whole font size that fits, without inspecting production metrics.
	for height := requested; height > 0; height-- {
		if windowsTestRawGlyph(t, "100", face, height).Bounds().Dx() <= size {
			return height
		}
	}
	t.Fatal("no readable substitute font fits")
	return 0
}

// Draw independently into an oversized DIB at the final font size. Comparing
// its cropped pixels with the emitted PNG detects clipping and any resampling.
func windowsTestRawGlyph(t *testing.T, label, face string, height int32) *image.Alpha {
	t.Helper()
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	dc, _, _ := iconCreateDC.Call(0)
	if dc == 0 {
		t.Fatal("cannot create independent glyph DC")
	}
	defer iconDeleteDC.Call(dc)
	const width, canvasHeight = 256, 96
	info := iconBitmapInfo{Size: 40, Width: width, Height: -canvasHeight, Planes: 1, BitCount: 32}
	var bits unsafe.Pointer
	bitmap, _, _ := iconCreateDIB.Call(dc, uintptr(unsafe.Pointer(&info)), 0, uintptr(unsafe.Pointer(&bits)), 0, 0)
	if bitmap == 0 || bits == nil {
		t.Fatal("cannot create independent glyph DIB")
	}
	defer iconDeleteObject.Call(bitmap)
	oldBitmap, _, _ := iconSelectObject.Call(dc, bitmap)
	if oldBitmap == 0 || oldBitmap == ^uintptr(0) {
		t.Fatal("cannot select independent glyph DIB")
	}
	defer iconSelectObject.Call(dc, oldBitmap)
	font, oldFont := windowsTestCreateFont(dc, face, height)
	if font == 0 || oldFont == 0 || oldFont == ^uintptr(0) {
		t.Fatal("cannot select independent glyph font")
	}
	defer iconDeleteObject.Call(font)
	defer iconSelectObject.Call(dc, oldFont)
	pixels := unsafe.Slice((*byte)(bits), width*canvasHeight*4)
	clear(pixels)
	iconSetTextColor.Call(dc, 0xffffff)
	iconSetBkMode.Call(dc, 1)
	text, _ := syscall.UTF16PtrFromString(label)
	rect := iconTextRect{Right: width, Bottom: canvasHeight}
	result, _, _ := iconDrawText.Call(dc, uintptr(unsafe.Pointer(text)), uintptr(len(label)), uintptr(unsafe.Pointer(&rect)), 0x20|0x800)
	runtime.KeepAlive(text)
	flushed, _, _ := iconGdiFlush.Call()
	if result == 0 || flushed == 0 {
		t.Fatal("independent glyph drawing failed")
	}
	mask := image.NewAlpha(image.Rect(0, 0, width, canvasHeight))
	for y := 0; y < canvasHeight; y++ {
		for x := 0; x < width; x++ {
			mask.Pix[y*mask.Stride+x] = pixels[(y*width+x)*4+2]
		}
	}
	bounds := windowsNumberInkBounds(mask)
	if bounds.Empty() {
		t.Fatal("independent glyph has no ink")
	}
	return mask.SubImage(bounds).(*image.Alpha)
}

func windowsTestEqualAlpha(t *testing.T, got image.Image, want *image.Alpha) {
	t.Helper()
	if got.Bounds().Size() != want.Bounds().Size() {
		t.Fatalf("pixel dimensions %v, want %v", got.Bounds().Size(), want.Bounds().Size())
	}
	for y := 0; y < want.Bounds().Dy(); y++ {
		for x := 0; x < want.Bounds().Dx(); x++ {
			_, _, _, a := got.At(got.Bounds().Min.X+x, got.Bounds().Min.Y+y).RGBA()
			if uint8(a>>8) != want.AlphaAt(want.Bounds().Min.X+x, want.Bounds().Min.Y+y).A {
				t.Fatalf("alpha differs at %d,%d: got %d, want %d", x, y, a>>8, want.AlphaAt(want.Bounds().Min.X+x, want.Bounds().Min.Y+y).A)
			}
		}
	}
}

func TestWindowsNumberPixelsMatchFinalSizeGDI(t *testing.T) {
	face := windowsTestPreferredFace(t)
	for size, pixelHeight := range map[int]int32{16: 11, 20: 13, 24: 16, 32: 21} {
		pixelHeight = windowsTestExpectedPixelHeight(t, face, size, pixelHeight)
		t.Run(strconv.Itoa(size), func(t *testing.T) {
			previousWidth := 0
			for _, value := range []int{9, 99, 100} {
				t.Run(strconv.Itoa(value), func(t *testing.T) {
					raw := windowsTestRawGlyph(t, strconv.Itoa(value), face, pixelHeight)
					im := decodeStyle(t, trayStylePNG(value, colCodex, size, iconStyleNumber), size)
					bounds := windowsNumberInkBounds(im)
					if bounds.Size() != raw.Bounds().Size() || raw.Bounds().Dx() > size {
						t.Fatalf("final ink %v differs from unclipped native glyph %v", bounds.Size(), raw.Bounds().Size())
					}
					crop := image.NewAlpha(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
					for y := 0; y < bounds.Dy(); y++ {
						for x := 0; x < bounds.Dx(); x++ {
							_, _, _, a := im.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
							crop.Pix[y*crop.Stride+x] = uint8(a >> 8)
						}
					}
					windowsTestEqualAlpha(t, crop, raw)
					if bounds.Dx() <= previousWidth {
						t.Error("glyph widths do not increase from 9 to 99 to 100")
					}
					previousWidth = bounds.Dx()
				})
			}
		})
	}
}

func TestWindowsNumberLayoutConsistentAcrossValues(t *testing.T) {
	for _, size := range []int{16, 20, 24, 32} {
		t.Run(strconv.Itoa(size), func(t *testing.T) {
			up := int(math.Round(float64(size) / 16))
			reference := windowsNumberInkBounds(decodeStyle(t, trayStylePNG(20, colCodex, size, iconStyleNumber), size))
			for value := 0; value <= 100; value++ {
				im := decodeStyle(t, trayStylePNG(value, colCodex, size, iconStyleNumber), size)
				bounds := windowsNumberInkBounds(im)
				if bounds.Empty() || bounds.Min.Y <= 0 || bounds.Max.Y >= size {
					t.Errorf("%d: missing or vertically clipped ink %v", value, bounds)
				}
				if math.Abs(float64(bounds.Dy()-reference.Dy())) > 1 {
					t.Errorf("%d: height %d differs from common font height %d", value, bounds.Dy(), reference.Dy())
				}
				if math.Abs(float64(bounds.Min.X+bounds.Max.X-size)) > 1 || math.Abs(float64(bounds.Min.Y+bounds.Max.Y-(size-2*up))) > 1 {
					t.Errorf("%d: ink %v is not horizontally centered and %dpx above vertical center", value, bounds, up)
				}
			}
		})
	}
}

func TestWindowsNumberPNGUsesNativePixelsAndSizeCache(t *testing.T) {
	// Deliberately revisit smaller sizes after larger ones with the same label.
	for _, size := range []int{32, 16, 24, 20, 16, 32} {
		for _, value := range []int{9, 55, 99, 100} {
			label := strconv.Itoa(value)
			mask := renderWindowsIconText(label, size)
			if mask == nil || mask.Bounds() != image.Rect(0, 0, size, size) {
				t.Fatalf("size %d label %s: invalid final native mask", size, label)
			}
			im := decodeStyle(t, trayStylePNG(value, colCodex, size, iconStyleNumber), size)
			windowsTestEqualAlpha(t, im, mask)
			dst := image.NewNRGBA(image.Rect(0, 0, size, size))
			if !drawNativeIconNumber(dst, label, colCodex, size) {
				t.Fatal("native renderer unexpectedly fell back")
			}
			windowsTestEqualAlpha(t, dst, mask)
		}
	}
}

func TestWindowsGDIResourcesReleased(t *testing.T) {
	if renderWindowsIconText("55", 16) == nil {
		t.Fatal("initial GDI text rendering failed")
	}
	process, err := syscall.GetCurrentProcess()
	if err != nil {
		t.Fatal(err)
	}
	resources := iconUser.NewProc("GetGuiResources")
	renderBatch := func() {
		var wg sync.WaitGroup
		for i := 0; i < 24; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if renderWindowsIconText("55", 16) == nil {
					t.Error("parallel uncached GDI text rendering failed")
				}
			}()
		}
		wg.Wait()
	}
	renderBatch()
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if renderWindowsIconText("55", 16) == nil {
		t.Fatal("GDI text rendering failed before resource measurement")
	}
	before, _, _ := resources.Call(uintptr(process), 0)
	for i := 0; i < 24; i++ {
		if renderWindowsIconText("55", 16) == nil {
			t.Fatal("uncached GDI text rendering failed")
		}
	}
	after, _, _ := resources.Call(uintptr(process), 0)
	if after > before {
		t.Errorf("GDI resources grew after 24 uncached renders: %d -> %d", before, after)
	}
}
