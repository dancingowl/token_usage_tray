package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewerRelease(t *testing.T) {
	for _, tt := range []struct {
		current, latest string
		want            bool
	}{
		{"v1.2.3", "v1.3.0", true}, {"1.2.3", "v1.2.3+build", false},
		{"1.2.3-rc.1", "1.2.3", true}, {"1.2.3", "1.2.3-rc.1", false},
		{"1.2.3-alpha.2", "1.2.3-alpha.10", true}, {"1.2.3-alpha", "1.2.3-alpha.1", true},
		{"1.2.3-1", "1.2.3-alpha", true}, {"1.2.3-beta", "1.2.3-alpha", false},
		{"dev", "9.0.0", false}, {"unknown", "9.0.0", false}, {"1.2", "9.0.0", false},
		{"01.2.3", "9.0.0", false}, {"1.2.3-01", "9.0.0", false}, {"1.2.3", "", false},
		{"999999999999999999999.0.0", "1000000000000000000000.0.0", true},
	} {
		if got := newerRelease(tt.current, tt.latest); got != tt.want {
			t.Errorf("%q -> %q: got %v, want %v", tt.current, tt.latest, got, tt.want)
		}
	}
}

func TestValidReleaseURL(t *testing.T) {
	for _, raw := range []string{
		"http://github.com/dancingowl/token_usage_tray/releases/tag/v1.0.0",
		"https://github.com.evil.com/dancingowl/token_usage_tray/releases/tag/v1.0.0",
		"https://github.com:443/dancingowl/token_usage_tray/releases/tag/v1.0.0",
		"https://someone@github.com/dancingowl/token_usage_tray/releases/tag/v1.0.0",
		"https://github.com/other/repo/releases/tag/v1.0.0",
		"https://github.com/dancingowl/token_usage_tray/releases/../issues",
		"https://github.com/dancingowl/token_usage_tray/releases/%2e%2e/issues",
		"https://github.com/dancingowl/token_usage_tray/releases/tag/v1.0.0?next=evil",
		"https://github.com/dancingowl/token_usage_tray/releases/",
	} {
		if validReleaseURL(raw) {
			t.Errorf("accepted unsafe URL: %s", raw)
		}
	}
	if !validReleaseURL("https://github.com/dancingowl/token_usage_tray/releases/tag/v1.0.0") {
		t.Fatal("rejected release URL")
	}
}

func testUpdateChecker(t *testing.T, handler http.HandlerFunc) *updateChecker {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return &updateChecker{client: srv.Client(), endpoint: srv.URL}
}

const testReleaseJSON = `{"tag_name":"v2.0.0","html_url":"https://github.com/dancingowl/token_usage_tray/releases/tag/v2.0.0","body":"Release notes"}`

func TestUpdateCheckSuccessAndCache(t *testing.T) {
	var calls atomic.Int32
	c := testUpdateChecker(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Header.Get("Authorization") != "" || r.Method != "GET" || r.Header.Get("Accept") != "application/vnd.github+json" {
			t.Error("unexpected request")
		}
		w.Write([]byte(testReleaseJSON))
	})
	if s := c.current("1.0.0"); !s.CheckedAt.IsZero() || calls.Load() != 0 {
		t.Fatal("cached read made a request")
	}
	s := c.check(context.Background(), "1.0.0")
	if s.Error != "" || !s.Available || s.LatestVersion != "v2.0.0" || s.CheckedAt.IsZero() {
		t.Fatalf("unexpected status: %+v", s)
	}
	c.check(context.Background(), "1.0.0")
	if calls.Load() != 1 {
		t.Fatalf("repeat made %d requests", calls.Load())
	}
	if c.current("dev").Available {
		t.Fatal("development build claimed an available update")
	}
}

func TestUpdateCheckErrors(t *testing.T) {
	for _, tt := range []struct {
		name    string
		code    int
		body    string
		headers map[string]string
		want    string
	}{
		{"no release", 404, `{"message":"Not Found"}`, nil, "공개된 최신 릴리스"},
		{"rate limit", 403, `{"message":"secret-token"}`, map[string]string{"X-RateLimit-Remaining": "0"}, "요청 한도"},
		{"secondary limit", 403, "", map[string]string{"Retry-After": "30"}, "요청 한도"},
		{"too many", 429, "", nil, "요청 한도"},
		{"server", 502, "secret-token", nil, "HTTP 502"},
		{"json", 200, "broken", nil, "올바른 공개 릴리스"},
		{"empty release", 200, "{}", nil, "올바른 공개 릴리스"},
		{"unsafe URL", 200, `{"tag_name":"v2.0.0","html_url":"https://evil.com/releases/v2"}`, nil, "올바른 공개 릴리스"},
		{"draft", 200, `{"tag_name":"v2.0.0","html_url":"https://github.com/dancingowl/token_usage_tray/releases/v2","draft":true}`, nil, "올바른 공개 릴리스"},
		{"invalid version", 200, `{"tag_name":"nightly","html_url":"https://github.com/dancingowl/token_usage_tray/releases/nightly"}`, nil, "버전 형식"},
		{"large", 200, strings.Repeat(" ", (1<<20)+1), nil, "너무 큽니다"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			c := testUpdateChecker(t, func(w http.ResponseWriter, r *http.Request) {
				for k, v := range tt.headers {
					w.Header().Set(k, v)
				}
				w.WriteHeader(tt.code)
				w.Write([]byte(tt.body))
			})
			s := c.check(context.Background(), "1.0.0")
			if !strings.Contains(s.Error, tt.want) || strings.Contains(s.Error, "secret-token") || s.Available || s.ReleaseURL != "" {
				t.Fatalf("unexpected error status: %+v", s)
			}
		})
	}
}

