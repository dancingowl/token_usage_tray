//go:build windows

package systray

import (
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	pBeginPaint = u32.NewProc("BeginPaint")
	pEndPaint   = u32.NewProc("EndPaint")
)

const clickTipDetailColor = 0x00786b62 // #626B78

type clickRowsLayout struct {
	width, height, padding, gap, firstHeight, detailHeight int32
}

func clickMax32(values ...int32) int32 {
	result := values[0]
	for _, value := range values[1:] {
		if value > result {
			result = value
		}
	}
	return result
}

// The severity comes solely from the numeric quota, never from display text.
func clickRowValueColor(remaining int) uintptr {
	if remaining < 0 {
		return clickTipTextColor
	}
	if remaining <= 10 {
		return 0x002626b9
	} // #B92626
	if remaining <= 20 {
		return 0x00006e96
	} // #966E00
	return clickTipTextColor
}

func clickRowMetrics(dpi uintptr) clickRowsLayout {
	if dpi == 0 {
		dpi = 96
	}
	px := func(dip int32) int32 { return int32((uintptr(dip)*dpi + 48) / 96) }
	return clickRowsLayout{width: px(340), padding: px(14), gap: px(12), firstHeight: px(26), detailHeight: px(18)}
}

func clickMeasureText(dc uintptr, font windows.Handle, text string) tooltipRect {
	if text == "" {
		text = " "
	}
	old, _, _ := pSelectObject.Call(dc, uintptr(font))
	defer pSelectObject.Call(dc, old)
	encoded, _ := windows.UTF16FromString(text)
	var rect tooltipRect
	pDrawText.Call(dc, uintptr(unsafe.Pointer(&encoded[0])), uintptr(len(encoded)-1), uintptr(unsafe.Pointer(&rect)), 0x0020|0x0400|0x0800) // single line, calculate, no prefix
	runtime.KeepAlive(encoded)
	return rect
}

func (tip *trackingTooltip) measureRows() clickRowsLayout {
	layout := clickRowMetrics(tip.fontDPI)
	dc, _, _ := pGetDC.Call(uintptr(tip.window))
	if dc != 0 {
		defer pReleaseDC.Call(uintptr(tip.window), dc)
		for _, row := range tip.rows {
			label := clickMeasureText(dc, tip.font, row.Label)
			value := clickMeasureText(dc, tip.valueFont, row.Value)
			detail := clickMeasureText(dc, tip.detailFont, row.Detail)
			layout.firstHeight = clickMax32(layout.firstHeight, label.Bottom, value.Bottom)
			layout.detailHeight = clickMax32(layout.detailHeight, detail.Bottom)
			layout.width = clickMax32(layout.width, label.Right+value.Right+layout.gap+2*layout.padding+2, detail.Right+2*layout.padding+2)
		}
	}
	// Two lines per quota with a 12 DIP gap, then the unchanged 36 DIP actions.
	footer := int32(36 * uintptr(clickMax32(int32(tip.fontDPI), 96)) / 96)
	layout.height = 2*layout.padding + int32(len(tip.rows))*(layout.firstHeight+layout.detailHeight+layout.gap) + footer + 2
	return layout
}

func clickDrawRowText(dc uintptr, font windows.Handle, text string, rect tooltipRect, color, align uintptr) {
	if text == "" {
		return
	}
	pSelectObject.Call(dc, uintptr(font))
	pSetTextColor.Call(dc, color)
	encoded, _ := windows.UTF16FromString(text)
	pDrawText.Call(dc, uintptr(unsafe.Pointer(&encoded[0])), uintptr(len(encoded)-1), uintptr(unsafe.Pointer(&rect)), align|0x0004|0x0020|0x0800) // vcenter, single line, no prefix
	runtime.KeepAlive(encoded)
}

// drawRows is shared by WM_PAINT and offscreen WM_PRINTCLIENT. All handles and
// strings are UI-thread snapshots. Restore the borrowed DC before freeing GDI.
func (tip *trackingTooltip) drawRows(dc uintptr, client tooltipRect) bool {
	saved, _, _ := pSaveDC.Call(dc)
	if saved == 0 {
		return false
	}
	defer pRestoreDC.Call(dc, saved)
	brush, _, _ := pCreateSolidBrush.Call(clickTipBackground)
	if brush == 0 {
		return false
	}
	pFillRect.Call(dc, uintptr(unsafe.Pointer(&client)), brush)
	pDeleteObject.Call(brush)
	pSetBkMode.Call(dc, 1)
	layout := tip.rowLayout
	y := client.Top + layout.padding
	for _, row := range tip.rows {
		value := clickMeasureText(dc, tip.valueFont, row.Value)
		first := tooltipRect{client.Left + layout.padding, y, client.Right - layout.padding, y + layout.firstHeight}
		label := first
		label.Right -= value.Right + layout.gap
		clickDrawRowText(dc, tip.font, row.Label, label, clickTipTextColor, 0)
		clickDrawRowText(dc, tip.valueFont, row.Value, first, clickRowValueColor(row.Remaining), 0x0002) // right
		detail := tooltipRect{first.Left, first.Bottom, first.Right, first.Bottom + layout.detailHeight}
		clickDrawRowText(dc, tip.detailFont, row.Detail, detail, clickTipDetailColor, 0)
		y = detail.Bottom + layout.gap
	}
	return true
}

// PAINTSTRUCT contains only native scalar fields. BeginPaint/EndPaint consume
// it synchronously and do not retain a Go pointer.
type clickPaintStruct struct {
	DC                 windows.Handle
	Erase              int32
	Rect               tooltipRect
	Restore, IncUpdate int32
	Reserved           [32]byte
}

func (tip *trackingTooltip) paintRows() {
	var paint clickPaintStruct
	dc, _, _ := pBeginPaint.Call(uintptr(tip.window), uintptr(unsafe.Pointer(&paint)))
	if dc == 0 {
		return
	}
	defer pEndPaint.Call(uintptr(tip.window), uintptr(unsafe.Pointer(&paint)))
	var client tooltipRect
	pGetClientRect.Call(uintptr(tip.window), uintptr(unsafe.Pointer(&client)))
	tip.drawRows(dc, client)
}
