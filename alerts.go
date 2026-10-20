package main

import (
	"fmt"
	"maps"
	"time"
)

type usageAlert struct{ Title, Message string }

// Pure transitions make threshold, deduplication and recovery behavior testable.
func alertTransitions(s *State, settings traySettings) []usageAlert {
	return alertTransitionsAt(s, settings, time.Now())
}

func alertTransitionsAt(s *State, settings traySettings, now time.Time) []usageAlert {
	// A pause preserves deduplication state; after expiry the next sample resumes normally.
	if settings.AlertsPausedUntil > now.Unix() {
		return nil
	}
	s.Alerts = maps.Clone(s.Alerts)
	if s.Alerts == nil {
		s.Alerts = map[string]bool{}
	}
	var events []usageAlert
	for _, t := range thresholds {
		rule := settings.AlertRules[t.key]
		if !settings.AlertsEnabled || !rule.Enabled {
			delete(s.Alerts, t.key)
			continue
		}
		if !t.on(s) {
			continue
		}
		v := t.left(s)
		if v < 0 {
			continue
		}
		fired := s.Alerts[t.key]
		if !fired && v < rule.Threshold {
			s.Alerts[t.key] = true
			events = append(events, usageAlert{t.label + " 한도 임박", fmt.Sprintf("%d%% 남음 · 알림 기준 %d%%", v, rule.Threshold)})
		} else if fired && v >= min(100, rule.Threshold+10) {
			s.Alerts[t.key] = false
			if rule.Recovery {
				events = append(events, usageAlert{t.label + " 한도 회복", fmt.Sprintf("%d%% 남음 · 다시 여유가 생겼습니다", v)})
			}
		}
	}
	if !settings.AlertsEnabled || !settings.NotifyLimits {
		delete(s.Alerts, "codexLimit")
	} else if s.Codex.Available {
		fired := s.Alerts["codexLimit"]
		if s.Codex.Limit != "" && !fired {
			s.Alerts["codexLimit"] = true
			events = append(events, usageAlert{"Codex 한도 도달", "사용 한도에 도달했습니다. 상세 화면을 확인해 주세요."})
		} else if s.Codex.Limit == "" && fired {
			s.Alerts["codexLimit"] = false
			events = append(events, usageAlert{"Codex 한도 해제", "다시 사용할 수 있습니다"})
		}
	}
	return events
}
