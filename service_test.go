package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func apiRequest(method, path, body string) *http.Request {
	r := httptest.NewRequest(method, "http://127.0.0.1:47113"+path, strings.NewReader(body))
	r.Host = "127.0.0.1:47113"
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-Usage-Tray", "1")
	return r
}

func TestSettingsAPIValidationAndPersistence(t *testing.T) {
	t.Setenv("USAGE_TRAY_DATA_DIR", t.TempDir())
	h := appHTTPHandler()
	settings := defaultTraySettings()
	settings.PollSeconds = 600
	settings.DisplayWindow = "urgent"
	settings.VisibleAgents.Claude = false
	b, _ := json.Marshal(settings)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, apiRequest("PUT", "/api/settings", string(b)))
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	got, err := loadTraySettings()
	if err != nil || got.PollSeconds != 600 || got.DisplayWindow != "urgent" || got.VisibleAgents.Claude {
		t.Fatalf("not persisted: %+v %v", got, err)
	}
	for _, invalid := range []string{`{"poll_seconds":1}`, `{"unknown":true}`, `{} {}`, strings.Repeat("a", 70<<10)} {
		w = httptest.NewRecorder()
		h.ServeHTTP(w, apiRequest("PUT", "/api/settings", invalid))
		if w.Code != 400 {
			t.Errorf("invalid update accepted: %d", w.Code)
		}
	}
	got, _ = loadTraySettings()
	if got.PollSeconds != 600 {
		t.Fatal("invalid request overwrote saved settings")
	}
}

func TestLocalAPIRejectsForeignMutationAndDNSRebinding(t *testing.T) {
	t.Setenv("USAGE_TRAY_DATA_DIR", t.TempDir())
	h := appHTTPHandler()
	for _, kind := range []string{"foreignOrigin", "missingHeader", "foreignHost", "wrongPort", "simpleForm"} {
		r := apiRequest("PUT", "/api/settings", `{"icon_style":"number"}`)
		switch kind {
		case "foreignOrigin":
			r.Header.Set("Origin", "https://example.com")
		case "missingHeader":
			r.Header.Del("X-Usage-Tray")
		case "foreignHost":
			r.Host = "evil.example:47113"
		case "wrongPort":
			r.Host = "127.0.0.1:47114"
		case "simpleForm":
			r.Header.Set("Content-Type", "text/plain")
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code < 400 {
			t.Errorf("%s accepted", kind)
		}
	}
	for _, origin := range []string{"http://127.0.0.1:47113", "http://localhost:47113"} {
		r := apiRequest("PUT", "/api/settings", `{"icon_style":"number"}`)
		r.Header.Set("Origin", origin)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 200 {
			t.Error(origin, w.Code, w.Body.String())
		}
	}
}

func TestStatusNeverExposesProviderResponseSecrets(t *testing.T) {
	now := time.Now()
	u := AgentUsage{Available: true, Error: "401 access_token=secret body=private", FetchedAt: now.Add(-10 * time.Minute)}
	v := statusForAgent("Codex", u, now)
	if v.Status != "auth_error" || v.AgeMinutes != 10 || v.LastSuccess == nil {
		t.Fatalf("status: %+v", v)
	}
	b, _ := json.Marshal(v)
	if bytes.Contains(b, []byte("secret")) || bytes.Contains(b, []byte("private")) {
		t.Fatal("response content leaked")
	}
	u.Error = ""
	u.OK = true
	u.Src = "직전값"
	if statusForAgent("Codex", u, now).Status != "cache" {
		t.Fatal("cache shown as live")
	}
}

func TestActionAPIQueuesAndRejectsUnknownActions(t *testing.T) {
	for len(appActions) > 0 {
		<-appActions
	}
	defer func() {
		for len(appActions) > 0 {
			<-appActions
		}
	}()
	h := appHTTPHandler()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, apiRequest("POST", "/api/actions", `{"action":"refresh"}`))
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
	if action := <-appActions; action != "refresh" {
		t.Fatal(action)
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, apiRequest("POST", "/api/actions", `{"action":"execute"}`))
	if w.Code != 400 {
		t.Fatal(w.Code)
	}
}
