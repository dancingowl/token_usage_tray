package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Optional fixture export for scripts/test-credit-ui.cjs; it never calls a provider.
func TestWriteCreditUIFixtures(t *testing.T) {
	dir := os.Getenv("CREDIT_UI_FIXTURES_DIR")
	if dir == "" {
		t.Skip("isolated UI fixture generation only")
	}
	now := time.Now().Truncate(time.Second)
	ptr := func(n float64) *float64 { return &n }
	for _, scenario := range []string{"normal", "zero", "unknown", "stale", "unlimited", "multiple"} {
		c := &CreditBalance{Balance: ptr(60087.0958375), CheckedAt: now, Account: "fixture-account"}
		switch scenario {
		case "zero":
			c.Balance = ptr(0)
		case "unknown":
			c.Balance = nil
		case "stale":
			c.CheckedAt = now.Add(-time.Hour)
		case "unlimited":
			c.Balance = nil
			c.Unlimited = true
		}
		s := State{Updated: now, Codex: emptyAgent()}
		s.Codex.Available, s.Codex.OK, s.Codex.Src, s.Codex.FetchedAt = true, true, "live", now
		s.Codex.Plan, s.Codex.Week, s.Codex.Credits = "pro", Window{Left: 65, ResetAt: now.Add(6 * 24 * time.Hour).Unix(), WindowS: 7 * 86400}, c
		if scenario == "multiple" {
			s.Codex.Short = Window{Left: 90, ResetAt: now.Add(3 * time.Hour).Unix()}
			s.Claude = emptyAgent()
			s.Claude.Available, s.Claude.Plan, s.Claude.Src, s.Claude.AgeMin = true, "pro", "cache", 12
			s.Claude.Week = Window{Left: 80, ResetAt: now.Add(5 * 24 * time.Hour).Unix()}
		}
		if err := os.WriteFile(filepath.Join(dir, scenario+".html"), []byte(renderPageView(&s, nil, "usage")), 0600); err != nil {
			t.Fatal(err)
		}
		if scenario == "normal" {
			if err := os.WriteFile(filepath.Join(dir, "summary.html"), []byte(renderPageView(&s, nil, "usage")), 0600); err != nil {
				t.Fatal(err)
			}
		}
	}
	samples := []usageHistorySample{}
	for i, value := range []float64{60200, 60100, 60087.0958375} {
		at := now.Add(time.Duration(i-2) * 5 * time.Minute)
		samples = append(samples, usageHistorySample{At: at, Agent: "codex", Window: "credits", Balance: ptr(value), CreditAccount: "fixture-account", Source: "live"}, usageHistorySample{At: at, Agent: "codex", Window: "week", Left: 75 - i*5, WindowS: 7 * 86400, ResetAt: now.Add(6 * 24 * time.Hour).Unix(), Source: "live"})
	}
	data := map[string]any{"samples": samples, "forecasts": []usageForecast{}, "range_end": now}
	b, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "history.json"), b, 0600); err != nil {
		t.Fatal(err)
	}
}
