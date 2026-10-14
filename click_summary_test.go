package main

import (
	"strings"
	"testing"
	"time"
)

func TestClickSummaryLabelsValuesAndResetTimes(t *testing.T) {
	now := time.Date(2026, 10, 13, 15, 0, 0, 0, time.FixedZone("KST", 9*3600))
	s := State{Codex: AgentUsage{
		Available: true, OK: true, Src: "live",
		Short: Window{Left: 10, ResetAt: now.Add(2*time.Hour + 15*time.Minute).Unix(), WindowS: 5 * 3600},
		Week:  Window{Left: 18, ResetAt: now.Add(5 * 24 * time.Hour).Unix(), WindowS: 7 * 86400},
	}}
	got := clickSummaryAt(&s, now)
	for _, want := range []string{"Codex", "5시간 한도: 10% 남음", "주간 한도: 18% 남음", "초기화: 10월 13일 (화) 17:15", "초기화까지: 2시간 15분", "초기화: 10월 18일 (일) 15:00", "초기화까지: 5일"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in summary:\n%s", want, got)
		}
	}
	if strings.Contains(got, "10/18") || len([]rune(got)) <= tipLimit {
		t.Fatalf("click summary must retain explicit labels beyond hover limit: %s", got)
	}
}

func TestClickSummaryUnknownAndElapsedReset(t *testing.T) {
	now := time.Date(2026, 10, 13, 15, 0, 0, 0, time.UTC)
	s := State{Codex: emptyAgent()}
	s.Codex.Available = true
	s.Codex.Week.ResetAt = now.Add(-time.Minute).Unix()
	got := clickSummaryAt(&s, now)
	for _, want := range []string{"주간 한도: 확인할 수 없음", "초기화 시각 경과 · 갱신 대기"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}
	if strings.Contains(got, "0% 남음") {
		t.Fatalf("unknown must not look exhausted: %s", got)
	}
	if strings.Contains(got, "5시간") {
		t.Fatalf("absent short quota must be hidden: %s", got)
	}
}

func TestClickSummaryKeepsExhaustedOrResetKnownShortQuota(t *testing.T) {
	now := time.Date(2026, 10, 13, 15, 0, 0, 0, time.UTC)
	for _, short := range []Window{{Left: 0}, {Left: -1, ResetAt: now.Add(time.Hour).Unix()}} {
		s := State{Codex: emptyAgent()}
		s.Codex.Available = true
		s.Codex.Short = short
		got := clickSummaryAt(&s, now)
		if !strings.Contains(got, "5시간 한도:") {
			t.Fatalf("known short quota was hidden: %s", got)
		}
		if short.Left == 0 && !strings.Contains(got, "5시간 한도: 0% 남음") {
			t.Fatalf("exhausted quota must retain zero: %s", got)
		}
	}
}

func TestClickSummaryRoundsShortWaitAndShowsNextYear(t *testing.T) {
	now := time.Date(2026, 12, 31, 23, 59, 40, 0, time.UTC)
	lines := strings.Join(clickWindowLines("5시간", Window{Left: 100, ResetAt: now.Add(30 * time.Second).Unix()}, now), "\n")
	if !strings.Contains(lines, "초기화: 2027년 1월 1일") || !strings.Contains(lines, "초기화까지: 1분") {
		t.Fatalf("year transition/short wait: %s", lines)
	}
}
