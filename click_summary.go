package main

import (
	"fmt"
	"strings"
	"time"
)

// 클릭 팝업은 셸의 짧은 hover 툴팁과 달리 각 숫자의 뜻을 모두 적는다.
func clickSummary(s *State) string { return clickSummaryAt(s, time.Now()) }

func clickSummaryAt(s *State, now time.Time) string {
	var sections []string
	for _, agent := range []struct {
		name string
		u    AgentUsage
	}{{"Claude", s.Claude}, {"Codex", s.Codex}, {"Antigravity", s.Antigravity}} {
		if !agent.u.Available {
			continue
		}
		lines := []string{agent.name}
		if agent.name == "Antigravity" && len(agent.u.Models) > 0 {
			for _, w := range agent.u.Models {
				lines = append(lines, clickWindowLines(w.Label, w, now)...)
			}
		} else {
			if agent.u.Short.Left >= 0 || agent.u.Short.ResetAt > 0 {
				lines = append(lines, clickWindowLines(windowName(agent.u.Short, "5시간"), agent.u.Short, now)...)
				lines = append(lines, "")
			}
			lines = append(lines, clickWindowLines(windowName(agent.u.Week, "주간"), agent.u.Week, now)...)
		}
		if agent.u.Limit != "" {
			lines = append(lines, "사용 한도에 도달했습니다")
		}
		if agent.u.Src != "" && agent.u.Src != "live" && agent.u.Src != "api" {
			line := "출처: " + agent.u.Src
			if agent.u.AgeMin >= 0 {
				line += fmt.Sprintf(" (%d분 전 기록)", agent.u.AgeMin)
			}
			lines = append(lines, line)
		}
		sections = append(sections, strings.Join(lines, "\n"))
	}
	if len(sections) == 0 {
		return "사용량을 확인하고 있습니다"
	}
	return strings.Join(sections, "\n\n")
}

func clickWindowLines(label string, w Window, now time.Time) []string {
	value := "확인할 수 없음"
	if w.Left >= 0 {
		value = fmt.Sprintf("%d%% 남음", w.Left)
	}
	lines := []string{label + " 한도: " + value}
	if w.ResetAt <= 0 {
		return append(lines, "초기화: 시각 정보 없음")
	}
	reset := time.Unix(w.ResetAt, 0).In(now.Location())
	date := fmt.Sprintf("%d월 %d일 (%s) %s", int(reset.Month()), reset.Day(), weekdayKo[int(reset.Weekday())], reset.Format("15:04"))
	if reset.Year() != now.Year() {
		date = fmt.Sprintf("%d년 %s", reset.Year(), date)
	}
	lines = append(lines, "초기화: "+date)
	d := reset.Sub(now)
	if d <= 0 {
		return append(lines, "초기화 시각 경과 · 갱신 대기")
	}
	// 분 단위 올림으로 곧 초기화될 값도 '0분'으로 보이지 않게 한다.
	minutes := int((d + time.Minute - 1) / time.Minute)
	days, hours, mins := minutes/(24*60), minutes/60%24, minutes%60
	var remaining []string
	if days > 0 {
		remaining = append(remaining, fmt.Sprintf("%d일", days))
	}
	if hours > 0 {
		remaining = append(remaining, fmt.Sprintf("%d시간", hours))
	}
	if days == 0 && mins > 0 {
		remaining = append(remaining, fmt.Sprintf("%d분", mins))
	}
	return append(lines, "초기화까지: "+strings.Join(remaining, " "))
}
