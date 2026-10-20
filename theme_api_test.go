package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestThemeAPIOnlyUpdatesTheme(t *testing.T) {
	t.Setenv("USAGE_TRAY_DATA_DIR", t.TempDir())
	want := defaultTraySettings()
	want.IconStyle = iconStyleNumber
	want.DisplayWindow = "urgent"
	want.VisibleAgents.Claude = false
	want.PollSeconds = 600
	want.AlertsEnabled = false
	want.NotifyLimits = false
	want.AlertsPausedUntil = 2000000000
	want.AlertRules["codex5h"] = alertRule{false, 35, false}
	if err := saveTraySettings(want); err != nil {
		t.Fatal(err)
	}
	h := appHTTPHandler()
	for _, theme := range []string{"dark", "light", "system"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, apiRequest(http.MethodPut, "/api/theme", `{"theme":"`+theme+`"}`))
		if w.Code != http.StatusOK || strings.TrimSpace(w.Body.String()) != `{"theme":"`+theme+`"}` {
			t.Fatalf("PUT theme %s: %d %s", theme, w.Code, w.Body.String())
		}
		want.Theme = theme
		got, err := loadTraySettings()
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("theme update changed other settings: got %+v, want %+v; %v", got, want, err)
		}
		w = httptest.NewRecorder()
		h.ServeHTTP(w, apiRequest(http.MethodGet, "/api/theme", ""))
		if w.Code != http.StatusOK || strings.TrimSpace(w.Body.String()) != `{"theme":"`+theme+`"}` {
			t.Fatalf("GET theme %s: %d %s", theme, w.Code, w.Body.String())
		}
	}
}

func TestThemeAPIDefaultAndValidation(t *testing.T) {
	t.Setenv("USAGE_TRAY_DATA_DIR", t.TempDir())
	h := appHTTPHandler()
	for _, persisted := range []string{"", `{"theme":""}`} {
		if persisted != "" {
			if err := os.WriteFile(settingsPath(), []byte(persisted), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, apiRequest(http.MethodGet, "/api/theme", ""))
		if w.Code != http.StatusOK || strings.TrimSpace(w.Body.String()) != `{"theme":"system"}` {
			t.Fatalf("default theme: %d %s", w.Code, w.Body.String())
		}
	}
	for _, body := range []string{`{}`, `null`, `{"theme":""}`, `{"theme":null}`, `{"theme":"invalid"}`, `{"theme":"Dark"}`, `{"theme":true}`, `{"theme":"dark","poll_seconds":300}`, `{"theme":"dark"} {}`, strings.Repeat("a", 70<<10)} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, apiRequest(http.MethodPut, "/api/theme", body))
		if w.Code != http.StatusBadRequest {
			t.Errorf("invalid theme body %q: %d", body[:min(len(body), 100)], w.Code)
		}
	}
	for _, method := range []string{http.MethodPost, http.MethodDelete, http.MethodHead, http.MethodPatch} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, apiRequest(method, "/api/theme", `{"theme":"dark"}`))
		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s theme: %d", method, w.Code)
		}
	}
}

func TestThemeAPIRejectsForeignMutation(t *testing.T) {
	t.Setenv("USAGE_TRAY_DATA_DIR", t.TempDir())
	h := appHTTPHandler()
	for _, kind := range []string{"origin", "header", "host", "port", "contentType"} {
		r := apiRequest(http.MethodPut, "/api/theme", `{"theme":"dark"}`)
		switch kind {
		case "origin":
			r.Header.Set("Origin", "https://example.com")
		case "header":
			r.Header.Del("X-Usage-Tray")
		case "host":
			r.Host = "evil.example:47113"
		case "port":
			r.Host = "127.0.0.1:47114"
		case "contentType":
			r.Header.Set("Content-Type", "text/plain")
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != http.StatusForbidden && !(kind == "contentType" && w.Code == http.StatusUnsupportedMediaType) {
			t.Errorf("%s mutation: %d %s", kind, w.Code, w.Body.String())
		}
	}
	got, err := loadTraySettings()
	if err != nil || got.Theme != "system" {
		t.Fatalf("rejected mutation changed theme: %+v %v", got, err)
	}
}

func TestSettingsAPIPreservesLatestTheme(t *testing.T) {
	t.Setenv("USAGE_TRAY_DATA_DIR", t.TempDir())
	h := appHTTPHandler()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, apiRequest(http.MethodPut, "/api/theme", `{"theme":"dark"}`))
	if w.Code != http.StatusOK {
		t.Fatal(w.Code, w.Body.String())
	}
	for _, staleTheme := range []string{"light", "system", "", "invalid", "omitted"} {
		body := map[string]any{"poll_seconds": 300, "alerts_paused_until": 2000000000}
		if staleTheme != "omitted" {
			body["theme"] = staleTheme
		}
		encoded, _ := json.Marshal(body)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, apiRequest(http.MethodPut, "/api/settings", string(encoded)))
		got, err := loadTraySettings()
		if w.Code != http.StatusOK || err != nil || got.Theme != "dark" || got.PollSeconds != 300 || got.AlertsPausedUntil != 2000000000 {
			t.Fatalf("stale %q settings save: %d %+v %v", staleTheme, w.Code, got, err)
		}
		var response traySettings
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil || response.Theme != "dark" {
			t.Fatalf("settings response omitted saved theme: %+v %v", response, err)
		}
	}
}

