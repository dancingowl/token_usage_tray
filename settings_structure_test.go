package main

import (
	"strings"
	"testing"
)

func TestSettingsStructureShowsInputsAndInformationWithoutDisclosures(t *testing.T) {
	start := strings.Index(dashboardHTML, `<section id="settings"`)
	if start < 0 {
		t.Fatal("settings section missing")
	}
	settings := dashboardHTML[start:]
	for _, removed := range []string{"<details", "theme-setting", "advanced-settings", "other-alerts"} {
		if strings.Contains(settings, removed) {
			t.Errorf("settings still contains obsolete disclosure/input %q", removed)
		}
	}
	general := strings.Index(settings, `aria-labelledby="general-settings-title"`)
	poll := strings.Index(settings, `id="poll-minutes"`)
	end := strings.Index(settings[general:], `</section>`)
	if poll < general || poll >= general+end {
		t.Fatal("collection interval must be directly in General")
	}
	if !strings.Contains(settings, `<section id="system-details"`) || !strings.Contains(settings, `<p class="dash-note" id="poll-help">`) {
		t.Fatal("app information and collection guidance must be directly visible")
	}
}

func TestThemeMenuIsSharedByEveryPageAndOutsideSettingsForm(t *testing.T) {
	for _, view := range []string{"usage", "tokens", "settings"} {
		page := renderPageView(&State{}, nil, view)
		if strings.Count(page, `id="theme-control"`) != 1 || strings.Count(page, `role="menuitemradio"`) != 3 {
			t.Fatalf("%s must have one shared three-choice theme control", view)
		}
		if strings.Index(page, `id="theme-control"`) > strings.Index(page, `id="settings-form"`) {
			t.Fatal("theme control must remain outside the settings form")
		}
	}
}

func TestSettingsHaveAutomaticFeedbackWithoutManualSaveOrUnloadWarning(t *testing.T) {
	for _, removed := range []string{`id="save-settings"`, `id="reset-settings"`, "beforeunload", "저장하지 않은 변경사항", "변경사항은 자동으로 저장·반영됩니다.", `class="settings-footer"`} {
		if strings.Contains(dashboardHTML+dashboardJS, removed) {
			t.Errorf("obsolete manual settings flow remains: %s", removed)
		}
	}
	for _, required := range []string{`id="retry-settings"`, `id="settings-message"`, `aria-live="polite"`, `id="auto-update-state"`} {
		if !strings.Contains(dashboardHTML, required) {
			t.Errorf("automatic settings feedback missing: %s", required)
		}
	}
}
