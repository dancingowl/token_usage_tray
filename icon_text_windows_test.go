package main

import (
	"image"
	"runtime"
	"sync"
	"syscall"
	"testing"
)

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
