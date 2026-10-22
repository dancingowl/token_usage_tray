package main

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestSettingsPatchMergesLeavesAndRejectsInvalidChanges(t *testing.T) {
	t.Setenv("USAGE_TRAY_DATA_DIR", t.TempDir())
	h := appHTTPHandler()
	for _, body := range []string{
		`{"visible_agents":{"claude":false}}`,
		`{"alert_rules":{"claude5h":{"threshold":23}}}`,
		`{"alert_rules":{"claude5h":{"recovery":false}}}`,
		`{"poll_seconds":150}`,
	} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, apiRequest(http.MethodPatch, "/api/settings", body))
		if w.Code != http.StatusOK {
			t.Fatalf("%s: %d %s", body, w.Code, w.Body.String())
		}
	}
	got, err := loadTraySettings()
	if err != nil || got.VisibleAgents.Claude || !got.VisibleAgents.Codex || got.AlertRules["claude5h"].Threshold != 23 || got.AlertRules["claude5h"].Recovery || !got.AlertRules["claude5h"].Enabled || got.PollSeconds != 150 {
		t.Fatalf("leaf changes lost another preference: %+v %v", got, err)
	}
	for _, body := range []string{
		`{}`, `null`, `{"unknown":true}`, `{"theme":"dark"}`, `{"poll_seconds":null}`, `{"poll_seconds":1}`, `{"poll_seconds":60.5}`,
		`{"visible_agents":{"unknown":true}}`, `{"visible_agents":null}`, `{"visible_agents":[]}`, `{"visible_agents":{}}`,
		`{"alert_rules":{"claude5h":{"threshold":100}}}`, `{"alert_rules":{"claude5h":{"extra":true}}}`, `{"alert_rules":{"unknown":{"enabled":true}}}`,
		`{"notify_limits":false,"alert_rules":{"claude5h":{"threshold":0}}}`, `{"notify_limits":null}`,
	} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, apiRequest(http.MethodPatch, "/api/settings", body))
		if w.Code != http.StatusBadRequest {
			t.Errorf("accepted %s: %d %s", body, w.Code, w.Body.String())
		}
	}
	after, _ := loadTraySettings()
	if !after.NotifyLimits || after.PollSeconds != got.PollSeconds || after.AlertRules["claude5h"] != got.AlertRules["claude5h"] {
		t.Fatal("invalid patch partially persisted")
	}
}

func TestConcurrentSettingsPatchAndThemePreserveIndependentLeaves(t *testing.T) {
	t.Setenv("USAGE_TRAY_DATA_DIR", t.TempDir())
	h := appHTTPHandler()
	changes := []struct{ path, method, body string }{
		{"/api/settings", http.MethodPatch, `{"visible_agents":{"claude":false}}`},
		{"/api/settings", http.MethodPatch, `{"visible_agents":{"codex":false}}`},
		{"/api/settings", http.MethodPatch, `{"alert_rules":{"claude5h":{"threshold":29}}}`},
		{"/api/settings", http.MethodPatch, `{"alert_rules":{"claude5h":{"recovery":false}}}`},
		{"/api/settings", http.MethodPatch, `{"poll_seconds":600}`},
		{"/api/theme", http.MethodPut, `{"theme":"dark"}`},
	}
	var group sync.WaitGroup
	start := make(chan struct{})
	for _, change := range changes {
		group.Add(1)
		go func() {
			defer group.Done()
			<-start
			w := httptest.NewRecorder()
			h.ServeHTTP(w, apiRequest(change.method, change.path, change.body))
			if w.Code != http.StatusOK {
				t.Errorf("concurrent update failed: %d %s", w.Code, w.Body.String())
			}
		}()
	}
	close(start)
	group.Wait()
	got, err := loadTraySettings()
	if err != nil || got.VisibleAgents.Claude || got.VisibleAgents.Codex || !got.VisibleAgents.Antigravity || got.AlertRules["claude5h"].Threshold != 29 || got.AlertRules["claude5h"].Recovery || got.PollSeconds != 600 || got.Theme != "dark" {
		t.Fatalf("concurrent independent changes lost: %+v %v", got, err)
	}
}
