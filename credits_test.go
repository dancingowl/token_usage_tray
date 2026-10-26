package main

import (
	"encoding/json"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func creditNumber(value float64) *float64 { return &value }

func TestCreditBalanceParsingIndependentFromQuota(t *testing.T) {
	now := time.Now()
	for _, tt := range []struct {
		name, raw string
		balance   *float64
		unlimited bool
	}{
		{"numeric", `{"balance":60087.0958375000}`, creditNumber(60087.0958375), false},
		{"string", `{"balance":"60087.0958375000","unlimited":false}`, creditNumber(60087.0958375), false},
		{"zero", `{"balance":"0"}`, creditNumber(0), false},
		{"fraction", `{"balance":0.125}`, creditNumber(.125), false},
		{"unlimited", `{"unlimited":true}`, nil, true},
		{"missing balance", `{}`, nil, false},
		{"null", `null`, nil, false},
		{"array", `[]`, nil, false},
		{"scalar", `false`, nil, false},
		{"bad balance", `{"balance":{"value":10}}`, nil, false},
		{"negative", `{"balance":-1}`, nil, false},
		{"nan", `{"balance":"NaN"}`, nil, false},
		{"infinity", `{"balance":"Infinity"}`, nil, false},
		{"overflow", `{"balance":"1e400"}`, nil, false},
		{"numeric overflow", `{"balance":1e400}`, nil, false},
		{"bad unlimited", `{"balance":10,"unlimited":"yes"}`, nil, false},
		{"null unlimited", `{"balance":10,"unlimited":null}`, nil, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var response codexUsageResp
			if err := json.Unmarshal([]byte(`{"credits":`+tt.raw+`,"rate_limit":{"primary_window":{"used_percent":18}}}`), &response); err != nil {
				t.Fatal("optional credits rejected quotas", err)
			}
			if response.RateLimit.Primary == nil || *response.RateLimit.Primary.UsedPercent != 18 {
				t.Fatal("quota lost")
			}
			c := parseCreditBalance(response.Credits, "synthetic-account", now)
			if c.CheckedAt != now || c.Account == "synthetic-account" || len(c.Account) != 64 {
				t.Fatal("observation metadata")
			}
			if c.Unlimited != tt.unlimited || (c.Balance == nil) != (tt.balance == nil) {
				t.Fatalf("unexpected balance: %+v", c)
			}
			if c.Balance != nil && *c.Balance != *tt.balance {
				t.Fatal("balance changed")
			}
		})
	}
}

func TestCreditBalanceDisplayAndFreshness(t *testing.T) {
	now := time.Now()
	for _, tt := range []struct {
		value float64
		want  string
	}{
		{0, "0.00"}, {.125, "0.13"}, {1.005, "1.01"}, {2.675, "2.68"}, {999.995, "1,000.00"},
		{60087.0958375, "60,087.10"}, {1234567890.5, "1,234,567,890.50"},
	} {
		if got := formatCreditBalance(tt.value); got != tt.want {
			t.Fatalf("%v: %q != %q", tt.value, got, tt.want)
		}
	}
	if got := formatCreditBalance(math.MaxFloat64); got == "" || strings.Contains(got, "Inf") {
		t.Fatal("finite maximum overflowed")
	}
	for _, value := range []float64{-1, math.NaN(), math.Inf(1)} {
		if formatCreditBalance(value) != "" {
			t.Fatal("invalid balance formatted")
		}
	}
	u := AgentUsage{Credits: &CreditBalance{Balance: creditNumber(60087.0958375), CheckedAt: now}}
	if creditBalanceTextAt(u, now) != "60,087.10 크레딧" {
		t.Fatal("balance text")
	}
	if creditBalanceTextAt(u, now.Add(historyFreshness)) == "확인 불가" {
		t.Fatal("freshness boundary")
	}
	if creditBalanceTextAt(u, now.Add(historyFreshness+time.Nanosecond)) != "확인 불가" {
		t.Fatal("stale balance displayed")
	}
	if creditBalanceTextAt(u, now.Add(-time.Nanosecond)) != "확인 불가" {
		t.Fatal("future observation displayed")
	}
	u.Credits.Unlimited = true
	if creditBalanceTextAt(u, now) != "무제한" {
		t.Fatal("unlimited")
	}
	u.Credits = &CreditBalance{CheckedAt: now}
	if creditBalanceTextAt(u, now) != "확인 불가" {
		t.Fatal("missing balance fabricated")
	}
	if creditBalanceTextAt(AgentUsage{}, now) != "확인 불가" {
		t.Fatal("never checked balance fabricated")
	}
}

func TestLastGoodDoesNotRestoreCreditBalanceAfterFailure(t *testing.T) {
	now := time.Now()
	prev := emptyAgent()
	prev.OK, prev.Short.Left, prev.FetchedAt = true, 64, now.Add(-time.Minute)
	prev.Credits = &CreditBalance{CheckedAt: prev.FetchedAt, Balance: creditNumber(100)}
	cur := emptyAgent()
	cur.Credits = &CreditBalance{CheckedAt: now, Account: "fingerprint"}
	got := keepLastGood(cur, prev, now)
	if !got.OK || got.Short.Left != 64 || got.Credits != cur.Credits || creditBalanceTextAt(got, now) != "확인 불가" {
		t.Fatal("fallback hid credit failure")
	}
	skipped := cachedUsage(prev)
	if skipped.Credits != prev.Credits || !skipped.Credits.CheckedAt.Equal(prev.Credits.CheckedAt) {
		t.Fatal("poll skip changed credit observation")
	}
}

