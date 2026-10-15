package main

import (
	"strings"
	"testing"
	"time"
)

func TestCompactQuotaCardsKeepKnownZeroAndUnknownDistinct(t *testing.T) {
	now := time.Date(2026, time.October, 15, 10, 0, 0, 0, time.FixedZone("KST", 9*3600))
	for _, tt := range []struct {
		left         int
		class, value string
	}{{-1, " unknown", "확인 중"}, {0, " bad", "0%<span>남음</span>"}, {10, " bad", "10%<span>남음</span>"}, {11, " warn", "11%<span>남음</span>"}, {20, " warn", "20%<span>남음</span>"}, {21, "", "21%<span>남음</span>"}} {
		c := gaugeCard{Title: "Claude · 주간 한도", Left: tt.left, ResetAt: now.Add(5*24*time.Hour + 3*time.Hour).Unix(), Mark: markClaude}
		got := cardHTMLAt(c, now)
		if !strings.Contains(got, `<div class="cl`+tt.class+`">`+tt.value) {
			t.Errorf("remaining %d has wrong value or severity: %s", tt.left, got)
		}
		if !strings.Contains(got, "10/20(화) 13:00 초기화 · 5일 3시간 후") || strings.Contains(got, "<img") {
			t.Errorf("remaining %d lost the native reset detail or still renders a ring", tt.left)
		}
	}
}

func TestCompactQuotaCardsOmitAbsentShortWindowAndRetainServiceMetadata(t *testing.T) {
	s := &State{Claude: emptyAgent(), Codex: emptyAgent()}
	s.Claude.Available, s.Codex.Available = true, true
	s.Codex.Plan, s.Codex.Src, s.Codex.Limit = "self_serve_pro", "cache", "account restriction"
	s.Codex.AgeMin = 12
	s.Codex.Week.Left = 74
	cards := gaugeCards(s)
	if len(cards) != 2 {
		t.Fatalf("absent short windows should be omitted for both services: %d cards", len(cards))
	}
	got := cardHTML(cards[1])
	for _, text := range []string{"Pro", "캐시", "12분 전", "Codex 이용 제한: account restriction", "74%"} {
		if !strings.Contains(got, text) {
			t.Errorf("service metadata missing %q", text)
		}
	}
	if strings.Contains(got, "한도 도달") {
		t.Fatal("a service-wide restriction must not be attributed to the weekly quota")
	}
	s.Claude.Short.ResetAt = time.Now().Add(time.Hour).Unix()
	if cards = gaugeCards(s); len(cards) != 3 || cards[0].Left != -1 {
		t.Fatal("unknown short quota with a reset time should remain visible")
	}
}

func TestTokenDisclosureSurroundsReplaceableSnapshot(t *testing.T) {
	for _, tok := range []*TokenStats{nil, {}} {
		page := renderPage(&State{}, tok)
		start := strings.Index(page, `<details id="token-details"`)
		snapshot := strings.Index(page, `<section id="token-snapshot">`)
		close := strings.LastIndex(page, `</section></details>`)
		if start < 0 || snapshot < start || close < snapshot || strings.Contains(page[start:snapshot], " open") {
			t.Fatal("token disclosure must start collapsed and surround the replaceable snapshot")
		}
		if strings.Count(page, "<script>") != 1 || strings.Contains(page[snapshot:close], `id="settings-form"`) {
			t.Fatal("token refresh must preserve dashboard controls and initialize the script once")
		}
	}
}