func TestUpdateCheckTimeout(t *testing.T) {
	c := testUpdateChecker(t, func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() })
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	start := time.Now()
	s := c.check(ctx, "1.0.0")
	if s.Error == "" || time.Since(start) > time.Second {
		t.Fatalf("timeout not honored: %+v", s)
	}
}

func TestUpdateCheckConcurrent(t *testing.T) {
	var calls atomic.Int32
	entered, release := make(chan struct{}), make(chan struct{})
	c := testUpdateChecker(t, func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			close(entered)
		}
		<-release
		w.Write([]byte(testReleaseJSON))
	})
	var wg sync.WaitGroup
	results := make(chan updateStatus, 10)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); results <- c.check(context.Background(), "1.0.0") }()
	}
	<-entered
	// A cached status read remains immediate while the HTTP request is running.
	read := make(chan struct{})
	go func() { c.current("1.0.0"); close(read) }()
	select {
	case <-read:
	case <-time.After(time.Second):
		t.Error("cached read blocked on request")
	}
	close(release)
	wg.Wait()
	close(results)
	for s := range results {
		if s.Error != "" || !s.Available {
			t.Errorf("unexpected concurrent result: %+v", s)
		}
	}
	if calls.Load() != 1 {
		t.Errorf("made %d requests", calls.Load())
	}
}

func TestUpdateCheckPreservesSuccessAfterFailure(t *testing.T) {
	var fail atomic.Bool
	c := testUpdateChecker(t, func(w http.ResponseWriter, r *http.Request) {
		if fail.Load() {
			w.WriteHeader(503)
			return
		}
		w.Write([]byte(testReleaseJSON))
	})
	first := c.check(context.Background(), "1.0.0")
	fail.Store(true)
	c.mu.Lock()
	c.status.CheckedAt = time.Now().Add(-2 * time.Minute)
	c.mu.Unlock()
	s := c.check(context.Background(), "1.0.0")
	if s.Error == "" || s.ReleaseURL != first.ReleaseURL || s.Notes != first.Notes || !s.Available {
		t.Fatalf("lost last success: %+v", s)
	}
	fail.Store(false)
	c.mu.Lock()
	c.status.CheckedAt = time.Now().Add(-2 * time.Minute)
	c.mu.Unlock()
	if s = c.check(context.Background(), "1.0.0"); s.Error != "" {
		t.Fatalf("failed to clear error: %+v", s)
	}
}

func TestUpdateNotesTruncatedAndJSONSafe(t *testing.T) {
	c := testUpdateChecker(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"tag_name": "v2.0.0", "html_url": "https://github.com/dancingowl/token_usage_tray/releases/tag/v2", "body": strings.Repeat("한", 9000) + "<script>"})
	})
	s := c.check(context.Background(), "dev")
	if s.Error != "" || len([]rune(s.Notes)) > 8100 || !strings.Contains(s.Notes, "전체 내용") {
		t.Fatalf("unexpected notes: %d %s", len([]rune(s.Notes)), s.Error)
	}
	if _, err := json.Marshal(s); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateRateLimitBackoff(t *testing.T) {
	var calls atomic.Int32
	c := testUpdateChecker(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(5*time.Minute).Unix(), 10))
		w.Header().Set("Retry-After", "120")
		w.WriteHeader(403)
	})
	c.check(context.Background(), "1.0.0")
	c.mu.Lock()
	c.status.CheckedAt = time.Now().Add(-2 * time.Minute)
	c.mu.Unlock()
	if s := c.check(context.Background(), "1.0.0"); !strings.Contains(s.Error, "요청 한도") || calls.Load() != 1 {
		t.Fatalf("rate-limit backoff ignored: %+v, %d requests", s, calls.Load())
	}
}

func TestUpdateConcurrentWaiterCancellation(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	c := testUpdateChecker(t, func(w http.ResponseWriter, r *http.Request) {
		close(entered)
		<-release
		w.Write([]byte(testReleaseJSON))
	})
	done := make(chan struct{})
	go func() { c.check(context.Background(), "1.0.0"); close(done) }()
	<-entered
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if s := c.check(ctx, "1.0.0"); !strings.Contains(s.Error, "취소") {
		t.Errorf("waiter cancellation ignored: %+v", s)
	}
	close(release)
	<-done
	if s := c.current("1.0.0"); s.Error != "" || !s.Available {
		t.Fatalf("waiter cancellation corrupted shared result: %+v", s)
	}
}

func TestUpdateRejectsRedirect(t *testing.T) {
	var redirected atomic.Int32
	destination := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { redirected.Add(1); w.Write([]byte(testReleaseJSON)) }))
	defer destination.Close()
	c := testUpdateChecker(t, func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, destination.URL, 302) })
	c.client.CheckRedirect = releaseChecker.client.CheckRedirect
	if s := c.check(context.Background(), "1.0.0"); s.Error == "" || redirected.Load() != 0 {
		t.Fatalf("followed redirect: %+v", s)
	}
}
