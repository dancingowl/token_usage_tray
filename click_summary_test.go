package main

import (
	"strings"
	"testing"
	"time"
)

func TestClickSummaryCompactsEachQuotaIntoTwoLines(t *testing.T) {
	now := time.Date(2026, 10, 13, 15, 0, 0, 0, time.FixedZone("KST", 9*3600))
	s := State{Codex: AgentUsage{
		Available: true, OK: true, Src: "live",
		Short: Window{Left: 10, ResetAt: now.Add(2*time.Hour + 15*time.Minute).Unix(), WindowS: 5 * 3600},
		Week:  Window{Left: 18, ResetAt: now.Add(5 * 24 * time.Hour).Unix(), WindowS: 7 * 86400},
	}}
	rows := clickSummaryRowsAt(&s, now)
	if len(rows) != 2 {
		t.Fatalf("want two quota rows, got %+v", rows)
	}
	for i, want := range []struct {
		label, value, detail string
		left                 int
	}{
		{"Codex · 5시간 한도", "10% 남음", "10/13(화) 17:15 초기화 · 2시간 15분 후", 10},
		{"Codex · 주간 한도", "18% 남음", "10/18(일) 15:00 초기화 · 5일 후", 18},
	} {
		got := rows[i]
		if got.Label != want.label || got.Value != want.value || got.Detail != want.detail || got.Remaining != want.left {
			t.Errorf("quota %d: got %+v, want %+v", i, got, want)
		}
		if strings.ContainsAny(got.Label+got.Value+got.Detail, "\r\n") {
			t.Errorf("quota row contains extra lines: %+v", got)
		}
	}
}

func TestClickSummaryUnknownAndElapsedReset(t *testing.T) {
	now := time.Date(2026, 10, 13, 15, 0, 0, 0, time.UTC)
	s := State{Codex: emptyAgent()}
	s.Codex.Available = true
	s.Codex.Week.ResetAt = now.Add(-time.Minute).Unix()
	rows := clickSummaryRowsAt(&s, now)
	if len(rows) != 1 || rows[0].Label != "Codex · 주간 한도" || rows[0].Value != "확인 중" || rows[0].Remaining != -1 {
		t.Fatalf("unknown quota/absent short quota: %+v", rows)
	}
	if !strings.Contains(rows[0].Detail, "시각 경과, 갱신 대기") {
		t.Fatalf("elapsed reset status missing: %+v", rows[0])
	}
}

func TestClickSummaryKeepsExhaustedOrResetKnownShortQuota(t *testing.T) {
	now := time.Date(2026, 10, 13, 15, 0, 0, 0, time.UTC)
	for _, short := range []Window{{Left: 0}, {Left: -1, ResetAt: now.Add(time.Hour).Unix()}} {
		s := State{Codex: emptyAgent()}
		s.Codex.Available = true
		s.Codex.Short = short
		rows := clickSummaryRowsAt(&s, now)
		if len(rows) != 2 || rows[0].Label != "Codex · 5시간 한도" {
			t.Fatalf("known short quota was hidden: %+v", rows)
		}
		if short.Left == 0 && (rows[0].Value != "0% 남음" || rows[0].Remaining != 0) {
			t.Fatalf("exhausted quota must retain zero: %+v", rows[0])
		}
	}
}

func TestClickSummaryRoundsShortWaitAndShowsNextYear(t *testing.T) {
	now := time.Date(2026, 12, 31, 23, 59, 40, 0, time.UTC)
	got := clickResetDetail(Window{Left: 100, ResetAt: now.Add(30 * time.Second).Unix()}, now)
	if got != "2027/1/1(금) 00:00 초기화 · 1분 후" {
		t.Fatalf("year transition/short wait: %s", got)
	}
}

func TestClickSummaryModelsAndStaleStatusStayWithQuota(t *testing.T) {
	now := time.Date(2026, 10, 13, 15, 0, 0, 0, time.UTC)
	s := State{Antigravity: AgentUsage{Available: true, Src: "cache", AgeMin: 12, Limit: "limited",
		Models: []Window{{Label: "Claude\nSonnet", Left: 5}, {Label: "Gemini Pro", Left: 40}}}}
	rows := clickSummaryRowsAt(&s, now)
	if len(rows) != 2 || rows[0].Label != "Antigravity · Claude Sonnet 한도" || rows[1].Label != "Antigravity · Gemini Pro 한도" {
		t.Fatalf("model quota labels: %+v", rows)
	}
	if rows[0].Detail != "초기화 시각 확인 중 · 캐시 12분 전 · Antigravity 이용 제한" || rows[1].Detail != "초기화 시각 확인 중 · 캐시 12분 전" {
		t.Fatalf("status should remain in quota detail: %+v", rows)
	}
	if got := clickSummaryRowsAt(&State{}, now); len(got) != 0 {
		t.Fatalf("unavailable agents should not add rows: %+v", got)
	}
}