type creditTestTransport func(*http.Request) (*http.Response, error)

func (f creditTestTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestFetchCodexCreditsFailureAndRetry(t *testing.T) {
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".codex"), 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(home, ".codex", "auth.json")
	if err := os.WriteFile(path, []byte(`{"tokens":{"access_token":"synthetic-token","account_id":"synthetic-account"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	old := httpClient
	t.Cleanup(func() { httpClient = old })
	calls, raw, status := 0, `{"balance":"25.125"}`, 200
	quota := `{"primary_window":{"used_percent":20,"limit_window_seconds":18000}}`
	httpClient = &http.Client{Transport: creditTestTransport(func(*http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(`{"credits":` + raw + `,"rate_limit":` + quota + `}`)), Header: make(http.Header)}, nil
	})}
	first := fetchCodex()
	if !first.OK || first.Credits.Balance == nil || *first.Credits.Balance != 25.125 || calls != 1 {
		t.Fatal("credits not fetched with quotas")
	}
	raw = `{"balance":false}`
	failed := fetchCodex()
	if !failed.OK || failed.Credits == nil || failed.Credits.Balance != nil || failed.Credits.CheckedAt.Before(first.Credits.CheckedAt) || calls != 2 {
		t.Fatal("malformed credits affected quota")
	}
	raw = `null`
	missing := fetchCodex()
	if !missing.OK || missing.Credits.Balance != nil {
		t.Fatal("missing credit payload invented balance")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	noAuth := fetchCodex()
	if noAuth.Credits == nil || noAuth.Credits.CheckedAt.IsZero() || noAuth.Credits.Balance != nil || calls != 3 {
		t.Fatal("missing credentials did not record failed check")
	}
	if err := os.WriteFile(path, []byte(`{"tokens":{"access_token":"synthetic-token","account_id":"synthetic-account"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	raw = `{"balance":0}`
	retry := fetchCodex()
	if !retry.OK || retry.Credits.Balance == nil || *retry.Credits.Balance != 0 || calls != 4 {
		t.Fatal("retry did not recover zero balance")
	}
	t.Setenv("PATH", "") // Force curl unavailable so the failed HTTP check stays entirely mocked.
	status = 503
	offline := fetchCodex()
	if offline.Credits == nil || offline.Credits.CheckedAt.IsZero() || offline.Credits.Balance != nil || calls != 5 {
		t.Fatal("failed API check did not record a credit gap")
	}
	status = 200
	recovered := fetchCodex()
	if recovered.Credits.Balance == nil || *recovered.Credits.Balance != 0 || calls != 6 {
		t.Fatal("API retry did not recover")
	}
	quota = `null`
	creditOnly := fetchCodex()
	if !creditOnly.OK || creditOnly.Short.Left != -1 || creditOnly.Week.Left != -1 || creditOnly.Credits.Balance == nil || *creditOnly.Credits.Balance != 0 {
		t.Fatal("a valid credit-only response must succeed without inventing quotas")
	}
	raw = `{"unlimited":true}`
	if unlimited := fetchCodex(); !unlimited.OK || !unlimited.Credits.Unlimited {
		t.Fatal("unlimited credit-only response must succeed")
	}
	raw = `null`
	if unavailable := fetchCodex(); unavailable.OK {
		t.Fatal("missing quotas and credits must not appear successful")
	}
}

func TestCreditHistoryPreservesGapsAccountsAndQuotaCompatibility(t *testing.T) {
	isolateUsageHistory(t)
	now := time.Now().Add(-time.Minute)
	for i, c := range []*CreditBalance{
		{CheckedAt: now, Balance: creditNumber(10.125), Account: "a"},
		{CheckedAt: now, Balance: creditNumber(20), Account: "b"},
		{CheckedAt: now.Add(time.Second), Account: "a"},
		{CheckedAt: now.Add(2 * time.Second), Balance: creditNumber(0), Account: "a"},
		{CheckedAt: now.Add(3 * time.Second), Unlimited: true, Account: "a"},
	} {
		s := &State{Codex: AgentUsage{Credits: c}}
		if err := recordUsageHistory(s); err != nil {
			t.Fatal(i, err)
		}
		if err := recordUsageHistory(s); err != nil {
			t.Fatal("same cached event", err)
		}
	}
	samples, err := readUsageHistory(historyPath())
	if err != nil {
		t.Fatal(err)
	}
	if len(samples) != 5 {
		t.Fatalf("accounts or gaps lost / polling duplicated: %+v", samples)
	}
	for i, s := range samples {
		if s.Window != "credits" || s.Agent != "codex" {
			t.Fatal("wrong credit stream")
		}
		if (i == 2 || i == 4) && (s.Source != "unavailable" || s.Balance != nil) {
			t.Fatal("gap replaced with balance")
		}
	}
	if samples[3].Balance == nil || *samples[3].Balance != 0 {
		t.Fatal("zero balance omitted")
	}
	if forecasts := forecastUsageHistory(samples, time.Now()); len(forecasts) != 0 {
		t.Fatal("credits entered quota forecast")
	}
	quota := historySamples(time.Now())
	merged := append(samples, quota...)
	if forecasts := forecastUsageHistory(merged, time.Now()); len(forecasts) != 1 || forecasts[0].Samples != 3 {
		t.Fatal("credits altered quota history")
	}
	var legacy []usageHistorySample
	if err := json.Unmarshal([]byte(`[{"at":"2026-10-26T00:00:00Z","agent":"codex","window":"week","left":25,"source":"live"}]`), &legacy); err != nil || legacy[0].Balance != nil {
		t.Fatal("legacy history compatibility")
	}
}
