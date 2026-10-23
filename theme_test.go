package main

import (
	"bytes"
	"image/color"
	"math"
	"strings"
	"testing"
)

func TestResolvedTheme(t *testing.T) {
	for _, tc := range []struct {
		preference string
		system     bool
		want       string
	}{
		{"", false, "light"}, {"", true, "dark"}, {"system", true, "dark"}, {"system", false, "light"},
		{"light", true, "light"}, {"dark", false, "dark"}, {"invalid", true, "dark"},
	} {
		if got := resolvedTheme(tc.preference, tc.system); got != tc.want {
			t.Errorf("resolvedTheme(%q,%v)=%q, want %q", tc.preference, tc.system, got, tc.want)
		}
	}
}

func themeContrast(a, b color.NRGBA) float64 {
	luminance := func(c color.NRGBA) float64 {
		linear := func(v uint8) float64 {
			f := float64(v) / 255
			if f <= .04045 {
				return f / 12.92
			}
			return math.Pow((f+.055)/1.055, 2.4)
		}
		return .2126*linear(c.R) + .7152*linear(c.G) + .0722*linear(c.B)
	}
	l1, l2 := luminance(a), luminance(b)
	if l1 < l2 {
		l1, l2 = l2, l1
	}
	return (l1 + .05) / (l2 + .05)
}

func TestTaskbarNumberPaletteAndContrast(t *testing.T) {
	if got := trayNumberColor(colCodex, false); got != (color.NRGBA{8, 127, 104, 255}) {
		t.Fatalf("light Codex color=%v, want existing #087F68", got)
	}
	for _, brand := range []color.NRGBA{colCodex, colClaude, colAntigravity} {
		ink := trayNumberColor(brand, true)
		if ratio := themeContrast(ink, color.NRGBA{32, 32, 32, 255}); ratio < 7 {
			t.Errorf("dark taskbar brand %v contrast=%.2f, want >=7", brand, ratio)
		}
	}
	// Preserve the specifically requested existing light-taskbar color. Its
	// contrast is about 4.46 against Windows' light #F3F3F3 taskbar.
	if ratio := themeContrast(trayNumberColor(colCodex, false), color.NRGBA{243, 243, 243, 255}); ratio < 4.4 {
		t.Errorf("light taskbar Codex contrast=%.2f", ratio)
	}
}

func TestNumberThemePreservesMaskAndLowQuota(t *testing.T) {
	for _, size := range []int{16, 20, 24, 32, 44} {
		for _, left := range []int{-1, 0, 8, 19, 20, 75, 100} {
			lightBytes := trayStylePNGForTheme(left, colCodex, size, iconStyleNumber, false)
			darkBytes := trayStylePNGForTheme(left, colCodex, size, iconStyleNumber, true)
			light, dark := decodeStyle(t, lightBytes, size), decodeStyle(t, darkBytes, size)
			for y := 0; y < size; y++ {
				for x := 0; x < size; x++ {
					_, _, _, a := light.At(x, y).RGBA()
					_, _, _, b := dark.At(x, y).RGBA()
					if a != b {
						t.Fatalf("theme changed mask/alignment left=%d size=%d at %d,%d", left, size, x, y)
					}
				}
			}
			if left >= 0 && left < 20 && !bytes.Equal(lightBytes, darkBytes) {
				t.Fatalf("theme changed low-quota red left=%d size=%d", left, size)
			}
			if !bytes.Equal(trayStylePNGForTheme(left, colCodex, size, iconStyleDonut, true), ringPNG(left, colCodex, size)) {
				t.Fatal("theme changed donut")
			}
		}
	}
}

func TestThemeBootstrapAndSurfaceCoverage(t *testing.T) {
	for _, want := range []string{"usage-tray-theme", "prefers-color-scheme: dark", "applyUsageTrayTheme", "addEventListener('storage'", "addEventListener('change'", "document.documentElement.dataset.theme", "try{localStorage"} {
		if !strings.Contains(themeJS, want) {
			t.Errorf("bootstrap missing %s", want)
		}
	}
	for _, want := range []string{"html{background:var(--bg)}", ".history-help-content", ".history-tooltip", ".token-tooltip", ".history-selected-point", ".history-overview rect", ".icon-option-face", "button.primary", "[type=number]", "--bad:#ff9292"} {
		if !strings.Contains(themeCSS, want) {
			t.Errorf("theme coverage missing %s", want)
		}
	}
}
