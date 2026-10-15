package main

import (
	"fmt"
	"strings"
	"time"

	"fyne.io/systray"
)

// Keep each quota's value separate so native rendering can emphasize it without parsing text.
func clickSummaryRows(s *State) []systray.ClickTooltipRow {
	return clickSummaryRowsAt(s, time.Now())
}

func clickSummaryRowsAt(s *State, now time.Time) []systray.ClickTooltipRow {
	var rows []systray.ClickTooltipRow
	for _, agent := range []struct {
		name string
		u    AgentUsage
	}{{"Claude", s.Claude}, {"Codex", s.Codex}, {"Antigravity", s.Antigravity}} {
		if !agent.u.Available {
			continue
		}
		start := len(rows)
		appendWindow := func(label string, w Window) {
			value := "확인 중"
			if w.Left >= 0 {
				value = fmt.Sprintf("%d%% 남음", w.Left)
			}
			detail := clickResetDetail(w, now)
			if agent.u.Src != "" && agent.u.Src != "live" && agent.u.Src != "api" {
				status := "기록"
				if agent.u.Src == "cache" {
					status = "캐시"
				}
				if agent.u.AgeMin >= 0 {
					status += fmt.Sprintf(" %d분 전", agent.u.AgeMin)
				}
				detail += " · " + status
			}
			rows = append(rows, systray.ClickTooltipRow{
				Label: agent.name + " · " + strings.Join(strings.Fields(label), " ") + " 한도",
				Value: value, Detail: detail, Remaining: w.Left,
			})
		}
		if agent.name == "Antigravity" && len(agent.u.Models) > 0 {
			for _, w := range agent.u.Models {
				appendWindow(w.Label, w)
			}
		} else {
			if agent.u.Short.Left >= 0 || agent.u.Short.ResetAt > 0 {
				appendWindow(windowName(agent.u.Short, "5시간"), agent.u.Short)
			}
			appendWindow(windowName(agent.u.Week, "주간"), agent.u.Week)
		}
		if agent.u.Limit != "" && len(rows) > start {
			matched := false
			for i := start; i < len(rows); i++ {
				if rows[i].Remaining == 0 {
					rows[i].Detail += " · 한도 도달"
					matched = true
				}
			}
			if !matched {
				// A service-wide limit does not identify a particular quota.
				rows[start].Detail += " · " + agent.name + " 이용 제한"
			}
		}
	}
	return rows
}

func clickResetDetail(w Window, now time.Time) string {
	if w.ResetAt <= 0 {
		return "초기화 시각 확인 중"
	}
	reset := time.Unix(w.ResetAt, 0).In(now.Location())
	date := fmt.Sprintf("%d/%d(%s) %s", int(reset.Month()), reset.Day(), weekdayKo[int(reset.Weekday())], reset.Format("15:04"))
	if reset.Year() != now.Year() {
		date = fmt.Sprintf("%d/%s", reset.Year(), date)
	}
	if !reset.After(now) {
		return date + " 초기화 · 시각 경과, 갱신 대기"
	}
	// Round up to minutes so an imminent reset never reads as zero minutes.
	minutes := int((reset.Sub(now) + time.Minute - 1) / time.Minute)
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
	return date + " 초기화 · " + strings.Join(remaining, " ") + " 후"
}
