package main

import "testing"

func TestIconSelectionRespectsQuotaAndVisibility(t *testing.T) {
	s := State{Claude: AgentUsage{Available: true, Week: Window{Left: 80}, Short: Window{Left: 5}}, Codex: AgentUsage{Available: true, Week: Window{Left: 20}, Short: Window{Left: -1}}}
	if left, _ := iconChoiceFor(&s, "week"); left != 20 {
		t.Fatal(left)
	}
	if left, _ := iconChoiceFor(&s, "short"); left != 5 {
		t.Fatal(left)
	}
	if left, _ := iconChoiceFor(&s, "urgent"); left != 5 {
		t.Fatal(left)
	}
	unknownWeek := State{Codex: AgentUsage{Available: true, Week: Window{Left: -1}, Short: Window{Left: 50}}}
	if left, _ := iconChoiceFor(&unknownWeek, "week"); left != -1 {
		t.Fatal("short quota incorrectly labeled weekly", left)
	}
	p := defaultTraySettings()
	p.VisibleAgents.Claude = false
	shown := trayVisibleState(s, p)
	if left, _ := iconChoiceFor(&shown, "short"); left != -1 {
		t.Fatal("absent short quota replaced by weekly", left)
	}
	p.VisibleAgents.Codex = false
	shown = trayVisibleState(s, p)
	if !trayShouldBeVisible(shown, s, p, "") || !trayShouldBeVisible(shown, s, p, "claude") || trayShouldBeVisible(shown, s, p, "codex") {
		t.Fatal("settings entry not uniquely retained")
	}
}

func TestAlertRulesDeduplicateRearmAndDisable(t *testing.T) {
	p := defaultTraySettings()
	p.NotifyLimits = false
	s := State{Codex: AgentUsage{Available: true, Week: Window{Left: 15}, Short: Window{Left: -1}}}
	r := p.AlertRules["codex7d"]
	r.Threshold = 25
	p.AlertRules["codex7d"] = r
	if events := alertTransitions(&s, p); len(events) != 1 {
		t.Fatal(events)
	}
	if events := alertTransitions(&s, p); len(events) != 0 {
		t.Fatal("duplicate", events)
	}
	s.Codex.Week.Left = 34
	if events := alertTransitions(&s, p); len(events) != 0 {
		t.Fatal("rearmed before hysteresis", events)
	}
	s.Codex.Week.Left = 35
	if events := alertTransitions(&s, p); len(events) != 1 {
		t.Fatal("no recovery", events)
	}
	s.Codex.Week.Left = 0
	if events := alertTransitions(&s, p); len(events) != 1 {
		t.Fatal("zero quota missed", events)
	}
	p.AlertsEnabled = false
	if events := alertTransitions(&s, p); len(events) != 0 || s.Alerts["codex7d"] {
		t.Fatal("disabled alert", events)
	}
	p.AlertsEnabled = true
	r.Enabled = false
	p.AlertRules["codex7d"] = r
	if events := alertTransitions(&s, p); len(events) != 0 {
		t.Fatal(events)
	}
}

func TestRecoveryCanBeSilentAndUnknownDoesNotFire(t *testing.T) {
	p := defaultTraySettings()
	r := p.AlertRules["claude5h"]
	r.Recovery = false
	p.AlertRules["claude5h"] = r
	s := State{Claude: AgentUsage{Available: true, Short: Window{Left: -1}, Week: Window{Left: -1}}}
	if len(alertTransitions(&s, p)) != 0 {
		t.Fatal("unknown alert")
	}
	s.Claude.Short.Left = 5
	alertTransitions(&s, p)
	s.Claude.Short.Left = 50
	if len(alertTransitions(&s, p)) != 0 || s.Alerts["claude5h"] {
		t.Fatal("silent recovery not rearmed")
	}
}
