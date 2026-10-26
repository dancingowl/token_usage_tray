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
		if !strings.Contains(got, `<div class="cs"><strong>5일 3시간 후 초기화</strong><time datetime="2026-10-20T13:00:00+09:00">10/20(화) 13:00</time></div>`) || strings.Contains(got, "<img") {
			t.Errorf("remaining %d must emphasize the countdown before a readable reset date: %s", tt.left, got)
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

func TestCompactQuotaCardsGroupPlanWithTitleAndOmitLiveNotes(t *testing.T) {
	for _, source := range []string{"live", "api"} {
		state := &State{Codex: emptyAgent()}
		state.Codex.Available, state.Codex.Plan, state.Codex.Src = true, "pro", source
		state.Codex.AgeMin = 0
		state.Codex.Short.Left, state.Codex.Week.Left = 90, 80
		for _, card := range gaugeCards(state) {
			got := cardHTML(card)
			if !strings.Contains(got, `<span class="card-plan"> · Pro</span></span></div>`) || strings.Count(got, "Pro") != 1 {
				t.Errorf("plan must appear once beside the title: %s", got)
			}
			if strings.Contains(got, "실시간") || strings.Contains(got, "기록 0분 전") || strings.Contains(got, `class="card-notes"`) {
				t.Errorf("fresh %s must not add a redundant metadata row: %s", source, got)
			}
		}
	}
}

func TestTokenPageSectionSurroundsReplaceableSnapshot(t *testing.T) {
	for _, tok := range []*TokenStats{nil, {}} {
		page := renderPageView(&State{}, tok, "tokens")
		start := strings.Index(page, `<section id="token-details"`)
		snapshot := strings.Index(page, `<section id="token-snapshot">`)
		close := strings.LastIndex(page, `</section></section>`)
		if start < 0 || snapshot < start || close < snapshot || !strings.Contains(page[start:snapshot], `<h2 id="token-title">토큰 통계`) || strings.Contains(page, `<details id="token-details"`) {
			t.Fatal("token statistics must have a direct heading and surround the replaceable snapshot")
		}
		if strings.Count(page, "<script>") != 1 || strings.Contains(page[snapshot:close], `id="settings-form"`) {
			t.Fatal("token refresh must preserve dashboard controls and initialize the script once")
		}
	}
}

func TestPageViewsHaveSeparateNavigationAndPreservePollingTargets(t *testing.T) {
	for _, tt := range []struct{ view, title, path string }{
		{"usage", "한도 현황", "/"}, {"tokens", "토큰 통계", "/tokens"}, {"settings", "설정", "/settings"},
		{"invalid", "한도 현황", "/"},
	} {
		page := renderPageView(&State{}, nil, tt.view)
		view := tt.view
		if view == "invalid" {
			view = "usage"
		}
		for _, want := range []string{
			`<title>Usage Tray · ` + tt.title + `</title>`, `<h1>Usage Tray</h1>`,
			`<body data-page="` + view + `">`, `<a href="` + tt.path + `" aria-current="page">` + tt.title + `</a>`,
			`id="usage-snapshot"`, `id="token-snapshot"`, `id="settings-form"`,
		} {
			if !strings.Contains(page, want) {
				t.Errorf("view %s missing %s", tt.view, want)
			}
		}
		navStart, navEnd := strings.Index(page, `<nav class="page-nav"`), strings.Index(page, `</nav>`)
		if navStart < 0 || navEnd < navStart || strings.Count(page[navStart:navEnd], `aria-current="page"`) != 1 {
			t.Errorf("view %s should have exactly one active navigation link", tt.view)
		}
		for _, link := range []string{`href="/"`, `href="/tokens"`, `href="/settings"`} {
			if !strings.Contains(page, link) {
				t.Errorf("view %s missing navigation %s", tt.view, link)
			}
		}
	}
	if got := renderPage(&State{}, nil); !strings.Contains(got, `<body data-page="usage">`) {
		t.Fatal("renderPage must default to the quota view")
	}
	for _, selector := range []string{`body:not([data-page="usage"]) #usage-snapshot`, `body:not([data-page="tokens"]) #token-details`} {
		if !strings.Contains(pageCSS, selector) {
			t.Errorf("missing view visibility selector %s", selector)
		}
	}
}

func TestQuotaResetRowsHandleUnknownElapsedAndYearChange(t *testing.T) {
	now := time.Date(2026, 12, 31, 23, 59, 40, 0, time.UTC)
	for _, tt := range []struct {
		name         string
		reset        int64
		strong, date string
	}{
		{"unknown", 0, "초기화 시각 확인 중", ""},
		{"elapsed", now.Add(-time.Minute).Unix(), "시각 경과, 갱신 대기", "12/31(목) 23:58"},
		{"imminent next year", now.Add(30 * time.Second).Unix(), "1분 후 초기화", "2027/1/1(금) 00:00"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := cardHTMLAt(gaugeCard{Title: `<quota>`, Left: -1, ResetAt: tt.reset, Mark: markClaude, Chips: []chip{{Text: `<metadata>`}}}, now)
			if !strings.Contains(got, `<strong>`+tt.strong+`</strong>`) {
				t.Fatalf("wrong reset countdown/status: %s", got)
			}
			if tt.date == "" && strings.Contains(got, `<time`) || tt.date != "" && !strings.Contains(got, `>`+tt.date+`</time>`) {
				t.Fatalf("wrong reset date: %s", got)
			}
			if strings.Contains(got, `<quota>`) || strings.Contains(got, `<metadata>`) || !strings.Contains(got, `&lt;quota&gt;`) || !strings.Contains(got, `&lt;metadata&gt;`) {
				t.Fatalf("quota text must remain escaped: %s", got)
			}
		})
	}
}

