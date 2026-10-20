package main

import (
	"html"
	"image/color"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestTokenChartNarrowLabelsKeepEndpointsWithoutHidingBars(t *testing.T) {
	for _, count := range []int{7, 14, 30} {
		days := make([]DayRow, count)
		for i := range days {
			days[i].Day = time.Date(2026, 10, 20-count+i+1, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
		}
		chart := barChartHTML(days, color.NRGBA{}, func(d DayRow) int64 { return d.Input }, "Codex")
		labels := regexp.MustCompile(`<div class="ch-x(?: sec)?">[^<]+</div>`).FindAllString(chart, -1)
		if len(tokenChartColumns(t, chart)) != count || len(labels) != count || strings.Count(chart, `<div class="ch-x">`) != 7 {
			t.Fatalf("%d-day chart must keep all bars and seven mobile date labels", count)
		}
		if strings.Contains(labels[0], " sec") || strings.Contains(labels[len(labels)-1], " sec") {
			t.Fatal("first and last dates must stay visible")
		}
	}
}

// Read only the interactive columns: decorative bars/scales should not become
// extra tab stops, and every column must retain its unrounded source data.
func tokenChartColumns(t *testing.T, chart string) []map[string]string {
	t.Helper()
	var columns []map[string]string
	tags := regexp.MustCompile(`<[^>]+>`).FindAllString(chart, -1)
	attributes := regexp.MustCompile(`([a-z-]+)="([^"]*)"`)
	for _, tag := range tags {
		attrs := map[string]string{}
		for _, pair := range attributes.FindAllStringSubmatch(tag, -1) {
			attrs[pair[1]] = html.UnescapeString(pair[2])
		}
		if attrs["class"] == "ch-col" || strings.HasPrefix(attrs["class"], "ch-col ") {
			columns = append(columns, attrs)
		} else if attrs["tabindex"] == "0" {
			t.Errorf("decorative chart element should not create an extra tab stop: %s", tag)
		}
	}
	return columns
}

func TestTokenChartCodexPreservesExactDailyBreakdown(t *testing.T) {
	days := []DayRow{
		{Day: "2026-10-19", Row: Row{Input: 9876543210, Output: 123456789, CacheRead: 8765432100, CacheCreate: 52, Requests: 12345}},
		{Day: "2026-10-20"},
	}
	chart := barChartHTML(days, color.NRGBA{}, func(d DayRow) int64 { return d.Input + d.Output }, "Codex")
	columns := tokenChartColumns(t, chart)
	if len(columns) != len(days) {
		t.Fatalf("got %d interactive columns, want %d", len(columns), len(days))
	}
	for i, day := range days {
		attrs := columns[i]
		for key, want := range map[string]string{
			"tabindex": "0", "data-agent": "Codex", "data-day": day.Day,
			"data-total": strconv.FormatInt(day.Input+day.Output, 10),
			"data-input": strconv.FormatInt(day.Input, 10), "data-output": strconv.FormatInt(day.Output, 10),
			"data-cache-read": strconv.FormatInt(day.CacheRead, 10), "data-cache-create": strconv.FormatInt(day.CacheCreate, 10),
			"data-requests": strconv.FormatInt(day.Requests, 10),
		} {
			if attrs[key] != want {
				t.Errorf("column %d %s = %q, want %q", i, key, attrs[key], want)
			}
		}
		label := strings.ReplaceAll(attrs["aria-label"], ",", "")
		for _, want := range []string{day.Day, "Codex", strconv.FormatInt(day.Input+day.Output, 10)} {
			if !strings.Contains(label, want) {
				t.Errorf("column %d accessible label lacks full date/service/exact total %q: %q", i, want, attrs["aria-label"])
			}
		}
		if _, legacyTitle := attrs["title"]; legacyTitle {
			t.Error("native title must not compete with the detailed tooltip")
		}
	}
}

func TestTokenChartClaudeIncludesBothCacheCategories(t *testing.T) {
	day := DayRow{Day: "2026-10-20", Row: Row{Input: 101, Output: 202, CacheRead: 303, CacheCreate: 404, Requests: 5}}
	chart := barChartHTML([]DayRow{day}, color.NRGBA{}, func(d DayRow) int64 { return d.total() }, "Claude")
	columns := tokenChartColumns(t, chart)
	if len(columns) != 1 {
		t.Fatalf("got %d columns, want one", len(columns))
	}
	if got := columns[0]["data-total"]; got != "1010" {
		t.Errorf("Claude total = %q, want input + output + cache read + cache create = 1010", got)
	}
}

func TestTokenChartKeepsIntegersAboveJavaScriptSafeRange(t *testing.T) {
	const total int64 = 9007199254740993
	chart := barChartHTML([]DayRow{{Day: "2026-10-20", Row: Row{Input: total}}}, color.NRGBA{}, func(d DayRow) int64 { return d.Input }, "Codex")
	columns := tokenChartColumns(t, chart)
	if len(columns) != 1 || columns[0]["data-total"] != "9007199254740993" || !strings.Contains(columns[0]["aria-label"], "9007199254740993") {
		t.Fatalf("daily total must retain integer precision in data and accessible text: %s", chart)
	}
}

func TestTokenChartHasVisibleFocusAndInspectedBar(t *testing.T) {
	for _, selector := range []string{".ch-col:focus-visible{", ".ch-col:focus-visible .ch-bar", ".ch-col.is-inspected .ch-bar"} {
		if !strings.Contains(pageCSS, selector) {
			t.Errorf("missing chart focus/selection treatment %q", selector)
		}
	}
}

func TestTokenChartEscapesUntrustedDateAndAgent(t *testing.T) {
	day := `2026-10-20"><img src=x onerror="bad">`
	agent := `Codex"><script>bad</script>&`
	chart := barChartHTML([]DayRow{{Day: day}}, color.NRGBA{}, func(DayRow) int64 { return 0 }, agent)
	columns := tokenChartColumns(t, chart)
	if len(columns) != 1 {
		t.Fatalf("got %d columns, want one", len(columns))
	}
	for key, want := range map[string]string{"data-agent": agent, "data-day": day} {
		if got := columns[0][key]; got != want {
			t.Errorf("escaped %s did not round-trip: got %q, want %q", key, got, want)
		}
	}
	if strings.Contains(chart, "<img") || strings.Contains(chart, "<script>") || strings.Contains(chart, `onerror="bad"`) {
		t.Fatalf("untrusted date or agent injected markup: %s", chart)
	}
}
