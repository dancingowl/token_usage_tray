//go:build windows

package systray

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

func TestLoadIconFromPreservesNativeSizeAndCache(t *testing.T) {
	tray := &winTray{loadedImages: make(map[string]windows.Handle)}
	tray.initialized.Store(true)
	destroyIcon := u32.NewProc("DestroyIcon")
	// Each cached handle is owned by this fixture; GetIconInfo's separate
	// bitmap copies are released by nativeLoadedIconSize below.
	t.Cleanup(func() {
		for src, icon := range tray.loadedImages {
			if ok, _, err := destroyIcon.Call(uintptr(icon)); ok == 0 {
				t.Errorf("DestroyIcon(%s): %v", src, err)
			}
			delete(tray.loadedImages, src)
		}
	})
	dir := t.TempDir()
	for _, size := range []int{16, 20, 24, 32} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			src := filepath.Join(dir, fmt.Sprintf("icon-%d.ico", size))
			writeNativeSizeICO(t, src, size)
			icon, err := tray.loadIconFrom(src)
			if err != nil || icon == 0 {
				t.Fatalf("loadIconFrom: handle=%v err=%v", icon, err)
			}
			width, height := nativeLoadedIconSize(t, icon)
			if width != int32(size) || height != int32(size) {
				t.Fatalf("source %dx%d loaded as %dx%d", size, size, width, height)
			}
			for n := 0; n < 32; n++ {
				cached, err := tray.loadIconFrom(src)
				if err != nil || cached != icon {
					t.Fatalf("cache changed: got %v, want %v, err=%v", cached, icon, err)
				}
			}
		})
	}
	if len(tray.loadedImages) != 4 {
		t.Fatalf("cache has %d entries, want 4", len(tray.loadedImages))
	}
}

func writeNativeSizeICO(t *testing.T, src string, size int) {
	t.Helper()
	im := image.NewNRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			im.SetNRGBA(x, y, color.NRGBA{R: uint8(x * 255 / size), G: uint8(y * 255 / size), B: 128, A: 255})
		}
	}
	var payload bytes.Buffer
	if err := png.Encode(&payload, im); err != nil {
		t.Fatal(err)
	}
	data := make([]byte, 22, 22+payload.Len())
	binary.LittleEndian.PutUint16(data[2:4], 1) // IMAGE_ICON
	binary.LittleEndian.PutUint16(data[4:6], 1) // One frame
	data[6], data[7] = byte(size), byte(size)
	binary.LittleEndian.PutUint16(data[10:12], 1)
	binary.LittleEndian.PutUint16(data[12:14], 32)
	binary.LittleEndian.PutUint32(data[14:18], uint32(payload.Len()))
	binary.LittleEndian.PutUint32(data[18:22], 22)
	if err := os.WriteFile(src, append(data, payload.Bytes()...), 0600); err != nil {
		t.Fatal(err)
	}
}

func nativeLoadedIconSize(t *testing.T, icon windows.Handle) (int32, int32) {
	t.Helper()
	var info struct {
		Icon                    int32
		XHotspot, YHotspot      uint32
		MaskBitmap, ColorBitmap windows.Handle
	}
	if ok, _, err := u32.NewProc("GetIconInfo").Call(uintptr(icon), uintptr(unsafe.Pointer(&info))); ok == 0 {
		t.Fatalf("GetIconInfo: %v", err)
	}
	for _, bitmap := range []windows.Handle{info.MaskBitmap, info.ColorBitmap} {
		if bitmap != 0 {
			defer func(handle windows.Handle) {
				if ok, _, err := pDeleteObject.Call(uintptr(handle)); ok == 0 {
					t.Errorf("DeleteObject: %v", err)
				}
			}(bitmap)
		}
	}
	var bitmap struct {
		Type, Width, Height, WidthBytes int32
		Planes, BitsPixel               uint16
		Bits                            uintptr
	}
	if info.ColorBitmap == 0 {
		t.Fatal("expected a color icon bitmap")
	}
	if n, _, err := g32.NewProc("GetObjectW").Call(uintptr(info.ColorBitmap), unsafe.Sizeof(bitmap), uintptr(unsafe.Pointer(&bitmap))); n == 0 {
		t.Fatalf("GetObjectW: %v", err)
	}
	return bitmap.Width, bitmap.Height
}