func TestCodexCreditSupplementBelongsToWeeklyQuotaCard(t *testing.T) {
	now := time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	balance := 60087.10
	for _, tt := range []struct {
		name    string
		credits *CreditBalance
		want    string
	}{
		{"decimal", &CreditBalance{Balance: &balance, CheckedAt: now}, "60,087.10"},
		{"missing", nil, "확인 불가"},
		{"stale", &CreditBalance{Balance: &balance, CheckedAt: now.Add(-historyFreshness - time.Second)}, "확인 불가"},
		{"unlimited", &CreditBalance{Unlimited: true, CheckedAt: now}, "무제한"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			state := &State{Claude: emptyAgent(), Codex: emptyAgent()}
			state.Claude.Available, state.Codex.Available = true, true
			state.Codex.Short.Left, state.Codex.Week.Left = 87, 74
			state.Codex.Credits = tt.credits
			count := 0
			for _, card := range gaugeCards(state) {
				got := cardHTMLAt(card, now)
				if strings.Contains(got, `class="card-credit"`) {
					count++
					for _, want := range []string{`74%<span>남음</span>`, `크레딧 잔액 <strong>` + tt.want + `</strong>`} {
						if !strings.Contains(got, want) {
							t.Errorf("credit card missing %q: %s", want, got)
						}
					}
					if card.Title != "Codex · 주간 한도" {
						t.Fatalf("credit supplement attached to %q", card.Title)
					}
				}
			}
			if count != 1 {
				t.Fatalf("want one weekly supplement, got %d", count)
			}
		})
	}
	zero := 0.0
	usage := AgentUsage{Credits: &CreditBalance{Balance: &zero, CheckedAt: now}}
	got := cardHTMLAt(gaugeCard{Title: "Codex · 주간 한도", Left: 74, Mark: markCodex, CreditUsage: &usage}, now)
	if !strings.Contains(got, `크레딧 잔액 <strong>0.00</strong>`) || strings.Contains(got, "확인 불가") {
		t.Fatalf("known zero was lost: %s", got)
	}
}

func TestCreditHistoryIsContainedInExistingUsageChart(t *testing.T) {
	for _, want := range []string{`for="history-window">항목`, `id="credit-history-summary"`, `id="history-title"`, `const compact = page==='usage'`, `if(credits){$('forecast-state').replaceChildren();return;}`} {
		if !strings.Contains(dashboardHTML+dashboardJS, want) {
			t.Errorf("missing existing chart integration %q", want)
		}
	}
	state := &State{Codex: emptyAgent()}
	state.Codex.Available = true
	page := renderPageView(state, &TokenStats{}, "tokens")
	start := strings.Index(page, `<section id="token-snapshot">`)
	end := strings.LastIndex(page, `</section></section>`)
	if start < 0 || end < start {
		t.Fatal("missing token snapshot")
	}
	if strings.Contains(page[start:end], "크레딧") || strings.Contains(page[start:end], "credit-history") {
		t.Fatal("credit UI added to token statistics")
	}
}
