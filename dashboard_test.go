package main

import (
	"strings"
	"testing"
)

func TestDashboardAvailableBeforeFirstTokenScan(t *testing.T) {
	page := renderPage(&State{}, nil)
	for _, id := range []string{"settings-form", "agent-status", "history-chart", "forecast-state", "check-update", "save-startup", "usage-snapshot", "token-snapshot"} {
		if !strings.Contains(page, `id="`+id+`"`) {
			t.Errorf("dashboard control %q missing before first token scan", id)
		}
	}
	if strings.Contains(page, `http-equiv="refresh"`) {
		t.Fatal("full-page refresh would discard unsaved settings")
	}
	if strings.Count(page, "<script>") != 1 || strings.Count(page, "</script>") != 1 || !strings.HasSuffix(page, "</body></html>") {
		t.Fatal("dashboard must initialize once and close the document before the first scan")
	}
	if strings.Contains(page, `<script src=`) || strings.Contains(page, "cdn.") {
		t.Fatal("local dashboard must not depend on a remote script")
	}
}

func TestDashboardPreservesExistingTokenCharts(t *testing.T) {
	s := &State{}
	s.Claude.Available = true
	s.Codex.Available = true
	page := renderPage(s, &TokenStats{})
	for _, expected := range []string{"Claude · 14일 토큰", "Codex · 14일 토큰", "모델별 · 7일", "프로젝트별 · 7일", `data-agent="claude"`, `data-agent="codex"`} {
		if !strings.Contains(page, expected) {
			t.Errorf("existing usage view lost %q", expected)
		}
	}
	// Settings live outside the replaceable snapshots, so background collection
	// can update charts without resetting focused controls or unsaved edits.
	formAt := strings.Index(page, `id="settings-form"`)
	usageAt := strings.Index(page, `id="usage-snapshot"`)
	tokensAt := strings.Index(page, `id="token-snapshot"`)
	if usageAt < 0 || formAt <= usageAt || tokensAt <= formAt {
		t.Fatal("current quota cards must precede the dashboard and token charts")
	}
	usageEnd := usageAt + strings.Index(page[usageAt:], "</section>")
	if usageEnd <= usageAt || usageEnd >= formAt {
		t.Fatal("settings must remain outside independently refreshed usage sections")
	}
}
