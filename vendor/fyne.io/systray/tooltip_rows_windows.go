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
	return clickRowsLayout{width: px(360), padding: px(16), gap: px(12), firstHeight: px(34), detailHeight: px(20)}
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

// Legacy callers can still supply a single detail line. Structured callers
// never need their localized display text parsed to recover dates or status.
func clickRowSecondLine(row ClickTooltipRow) (string, string) {
	if row.Countdown == "" && row.ResetDate == "" {
		return row.Detail, ""
	}
	date := row.ResetDate
	if row.Countdown != "" && date != "" {
		date = " · " + date
	}
	return row.Countdown, date
}

func (tip *trackingTooltip) rowCountdownFont(row ClickTooltipRow) windows.Handle {
	if row.Countdown != "" {
		return tip.countdownFont
	}
	return tip.detailFont
}

func (layout clickRowsLayout) rowHeight(row ClickTooltipRow) int32 {
	height := layout.firstHeight + layout.detailHeight + layout.gap
	if row.Status != "" {
		height += layout.detailHeight
	}
	return height
}

func (tip *trackingTooltip) measureRows() clickRowsLayout {
	layout := clickRowMetrics(tip.fontDPI)
	dc, _, _ := pGetDC.Call(uintptr(tip.window))
	if dc != 0 {
		defer pReleaseDC.Call(uintptr(tip.window), dc)
		for _, row := range tip.rows {
			label := clickMeasureText(dc, tip.font, row.Label)
			value := clickMeasureText(dc, tip.valueFont, row.Value)
			left, right := clickRowSecondLine(row)
			countdown := clickMeasureText(dc, tip.rowCountdownFont(row), left)
			date := clickMeasureText(dc, tip.detailFont, right)
			status := clickMeasureText(dc, tip.detailFont, row.Status)
			layout.firstHeight = clickMax32(layout.firstHeight, label.Bottom, value.Bottom)
			layout.detailHeight = clickMax32(layout.detailHeight, countdown.Bottom, date.Bottom, status.Bottom)
			layout.width = clickMax32(layout.width,
				label.Right+value.Right+layout.gap+2*layout.padding+2,
				countdown.Right+date.Right+2*layout.padding+2,
				status.Right+2*layout.padding+2)
		}
	}
	// Each exceptional status receives its own measured line; the footer stays
	// compact regardless of how many service quotas are visible.
	footer := int32(30 * uintptr(clickMax32(int32(tip.fontDPI), 96)) / 96)
	layout.height = 2*layout.padding + footer + 2
	for _, row := range tip.rows {
		layout.height += layout.rowHeight(row)
	}
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
		label.Left += value.Right + layout.gap
		clickDrawRowText(dc, tip.valueFont, row.Value, first, clickRowValueColor(row.Remaining), 0)
		clickDrawRowText(dc, tip.font, row.Label, label, clickTipDetailColor, 0x0002)
		detail := tooltipRect{first.Left, first.Bottom, first.Right, first.Bottom + layout.detailHeight}
		left, right := clickRowSecondLine(row)
		countdownWidth := clickMeasureText(dc, tip.rowCountdownFont(row), left).Right
		if left == "" {
			countdownWidth = 0
		}
		countdown := detail
		countdown.Right = countdown.Left + countdownWidth
		date := detail
		date.Left += countdownWidth
		countdownColor := uintptr(clickTipDetailColor)
		if row.Countdown != "" {
			countdownColor = clickTipTextColor
		}
		clickDrawRowText(dc, tip.rowCountdownFont(row), left, countdown, countdownColor, 0)
		clickDrawRowText(dc, tip.detailFont, right, date, clickTipDetailColor, 0)
		if row.Status != "" {
			detail.Top, detail.Bottom = detail.Bottom, detail.Bottom+layout.detailHeight
			clickDrawRowText(dc, tip.detailFont, row.Status, detail, clickTipDetailColor, 0)
		}
		y = detail.Bottom + layout.gap
	}
	// A quiet rule separates the summary from its ghost actions.
	rule := tooltipRect{client.Left + layout.padding, y - layout.gap/2, client.Right - layout.padding, y - layout.gap/2 + 1}
	border, _, _ := pCreateSolidBrush.Call(clickTipBorderColor)
	if border != 0 {
		pFillRect.Call(dc, uintptr(unsafe.Pointer(&rule)), border)
		pDeleteObject.Call(border)
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