func TestThemeAPIReportsSettingsReadFailure(t *testing.T) {
	t.Setenv("USAGE_TRAY_DATA_DIR", t.TempDir())
	if err := os.WriteFile(settingsPath(), []byte(`{"theme":`), 0o600); err != nil {
		t.Fatal(err)
	}
	h := appHTTPHandler()
	for _, path := range []string{"/api/theme", "/api/settings"} {
		for _, method := range []string{http.MethodGet, http.MethodPut} {
			w := httptest.NewRecorder()
			h.ServeHTTP(w, apiRequest(method, path, `{"theme":"dark"}`))
			if w.Code != http.StatusInternalServerError {
				t.Errorf("%s %s with unreadable settings: %d", method, path, w.Code)
			}
		}
	}
}

type themeBodyBeforeRead struct {
	io.Reader
	before func()
}

func (r *themeBodyBeforeRead) Read(p []byte) (int, error) {
	if r.before != nil {
		before := r.before
		r.before = nil
		before()
	}
	return r.Reader.Read(p)
}

func TestThemeAPIReportsSettingsSaveFailure(t *testing.T) {
	t.Setenv("USAGE_TRAY_DATA_DIR", t.TempDir())
	r := apiRequest(http.MethodPut, "/api/theme", "")
	// Loading a missing settings file succeeds; replacing it with a directory
	// during body decoding makes the subsequent atomic rename fail on all OSes.
	r.Body = io.NopCloser(&themeBodyBeforeRead{Reader: strings.NewReader(`{"theme":"dark"}`), before: func() {
		if err := os.Mkdir(settingsPath(), 0o700); err != nil {
			t.Fatal(err)
		}
	}})
	w := httptest.NewRecorder()
	appHTTPHandler().ServeHTTP(w, r)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("save failure: %d %s", w.Code, w.Body.String())
	}
}

func TestThemeAndSettingsAPIUpdatesAreSerialized(t *testing.T) {
	for _, firstPath := range []string{"/api/theme", "/api/settings"} {
		t.Run(firstPath, func(t *testing.T) {
			t.Setenv("USAGE_TRAY_DATA_DIR", t.TempDir())
			bodies := map[string]string{"/api/theme": `{"theme":"dark"}`, "/api/settings": `{"poll_seconds":300,"theme":"light"}`}
			secondPath := "/api/settings"
			if firstPath == secondPath {
				secondPath = "/api/theme"
			}
			h := appHTTPHandler()
			reading, release := make(chan struct{}), make(chan struct{})
			first := apiRequest(http.MethodPut, firstPath, "")
			first.Body = io.NopCloser(&themeBodyBeforeRead{Reader: strings.NewReader(bodies[firstPath]), before: func() { close(reading); <-release }})
			firstResult, secondResult := make(chan *httptest.ResponseRecorder, 1), make(chan *httptest.ResponseRecorder, 1)
			go func() { w := httptest.NewRecorder(); h.ServeHTTP(w, first); firstResult <- w }()
			<-reading
			go func() {
				w := httptest.NewRecorder()
				h.ServeHTTP(w, apiRequest(http.MethodPut, secondPath, bodies[secondPath]))
				secondResult <- w
			}()
			var second *httptest.ResponseRecorder
			select {
			case second = <-secondResult:
				t.Error("second settings mutation completed while first mutation was decoding")
			case <-time.After(100 * time.Millisecond):
			}
			close(release)
			firstResponse := <-firstResult
			if second == nil {
				second = <-secondResult
			}
			if firstResponse.Code != http.StatusOK || second.Code != http.StatusOK {
				t.Fatalf("concurrent updates failed: %d %s; %d %s", firstResponse.Code, firstResponse.Body.String(), second.Code, second.Body.String())
			}
			got, err := loadTraySettings()
			if err != nil || got.Theme != "dark" || got.PollSeconds != 300 {
				t.Fatalf("concurrent update lost changes: %+v %v", got, err)
			}
		})
	}
}
