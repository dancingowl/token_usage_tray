package main

import (
	"image"
	"math"
	"runtime"
	"strconv"
	"sync"
	"syscall"
	"testing"
)

// Measure the emitted pixels, including antialiased edges, rather than the font
// metrics or the renderer's requested rectangle.
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

func TestWindowsNumberLayoutConsistentAcrossValues(t *testing.T) {
	for _, size := range []int{16, 20, 24, 32} {
		t.Run(strconv.Itoa(size), func(t *testing.T) {
			// 20 is an ordinary two-digit value. Every other value, including
			// the widest label 100, should retain its displayed cap height.
			reference := windowsNumberInkBounds(decodeStyle(t, trayStylePNG(20, colCodex, size, iconStyleNumber), size))
			if reference.Empty() {
				t.Fatal("reference number has no ink")
			}
			for value := 0; value <= 100; value++ {
				im := decodeStyle(t, trayStylePNG(value, colCodex, size, iconStyleNumber), size)
				bounds := windowsNumberInkBounds(im)
				if bounds.Empty() {
					t.Errorf("%d: missing number", value)
					continue
				}
				if bounds.Min.X < 1 || bounds.Min.Y < 1 || bounds.Max.X > size-1 || bounds.Max.Y > size-1 {
					t.Errorf("%d: number reaches the icon boundary: %v", value, bounds)
				}
				if math.Abs(float64(bounds.Dy()-reference.Dy())) > 1 {
					t.Errorf("%d: ink height %d differs from two-digit height %d", value, bounds.Dy(), reference.Dy())
				}
				// Half a final pixel allows unavoidable rounding on the small
				// output grid, while detecting a visibly shifted baseline.
				if math.Abs(float64(bounds.Min.Y+bounds.Max.Y-reference.Min.Y-reference.Max.Y)) > 1 {
					t.Errorf("%d: vertical center differs from two-digit number: %v versus %v", value, bounds, reference)
				}
				if math.Abs(float64(bounds.Min.X+bounds.Max.X-size)) > 1 || math.Abs(float64(bounds.Min.Y+bounds.Max.Y-size)) > 1 {
					t.Errorf("%d: ink is not centered in %dpx icon: %v", value, size, bounds)
				}
			}
		})
	}
}

func TestWindowsGDIResourcesReleased(t *testing.T) {
	if renderWindowsIconText("55") == nil {
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
				if renderWindowsIconText("55") == nil {
					t.Error("parallel uncached GDI text rendering failed")
				}
			}()
		}
		wg.Wait()
	}
	// 첫 병렬 호출에서 초기화되는 Windows 폰트 리소스는 측정 전에 준비한다.
	renderBatch()
	// 스레드별 Windows 초기화를 핸들 누수로 오판하지 않도록 같은 스레드에서 센다.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if renderWindowsIconText("55") == nil {
		t.Fatal("GDI text rendering failed before resource measurement")
	}
	before, _, _ := resources.Call(uintptr(process), 0)
	for i := 0; i < 24; i++ {
		if renderWindowsIconText("55") == nil {
			t.Fatal("uncached GDI text rendering failed")
		}
	}
	after, _, _ := resources.Call(uintptr(process), 0)
	if after > before {
		t.Errorf("GDI resources grew after 24 uncached renders: %d -> %d", before, after)
	}
}

func TestWindowsNativeNumbersFontAndSize(t *testing.T) {
	for _, label := range []string{"-", "0", "8", "19", "20", "55", "90", "100"} {
		mask := renderWindowsIconText(label)
		if mask == nil {
			t.Fatalf("GDI text rendering failed for %s", label)
		}
		antialiased := false
		for _, a := range mask.Pix {
			antialiased = antialiased || (a > 0 && a < 255)
		}
		if label != "-" && !antialiased {
			t.Errorf("%s did not produce an antialiased font outline", label)
		}
		dst := image.NewNRGBA(image.Rect(0, 0, 64, 64))
		if !drawNativeIconNumber(dst, label, colCodex, 16) {
			t.Fatal("native renderer unexpectedly fell back")
		}
		bounds := image.Rectangle{}
		for y := 0; y < 64; y++ {
			for x := 0; x < 64; x++ {
				if dst.NRGBAAt(x, y).A > 0 {
					bounds = bounds.Union(image.Rect(x, y, x+1, y+1))
				}
			}
		}
		if bounds.Empty() || bounds.Min.X <= 0 || bounds.Min.Y <= 0 || bounds.Max.X >= 64 || bounds.Max.Y >= 64 {
			t.Errorf("%s: clipped or missing ink: %v", label, bounds)
		}
		if label == "-" && (bounds.Dx() < 3*bounds.Dy() || bounds.Dy() > 8) {
			t.Errorf("native unknown label became a rectangle rather than a thin dash: %v", bounds)
		}
		if label != "-" && len(label) <= 2 && bounds.Dy() < 30 {
			t.Errorf("%s: numeric height=%gpx, want at least 7.5px", label, float64(bounds.Dy())/4)
		}
	}
}
