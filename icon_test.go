package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"runtime"
	"sync"
	"testing"
)

func decodeStyle(t *testing.T, data []byte, size int) image.Image {
	t.Helper()
	im, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("decode PNG: %v", err)
	}
	if im.Bounds() != image.Rect(0, 0, size, size) {
		t.Fatalf("bounds = %v, want %d square", im.Bounds(), size)
	}
	return im
}

func TestTrayStylesLegibleRegions(t *testing.T) {
	brand := color.NRGBA{16, 163, 127, 255}
	for _, left := range []int{-1, 0, 8, 19, 20, 100} {
		for _, size := range []int{16, 32, 44} {
			for _, style := range []iconStyle{iconStyleDonut, iconStyleNumber} {
				im := decodeStyle(t, trayStylePNG(left, brand, size, style), size)
				for _, p := range []image.Point{{0, 0}, {size - 1, 0}, {0, size - 1}, {size - 1, size - 1}} {
					if _, _, _, a := im.At(p.X, p.Y).RGBA(); a != 0 {
						t.Errorf("%s left=%d size=%d: painted corner %v", style, left, size, p)
					}
				}
				_, _, _, top := im.At(size/2, size/16).RGBA()
				if (top > 0) != (style != iconStyleNumber) {
					t.Errorf("%s left=%d size=%d: outer ring presence = %v", style, left, size, top > 0)
				}
				centerInk := false
				for y := size * 6 / 16; y < size*10/16; y++ {
					for x := size * 6 / 16; x < size*10/16; x++ {
						_, _, _, a := im.At(x, y).RGBA()
						centerInk = centerInk || a > 0
					}
				}
				if centerInk != (style != iconStyleDonut) {
					t.Errorf("%s left=%d size=%d: center label presence = %v", style, left, size, centerInk)
				}
				if style != iconStyleDonut {
					want := brand
					if left >= 0 && left < 20 {
						want = colDanger
					}
					found := false
					for y := 0; y < size; y++ {
						for x := 0; x < size; x++ {
							got := color.NRGBAModel.Convert(im.At(x, y)).(color.NRGBA)
							found = found || (got.R == want.R && got.G == want.G && got.B == want.B && got.A >= 160)
						}
					}
					if !found {
						t.Errorf("%s left=%d size=%d: no strong readable foreground of expected color", style, left, size)
					}
				}
			}
		}
	}
}

func TestUnknownNumberIsDashAndPercentIsClamped(t *testing.T) {
	brand := color.NRGBA{111, 145, 255, 255}
	im := decodeStyle(t, trayStylePNG(-1, brand, 16, iconStyleNumber), 16)
	var count int
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			_, _, _, a := im.At(x, y).RGBA()
			if a > 0 {
				count++
				if x < 5 || x >= 11 || y < 7 || y >= 9 {
					t.Errorf("unknown label has ink outside its centered dash: %d,%d", x, y)
				}
			}
		}
	}
	if count != 12 {
		t.Errorf("dash painted pixels = %d, want 12", count)
	}
	for _, style := range []iconStyle{iconStyleNumber} {
		if !bytes.Equal(trayStylePNG(101, brand, 16, style), trayStylePNG(100, brand, 16, style)) {
			t.Errorf("%s does not clamp above 100", style)
		}
		if !bytes.Equal(trayStylePNG(-5, brand, 16, style), trayStylePNG(-1, brand, 16, style)) {
			t.Errorf("%s does not treat negative usage as unknown", style)
		}
	}
}

func TestTrayIconPlatformFormat(t *testing.T) {
	for _, style := range []iconStyle{iconStyleDonut, iconStyleNumber} {
		data := trayIconStyle(100, colCodex, style)
		if runtime.GOOS != "windows" {
			decodeStyle(t, data, 44)
			continue
		}
		if len(data) < 22 || binary.LittleEndian.Uint16(data[2:4]) != 1 || binary.LittleEndian.Uint16(data[4:6]) != 1 {
			t.Fatal("invalid single-image ICO header")
		}
		wantSize := 32
		if style == iconStyleNumber {
			wantSize = nativeTrayIconSize()
		}
		if data[6] != byte(wantSize) || data[7] != byte(wantSize) || binary.LittleEndian.Uint32(data[18:22]) != 22 {
			t.Fatalf("invalid %dpx ICO dimensions or image offset", wantSize)
		}
		if int(binary.LittleEndian.Uint32(data[14:18])) != len(data)-22 {
			t.Fatal("ICO payload length mismatch")
		}
		decodeStyle(t, data[22:], wantSize)
	}
	if !bytes.Equal(trayIcon(19, colClaude), trayIconStyle(19, colClaude, iconStyleDonut)) {
		t.Fatal("legacy trayIcon no longer defaults to donut")
	}
	if !bytes.Equal(trayStylePNG(20, colCodex, 32, iconStyle("invalid")), ringPNG(20, colCodex, 32)) {
		t.Fatal("unknown style no longer falls back to donut")
	}
}

func TestTrayStyleConcurrentRendering(t *testing.T) {
	baseline := trayStylePNG(19, colClaude, 32, iconStyleNumber)
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if got := trayStylePNG(19, colClaude, 32, iconStyleNumber); !bytes.Equal(got, baseline) {
				t.Error("parallel rendering changed PNG output")
			}
		}()
	}
	wg.Wait()
}
