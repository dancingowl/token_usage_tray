package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var appActions = make(chan string, 16)
var collectingRefresh, collectingTokens atomic.Bool
var startupMu sync.Mutex

type agentStatus struct {
	Name        string     `json:"name"`
	Status      string     `json:"status"`
	Label       string     `json:"label"`
	Source      string     `json:"source"`
	LastSuccess *time.Time `json:"last_success"`
	AgeMinutes  int        `json:"age_minutes"`
	Error       string     `json:"error"`
}

func safeAgentError(err string) string {
	low := strings.ToLower(err)
	if err == "" {
		return ""
	}
	if strings.Contains(low, "401") || strings.Contains(low, "403") || strings.Contains(low, "auth") || strings.Contains(low, "credential") || strings.Contains(low, "로그인") {
		return "로그인이 만료됐거나 인증을 확인할 수 없습니다. 해당 에이전트에 다시 로그인해 주세요."
	}
	if strings.Contains(low, "429") {
		return "요청 제한으로 갱신을 기다리고 있습니다."
	}
	return "사용량을 갱신하지 못했습니다. 연결 상태와 에이전트 로그인을 확인해 주세요."
}

func statusForAgent(name string, u AgentUsage, now time.Time) agentStatus {
	v := agentStatus{Name: name, Status: "waiting", Label: "수집 대기", Source: u.Src, AgeMinutes: -1, Error: safeAgentError(u.Error)}
	if !u.FetchedAt.IsZero() {
		at := u.FetchedAt
		v.LastSuccess = &at
		v.AgeMinutes = max(0, int(now.Sub(at).Minutes()))
	}
	if !u.Available {
		v.Label = "연결된 계정 없음"
		return v
	}
	switch {
	case v.Error != "" && strings.Contains(v.Error, "인증"):
		v.Status, v.Label = "auth_error", "로그인 확인 필요"
	case u.Error != "":
		v.Status, v.Label = "error", "수집 실패"
	case u.OK && (u.Src == "live" || u.Src == "api" || u.Src == "local"):
		v.Status, v.Label = "live", "정상 수집"
	case u.OK || u.Src != "":
		v.Status, v.Label = "cache", "기록·캐시 사용"
	}
	return v
}

func sanitizedState(s State) State {
	s.Claude.Error = safeAgentError(s.Claude.Error)
	s.Codex.Error = safeAgentError(s.Codex.Error)
	s.Antigravity.Error = safeAgentError(s.Antigravity.Error)
	return s
}

func appHTTPHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		view, exists := map[string]string{"/": "usage", "/tokens": "tokens", "/settings": "settings"}[r.URL.Path]
		if !exists {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodGet {
			apiError(w, 405, "GET 요청만 지원합니다")
			return
		}
		s := sanitizedState(mergedState())
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, renderPageView(&s, mergedTokens(), view))
	})
	mux.HandleFunc("/state.json", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			apiError(w, 405, "GET 요청만 지원합니다")
			return
		}
		writeJSON(w, sanitizedState(mergedState()))
	})
	mux.HandleFunc("/api/settings", func(w http.ResponseWriter, r *http.Request) {
		s, err := loadTraySettings()
		if err != nil {
			apiError(w, 500, "설정 파일을 읽지 못했습니다")
			return
		}
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, s)
		case http.MethodPut:
			if !allowMutation(w, r) {
				return
			}
			if !readJSON(w, r, &s) {
				return
			}
			if err := validateTraySettings(s); err != nil {
				apiError(w, 400, err.Error())
				return
			}
			if err := saveTraySettings(s); err != nil {
				apiError(w, 500, "설정을 저장하지 못했습니다")
				return
			}
			writeJSON(w, s)
		default:
			apiError(w, 405, "GET 또는 PUT 요청만 지원합니다")
		}
	})
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			apiError(w, 405, "GET 요청만 지원합니다")
			return
		}
		s := mergedState()
		now := time.Now()
		writeJSON(w, map[string]any{"agents": []agentStatus{statusForAgent("Claude", s.Claude, now), statusForAgent("Codex", s.Codex, now), statusForAgent("Antigravity", s.Antigravity, now)}, "refreshing": collectingRefresh.Load(), "scanning": collectingTokens.Load()})
	})
	mux.HandleFunc("/api/actions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			apiError(w, 405, "POST 요청만 지원합니다")
			return
		}
		if !allowMutation(w, r) {
			return
		}
		var body struct {
			Action string `json:"action"`
		}
		if !readJSON(w, r, &body) {
			return
		}
		switch body.Action {
		case "refresh", "detail", "settings":
		default:
			apiError(w, 400, "지원하지 않는 동작입니다")
			return
		}
		select {
		case appActions <- body.Action:
			writeJSON(w, map[string]bool{"ok": true})
		default:
			apiError(w, 409, "앞선 동작을 처리하고 있습니다")
		}
	})
	mux.HandleFunc("/api/history", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			apiError(w, 405, "GET 요청만 지원합니다")
			return
		}
		days := 7
		switch r.URL.Query().Get("days") {
		case "1":
			days = 1
		case "30":
			days = 30
		case "", "7":
		default:
			apiError(w, 400, "기록 범위는 1, 7, 30일입니다")
			return
		}
		rangeEnd := time.Now()
		samples, forecasts, err := usageHistorySnapshot(rangeEnd, days)
		if err != nil {
			logf("history read: %v", err)
		}
		if samples == nil {
			samples = []usageHistorySample{}
		}
		if forecasts == nil {
			forecasts = []usageForecast{}
		}
		writeJSON(w, map[string]any{"samples": samples, "forecasts": forecasts, "range_end": rangeEnd})
	})
	mux.HandleFunc("/api/startup", func(w http.ResponseWriter, r *http.Request) {
		startupMu.Lock()
		defer startupMu.Unlock()
		if r.Method == http.MethodPost {
			if !allowMutation(w, r) {
				return
			}
			var body struct {
				Enabled *bool `json:"enabled"`
			}
			if !readJSON(w, r, &body) {
				return
			}
			if body.Enabled == nil {
				apiError(w, 400, "자동 실행 여부를 지정해 주세요")
				return
			}
			if err := setStartupEnabled(*body.Enabled); err != nil {
				apiError(w, 500, "자동 실행 설정을 바꾸지 못했습니다")
				return
			}
		} else if r.Method != http.MethodGet {
			apiError(w, 405, "GET 또는 POST 요청만 지원합니다")
			return
		}
		enabled, supported := startupEnabled()
		writeJSON(w, map[string]bool{"enabled": enabled, "supported": supported})
	})
	mux.HandleFunc("/api/update", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, currentUpdateStatus())
		case http.MethodPost:
			if !allowMutation(w, r) {
				return
			}
			writeJSON(w, checkForUpdates(r.Context()))
		default:
			apiError(w, 405, "GET 또는 POST 요청만 지원합니다")
		}
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !localHostAllowed(r.Host) {
			apiError(w, 403, "로컬 주소로 접속해 주세요")
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'")
		mux.ServeHTTP(w, r)
	})
}

func localHostAllowed(host string) bool {
	name, port, err := net.SplitHostPort(host)
	if err != nil {
		return false
	}
	_, expected, _ := net.SplitHostPort(listenAddr())
	return port == expected && (name == "127.0.0.1" || name == "localhost" || name == "::1")
}

func allowMutation(w http.ResponseWriter, r *http.Request) bool {
	if r.Header.Get("X-Usage-Tray") != "1" {
		apiError(w, 403, "앱 화면에서 실행해 주세요")
		return false
	}
	if origin := r.Header.Get("Origin"); origin != "" {
		u, err := url.Parse(origin)
		if err != nil || u.Scheme != "http" || !localHostAllowed(u.Host) {
			apiError(w, 403, "외부 페이지의 조작 요청은 허용하지 않습니다")
			return false
		}
	}
	return true
}

func readJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		apiError(w, 415, "JSON 요청이 필요합니다")
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		apiError(w, 400, "설정 요청을 읽지 못했습니다")
		return false
	}
	if err := dec.Decode(new(any)); err != io.EOF {
		apiError(w, 400, "요청에 불필요한 데이터가 있습니다")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(value)
}
func apiError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