func TestClickSummaryLimitDoesNotMislabelHealthyQuota(t *testing.T) {
	s := State{Codex: AgentUsage{Available: true, Limit: "rate_limit", Short: Window{Left: 0}, Week: Window{Left: 70}}}
	rows := clickSummaryRowsAt(&s, time.Now())
	if len(rows) != 2 || !strings.Contains(rows[0].Detail, "한도 도달") || strings.Contains(rows[1].Detail, "한도 도달") {
		t.Fatalf("short limit must not be attached to healthy week: %+v", rows)
	}
}

func TestClickSummaryStructuredResetColumnsAndExceptions(t *testing.T) {
	now := time.Date(2026, 12, 31, 23, 59, 40, 0, time.UTC)
	s := State{Codex: AgentUsage{Available: true, Src: "cache", AgeMin: 12, Limit: "rate_limit",
		Short: Window{Left: 0, ResetAt: now.Add(30 * time.Second).Unix()},
		Week:  Window{Left: 70, ResetAt: now.Add(-time.Minute).Unix()},
	}}
	rows := clickSummaryRowsAt(&s, now)
	if len(rows) != 2 {
		t.Fatalf("want both quotas: %+v", rows)
	}
	if rows[0].Countdown != "1분 후 초기화" || rows[0].ResetDate != "2027/1/1(금) 00:00" || rows[0].Status != "캐시 12분 전 · 한도 도달" {
		t.Fatalf("future reset columns/status: %+v", rows[0])
	}
	if rows[1].Countdown != "초기화 갱신 대기" || rows[1].ResetDate != "12/31(목) 23:58" || rows[1].Status != "초기화 시각 경과 · 캐시 12분 전" {
		t.Fatalf("elapsed reset columns/status: %+v", rows[1])
	}
	unknown := clickSummaryRowsAt(&State{Codex: AgentUsage{Available: true, Short: Window{Left: -1}, Week: Window{Left: -1}}}, now)
	if len(unknown) != 1 || unknown[0].Countdown != "초기화 시각 확인 중" || unknown[0].ResetDate != "" || unknown[0].Status != "" {
		t.Fatalf("unknown reset columns: %+v", unknown)
	}
}

func TestClickSummaryCreditsAreSeparateAndUnknownIsNotZero(t *testing.T) {
	now := time.Date(2026, 10, 26, 15, 0, 0, 0, time.UTC)
	balance := 60087.0958375
	s := State{Claude: AgentUsage{Available: true, Short: Window{Left: -1}, Week: Window{Left: 90}},
		Codex: AgentUsage{Available: true, Short: Window{Left: 50}, Week: Window{Left: 70},
			Credits: &CreditBalance{Balance: &balance, CheckedAt: now}}}
	rows := clickSummaryRowsAt(&s, now)
	if len(rows) != 3 || rows[0].Supplement != "" || rows[1].Supplement != "" || rows[2].Supplement != "크레딧 잔액  60,087.10 크레딧" {
		t.Fatalf("credit balance must appear once below Codex quota: %+v", rows)
	}
	if rows[2].Remaining != 70 || rows[2].Value != "70% 남음" {
		t.Fatal("credits must not change quota value or severity")
	}
	balance = 0
	if got := clickSummaryRowsAt(&s, now)[2].Supplement; got != "크레딧 잔액  0.00 크레딧" {
		t.Fatalf("real zero lost: %s", got)
	}
	s.Codex.Credits = &CreditBalance{CheckedAt: now}
	if got := clickSummaryRowsAt(&s, now)[2].Supplement; got != "크레딧 잔액  확인 불가" {
		t.Fatalf("failed lookup must be unknown: %s", got)
	}
	s.Codex.Credits = &CreditBalance{Balance: &balance, CheckedAt: now.Add(-historyFreshness - time.Minute)}
	if got := clickSummaryRowsAt(&s, now)[2].Supplement; got != "크레딧 잔액  확인 불가" {
		t.Fatalf("stale balance must be unknown: %s", got)
	}
}
