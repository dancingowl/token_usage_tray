//go:build windows

package systray

import (
	"bytes"
	_ "embed"
	"image"
	"image/png"
	"runtime"
	"unsafe"
)

var (
	// Sources and license are bundled with the binary; painting needs no files,
	// fonts, SVG renderer, or additional native graphics runtime.
	//go:embed icons/refresh-cw.png
	clickRefreshIconPNG []byte
	//go:embed icons/settings.png
	clickSettingsIconPNG []byte
	//go:embed icons/LICENSE
	clickActionIconLicense string

	clickRefreshIconMask  = decodeClickActionIcon(clickRefreshIconPNG)
	clickSettingsIconMask = decodeClickActionIcon(clickSettingsIconPNG)
	pSetDIBitsToDevice    = g32.NewProc("SetDIBitsToDevice")
)

// Decode only once. Keep the original antialiased alpha independently of its
// black source color so every native button state can use its own palette.
func decodeClickActionIcon(data []byte) *image.Alpha {
	decoded, err := png.Decode(bytes.NewReader(data))
	if err != nil || decoded.Bounds() != image.Rect(0, 0, 128, 128) {
		return nil
	}
	mask := image.NewAlpha(decoded.Bounds())
	for y := 0; y < 128; y++ {
		for x := 0; x < 128; x++ {
			_, _, _, alpha := decoded.At(x, y).RGBA()
			mask.Pix[y*mask.Stride+x] = uint8(alpha >> 8)
		}
	}
	return mask
}

// Area averaging retains SVG coverage at fractional scale factors, unlike
// nearest-neighbor resizing or an aliased GDI outline. Coordinates use integer
// subpixels to integrate every overlapping source pixel exactly.
func clickActionIconPixels(mask *image.Alpha, size int, color, background uintptr) []byte {
	width, height := mask.Rect.Dx(), mask.Rect.Dy()
	pixels := make([]byte, size*size*4)
	for y := 0; y < size; y++ {
		top, bottom := y*height, (y+1)*height
		for x := 0; x < size; x++ {
			left, right := x*width, (x+1)*width
			coverage := 0
			for sy := top / size; sy < (bottom+size-1)/size; sy++ {
				wy := clickIconPixelOverlap(top, bottom, sy, size)
				for sx := left / size; sx < (right+size-1)/size; sx++ {
					wx := clickIconPixelOverlap(left, right, sx, size)
					coverage += int(mask.Pix[sy*mask.Stride+sx]) * wx * wy
				}
			}
			alpha := (coverage + width*height/2) / (width * height)
			offset := (y*size + x) * 4
			// COLORREF is 0x00BBGGRR; 32-bit BI_RGB DIB memory is BGRA.
			for channel := 0; channel < 3; channel++ {
				shift := uint((2 - channel) * 8)
				ink, fill := int(color>>shift&255), int(background>>shift&255)
				pixels[offset+channel] = byte((ink*alpha + fill*(255-alpha) + 127) / 255)
			}
			pixels[offset+3] = 255
		}
	}
	return pixels
}

func clickIconPixelOverlap(start, end, pixel, scale int) int {
	if start < pixel*scale {
		start = pixel * scale
	}
	if end > (pixel+1)*scale {
		end = (pixel + 1) * scale
	}
	return end - start
}

// Native BUTTON text remains the accessible name. This copies a small opaque
// DIB without selecting GDI objects or changing the borrowed DC's state.
func drawClickActionIcon(dc uintptr, bounds tooltipRect, dpi uintptr, action string, color, background uintptr) bool {
	var mask *image.Alpha
	switch action {
	case "refresh":
		mask = clickRefreshIconMask
	case "settings":
		mask = clickSettingsIconMask
	default:
		return false
	}
	if dc == 0 || mask == nil {
		return false
	}
	if dpi == 0 {
		dpi = 96
	}
	// Divide before multiplying so even an invalid, very large DPI cannot
	// overflow uintptr. The 20-DIP icon sits inside the 28-DIP native button.
	size := int(dpi/96)*20 + int(dpi%96)*20/96
	if size <= 0 || int64(size) > int64(bounds.Right)-int64(bounds.Left) || int64(size) > int64(bounds.Bottom)-int64(bounds.Top) {
		return false
	}
	pixels := clickActionIconPixels(mask, size, color, background)
	info := bitmapInfo{BmiHeader: bitmapInfoHeader{BiSize: 40, BiWidth: int32(size), BiHeight: -int32(size), BiPlanes: 1, BiBitCount: 32}}
	left := int64(bounds.Left) + (int64(bounds.Right)-int64(bounds.Left)-int64(size))/2
	top := int64(bounds.Top) + (int64(bounds.Bottom)-int64(bounds.Top)-int64(size))/2
	written, _, _ := pSetDIBitsToDevice.Call(dc, uintptr(left), uintptr(top), uintptr(size), uintptr(size),
		0, 0, 0, uintptr(size), uintptr(unsafe.Pointer(&pixels[0])), uintptr(unsafe.Pointer(&info)), 0)
	runtime.KeepAlive(pixels)
	runtime.KeepAlive(info)
	// Keep the bundled attribution reachable as well as the raster assets.
	runtime.KeepAlive(clickActionIconLicense)
	return written != 0
}
