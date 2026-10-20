package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type agentVisibility struct {
	Claude      bool `json:"claude"`
	Codex       bool `json:"codex"`
	Antigravity bool `json:"antigravity"`
}

type alertRule struct {
	Enabled   bool `json:"enabled"`
	Threshold int  `json:"threshold"`
	Recovery  bool `json:"recovery"`
}

// All agent processes share preferences. Credentials never enter this file.
type traySettings struct {
	IconStyle         iconStyle            `json:"icon_style"`
	DisplayWindow     string               `json:"display_window"`
	VisibleAgents     agentVisibility      `json:"visible_agents"`
	PollSeconds       int                  `json:"poll_seconds"`
	AlertsEnabled     bool                 `json:"alerts_enabled"`
	NotifyLimits      bool                 `json:"notify_limits"`
	AlertsPausedUntil int64                `json:"alerts_paused_until"`
	Theme             string               `json:"theme"`
	AlertRules        map[string]alertRule `json:"alert_rules"`
}

func defaultTraySettings() traySettings {
	return traySettings{
		IconStyle: iconStyleDonut, DisplayWindow: "week", PollSeconds: 60, Theme: "system",
		VisibleAgents: agentVisibility{true, true, true}, AlertsEnabled: true, NotifyLimits: true,
		AlertRules: map[string]alertRule{
			"claude5h": {true, 10, true}, "claude7d": {true, 20, true},
			"codex5h": {true, 10, true}, "codex7d": {true, 20, true}, "antigravity": {true, 20, true},
		},
	}
}

func validIconStyle(style iconStyle) bool {
	return style == iconStyleDonut || style == iconStyleNumber
}

func validateTraySettings(settings traySettings) error {
	if settings.Theme != "" && settings.Theme != "system" && settings.Theme != "light" && settings.Theme != "dark" {
		return fmt.Errorf("테마는 시스템, 밝게 또는 어둡게여야 합니다")
	}
	if settings.AlertsPausedUntil < 0 {
		return fmt.Errorf("알림 중지 시각이 올바르지 않습니다")
	}
	if !validIconStyle(settings.IconStyle) {
		return fmt.Errorf("지원하지 않는 아이콘 모양: %q", settings.IconStyle)
	}
	if settings.DisplayWindow != "week" && settings.DisplayWindow != "short" && settings.DisplayWindow != "urgent" {
		return fmt.Errorf("표시 한도는 주간, 단기 또는 가장 부족한 한도여야 합니다")
	}
	if settings.PollSeconds < 60 || settings.PollSeconds > 3600 {
		return fmt.Errorf("갱신 주기는 60~3600초여야 합니다")
	}
	defaults := defaultTraySettings()
	if len(settings.AlertRules) != len(defaults.AlertRules) {
		return fmt.Errorf("한도별 알림 설정이 누락됐습니다")
	}
	for key, rule := range settings.AlertRules {
		if _, ok := defaults.AlertRules[key]; !ok {
			return fmt.Errorf("지원하지 않는 알림 한도: %s", key)
		}
		if rule.Threshold < 1 || rule.Threshold > 99 {
			return fmt.Errorf("알림 기준은 1~99%%여야 합니다")
		}
	}
	return nil
}

func settingsPath() string { return filepath.Join(dataDir(), "settings.json") }

func loadTraySettings() (traySettings, error) {
	settings := defaultTraySettings()
	b, err := os.ReadFile(settingsPath())
	if os.IsNotExist(err) {
		return settings, nil
	}
	if err != nil {
		return settings, err
	}
	if err := json.Unmarshal(b, &settings); err != nil {
		return defaultTraySettings(), fmt.Errorf("아이콘 설정을 읽지 못했다: %w", err)
	}
	// 폐기된 도넛+숫자 설정은 기본 도넛으로 읽는다.
	if settings.IconStyle == "donut-number" {
		settings.IconStyle = iconStyleDonut
	}
	if err := validateTraySettings(settings); err != nil {
		return defaultTraySettings(), err
	}
	return settings, nil
}

func saveTraySettings(settings traySettings) error {
	if err := validateTraySettings(settings); err != nil {
		return err
	}
	b, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return atomicWriteFile(settingsPath(), append(b, '\n'))
}

func atomicWriteFile(path string, data []byte) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".settings-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}

func trayVisibleState(s State, settings traySettings) State {
	s.Claude.Available = s.Claude.Available && settings.VisibleAgents.Claude
	s.Codex.Available = s.Codex.Available && settings.VisibleAgents.Codex
	s.Antigravity.Available = s.Antigravity.Available && settings.VisibleAgents.Antigravity
	return s
}
